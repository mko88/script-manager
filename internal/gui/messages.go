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
func (a *App) GetMessages() (map[string]interface{}, error) {
	return LoadOrSeedMessages(filepath.Join(a.exeDir, GUIMessagesFilename), a.defaultMessages)
}
