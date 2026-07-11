package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Reserved item keys. Every other key in an item map is free-form data for
// templates and the subprocess environment.
const (
	// KeyName is the item's display name, used in headers and titles.
	KeyName = "name"
	// KeyDisplay selects which DisplayConfig renders the item.
	KeyDisplay = "display"
	// KeyActions restricts the item to the global actions with these IDs.
	KeyActions = "actions"
	// KeyActionGroups restricts the item to global actions in these groups.
	KeyActionGroups = "actionGroups"
	// KeyCustomActions holds inline item-specific action definitions.
	KeyCustomActions = "customActions"
)

type Action struct {
	ID          string   `yaml:"id,omitempty"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	Cmd         string   `yaml:"cmd"`
	Groups      []string `yaml:"groups,omitempty"`
	NoWait      bool     `yaml:"noWait,omitempty"`
	// Interactive marks an action whose command needs to read from stdin
	// (e.g. a prompt), so it can only run in a real terminal — the GUI hides
	// "Run here" for it and inline.go's buildInlineCmd refuses to start it,
	// since an inline run's stdin is deliberately left disconnected.
	Interactive bool `yaml:"interactive,omitempty"`
}

// ActionGroup is the catalog entry for a group name: Action.Groups and an
// item's "actionGroups" key still just reference groups by plain string
// (matched against ID here), the same as before this existed — this only
// adds an optional friendlier title and a color for the UI to use instead
// of showing/coloring the bare ID everywhere.
type ActionGroup struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title,omitempty"`
	Color string `yaml:"color,omitempty"`
}

type DisplayConfig struct {
	Name    string `yaml:"name,omitempty"`
	List    string `yaml:"list,omitempty"`
	Details string `yaml:"details,omitempty"`
}

// DisplayList is a slice of DisplayConfig that can be unmarshalled from either
// a YAML sequence (new format) or a single mapping (legacy format).
type DisplayList []DisplayConfig

func (dl *DisplayList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var list []DisplayConfig
		if err := value.Decode(&list); err != nil {
			return err
		}
		*dl = list
	case yaml.MappingNode:
		var single DisplayConfig
		if err := value.Decode(&single); err != nil {
			return err
		}
		*dl = DisplayList{single}
	}
	return nil
}

// FindDisplay returns the DisplayConfig matching item["display"], or the first
// entry if no match is found or the item has no display key.
func FindDisplay(displays DisplayList, item map[string]any) DisplayConfig {
	if len(displays) == 0 {
		return DisplayConfig{}
	}
	if item != nil {
		if name, ok := item[KeyDisplay].(string); ok && name != "" {
			for _, d := range displays {
				if d.Name == name {
					return d
				}
			}
		}
	}
	return displays[0]
}

// TerminalConfig selects which terminal emulator the GUI's Run button opens
// actions in (internal/gui owns the built-in table and auto-detection; the
// TUI ignores this field entirely since it runs actions inline). The zero
// value means "auto-detect the most common terminal for this OS". A YAML
// scalar names one specific built-in terminal, skipping auto-detection; a
// YAML sequence gives a fully custom argv template for a terminal that isn't
// built in — the same string-or-list convention Shell already established.
type TerminalConfig struct {
	// Name is a key into the GUI's built-in terminal table (e.g. "wt",
	// "gnome-terminal", "alacritty"), set when the config gave a plain string.
	Name string
	// Argv is a custom launch command, set when the config gave a list: the
	// first element is the terminal binary, the rest are its flags. Elements
	// may contain the "{{title}}" and "{{dir}}" placeholders; the resolved
	// shell command is always appended as the final arguments.
	Argv []string
}

func (t *TerminalConfig) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Decode(&t.Name)
	case yaml.SequenceNode:
		return value.Decode(&t.Argv)
	}
	return fmt.Errorf("terminal: expected a string or a list, got YAML node kind %v", value.Kind)
}

// MarshalYAML is the write-side counterpart to UnmarshalYAML, keeping the
// same scalar-or-list convention: Name marshals as a plain string, Argv as a
// sequence. A zero value marshals to nil, which combined with Config's
// terminal,omitempty tag omits the key entirely — reflect-based omitempty
// checks the field's own zero-ness before this method ever runs, so that
// omission doesn't actually depend on this returning nil, but it's returned
// for a consistent result if MarshalYAML is ever called directly.
func (t TerminalConfig) MarshalYAML() (interface{}, error) {
	if t.Name != "" {
		return t.Name, nil
	}
	if len(t.Argv) > 0 {
		return t.Argv, nil
	}
	return nil, nil
}

