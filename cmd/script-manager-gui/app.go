package main

import (
	"bytes"
	"context"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"script-manager/internal/config"
	"script-manager/internal/render"

	"github.com/atotto/clipboard"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// App is the Wails-bound backend. Every exported method becomes a callable
// binding on the frontend.
type App struct {
	ctx context.Context
	cfg *config.Config
	md  goldmark.Markdown
}

func NewApp() *App {
	return &App{
		cfg: config.Load(),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ReloadConfig re-reads config.yaml (or config-win.yaml) from disk. On
// failure — a missing file or a YAML syntax error — the previously loaded
// config is kept and an error is returned so the frontend can surface it
// without losing the current view.
func (a *App) ReloadConfig() error {
	cfg, err := config.LoadWithError()
	if err != nil {
		return err
	}
	a.cfg = cfg
	return nil
}

// TitlesDTO mirrors config.TitlesConfig for the frontend pane headers.
type TitlesDTO struct {
	Items   string `json:"items"`
	Actions string `json:"actions"`
	Details string `json:"details"`
}

func (a *App) GetTitles() TitlesDTO {
	return TitlesDTO{
		Items:   orDefault(a.cfg.Titles.Items, "Items"),
		Actions: orDefault(a.cfg.Titles.Actions, "Actions"),
		Details: orDefault(a.cfg.Titles.Details, "Details"),
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ItemDTO is a row in the item list.
type ItemDTO struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

func (a *App) GetItems() []ItemDTO {
	items := make([]ItemDTO, len(a.cfg.Items))
	for i, item := range a.cfg.Items {
		items[i] = ItemDTO{Index: i, Label: a.renderListLabel(item)}
	}
	return items
}

func (a *App) renderListLabel(item map[string]any) string {
	d := config.FindDisplay(a.cfg.Display, item)
	tmpl, err := template.New("list").Parse(d.List)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, item)
	return buf.String()
}

// mergedItem returns a copy of the item with global env vars as defaults.
// Item-level keys always win over globals.
func (a *App) mergedItem(item map[string]any) map[string]any {
	merged := make(map[string]any, len(a.cfg.Env)+len(item))
	maps.Copy(merged, a.cfg.Env)
	maps.Copy(merged, item)
	return merged
}

func (a *App) itemAt(index int) map[string]any {
	if index < 0 || index >= len(a.cfg.Items) {
		return nil
	}
	return a.cfg.Items[index]
}

// ActionDTO is a row in the actions list for the selected item.
type ActionDTO struct {
	Index  int      `json:"index"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Groups []string `json:"groups"`
}

// GetActions returns the actions available for the item, in the same order
// GetActionDetail expects to receive back via ActionDTO.Index. Action IDs are
// optional (customActions rarely set one) so the index, not the ID, is the
// only reliable way to address a specific action.
func (a *App) GetActions(itemIndex int) []ActionDTO {
	item := a.itemAt(itemIndex)
	actions := config.ActionsForItem(a.cfg.Actions, item)
	out := make([]ActionDTO, len(actions))
	for i, act := range actions {
		out[i] = ActionDTO{Index: i, ID: act.ID, Title: act.Title, Groups: act.Groups}
	}
	return out
}

// ActionDetailDTO carries the expanded (but not yet run) command preview for
// the selected item/action pair.
type ActionDetailDTO struct {
	Description string `json:"description"`
	Cmd         string `json:"cmd"`
	NoWait      bool   `json:"noWait"`
}

func (a *App) GetActionDetail(itemIndex, actionIndex int) ActionDetailDTO {
	item := a.itemAt(itemIndex)
	if item == nil {
		return ActionDetailDTO{}
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return ActionDetailDTO{}
	}
	action := actions[actionIndex]
	merged := a.mergedItem(item)
	return ActionDetailDTO{
		Description: expandTemplate(action.Description, merged),
		Cmd:         expandTemplate(action.Cmd, merged),
		NoWait:      action.NoWait,
	}
}

func expandTemplate(src string, data map[string]any) string {
	if src == "" {
		return ""
	}
	tmpl, err := template.New("t").Parse(src)
	if err != nil {
		return src
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return src
	}
	return buf.String()
}

// DetailsDTO is the rendered details pane: HTML plus the copyable values
// found in the source template, in the order they appear.
type DetailsDTO struct {
	Html       string   `json:"html"`
	CopyValues []string `json:"copyValues"`
	CopyMasked []bool   `json:"copyMasked"`
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
		return DetailsDTO{}
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, merged)

	displayMd, copyValues, copyMasked := render.ProcessMaskSpans(buf.String())

	var htmlBuf bytes.Buffer
	if err := a.md.Convert([]byte(displayMd), &htmlBuf); err != nil {
		return DetailsDTO{Html: "<pre>" + strings.TrimSpace(displayMd) + "</pre>"}
	}

	idx := -1
	htmlOut := codeTagRe.ReplaceAllStringFunc(htmlBuf.String(), func(match string) string {
		idx++
		sub := codeTagRe.FindStringSubmatch(match)
		inner := sub[1]
		masked := idx < len(copyMasked) && copyMasked[idx]
		cls := "copy-value"
		if masked {
			cls += " copy-value-masked"
		}
		return `<code class="` + cls + `" data-copy-idx="` + strconv.Itoa(idx) + `">` + inner + `</code>`
	})

	return DetailsDTO{Html: htmlOut, CopyValues: copyValues, CopyMasked: copyMasked}
}

// CopyToClipboard writes value to the system clipboard.
func (a *App) CopyToClipboard(value string) error {
	return clipboard.WriteAll(value)
}
