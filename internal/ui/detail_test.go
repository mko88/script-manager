package ui

import (
	"testing"

	"script-manager/internal/config"

	tl "github.com/mko88/bubbletea-tilelayout"
)

func TestDescriptionTileMultilineIsMasked(t *testing.T) {
	displays := config.DisplayList{{Name: "default", List: "{{.name}}", Details: "`{{.cert}}`"}}
	tile := newDescriptionTile(displays)
	tile.Size = tl.Size{Width: 40, Height: 10}

	cert := "line1\nline2\nline3"
	tile.SetItem(map[string]any{"name": "x", "cert": cert})
	tile.View()

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
