package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const Filename = "sm-theme.json"

const legacyFilename = "sm-theme.txt"

type State struct {
	Active string                       `json:"active"`
	Themes map[string]map[string]string `json:"themes,omitempty"`

	Custom map[string]string `json:"custom,omitempty"`
}

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

func Save(dir string, s State) error {
	s.Custom = nil
	s.Active = normalizeActive(s.Active, s.Themes)
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Filename), out, 0o644)
}
