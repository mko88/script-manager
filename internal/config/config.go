package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Action struct {
	Title  string `yaml:"title"`
	Cmd    string `yaml:"cmd"`
	NoWait bool   `yaml:"noWait"`
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
