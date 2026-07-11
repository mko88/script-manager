// Package theme persists the dark/light theme choice to a file shared by
// both GUI apps' executable directory, so switching it in one app is picked
// up by the other the next time it starts — the two apps are separate
// WebView processes with separate localStorage, which this file bridges the
// same way internal/messages bridges each app's message overrides.
package theme

import (
	"os"
	"path/filepath"
	"strings"
)

// Filename is shared by both apps — unlike internal/messages there's only
// one value, not one per app, so there's nothing to namespace.
const Filename = "sm-theme.txt"

// Load returns the persisted theme ("dark" or "light"), defaulting to
// "dark" if the file doesn't exist or holds anything else.
func Load(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		return "dark"
	}
	if strings.TrimSpace(string(data)) == "light" {
		return "light"
	}
	return "dark"
}

// Save persists theme to dir. Best-effort by design (see callers): a write
// failure shouldn't block the UI from switching, since the theme has
// already been applied client-side by the time this is called.
func Save(dir string, t string) error {
	if t != "light" {
		t = "dark"
	}
	return os.WriteFile(filepath.Join(dir, Filename), []byte(t), 0o644)
}
