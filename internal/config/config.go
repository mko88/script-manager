package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Action struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Cmd         string   `yaml:"cmd"`
	Groups      []string `yaml:"groups"`
	NoWait      bool     `yaml:"noWait"`
}

type DisplayConfig struct {
	List    string `yaml:"list"`
	Details string `yaml:"details"`
}

type TitlesConfig struct {
	Items   string `yaml:"items"`
	Actions string `yaml:"actions"`
	Details string `yaml:"details"`
	Command string `yaml:"command"`
}

type Config struct {
	Shell   []string         `yaml:"shell"`
	Display DisplayConfig    `yaml:"display"`
	Titles  TitlesConfig     `yaml:"titles"`
	Env     map[string]any   `yaml:"env"`
	Items   []map[string]any `yaml:"items"`
	Actions []Action         `yaml:"actions"`
}

// ActionsForItem returns the actions available for the given item.
//
// If the item defines "actions" (list of IDs) or "actionGroups" (list of group
// names), only matching global actions are included — in that order, without
// duplicates. Item-level "customActions" are always appended at the end.
//
// If none of those keys are present the full allActions slice is returned as-is.
func ActionsForItem(allActions []Action, item map[string]any) []Action {
	if item == nil {
		return allActions
	}

	allowedIDs, hasIDs := asStringSlice(item["actions"])
	allowedGroups, hasGroups := asStringSlice(item["actionGroups"])
	customRaw := item["customActions"]

	if !hasIDs && !hasGroups && customRaw == nil {
		return allActions
	}

	seen := make(map[int]bool)
	var result []Action

	if hasIDs {
		idSet := make(map[string]bool)
		for _, id := range allowedIDs {
			idSet[id] = true
		}
		for i, a := range allActions {
			if a.ID != "" && idSet[a.ID] && !seen[i] {
				result = append(result, a)
				seen[i] = true
			}
		}
	}

	if hasGroups {
		groupSet := make(map[string]bool)
		for _, g := range allowedGroups {
			groupSet[g] = true
		}
		for i, a := range allActions {
			if seen[i] {
				continue
			}
			for _, g := range a.Groups {
				if groupSet[g] {
					result = append(result, a)
					seen[i] = true
					break
				}
			}
		}
	}

	result = append(result, parseCustomActions(customRaw)...)
	return result
}

func asStringSlice(v any) ([]string, bool) {
	if v == nil {
		return nil, false
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, elem := range raw {
		if s, ok := elem.(string); ok {
			out = append(out, s)
		}
	}
	return out, len(out) > 0
}

func parseCustomActions(v any) []Action {
	if v == nil {
		return nil
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []Action
	for _, elem := range raw {
		m, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}
		a := Action{
			ID:          strVal(m["id"]),
			Title:       strVal(m["title"]),
			Description: strVal(m["description"]),
			Cmd:         strVal(m["cmd"]),
		}
		if gs, ok := asStringSlice(m["groups"]); ok {
			a.Groups = gs
		}
		if noWait, ok := m["noWait"].(bool); ok {
			a.NoWait = noWait
		}
		if a.Title != "" || a.Cmd != "" {
			result = append(result, a)
		}
	}
	return result
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Load resolves the config file automatically: next to the binary first,
// then the working directory. On Windows, config-win.yaml takes precedence.
// Pass an explicit path via LoadFrom to override this behaviour.
func Load() *Config {
	name := "config.yaml"
	if runtime.GOOS == "windows" {
		name = "config-win.yaml"
	}

	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), name))
	}
	paths = append(paths, name)

	return loadPaths(paths)
}

// LoadFrom loads a config from an explicit file path.
func LoadFrom(path string) *Config {
	return loadPaths([]string{path})
}

func loadPaths(paths []string) *Config {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		return &cfg
	}
	return &Config{}
}
