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
