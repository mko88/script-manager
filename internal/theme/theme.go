// Package theme persists the active theme choice — and, once saved, a
// user-defined custom palette — to a file shared by both GUI apps'
// executable directory, so switching it in one app is picked up by the
// other the next time it starts — the two apps are separate WebView
// processes with separate localStorage, which this file bridges the same
// way internal/messages bridges each app's message overrides.
package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Filename is shared by both apps — unlike internal/messages there's only
// one value, not one per app, so there's nothing to namespace.
const Filename = "sm-theme.json"

// legacyFilename was the whole file's format before the custom-theme
// editor: just the literal text "dark" or "light", no JSON. Load falls
// back to it so an existing user's choice isn't silently reset by this
// upgrade.
const legacyFilename = "sm-theme.txt"

// State is the persisted theme choice. Custom is only meaningful when
// Active is "custom" but is kept even when switched away from, so
// switching back doesn't lose the saved palette.
type State struct {
	Active string            `json:"active"` // "dark" | "light" | "custom"
	Custom map[string]string `json:"custom,omitempty"`
}

func normalizeActive(active string) string {
	switch active {
	case "light", "custom":
		return active
	default:
		return "dark"
	}
}

// Load returns the persisted theme state, defaulting to dark if
// sm-theme.json doesn't exist, is invalid, or holds an unrecognized
// Active value. Falls back to the pre-custom-theme sm-theme.txt (just an
// active name, no palette) if sm-theme.json doesn't exist yet.
func Load(dir string) State {
	if data, err := os.ReadFile(filepath.Join(dir, Filename)); err == nil {
		var s State
		if json.Unmarshal(data, &s) == nil {
			s.Active = normalizeActive(s.Active)
			return s
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, legacyFilename)); err == nil {
		return State{Active: normalizeActive(strings.TrimSpace(string(data)))}
	}
	return State{Active: "dark"}
}

// Save persists s to dir. Best-effort by design (see callers): a write
// failure shouldn't block the UI from switching, since the theme has
// already been applied client-side by the time this is called.
func Save(dir string, s State) error {
	s.Active = normalizeActive(s.Active)
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Filename), out, 0o644)
}
