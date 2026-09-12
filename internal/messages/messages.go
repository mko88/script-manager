package messages

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed gui.json
var GUI []byte

//go:embed configedit.json
var ConfigEdit []byte

const (
	GUIFilename                = "script-manager-gui.messages.json"
	GUIDefaultsFilename        = "script-manager-gui.messages.defaults.json"
	ConfigEditFilename         = "sm-config-edit.messages.json"
	ConfigEditDefaultsFilename = "sm-config-edit.messages.defaults.json"
)

func DefaultsFor(target string) ([]byte, error) {
	switch target {
	case "gui":
		return GUI, nil
	case "configedit":
		return ConfigEdit, nil
	default:
		return nil, fmt.Errorf("unknown messages target %q", target)
	}
}

func FilenameFor(target string) (string, error) {
	switch target {
	case "gui":
		return GUIFilename, nil
	case "configedit":
		return ConfigEditFilename, nil
	default:
		return "", fmt.Errorf("unknown messages target %q", target)
	}
}

func RefreshDefaultsSnapshots(dir string) {
	_ = os.WriteFile(filepath.Join(dir, GUIDefaultsFilename), GUI, 0o644)
	_ = os.WriteFile(filepath.Join(dir, ConfigEditDefaultsFilename), ConfigEdit, 0o644)
}

func SyncKeys(override, defaults map[string]interface{}) (map[string]interface{}, bool) {
	changed := false

	for k, dv := range defaults {
		dsub, isCategory := dv.(map[string]interface{})
		if !isCategory {
			if _, exists := override[k]; !exists {
				override[k] = dv
				changed = true
			}
			continue
		}
		osub, ok := override[k].(map[string]interface{})
		if !ok || osub == nil {
			osub = map[string]interface{}{}
			changed = true
		}
		var subChanged bool
		osub, subChanged = SyncKeys(osub, dsub)
		if subChanged {
			changed = true
		}
		override[k] = osub
	}

	for k := range override {
		if _, ok := defaults[k]; !ok {
			delete(override, k)
			changed = true
		}
	}

	return override, changed
}

func LoadOrSync(path string, defaults []byte) (map[string]interface{}, error) {
	var defaultsMap map[string]interface{}
	if err := json.Unmarshal(defaults, &defaultsMap); err != nil {
		return nil, err
	}

	override := map[string]interface{}{}
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		if err := json.Unmarshal(data, &override); err != nil {
			return nil, err
		}
	case os.IsNotExist(readErr):
	default:
		return nil, readErr
	}

	synced, changed := SyncKeys(override, defaultsMap)
	if changed {
		if out, err := json.MarshalIndent(synced, "", "  "); err == nil {
			_ = os.WriteFile(path, out, 0o644)
		}
	}
	return synced, nil
}
