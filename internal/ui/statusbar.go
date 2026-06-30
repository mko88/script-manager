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
	statusMsgStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
	statusSepStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("240"))
)

type statusContext int

const (
	ctxItemSelect statusContext = iota
	ctxActionsFocused
	ctxDetailsFocused
	ctxDetailsCopyMode
	ctxCommandFocused
)

var contextHelp = map[statusContext][]struct{ key, desc string }{
	ctxItemSelect: {
		{"↑↓ / k j", "Navigate"},
		{"Enter / Tab", "Select item"},
		{"Q", "Quit"},
	},
	ctxActionsFocused: {
		{"↑↓ / k j", "Navigate"},
		{"[ ]", "Cycle group"},
		{"Enter", "Run action"},
		{"Tab / ←→", "Next pane"},
		{"Esc", "Back to items"},
	},
	ctxDetailsFocused: {
		{"↑↓ / k j", "Scroll"},
		{"Enter", "Select value to copy"},
		{"Tab / ←→", "Next pane"},
		{"Esc", "Back to items"},
	},
	ctxDetailsCopyMode: {
		{"↑↓ / k j", "Cycle value"},
		{"Enter", "Copy"},
		{"Esc", "Done"},
	},
	ctxCommandFocused: {
		{"↑↓ / k j", "Scroll"},
		{"y", "Copy command"},
		{"Tab / ←→", "Next pane"},
		{"Esc", "Back to items"},
	},
}

type StatusBarTile struct {
	*tl.BaseTile
	message string
	context statusContext
}

func newStatusBarTile() *StatusBarTile {
	return &StatusBarTile{
		BaseTile: &tl.BaseTile{
			Name: "statusbar",
			Size: tl.Size{FixedHeight: 1},
		},
	}
}

func (t *StatusBarTile) SetMessage(msg string)       { t.message = msg }
func (t *StatusBarTile) ClearMessage()               { t.message = "" }
func (t *StatusBarTile) SetContext(c statusContext)   { t.context = c }

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

	entries := contextHelp[t.context]
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
