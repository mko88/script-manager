package main

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
	detailContentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailCmdStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	detailCmdLblStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// DescriptionTile renders the details template for the selected item.
type DescriptionTile struct {
	*tl.BaseTile
	item         map[string]any
	tmpl         *template.Template
	scrollOffset int
	expandedCmd  string
}

func newDescriptionTile(item map[string]any, detailTmpl string) *DescriptionTile {
	tmpl, _ := template.New("detail").Parse(detailTmpl)
	return &DescriptionTile{
		BaseTile: &tl.BaseTile{
			Name: "description",
			Size: tl.Size{Weight: 2},
		},
		item: item,
		tmpl: tmpl,
	}
}

func (t *DescriptionTile) SetItem(item map[string]any) { t.item = item }
func (t *DescriptionTile) SetCmd(cmd string)           { t.expandedCmd = cmd }
func (t *DescriptionTile) ScrollUp() {
	if t.scrollOffset > 0 {
		t.scrollOffset--
	}
}
func (t *DescriptionTile) ScrollDown() { t.scrollOffset++ }
func (t *DescriptionTile) ResetScroll() { t.scrollOffset = 0 }

func (t *DescriptionTile) Init() tea.Cmd                             { return nil }
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
	if t.expandedCmd != "" {
		lines = append(lines, "")
		lines = append(lines, detailCmdLblStyle.Render("  Command:"))
		for i, cmdLine := range strings.Split(strings.TrimRight(t.expandedCmd, "\n"), "\n") {
			prefix := "  $ "
			if i > 0 {
				prefix = "    "
			}
			lines = append(lines, detailCmdStyle.Render(prefix+cmdLine))
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
	actions []Action
}

func newActionsTile(actions []Action) *ActionsTile {
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

func (t *ActionsTile) Selected() *Action {
	if t.selected >= 0 && t.selected < len(t.actions) {
		return &t.actions[t.selected]
	}
	return nil
}

func (t *ActionsTile) Init() tea.Cmd                             { return nil }
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

// renderBox wraps content in a rounded border with a title in the top edge.
// w and h are the outer dimensions including the border characters.
func renderBox(title, content string, w, h int, focused bool) string {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("12")
	}
	bs := lipgloss.NewStyle().Foreground(borderColor)

	innerW := w - 2

	titleStr := " " + title + " "
	dashCount := innerW - 1 - len(titleStr)
	if dashCount < 0 {
		if innerW-1 > 0 {
			titleStr = titleStr[:innerW-1]
		} else {
			titleStr = ""
		}
		dashCount = 0
	}
	top := bs.Render("╭─" + titleStr + strings.Repeat("─", dashCount) + "╮")
	bottom := bs.Render("╰" + strings.Repeat("─", innerW) + "╯")

	contentLines := strings.Split(content, "\n")
	rows := make([]string, 0, len(contentLines)+2)
	rows = append(rows, top)
	for _, line := range contentLines {
		if pad := innerW - lipgloss.Width(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		rows = append(rows, bs.Render("│")+line+bs.Render("│"))
	}
	rows = append(rows, bottom)

	return strings.Join(rows, "\n")
}

func padToLines(content string, h int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:h], "\n")
}
