package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"héllo wörld", 6, "héllo…"},
		// Wide (2-cell) characters: 日本語 is 6 cells.
		{"日本語", 6, "日本語"},
		{"日本語", 5, "日本…"},
		{"日本語", 4, "日…"},
	}
	for _, tt := range tests {
		if got := truncateToWidth(tt.in, tt.max); got != tt.want {
			t.Errorf("truncateToWidth(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
		if got := truncateToWidth(tt.in, tt.max); lipgloss.Width(got) > tt.max {
			t.Errorf("truncateToWidth(%q, %d) = %q is %d cells wide", tt.in, tt.max, got, lipgloss.Width(got))
		}
	}
}

func TestWrapLine(t *testing.T) {
	tests := []struct {
		in    string
		width int
		want  []string
	}{
		{"short", 10, []string{"short"}},
		{"abcdef", 3, []string{"abc", "def"}},
		{"abcdefg", 3, []string{"abc", "def", "g"}},
		{"anything", 0, []string{"anything"}},
		{"", 5, []string{""}},
	}
	for _, tt := range tests {
		if got := wrapLine(tt.in, tt.width); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("wrapLine(%q, %d) = %v, want %v", tt.in, tt.width, got, tt.want)
		}
	}
}

func TestRenderBoxWidths(t *testing.T) {
	for _, title := range []string{"Items", "日本語タイトル", "a very long title that will not fit in the box at all"} {
		out := renderBox(title, "content", 20, false)
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w != 20 {
				t.Errorf("renderBox(title=%q) line %d is %d cells wide, want 20", title, i, w)
			}
		}
	}
}

func TestVisibleLinesClampsScroll(t *testing.T) {
	s := &scrollableContent{scrollOffset: 99}
	lines := []string{"a", "b", "c", "d"}
	out := s.visibleLines(lines, 2)
	if s.scrollOffset != 2 {
		t.Errorf("scrollOffset clamped to %d, want 2", s.scrollOffset)
	}
	if want := "c\nd"; out != want {
		t.Errorf("visibleLines = %q, want %q", out, want)
	}
}
