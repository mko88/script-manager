package ui

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"script-manager/internal/config"
	"script-manager/internal/render"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var detailContentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

var brSplitRe = regexp.MustCompile(`(?i)\s*<br\s*/?>\s*\n?`)

const hlSentinel = "XXGLHIGHLIGHTXX"

func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }
func strPtr(s string) *string { return &s }

func newGlamourRenderer(width int) (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithStyles(gansi.StyleConfig{
			Document: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: strPtr("252"),
				},
				Margin: uintPtr(0),
			},
			Heading: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					BlockSuffix: "\n",
					Color:       strPtr("12"),
					Bold:        boolPtr(true),
				},
			},
			H1: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: strPtr("12"),
					Bold:  boolPtr(true),
				},
			},
			H2: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: strPtr("14"),
					Bold:  boolPtr(true),
				},
			},
			H3: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: strPtr("10"),
					Bold:  boolPtr(true),
				},
			},
			H4: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Bold: boolPtr(true),
				},
			},
			Strong: gansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: strPtr("255"),
			},
			Emph: gansi.StylePrimitive{
				Italic: boolPtr(true),
				Color:  strPtr("245"),
			},
			Code: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: strPtr("6"),
				},
			},
			CodeBlock: gansi.StyleCodeBlock{
				StyleBlock: gansi.StyleBlock{
					StylePrimitive: gansi.StylePrimitive{
						Color: strPtr("252"),
					},
					Margin: uintPtr(1),
				},
			},
			HorizontalRule: gansi.StylePrimitive{
				Color:  strPtr("240"),
				Format: "\n--------\n",
			},
			Item: gansi.StylePrimitive{
				BlockPrefix: "• ",
			},
			Enumeration: gansi.StylePrimitive{
				BlockPrefix: ". ",
			},
			List: gansi.StyleList{
				LevelIndent: 2,
			},
			BlockQuote: gansi.StyleBlock{
				Indent:      uintPtr(1),
				IndentToken: strPtr("│ "),
			},
			Strikethrough: gansi.StylePrimitive{
				CrossedOut: boolPtr(true),
			},
		}),
		glamour.WithWordWrap(width),
	)
}

func glamourRender(r *glamour.TermRenderer, content string) (string, error) {
	parts := brSplitRe.Split(content, -1)
	if len(parts) == 1 {
		return r.Render(content)
	}
	var segments []string
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			segments = append(segments, "")
			continue
		}
		rendered, err := r.Render(part)
		if err != nil {
			return "", err
		}
		segments = append(segments, strings.Trim(rendered, "\n"))
	}
	return strings.Join(segments, "\n") + "\n", nil
}

type DescriptionTile struct {
	*tl.BaseTile
	scrollableContent
	item          map[string]any
	displays      []config.DisplayConfig
	tmpls         map[string]*template.Template
	configPath    string
	title         string
	copyValues    []string
	copyMasked    []bool
	displayMd     string
	copyValuesSet bool
	copyIdx       int
	copyMode      bool
	renderer      *glamour.TermRenderer
	rendererWidth int
}

func newDescriptionTile(displays []config.DisplayConfig) *DescriptionTile {
	t := &DescriptionTile{
		BaseTile: &tl.BaseTile{
			Name: "description",
			Size: tl.Size{Weight: 1},
		},
		title: "Details",
	}
	t.SetDisplays(displays)
	return t
}

func (t *DescriptionTile) SetDisplays(displays config.DisplayList) {
	funcMap := template.FuncMap{"mask": render.MaskFunc}
	tmpls := make(map[string]*template.Template, len(displays))
	for _, d := range displays {
		tmpl, _ := template.New("detail").Funcs(funcMap).Parse(d.Details)
		tmpls[d.Name] = tmpl
	}
	t.displays = displays
	t.tmpls = tmpls
}

func (t *DescriptionTile) SetConfigPath(path string) {
	t.configPath = path
}

func (t *DescriptionTile) SetItem(item map[string]any) {
	t.item = item
	t.copyValuesSet = false
	t.copyIdx = 0
	t.copyMode = false
	t.displayMd = ""
	t.copyMasked = nil
}

func (t *DescriptionTile) IsCopyMode() bool    { return t.copyMode }
func (t *DescriptionTile) HasCopyValues() bool { return len(t.copyValues) > 0 }

func (t *DescriptionTile) EnterCopyMode() {
	t.copyMode = true
	t.copyIdx = 0
}

func (t *DescriptionTile) ExitCopyMode() { t.copyMode = false }

func (t *DescriptionTile) CycleCopy(delta int) {
	n := len(t.copyValues)
	if n == 0 {
		return
	}
	t.copyIdx = (t.copyIdx + delta + n) % n
}

func (t *DescriptionTile) CurrentCopyValue() (string, bool) {
	if len(t.copyValues) == 0 {
		return "", false
	}
	return t.copyValues[t.copyIdx], true
}

func (t *DescriptionTile) IsCurrentMasked() bool {
	return t.copyIdx < len(t.copyMasked) && t.copyMasked[t.copyIdx]
}

