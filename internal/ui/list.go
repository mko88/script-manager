package ui

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var (
	selFocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0")).Bold(true)
	selNormalStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// selectableList is embedded by tiles that show a scrollable, selectable list of rows.
type selectableList struct {
	selected int
	offset   int
}

func (s *selectableList) moveUp() {
	if s.selected > 0 {
		s.selected--
	}
}

func (s *selectableList) moveDown(count int) {
	if s.selected < count-1 {
		s.selected++
	}
}

func (s *selectableList) renderRows(labels []string, innerW, innerH int, focused bool) []string {
	if s.selected >= 0 {
		if s.selected < s.offset {
			s.offset = s.selected
		}
		if s.selected >= s.offset+innerH {
			s.offset = s.selected - innerH + 1
		}
	} else {
		s.offset = 0
	}

	var rows []string
	for i := s.offset; i < len(labels) && len(rows) < innerH; i++ {
		label := labels[i]
		maxLabel := innerW - 4
		if maxLabel < 1 {
			maxLabel = 1
		}
		if len([]rune(label)) > maxLabel {
			label = string([]rune(label)[:maxLabel-1]) + "…"
		}

		var row string
		if i == s.selected && focused {
			row = selFocusedStyle.Width(innerW).Render(" ▶ " + label)
		} else {
			row = selNormalStyle.Width(innerW).Render("   " + label)
		}
		rows = append(rows, row)
	}

	for len(rows) < innerH {
		rows = append(rows, selNormalStyle.Width(innerW).Render(""))
	}
	return rows
}

// ListTile renders a scrollable list of items whose labels come from a Go template.
type ListTile struct {
	*tl.BaseTile
	selectableList
	items []map[string]any
	tmpl  *template.Template
}

func newListTile(items []map[string]any, listTmpl string) *ListTile {
	tmpl, _ := template.New("list").Parse(listTmpl)
	return &ListTile{
		BaseTile: &tl.BaseTile{
			Name: "list",
			Size: tl.Size{Weight: 1},
		},
		items: items,
		tmpl:  tmpl,
	}
}

func (t *ListTile) renderLabel(item map[string]any) string {
	if t.tmpl == nil {
		return fmt.Sprint(item["name"])
	}
	var buf bytes.Buffer
	t.tmpl.Execute(&buf, item)
	return buf.String()
}

func (t *ListTile) Init() tea.Cmd                            { return nil }
func (t *ListTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *ListTile) View() string {
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

	labels := make([]string, len(t.items))
	for i, item := range t.items {
		labels[i] = t.renderLabel(item)
	}

	rows := t.renderRows(labels, innerW, innerH, t.IsFocused())
	return renderBox("Items", strings.Join(rows, "\n"), w, h, t.IsFocused())
}

func (t *ListTile) MoveUp()   { t.moveUp() }
func (t *ListTile) MoveDown() { t.moveDown(len(t.items)) }

func (t *ListTile) Selected() map[string]any {
	if t.selected >= 0 && t.selected < len(t.items) {
		return t.items[t.selected]
	}
	return nil
}
