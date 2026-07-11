// Package messages is the single source of truth for both GUI apps'
// default UI text, and the logic for keeping each app's on-disk override
// file in sync with it. Both script-manager-gui and sm-config-edit embed
// this package directly, so either one has compiled-in access to both
// apps' defaults regardless of which is launched first — no cross-process
// file bridging needed for sm-config-edit's "Restore defaults" or the
// startup sync described below.
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

// Filenames for each target's on-disk, user-editable override file, and a
// read-only compiled-defaults snapshot refreshed on every startup (see
// RefreshDefaultsSnapshots) purely for manual/on-disk reference.
const (
	GUIFilename                = "script-manager-gui.messages.json"
	GUIDefaultsFilename        = "script-manager-gui.messages.defaults.json"
	ConfigEditFilename         = "sm-config-edit.messages.json"
	ConfigEditDefaultsFilename = "sm-config-edit.messages.defaults.json"
)

// DefaultsFor returns the compiled default message bytes for target ("gui"
// or "configedit").
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

// FilenameFor returns the on-disk override filename for target.
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

// RefreshDefaultsSnapshots writes both apps' read-only compiled-defaults
// snapshots into dir, regardless of which app is running — so either is
// always available on disk for manual reference, even for the app that
// isn't the one currently running. Best-effort: a write failure here
// doesn't affect anything else (no code path depends on these files —
// GetDefaultMessages reads the compiled bytes directly).
func RefreshDefaultsSnapshots(dir string) {
	_ = os.WriteFile(filepath.Join(dir, GUIDefaultsFilename), GUI, 0o644)
	_ = os.WriteFile(filepath.Join(dir, ConfigEditDefaultsFilename), ConfigEdit, 0o644)
}

// SyncKeys reconciles an on-disk override against the current compiled
// defaults, recursively: a key defaults no longer has is deleted from
// override (a stale or renamed message), and a key defaults has that
// override doesn't is backfilled from defaults (a newly added message) —
// so an override file survives an app upgrade that adds or removes message
// keys without ever going stale (t() falling back to a missing key) or
// carrying dead entries forever. Only a value's presence is synced; an
// existing leaf value in override is never overwritten, so user edits
// always win. Returns the (mutated) override and whether anything changed.
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

// LoadOrSync reads the JSON message-override file at path — starting from
// an empty map if it doesn't exist yet — reconciles it against defaults via
// SyncKeys, and writes the result back to path only if something actually
// changed (so a file that's already fully in sync isn't rewritten on every
// startup).
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
		// override starts empty; SyncKeys backfills everything below.
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
