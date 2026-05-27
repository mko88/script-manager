package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var (
	cmdBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	cmdBarPfxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type CmdBarTile struct {
	*tl.BaseTile
	cmd          string
	scrollOffset int
}

func newCmdBarTile() *CmdBarTile {
	return &CmdBarTile{
		BaseTile: &tl.BaseTile{
			Name: "cmdbar",
			Size: tl.Size{FixedHeight: 5},
		},
	}
}

func (t *CmdBarTile) Cmd() string       { return t.cmd }
func (t *CmdBarTile) SetCmd(cmd string) { t.cmd = cmd }
func (t *CmdBarTile) ScrollUp() {
	if t.scrollOffset > 0 {
		t.scrollOffset--
	}
}
func (t *CmdBarTile) ScrollDown() { t.scrollOffset++ }
func (t *CmdBarTile) ResetScroll() { t.scrollOffset = 0 }

func (t *CmdBarTile) Init() tea.Cmd                            { return nil }
func (t *CmdBarTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *CmdBarTile) View() string {
	w := t.Size.Width
	h := t.Size.Height
	if w <= 0 || h <= 0 {
		return ""
	}

	innerH := h - 2

	var lines []string
	if t.cmd != "" {
		for i, line := range strings.Split(strings.TrimRight(t.cmd, "\n"), "\n") {
			prefix := "$ "
			if i > 0 {
				prefix = "  "
			}
			lines = append(lines, "  "+cmdBarPfxStyle.Render(prefix)+cmdBarStyle.Render(line))
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
	return renderBox("Command", content, w, h, t.IsFocused())
}
