package configedit

import (
	"encoding/json"
	"os"
	"path/filepath"

	"script-manager/internal/messages"
)

// GetMessages returns this app's own message-text overrides — see
// gui.App.GetMessages's doc comment for the full behavior (reconciled
// against defaults on every call, both apps' defaults snapshots
// refreshed); used by sm-config-edit's own startup bootstrap.
func (a *App) GetMessages() (map[string]interface{}, error) {
	messages.RefreshDefaultsSnapshots(a.appDataDir)
	return messages.LoadOrSync(filepath.Join(a.appDataDir, messages.ConfigEditFilename), messages.ConfigEdit)
}

// GetEditableMessages returns the current message text for the Messages
// section's editor, for either target. "configedit" delegates to
// GetMessages (self). "gui" reads script-manager-gui's own override file
// and reconciles it against that app's defaults in memory only — this
// process isn't script-manager-gui, so the reconciled view reaches disk
// only if the user clicks Save. A script-manager-gui that has never run
// (and so wrote no override file) isn't an error: this binary carries that
// app's compiled defaults too (see internal/messages), so the tab still
// shows real shipped text.
func (a *App) GetEditableMessages(target string) (map[string]interface{}, error) {
	if target == "configedit" {
		return a.GetMessages()
	}
	filename, err := messages.FilenameFor(target)
	if err != nil {
		return nil, err
	}
	override := map[string]interface{}{}
	data, readErr := os.ReadFile(filepath.Join(a.appDataDir, filename))
	switch {
	case readErr == nil:
		if err := json.Unmarshal(data, &override); err != nil {
			return nil, err
		}
	case os.IsNotExist(readErr):
		// No override yet — override starts empty; SyncKeys below backfills
		// every key from defaults.
	default:
		return nil, readErr
	}
	defaultsBytes, err := messages.DefaultsFor(target)
	if err != nil {
		return nil, err
	}
	var defaultsMap map[string]interface{}
	if err := json.Unmarshal(defaultsBytes, &defaultsMap); err != nil {
		return nil, err
	}
	synced, _ := messages.SyncKeys(override, defaultsMap)
	return synced, nil
}

// GetDefaultMessages returns the target's compiled-default message text —
// available regardless of whether that app has ever run, since its
// defaults are compiled into this binary too (see internal/messages) — for
// the Messages section's "Restore defaults" button. Populates the
// in-memory editor form only; the caller must still Save to persist it.
func (a *App) GetDefaultMessages(target string) (map[string]interface{}, error) {
	defaultsBytes, err := messages.DefaultsFor(target)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(defaultsBytes, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// SaveMessages writes the full message set back to disk for either target.
func (a *App) SaveMessages(target string, data map[string]interface{}) error {
	filename, err := messages.FilenameFor(target)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.appDataDir, filename), out, 0o644)
}
