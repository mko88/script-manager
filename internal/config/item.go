package config

import (
	"fmt"
	"maps"

	"gopkg.in/yaml.v3"
)

type Item struct {
	Name          string         `yaml:"name,omitempty"`
	Display       string         `yaml:"display,omitempty"`
	Actions       []string       `yaml:"actions,omitempty"`
	ActionGroups  []string       `yaml:"actionGroups,omitempty"`
	CustomActions []Action       `yaml:"customActions,omitempty"`
	Env           map[string]any `yaml:"env,omitempty"`
}

var itemKeys = map[string]bool{
	KeyName:          true,
	KeyDisplay:       true,
	KeyActions:       true,
	KeyActionGroups:  true,
	KeyCustomActions: true,
	KeyEnv:           true,
}

// ItemKey reports whether key is one of an item's six structural keys.
func ItemKey(key string) bool {
	return itemKeys[key]
}

// rawItem drops the UnmarshalYAML method so Decode doesn't recurse into it.
type rawItem Item

func (i *Item) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("item: expected a mapping, got YAML node kind %v", value.Kind)
	}
	var raw rawItem
	if err := value.Decode(&raw); err != nil {
		return err
	}
	for n := 0; n+1 < len(value.Content); n += 2 {
		key := value.Content[n].Value
		if itemKeys[key] {
			continue
		}
		name := raw.Name
		if name == "" {
			name = "(unnamed)"
		}
		return fmt.Errorf("item %q: unknown key %q — environment variables belong under %q", name, key, KeyEnv)
	}
	*i = Item(raw)
	return nil
}

// Values is the flat map templates and scripts see: the item's env, with name
// on top so a stray env key called "name" can't shadow the item's own.
func (i *Item) Values() map[string]any {
	out := make(map[string]any, len(i.Env)+1)
	maps.Copy(out, i.Env)
	if i.Name != "" {
		out[KeyName] = i.Name
	}
	return out
}
