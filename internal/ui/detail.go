package ui

import (
	"bytes"
	"fmt"
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
	item         map[string]any
	tmpl         *template.Template
	scrollOffset int
}

func newDescriptionTile(item map[string]any, detailTmpl string) *DescriptionTile {
	tmpl, _ := template.New("detail").Parse(detailTmpl)
	return &DescriptionTile{
		BaseTile: &tl.BaseTile{
			Name: "description",
			Size: tl.Size{Weight: 1},
		},
		item: item,
		tmpl: tmpl,
	}
}

func (t *DescriptionTile) SetItem(item map[string]any) { t.item = item }
func (t *DescriptionTile) ScrollUp() {
	if t.scrollOffset > 0 {
		t.scrollOffset--
	}
}
func (t *DescriptionTile) ScrollDown() { t.scrollOffset++ }
func (t *DescriptionTile) ResetScroll() { t.scrollOffset = 0 }

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
	} else if t.tmpl != nil {
		var buf bytes.Buffer
		t.tmpl.Execute(&buf, t.item)
		rendered := strings.TrimRight(buf.String(), "\n")
		for _, line := range strings.Split(rendered, "\n") {
			lines = append(lines, detailContentStyle.Render("  "+line))
		}
	}

	maxOffset := len(lines) - innerH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if t.scrollOffset > maxOffset {
		t.scrollOffset = maxOffset
	}

	content := padToLines(strings.Join(lines[t.scrollOffset:], "\n"), innerH)
	return renderBox("Details", content, w, h, t.IsFocused())
}

// ActionsTile shows the configured actions and tracks the highlighted selection.
type ActionsTile struct {
	*tl.BaseTile
	selectableList
	actions []config.Action
}

func newActionsTile(actions []config.Action) *ActionsTile {
	return &ActionsTile{
		BaseTile: &tl.BaseTile{
			Name: "actions",
			Size: tl.Size{Weight: 2},
		},
		actions: actions,
	}
}

func (t *ActionsTile) MoveUp()   { t.moveUp() }
func (t *ActionsTile) MoveDown() { t.moveDown(len(t.actions)) }

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
		labels[i] = fmt.Sprintf("%d  %s", i+1, a.Title)
	}

	rows := t.renderRows(labels, innerW, innerH, t.IsFocused())
	return renderBox("Actions", strings.Join(rows, "\n"), w, h, t.IsFocused())
}
