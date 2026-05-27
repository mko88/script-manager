package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

var (
	statusBarBgStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	statusKeyStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("11")).Bold(true)
	statusDescStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("245"))
	statusMsgStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("9")).Bold(true)
	statusSepStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("240"))
)

var itemModeHelp = []struct{ key, desc string }{
	{"↑↓ / k j", "Navigate items"},
	{"Enter", "Select item"},
	{"Q / Esc", "Quit"},
}

var actionModeHelp = []struct{ key, desc string }{
	{"↑↓ / k j", "Navigate / Scroll"},
	{"Tab / ←→", "Switch focus"},
	{"Enter / 1-9", "Run action"},
	{"y", "Copy command"},
	{"Esc", "Back to items"},
}

type StatusBarTile struct {
	*tl.BaseTile
	message string
	mode    appMode
}

func newStatusBarTile() *StatusBarTile {
	return &StatusBarTile{
		BaseTile: &tl.BaseTile{
			Name: "statusbar",
			Size: tl.Size{FixedHeight: 1},
		},
	}
}

func (t *StatusBarTile) SetMessage(msg string) { t.message = msg }
func (t *StatusBarTile) ClearMessage()         { t.message = "" }
func (t *StatusBarTile) SetMode(m appMode)     { t.mode = m }

func (t *StatusBarTile) Init() tea.Cmd                            { return nil }
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

	entries := itemModeHelp
	if t.mode == modeSelectAction {
		entries = actionModeHelp
	}

	sep := statusSepStyle.Render("  │  ")
	var parts []string
	for i, e := range entries {
		part := statusKeyStyle.Render(e.key) + statusDescStyle.Render(" "+e.desc)
		parts = append(parts, part)
		if i < len(entries)-1 {
			parts = append(parts, sep)
		}
	}

	return statusBarBgStyle.Width(w).Render("  " + strings.Join(parts, ""))
}
