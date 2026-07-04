package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// The status bar must always render exactly one row: content wider than the
// tile makes lipgloss wrap, which used to grow the bar to two lines on
// narrow terminals.
func TestStatusBarSingleLineAtAnyWidth(t *testing.T) {
	contexts := []statusContext{
		ctxItemSelect, ctxActionsFocused, ctxDetailsFocused, ctxDetailsCopyMode, ctxCommandFocused,
	}
	for _, ctx := range contexts {
		for w := 1; w <= 120; w++ {
			bar := newStatusBarTile()
			bar.SetContext(ctx)
			bar.Size.Width = w
			view := bar.View()
			if strings.Contains(view, "\n") {
				t.Fatalf("context %d width %d: view has multiple lines", ctx, w)
			}
			if got := lipgloss.Width(view); got != w {
				t.Fatalf("context %d width %d: view width = %d", ctx, w, got)
			}
		}
	}
}

func TestStatusBarMessageTruncated(t *testing.T) {
	bar := newStatusBarTile()
	bar.SetMessage(strings.Repeat("config reload failed: yaml error ", 5))
	for w := 1; w <= 60; w++ {
		bar.Size.Width = w
		view := bar.View()
		if strings.Contains(view, "\n") {
			t.Fatalf("width %d: message view has multiple lines", w)
		}
		if got := lipgloss.Width(view); got != w {
			t.Fatalf("width %d: message view width = %d", w, got)
		}
	}
}
