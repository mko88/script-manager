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

type DetailsDTO struct {
	Html          string   `json:"html"`
	CopyValues    []string `json:"copyValues"`
	CopyMasked    []bool   `json:"copyMasked"`
	MissingFields []string `json:"missingFields"`
}

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
			if idx < len(copyValues) && strings.Contains(copyValues[idx], "\n") {
				titleAttr = ` title="` + stdhtml.EscapeString(copyValues[idx]) + `"`
			}
		}
		return `<code class="` + cls + `"` + titleAttr + ` data-copy-idx="` + strconv.Itoa(idx) + `">` + inner + `</code>`
	})

	return DetailsDTO{Html: htmlOut, CopyValues: copyValues, CopyMasked: copyMasked, MissingFields: missing}
}
