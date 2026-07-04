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
		{"F5", "Reload config"},
		{"Q", "Quit"},
	},
	ctxActionsFocused: {
		{"↑↓ / k j", "Navigate"},
		{"[ ]", "Cycle group"},
		{"Enter", "Run action"},
		{"Tab / ←→", "Next pane"},
		{"F5", "Reload config"},
		{"Esc", "Back to items"},
	},
	ctxDetailsFocused: {
		{"↑↓ / k j", "Scroll"},
		{"Enter", "Select value to copy"},
		{"Tab / ←→", "Next pane"},
		{"F5", "Reload config"},
		{"Esc", "Back to items"},
	},
	ctxDetailsCopyMode: {
		{"Enter", "Copy"},
		{"Esc", "Done"},
	},
	ctxCommandFocused: {
		{"↑↓ / k j", "Scroll"},
		{"y", "Copy command"},
		{"Tab / ←→", "Next pane"},
		{"F5", "Reload config"},
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

func (t *StatusBarTile) SetMessage(msg string)      { t.message = msg }
func (t *StatusBarTile) ClearMessage()              { t.message = "" }
func (t *StatusBarTile) SetContext(c statusContext) { t.context = c }

func (t *StatusBarTile) Init() tea.Cmd                           { return nil }
func (t *StatusBarTile) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return t, nil }

func (t *StatusBarTile) View() string {
	w := t.Size.Width
	if w <= 0 {
		return ""
	}

	// Everything below must fit in exactly one row of w cells: lipgloss
	// wraps content wider than Width(w), which would grow the bar to two
	// lines on narrow terminals.
	if t.message != "" {
		content := truncateToWidth("  "+t.message, w)
		return statusBarBgStyle.Width(w).Render(statusMsgStyle.Render(content))
	}

	entries := contextHelp[t.context]
	sep := statusSepStyle.Render("  │  ")
	sepW := lipgloss.Width(sep)
	avail := w - 2 // leading padding
	used := 0
	var b strings.Builder
	for i, e := range entries {
		part := statusKeyStyle.Render(e.key) + statusDescStyle.Render(" "+e.desc)
		need := lipgloss.Width(part)
		if i > 0 {
			need += sepW
		}
		// Drop this entry and the rest rather than truncating mid-entry:
		// a partial key hint is worse than none.
		if used+need > avail {
			break
		}
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(part)
		used += need
	}

	return statusBarBgStyle.Width(w).Render("  " + b.String())
}