func (t *DescriptionTile) CopyValueLabel() string {
	if len(t.copyValues) == 0 || t.item == nil {
		return ""
	}
	target := t.copyValues[t.copyIdx]
	var matches []string
	for k, v := range t.item {
		if fmt.Sprintf("%v", v) == target {
			matches = append(matches, k)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

func (t *DescriptionTile) glamourRenderer(width int) *glamour.TermRenderer {
	if t.renderer == nil || t.rendererWidth != width {
		t.renderer, _ = newGlamourRenderer(width)
		t.rendererWidth = width
	}
	return t.renderer
}

func (t *DescriptionTile) Init() tea.Cmd                           { return nil }
func (t *DescriptionTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

var hlStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("6")).
	Foreground(lipgloss.Color("0")).
	Bold(true)

func (t *DescriptionTile) View() string {
	w := t.Size.Width
	h := t.Size.Height
	if w <= 0 || h <= 0 {
		return ""
	}

	innerW := w - 2
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	var lines []string
	if t.item == nil {
		lines = []string{"  No item selected"}
	} else {
		lines = t.renderItem(innerW, innerH)
	}

	return renderBox(t.title, t.visibleLines(lines, innerH), w, t.IsFocused())
}

func (t *DescriptionTile) renderItem(innerW, innerH int) []string {
	d := config.FindDisplay(t.displays, t.item)
	tmpl := t.tmpls[d.Name]
	if tmpl == nil {
		return nil
	}

	data, missing := render.FillMissingFields(tmpl, t.item)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return t.plainLines("details template error: "+err.Error(), innerW)
	}
	expanded := render.ExpandAllEnv(buf.String(), t.item)
	expanded = render.ExpandConfigFile(expanded, t.configPath)
	expanded = render.MissingFieldsWarning(missing) + expanded

	if !t.copyValuesSet {
		t.displayMd, t.copyValues, t.copyMasked = render.ProcessMaskSpans(expanded)
		t.copyValuesSet = true
	}

	if lines, ok := t.renderMarkdown(innerW, innerH); ok {
		return lines
	}
	return t.plainLines(t.displayMd, innerW)
}

func (t *DescriptionTile) renderMarkdown(innerW, innerH int) ([]string, bool) {
	highlighting := t.copyMode && len(t.copyValues) > 0
	forRender := t.displayMd
	if highlighting {
		forRender = render.MarkNthCodeSpan(t.displayMd, t.copyIdx, hlSentinel)
	}

	r := t.glamourRenderer(innerW - 2)
	if r == nil {
		return nil, false
	}
	rendered, err := glamourRender(r, forRender)
	if err != nil {
		return nil, false
	}
	rendered = strings.TrimRight(rendered, "\n")

	sentinelLine := -1
	if highlighting {
		rendered, sentinelLine = t.highlightSentinel(rendered)
	}

	lines := strings.Split(rendered, "\n")
	if sentinelLine >= 0 {
		t.scrollIntoView(sentinelLine, innerH)
	}
	return lines, true
}

func (t *DescriptionTile) highlightSentinel(rendered string) (string, int) {
	sentinelLine := -1
	for i, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, hlSentinel) {
			sentinelLine = i
			break
		}
	}

	displayTarget := t.copyValues[t.copyIdx]
	if t.copyIdx < len(t.copyMasked) && t.copyMasked[t.copyIdx] {
		displayTarget = render.MaskedDisplayText(displayTarget)
	}
	return strings.Replace(rendered, hlSentinel, hlStyle.Render(displayTarget), 1), sentinelLine
}

func (t *DescriptionTile) scrollIntoView(line, innerH int) {
	if line < t.scrollOffset {
		t.scrollOffset = line
	} else if line >= t.scrollOffset+innerH {
		t.scrollOffset = line - innerH + 1
	}
}

func (t *DescriptionTile) plainLines(text string, innerW int) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		for _, seg := range wrapLine(line, innerW-2) {
			lines = append(lines, detailContentStyle.Render("  "+seg))
		}
	}
	return lines
}

type ActionsTile struct {
	*tl.BaseTile
	selectableList
	actions []config.Action
	title   string
}

func newActionsTile(actions []config.Action) *ActionsTile {
	return &ActionsTile{
		BaseTile: &tl.BaseTile{
			Name: "actions",
			Size: tl.Size{Weight: 1},
		},
		actions: actions,
		title:   "Actions",
	}
}

func (t *ActionsTile) SetActions(actions []config.Action) { t.actions = actions }
func (t *ActionsTile) MoveUp()                            { t.moveUp() }
func (t *ActionsTile) MoveDown()                          { t.moveDown(len(t.actions)) }

func (t *ActionsTile) Selected() *config.Action {
	if t.selected >= 0 && t.selected < len(t.actions) {
		return &t.actions[t.selected]
	}
	return nil
}

func (t *ActionsTile) Init() tea.Cmd                           { return nil }
func (t *ActionsTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *ActionsTile) View() string {
	w := t.Size.Width
	h := t.Size.Height
	if w <= 0 || h <= 0 {
		return ""
	}

	innerW := w - 2
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	labels := make([]string, len(t.actions))
	for i, a := range t.actions {
		labels[i] = a.Title
	}

	rows := t.renderRows(labels, innerW, innerH, t.IsFocused())
	return renderBox(t.title, strings.Join(rows, "\n"), w, t.IsFocused())
}
