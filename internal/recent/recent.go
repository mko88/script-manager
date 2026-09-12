package recent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const Filename = "sm-recent.json"

const Max = 10

type state struct {
	Paths []string `json:"paths"`
}

func Load(dir string) []string {
	if dir == "" {
		return []string{}
	}
	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		return []string{}
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return []string{}
	}
	return normalize(s.Paths)
}

func Add(dir, path string) []string {
	if dir == "" || path == "" {
		return Load(dir)
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return save(dir, normalize(append([]string{path}, Load(dir)...)))
}

func Remove(dir, path string) []string {
	kept := []string{}
	for _, p := range Load(dir) {
		if !samePath(p, path) {
			kept = append(kept, p)
		}
	}
	return save(dir, kept)
}

func Clear(dir string) []string {
	return save(dir, []string{})
}

func normalize(paths []string) []string {
	out := []string{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || contains(out, p) {
			continue
		}
		out = append(out, p)
		if len(out) == Max {
			break
		}
	}
	return out
}

func contains(paths []string, path string) bool {
	for _, p := range paths {
		if samePath(p, path) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func save(dir string, paths []string) []string {
	if dir == "" {
		return paths
	}
	data, err := json.MarshalIndent(state{Paths: paths}, "", "  ")
	if err != nil {
		return paths
	}
	os.WriteFile(filepath.Join(dir, Filename), data, 0o644)
	return paths
}
