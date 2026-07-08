package ui

import (
	"testing"

	"script-manager/internal/config"

	tl "github.com/mko88/bubbletea-tilelayout"
)

// TestDescriptionTileMultilineIsMasked exercises the copy pipeline for a
// value referenced directly by a hand-written `{{.field}}` span (as opposed
// to one produced by #ALL_ENV_LIST#/#ALL_ENV_TABLE#, covered at the render
// package level): a multi-line value is treated as masked, same as a real
// secret, while the real value stays available to copy in full.
func TestDescriptionTileMultilineIsMasked(t *testing.T) {
	displays := config.DisplayList{{Name: "default", List: "{{.name}}", Details: "`{{.cert}}`"}}
	tile := newDescriptionTile(displays)
	tile.Size = tl.Size{Width: 40, Height: 10}

	cert := "line1\nline2\nline3"
	tile.SetItem(map[string]any{"name": "x", "cert": cert})
	tile.View() // triggers renderItem, populating copyValues/copyMasked

	if !tile.HasCopyValues() {
		t.Fatal("expected a copy value to have been found")
	}
	if !tile.IsCurrentMasked() {
		t.Error("expected a multi-line value to be treated as masked")
	}
	val, ok := tile.CurrentCopyValue()
	if !ok || val != cert {
		t.Errorf("CurrentCopyValue() = (%q, %v), want (%q, true)", val, ok, cert)
	}
}
