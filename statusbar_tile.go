package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var (
	statusBarBgStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	statusKeyStyle    = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("11")).Bold(true)
	statusDescStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("245"))
	statusMsgStyle    = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("9")).Bold(true)
	statusSepStyle    = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("240"))
)

var helpEntries = []struct{ key, desc string }{
	{"↑↓ / k j", "Navigate / Scroll"},
	{"Tab", "Switch focus"},
	{"Enter / 1-9", "Run action"},
	{"Q / Esc", "Quit"},
}

type StatusBarTile struct {
	*tl.BaseTile
	message string
}

func newStatusBarTile() *StatusBarTile {
	return &StatusBarTile{
		BaseTile: &tl.BaseTile{
			Name: "statusbar",
			Size: tl.Size{FixedHeight: 1},
		},
	}
}

func (t *StatusBarTile) SetMessage(msg string) {
	t.message = msg
}

func (t *StatusBarTile) ClearMessage() {
	t.message = ""
}

func (t *StatusBarTile) Init() tea.Cmd { return nil }

func (t *StatusBarTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *StatusBarTile) View() string {
	w := t.Size.Width
	if w <= 0 {
		return ""
	}

	if t.message != "" {
		content := "  " + t.message
		return statusBarBgStyle.Width(w).Render(statusMsgStyle.Render(content))
	}

	var parts []string
	sep := statusSepStyle.Render("  │  ")

	for i, e := range helpEntries {
		part := statusKeyStyle.Render(e.key) + statusDescStyle.Render(" "+e.desc)
		parts = append(parts, part)
		if i < len(helpEntries)-1 {
			parts = append(parts, sep)
		}
	}

	bar := strings.Join(parts, "")
	return statusBarBgStyle.Width(w).Render("  " + bar)
}
