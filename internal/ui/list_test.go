package ui

import (
	"testing"

	"script-manager/internal/config"
	"script-manager/internal/secret"
)

func TestListLabelRedactsSecrets(t *testing.T) {
	displays := config.DisplayList{{Name: "default", List: "{{.name}} {{.password}} {{.token}} {{.host}}"}}
	items := []config.Item{{Name: "one", Env: map[string]any{"password": secret.Prefix + "Y2lwaGVydGV4dA==", "token": secret.Unset}}}
	tile := newListTile(items, displays, map[string]any{"host": "example"})

	got := tile.renderLabel(&tile.items[0])
	want := "one " + secret.LockedDisplayText + " " + secret.UnsetDisplayText + " example"
	if got != want {
		t.Errorf("list label = %q, want %q", got, want)
	}
}
