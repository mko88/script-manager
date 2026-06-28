package ui

import (
	"bytes"
	"strings"
	"text/template"

	"script-manager/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var detailContentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

// DescriptionTile renders the details template for the selected item.
type DescriptionTile struct {
	*tl.BaseTile
	scrollableContent
	item     map[string]any
	displays []config.DisplayConfig
	tmpls    map[string]*template.Template // keyed by DisplayConfig.Name
	title    string
}

func newDescriptionTile(displays []config.DisplayConfig) *DescriptionTile {
	tmpls := make(map[string]*template.Template, len(displays))
	for _, d := range displays {
		tmpl, _ := template.New("detail").Parse(d.Details)
		tmpls[d.Name] = tmpl
	}
	return &DescriptionTile{
		BaseTile: &tl.BaseTile{
			Name: "description",
			Size: tl.Size{Weight: 1},
		},
		displays: displays,
		tmpls:    tmpls,
		title:    "Details",
	}
}

func (t *DescriptionTile) SetItem(item map[string]any) { t.item = item }

func (t *DescriptionTile) Init() tea.Cmd                            { return nil }
func (t *DescriptionTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

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
		lines = append(lines, "  No item selected")
	} else {
		d := config.FindDisplay(t.displays, t.item)
		tmpl := t.tmpls[d.Name]
		if tmpl != nil {
			var buf bytes.Buffer
			tmpl.Execute(&buf, t.item)
			rendered := strings.TrimRight(buf.String(), "\n")
			for _, line := range strings.Split(rendered, "\n") {
				for _, seg := range wrapLine(line, innerW-2) {
					lines = append(lines, detailContentStyle.Render("  "+seg))
				}
			}
		}
	}

	return renderBox(t.title, t.visibleLines(lines, innerH), w, t.IsFocused())
}

// ActionsTile shows the configured actions and tracks the highlighted selection.
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
func (t *ActionsTile) MoveUp()                           { t.moveUp() }
func (t *ActionsTile) MoveDown()                         { t.moveDown(len(t.actions)) }

func (t *ActionsTile) Selected() *config.Action {
	if t.selected >= 0 && t.selected < len(t.actions) {
		return &t.actions[t.selected]
	}
	return nil
}

func (t *ActionsTile) Init() tea.Cmd                            { return nil }
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
