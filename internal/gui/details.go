package gui

import (
	"bytes"
	stdhtml "html"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"script-manager/internal/config"
	"script-manager/internal/render"
)

// DetailsDTO is the rendered details pane: HTML plus the copyable values
// found in the source template, in the order they appear. MissingFields
// lists template fields the item lacks (rendered as <nil> in the HTML); the
// frontend shows them in a pinned warning bar rather than inline markdown.
type DetailsDTO struct {
	Html          string   `json:"html"`
	CopyValues    []string `json:"copyValues"`
	CopyMasked    []bool   `json:"copyMasked"`
	MissingFields []string `json:"missingFields"`
}

// codeTagRe matches a single inline <code>...</code> element as emitted by
// goldmark for a backtick span. Code fences render as <pre><code>, which this
// intentionally does not match since fenced blocks aren't used as copy targets.
var codeTagRe = regexp.MustCompile(`<code>(.*?)</code>`)

func (a *App) GetItemDetails(itemIndex int) DetailsDTO {
	item := a.itemAt(itemIndex)
	if item == nil {
		return DetailsDTO{}
	}
	merged := a.mergedItem(item)
	d := config.FindDisplay(a.cfg.Display, merged)
	funcMap := template.FuncMap{"mask": render.MaskFunc}
	tmpl, err := template.New("detail").Funcs(funcMap).Parse(d.Details)
	if err != nil {
		return DetailsDTO{Html: "<pre>details template error: " + err.Error() + "</pre>"}
	}
	data, missing := render.FillMissingFields(tmpl, merged)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return DetailsDTO{Html: "<pre>details template error: " + err.Error() + "</pre>"}
	}
	expanded := render.ExpandAllEnv(buf.String(), merged)
	expanded = render.ExpandConfigFile(expanded, a.cfg.SourcePath)

	displayMd, copyValues, copyMasked := render.ProcessMaskSpans(expanded)

	var htmlBuf bytes.Buffer
	if err := a.md.Convert([]byte(displayMd), &htmlBuf); err != nil {
		return DetailsDTO{Html: "<pre>" + strings.TrimSpace(displayMd) + "</pre>", MissingFields: missing}
	}

	idx := -1
	htmlOut := codeTagRe.ReplaceAllStringFunc(htmlBuf.String(), func(match string) string {
		idx++
		sub := codeTagRe.FindStringSubmatch(match)
		inner := sub[1]
		masked := idx < len(copyMasked) && copyMasked[idx]
		cls := "copy-value"
		titleAttr := ""
		if masked {
			cls += " copy-value-masked"
			// A genuine secret must never surface on hover, but a value
			// masked only because it spans multiple lines isn't sensitive —
			// just too long to inline — so it's safe to preview in full via
			// the native title tooltip.
			if idx < len(copyValues) && strings.Contains(copyValues[idx], "\n") {
				titleAttr = ` title="` + stdhtml.EscapeString(copyValues[idx]) + `"`
			}
		}
		return `<code class="` + cls + `"` + titleAttr + ` data-copy-idx="` + strconv.Itoa(idx) + `">` + inner + `</code>`
	})

	return DetailsDTO{Html: htmlOut, CopyValues: copyValues, CopyMasked: copyMasked, MissingFields: missing}
}
