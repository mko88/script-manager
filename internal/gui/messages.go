package gui

import (
	"path/filepath"

	"script-manager/internal/messages"
)

// GetMessages returns this app's message-text overrides, reconciling its
// runtime override file against the compiled defaults on every call (see
// messages.LoadOrSync — this both seeds the file on first run and syncs it
// against defaults on every later run, adding newly added keys and
// dropping ones that no longer exist). Also refreshes both apps' read-only
// defaults snapshots (see messages.RefreshDefaultsSnapshots) regardless of
// which app is running. The frontend merges the returned override over its
// own compiled messages.json at startup, falling back per-key on any error
// or missing key.
func (a *App) GetMessages() (map[string]interface{}, error) {
	messages.RefreshDefaultsSnapshots(a.appDataDir)
	return messages.LoadOrSync(filepath.Join(a.appDataDir, messages.GUIFilename), messages.GUI)
}
