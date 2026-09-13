package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"script-manager/internal/appdata"
	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

//go:embed templates/default.yaml
var defaultConfigUnix string

//go:embed templates/default-win.yaml
var defaultConfigWindows string

func defaultConfigYAML() string {
	if runtime.GOOS == "windows" {
		return defaultConfigWindows
	}
	return defaultConfigUnix
}

const (
	KeyName          = "name"
	KeyDisplay       = "display"
	KeyActions       = "actions"
	KeyActionGroups  = "actionGroups"
	KeyCustomActions = "customActions"
	KeyEnv           = "env"
)

type Action struct {
	ID          string   `yaml:"id,omitempty"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	Cmd         string   `yaml:"cmd,omitempty"`
	Script      string   `yaml:"script,omitempty"`
	Groups      []string `yaml:"groups,omitempty"`
	NoWait      bool     `yaml:"noWait,omitempty"`
	Interactive bool     `yaml:"interactive,omitempty"`
	RequiresPIN bool     `yaml:"requiresPin,omitempty"`
}

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

func FindDisplay(displays DisplayList, item *Item) DisplayConfig {
	if len(displays) == 0 {
		return DisplayConfig{}
	}
	if item != nil && item.Display != "" {
		for _, d := range displays {
			if d.Name == item.Display {
				return d
			}
		}
	}
	return displays[0]
}

type TerminalConfig struct {
	Name string
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
	Shell        []string       `yaml:"shell,omitempty"`
	Display      DisplayList    `yaml:"display,omitempty"`
	Terminal     TerminalConfig `yaml:"terminal,omitempty"`
	Secrets      *secret.Params `yaml:"secrets,omitempty"`
	Env          map[string]any `yaml:"env,omitempty"`
	Items        []Item         `yaml:"items,omitempty"`
	ActionGroups []ActionGroup  `yaml:"actionGroups,omitempty"`
	Actions      []Action       `yaml:"actions,omitempty"`

	SourcePath string `yaml:"-"`
}

func ActionsForItem(allActions []Action, item *Item) []Action {
	if item == nil {
		return allActions
	}
	if len(item.Actions) == 0 && len(item.ActionGroups) == 0 && len(item.CustomActions) == 0 {
		return allActions
	}

	seen := make(map[int]bool)
	var result []Action

	if len(item.Actions) > 0 {
		idSet := make(map[string]bool, len(item.Actions))
		for _, id := range item.Actions {
			idSet[id] = true
		}
		for i, a := range allActions {
			if a.ID != "" && idSet[a.ID] && !seen[i] {
				result = append(result, a)
				seen[i] = true
			}
		}
	}

	if len(item.ActionGroups) > 0 {
		groupSet := make(map[string]bool, len(item.ActionGroups))
		for _, g := range item.ActionGroups {
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

	return append(result, item.CustomActions...)
}

func configNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"config-win.yaml", "config.yaml"}
	}
	return []string{"config.yaml"}
}

func searchPaths() []string {
	names := configNames()

	var exeDir string
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	dataDir := appdata.Dir()

	var paths []string
	for _, name := range names {
		if exeDir != "" {
			paths = append(paths, filepath.Join(exeDir, name))
		}
		paths = append(paths, name)
	}
	if dataDir != "" {
		for _, name := range names {
			paths = append(paths, filepath.Join(dataDir, name))
		}
	}
	return paths
}

// ResolvePath is the file LoadWithError would read, creating the starter
// config if nothing exists yet. Callers that must inspect the file before
// loading it — the format check on startup — use this to find it.
func ResolvePath() (string, error) {
	paths := searchPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	dataDir := appdata.Dir()
	if dataDir == "" {
		return "", fmt.Errorf("no config file found (tried %s)", strings.Join(paths, ", "))
	}
	defaultPath := filepath.Join(dataDir, configNames()[0])
	if err := os.WriteFile(defaultPath, []byte(defaultConfigYAML()), 0o644); err != nil {
		return "", err
	}
	return defaultPath, nil
}

func LoadWithError() (*Config, error) {
	paths := searchPaths()
	dataDir := appdata.Dir()
	if dataDir == "" {
		return loadPaths(paths)
	}
	return loadOrCreate(paths, filepath.Join(dataDir, configNames()[0]), defaultConfigYAML())
}

func LoadFromWithError(path string) (*Config, error) {
	return loadPaths([]string{path})
}

func loadOrCreate(paths []string, defaultPath, defaultYAML string) (*Config, error) {
	cfg, err := loadPaths(paths)
	if cfg.SourcePath != "" || anyExists(paths) {
		return cfg, err
	}
	if writeErr := os.WriteFile(defaultPath, []byte(defaultYAML), 0o644); writeErr != nil {
		return cfg, err
	}
	return loadPaths([]string{defaultPath})
}

func anyExists(paths []string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

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
