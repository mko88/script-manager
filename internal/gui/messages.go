package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GUIMessagesFilename is the runtime message-override file this app
// self-seeds next to its own executable. Exported so internal/configedit
// (which edits this file from across the process boundary) has a single
// source of truth for the name rather than re-hardcoding it.
const GUIMessagesFilename = "script-manager-gui.messages.json"

// GUIMessagesDefaultsFilename is a read-only snapshot of this app's
// compiled message defaults, rewritten on every startup (see GetMessages).
// It exists so sm-config-edit's "Restore defaults" can reset this app's
// messages back to what it ships with, even though sm-config-edit's own
// process has no compiled copy of script-manager-gui's default text.
const GUIMessagesDefaultsFilename = "script-manager-gui.messages.defaults.json"

// LoadOrSeedMessages reads the JSON message-override file at path, writing
// defaults there first if it doesn't exist yet. This guarantees the file
// exists after an app's first run, which is what lets sm-config-edit's
// Messages editor show real current text for a sibling app it has no
// compiled defaults of its own for.
func LoadOrSeedMessages(path string, defaults []byte) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.WriteFile(path, defaults, 0o644); err != nil {
			return nil, err
		}
		data = defaults
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetMessages returns this app's message-text overrides, self-seeding its
// runtime file from the compiled defaults (see SetDefaultMessages) on first
// run. The frontend merges this over its own compiled messages.json at
// startup, falling back per-key on any error or missing key.
//
// It also refreshes GUIMessagesDefaultsFilename on every call (i.e. every
// startup) from the same compiled defaults — best-effort; a failure here
// doesn't affect loading the actual override, only a later "restore
// defaults" in sm-config-edit's editor.
func (a *App) GetMessages() (map[string]interface{}, error) {
	_ = os.WriteFile(filepath.Join(a.exeDir, GUIMessagesDefaultsFilename), a.defaultMessages, 0o644)
	return LoadOrSeedMessages(filepath.Join(a.exeDir, GUIMessagesFilename), a.defaultMessages)
}
