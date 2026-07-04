package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var (
	cmdBarDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cmdBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cmdBarPfxStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

type CmdBarTile struct {
	*tl.BaseTile
	scrollableContent
	description string
	cmd         string
	title       string
}

func newCmdBarTile() *CmdBarTile {
	return &CmdBarTile{
		BaseTile: &tl.BaseTile{
			Name: "cmdbar",
			Size: tl.Size{Weight: 1},
		},
		title: "Command",
	}
}

func (t *CmdBarTile) Cmd() string                { return t.cmd }
func (t *CmdBarTile) SetCmd(cmd string)          { t.cmd = cmd }
func (t *CmdBarTile) SetDescription(desc string) { t.description = desc }

func (t *CmdBarTile) Init() tea.Cmd                           { return nil }
func (t *CmdBarTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *CmdBarTile) View() string {
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

	if t.description != "" {
		for _, line := range strings.Split(strings.TrimRight(t.description, "\n"), "\n") {
			for _, seg := range wrapLine(line, innerW-2) {
				lines = append(lines, "  "+cmdBarDescStyle.Render(seg))
			}
		}
		if t.cmd != "" {
			lines = append(lines, "")
		}
	}

	if t.cmd != "" {
		for i, line := range strings.Split(strings.TrimRight(t.cmd, "\n"), "\n") {
			for j, seg := range wrapLine(line, innerW-4) {
				prefix := "  "
				if i == 0 && j == 0 {
					prefix = "$ "
				}
				lines = append(lines, "  "+cmdBarPfxStyle.Render(prefix)+cmdBarStyle.Render(seg))
			}
		}
	}

	return renderBox(t.title, t.visibleLines(lines, innerH), w, t.IsFocused())
}
