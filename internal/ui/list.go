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

var (
	selFocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0")).Bold(true)
	selBlurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
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
		label = truncateToWidth(label, maxLabel)

		var row string
		switch {
		case i == s.selected && focused:
			row = selFocusedStyle.Width(innerW).Render(" ▶ " + label)
		case i == s.selected:
			row = selBlurredStyle.Width(innerW).Render(" ▶ " + label)
		default:
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
	items    []map[string]any
	displays []config.DisplayConfig
	tmpls    map[string]*template.Template // keyed by DisplayConfig.Name
	title    string
}

func newListTile(items []map[string]any, displays []config.DisplayConfig) *ListTile {
	t := &ListTile{
		BaseTile: &tl.BaseTile{
			Name: "list",
			Size: tl.Size{Weight: 1},
		},
		title: "Items",
	}
	t.SetItems(items, displays)
	return t
}

// SetItems replaces the item list and display templates, e.g. after a config
// reload. The current selection is preserved when still in range.
func (t *ListTile) SetItems(items []map[string]any, displays []config.DisplayConfig) {
	tmpls := make(map[string]*template.Template, len(displays))
	for _, d := range displays {
		tmpl, _ := template.New("list").Parse(d.List)
		tmpls[d.Name] = tmpl
	}
	t.items = items
	t.displays = displays
	t.tmpls = tmpls
	switch {
	case len(items) == 0:
		t.selected = 0
	case t.selected >= len(items):
		t.selected = len(items) - 1
	case t.selected < 0:
		t.selected = 0
	}
}

// renderLabel expands the list template for the item, falling back to the
// item's name when the template failed to parse or execute.
func (t *ListTile) renderLabel(item map[string]any) string {
	d := config.FindDisplay(t.displays, item)
	tmpl := t.tmpls[d.Name]
	if tmpl == nil {
		return fmt.Sprint(item[config.KeyName])
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, item); err != nil {
		return fmt.Sprint(item[config.KeyName])
	}
	return buf.String()
}

func (t *ListTile) Init() tea.Cmd                           { return nil }
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
	return renderBox(t.title, strings.Join(rows, "\n"), w, t.IsFocused())
}

func (t *ListTile) MoveUp()   { t.moveUp() }
func (t *ListTile) MoveDown() { t.moveDown(len(t.items)) }

func (t *ListTile) Selected() map[string]any {
	if t.selected >= 0 && t.selected < len(t.items) {
		return t.items[t.selected]
	}
	return nil
}
