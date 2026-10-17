// Package presets stores named sets of script arguments per action, in a
// presets.yaml beside the config. The config itself is never written.
package presets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = "presets.yaml"

type Preset struct {
	Name string `yaml:"name" json:"name"`
	// Values maps a parameter name to a bool, a string or a list of strings.
	Values map[string]any `yaml:"values,omitempty" json:"values"`
	// Args is free text appended to the command as written.
	Args string `yaml:"args,omitempty" json:"args"`
}

// File is the whole presets.yaml. Global actions are keyed by ActionKey;
// an item's custom actions sit under the item's name, so they can't
// collide with global ones.
type File struct {
	Actions       map[string][]Preset            `yaml:"actions,omitempty"`
	CustomActions map[string]map[string][]Preset `yaml:"customActions,omitempty"`
}

// Key locates an action's presets. Item is empty for a global action.
// Action is ActionKey's form of the action's id or title.
type Key struct {
	Item   string
	Action string
	Custom bool
}

// ActionKey names an action in the file: "id:<id>", or "title:<title>"
// for an action without an id, so the two can't collide.
func ActionKey(id, title string) string {
	if id != "" {
		return "id:" + id
	}
	return "title:" + title
}

func PathFor(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), FileName)
}

// Load reads path. A missing file is an empty one. A file that doesn't
// parse also yields an empty one, with the error to report.
func Load(path string) (*File, error) {
	f := &File{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := yaml.Unmarshal(data, f); err != nil {
		return &File{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	f.normalize()
	return f, nil
}

func (f *File) normalize() {
	for k, list := range f.Actions {
		f.Actions[k] = cleanList(list)
	}
	for item, actions := range f.CustomActions {
		for k, list := range actions {
			actions[k] = cleanList(list)
		}
		f.CustomActions[item] = actions
	}
}

func cleanList(list []Preset) []Preset {
	out := list[:0]
	for _, p := range list {
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			continue
		}
		p.Values = cleanValues(p.Values)
		out = append(out, p)
	}
	return out
}

// cleanValues keeps the shapes a parameter can take: bools, strings and
// lists of strings. Numbers become strings; anything else is dropped.
func cleanValues(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch x := v.(type) {
		case bool, string:
			out[k] = x
		case int, int64, float64:
			out[k] = fmt.Sprint(x)
		case []any:
			list := make([]string, 0, len(x))
			for _, e := range x {
				switch e.(type) {
				case string, bool, int, int64, float64:
					list = append(list, fmt.Sprint(e))
				}
			}
			out[k] = list
		}
	}
	return out
}

func (f *File) List(k Key) []Preset {
	var list []Preset
	if k.Custom {
		list = f.CustomActions[k.Item][k.Action]
	} else {
		list = f.Actions[k.Action]
	}
	return slices.Clone(list)
}

func (f *File) set(k Key, list []Preset) {
	if k.Custom {
		if f.CustomActions == nil {
			f.CustomActions = map[string]map[string][]Preset{}
		}
		if f.CustomActions[k.Item] == nil {
			f.CustomActions[k.Item] = map[string][]Preset{}
		}
		if len(list) == 0 {
			delete(f.CustomActions[k.Item], k.Action)
			if len(f.CustomActions[k.Item]) == 0 {
				delete(f.CustomActions, k.Item)
			}
			return
		}
		f.CustomActions[k.Item][k.Action] = list
		return
	}
	if f.Actions == nil {
		f.Actions = map[string][]Preset{}
	}
	if len(list) == 0 {
		delete(f.Actions, k.Action)
		return
	}
	f.Actions[k.Action] = list
}

// Put adds p, or replaces the preset of the same name.
func (f *File) Put(k Key, p Preset) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("a preset needs a name")
	}
	p.Values = cleanValues(p.Values)
	list := f.List(k)
	for i := range list {
		if list[i].Name == p.Name {
			list[i] = p
			f.set(k, list)
			return nil
		}
	}
	f.set(k, append(list, p))
	return nil
}

func (f *File) Delete(k Key, name string) {
	list := f.List(k)
	f.set(k, slices.DeleteFunc(list, func(p Preset) bool { return p.Name == name }))
}

// Save writes f to path through a temporary file in the same directory,
// so a reader never sees half a file.
func Save(path string, f *File) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+FileName+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
