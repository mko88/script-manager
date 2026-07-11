// Package theme persists the active theme choice — and any number of
// user-defined named custom palettes — to a file shared by both GUI apps'
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

// State is the persisted theme choice. Active is "dark", "light", or the
// name of an entry in Themes — a custom theme's name doubles as its
// selector, so there's no separate "custom" sentinel.
type State struct {
	Active string                       `json:"active"`
	Themes map[string]map[string]string `json:"themes,omitempty"`

	// Custom is the pre-multi-theme format's single-palette slot. Load
	// migrates it into Themes["Custom"] the first time an old file is
	// read; Save never writes it again.
	Custom map[string]string `json:"custom,omitempty"`
}

// migrate folds the old single-palette Custom field into Themes the first
// time a pre-multi-theme file is read, so an existing user's saved
// palette isn't lost by this upgrade.
func migrate(s *State) {
	if s.Custom == nil {
		return
	}
	if s.Themes == nil {
		s.Themes = map[string]map[string]string{}
	}
	if _, exists := s.Themes["Custom"]; !exists {
		s.Themes["Custom"] = s.Custom
	}
	if s.Active == "custom" {
		s.Active = "Custom"
	}
	s.Custom = nil
}

func normalizeActive(active string, themes map[string]map[string]string) string {
	if active == "dark" || active == "light" {
		return active
	}
	if _, ok := themes[active]; ok {
		return active
	}
	return "dark"
}

// Load returns the persisted theme state, defaulting to dark if
// sm-theme.json doesn't exist, is invalid, or names a theme that no
// longer exists. Falls back to the pre-custom-theme sm-theme.txt (just an
// active name, no palette) if sm-theme.json doesn't exist yet.
func Load(dir string) State {
	if data, err := os.ReadFile(filepath.Join(dir, Filename)); err == nil {
		var s State
		if json.Unmarshal(data, &s) == nil {
			migrate(&s)
			s.Active = normalizeActive(s.Active, s.Themes)
			return s
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, legacyFilename)); err == nil {
		return State{Active: normalizeActive(strings.TrimSpace(string(data)), nil)}
	}
	return State{Active: "dark"}
}

// Save persists s to dir. Best-effort by design (see callers): a write
// failure shouldn't block the UI from switching, since the theme has
// already been applied client-side by the time this is called.
func Save(dir string, s State) error {
	s.Custom = nil
	s.Active = normalizeActive(s.Active, s.Themes)
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Filename), out, 0o644)
}
