package main

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Action struct {
	Title string `yaml:"title"`
	Cmd   string `yaml:"cmd"`
}

type DisplayConfig struct {
	List    string `yaml:"list"`
	Details string `yaml:"details"`
}

type Config struct {
	Shell   []string         `yaml:"shell"`
	Display DisplayConfig    `yaml:"display"`
	Items   []map[string]any `yaml:"items"`
	Actions []Action         `yaml:"actions"`
}

func loadConfig() *Config {
	name := "config.yaml"
	if runtime.GOOS == "windows" {
		name = "config-win.yaml"
	}

	// Look next to the executable first, fall back to the working directory.
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), name))
	}
	paths = append(paths, name)

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