type Config struct {
	Shell        []string         `yaml:"shell,omitempty"`
	Display      DisplayList      `yaml:"display,omitempty"`
	Terminal     TerminalConfig   `yaml:"terminal,omitempty"`
	Env          map[string]any   `yaml:"env,omitempty"`
	Items        []map[string]any `yaml:"items,omitempty"`
	ActionGroups []ActionGroup    `yaml:"actionGroups,omitempty"`
	Actions      []Action         `yaml:"actions,omitempty"`

	// SourcePath is the absolute path of the file this config was actually
	// loaded from — not part of the YAML itself, but set by loadPaths so
	// callers (e.g. the #CONFIG_FILE# template placeholder) can show which of
	// several candidate paths (config-win.yaml vs. config.yaml, exe dir vs.
	// working dir) won.
	SourcePath string `yaml:"-"`
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

	allowedIDs, hasIDs := AsStringSlice(item[KeyActions])
	allowedGroups, hasGroups := AsStringSlice(item[KeyActionGroups])
	customRaw := item[KeyCustomActions]

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

	result = append(result, ParseCustomActions(customRaw)...)
	return result
}

// AsStringSlice reads v as a YAML sequence of strings — the shape item keys
// like "actions"/"actionGroups" (and a custom action's "groups") take once
// decoded into a map[string]any. Non-string elements are dropped rather than
// erroring, since a malformed entry shouldn't break the whole list. The bool
// reports whether v was a non-empty sequence at all, distinguishing "absent"
// from "present but empty" for callers like ActionsForItem that treat the two
// differently.
func AsStringSlice(v any) ([]string, bool) {
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

// ParseCustomActions decodes an item's "customActions" key (v) into inline
// Actions. An entry without a title or cmd is dropped, matching the same
// "nothing useful to run" rule ActionsForItem otherwise relies on.
func ParseCustomActions(v any) []Action {
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
			ID:          StrVal(m["id"]),
			Title:       StrVal(m["title"]),
			Description: StrVal(m["description"]),
			Cmd:         StrVal(m["cmd"]),
		}
		if gs, ok := AsStringSlice(m["groups"]); ok {
			a.Groups = gs
		}
		if noWait, ok := m["noWait"].(bool); ok {
			a.NoWait = noWait
		}
		if interactive, ok := m["interactive"].(bool); ok {
			a.Interactive = interactive
		}
		if a.Title != "" || a.Cmd != "" {
			result = append(result, a)
		}
	}
	return result
}

// StrVal reads v as a string, returning "" for nil or any other type.
func StrVal(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// LoadWithError resolves the config file automatically — next to the binary
// first, then the working directory — and reports the last error encountered
// (e.g. a missing file or a YAML syntax error) so callers can reload without
// losing the previous config on failure. On Windows, config-win.yaml takes
// precedence in both locations, falling back to config.yaml when absent.
// Use LoadFromWithError to load an explicit path instead.
func LoadWithError() (*Config, error) {
	names := []string{"config.yaml"}
	if runtime.GOOS == "windows" {
		names = []string{"config-win.yaml", "config.yaml"}
	}

	var exeDir string
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}

	var paths []string
	for _, name := range names {
		if exeDir != "" {
			paths = append(paths, filepath.Join(exeDir, name))
		}
		paths = append(paths, name)
	}

	return loadPaths(paths)
}

// LoadFromWithError loads a config from an explicit file path.
func LoadFromWithError(path string) (*Config, error) {
	return loadPaths([]string{path})
}

// loadPaths tries each candidate path in order and returns the config from
// the first one that parses. A candidate missing entirely is expected —
// config-win.yaml/config.yaml are each tried in two locations — and is
// skipped quietly. A candidate that exists but fails to parse (a YAML syntax
// error) is a real problem worth knowing about even if a later candidate
// succeeds, so that error is returned alongside the fallback config instead
// of being silently swallowed; callers can tell "loaded with a warning" apart
// from "nothing loaded at all" via cfg.SourcePath being non-empty.
func loadPaths(paths []string) (*Config, error) {
	var parseErr, lastErr error
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			wrapped := fmt.Errorf("%s: %w", p, err)
			if parseErr == nil {
				parseErr = wrapped
			}
			lastErr = wrapped
			continue
		}
		if abs, err := filepath.Abs(p); err == nil {
			cfg.SourcePath = abs
		} else {
			cfg.SourcePath = p
		}
		return &cfg, parseErr
	}
	if parseErr != nil {
		return &Config{}, parseErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no config file found (tried %s)", strings.Join(paths, ", "))
	}
	return &Config{}, lastErr
}
