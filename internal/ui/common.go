package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderBox(title, content string, w int, focused bool) string {
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

// scrollableContent tracks a scroll position and is embedded by tiles that
// need scrollable text content.
type scrollableContent struct {
	scrollOffset int
}

func (s *scrollableContent) ScrollUp() {
	if s.scrollOffset > 0 {
		s.scrollOffset--
	}
}
func (s *scrollableContent) ScrollDown() { s.scrollOffset++ }
func (s *scrollableContent) ResetScroll() { s.scrollOffset = 0 }

// visibleLines clamps the scroll offset, then returns the visible window of
// lines joined and padded to innerH rows — ready to pass to renderBox.
func (s *scrollableContent) visibleLines(lines []string, innerH int) string {
	max := len(lines) - innerH
	if max < 0 {
		max = 0
	}
	if s.scrollOffset > max {
		s.scrollOffset = max
	}
	return padToLines(strings.Join(lines[s.scrollOffset:], "\n"), innerH)
}

// wrapLine splits a raw (unstyled) line into segments of at most width runes.
func wrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	runes := []rune(line)
	if len(runes) <= width {
		return []string{line}
	}
	var out []string
	for len(runes) > width {
		out = append(out, string(runes[:width]))
		runes = runes[width:]
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
	}
	return out
}
