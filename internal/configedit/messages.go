package configedit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"script-manager/internal/messages"
)

// GetMessages returns this app's own message-text overrides — see
// gui.App.GetMessages's doc comment for the full behavior (reconciled
// against defaults on every call, both apps' defaults snapshots
// refreshed); used by sm-config-edit's own startup bootstrap.
func (a *App) GetMessages() (map[string]interface{}, error) {
	messages.RefreshDefaultsSnapshots(a.exeDir)
	return messages.LoadOrSync(filepath.Join(a.exeDir, messages.ConfigEditFilename), messages.ConfigEdit)
}

// GetEditableMessages returns the current message text for the Messages
// section's editor, for either target. The "configedit" case delegates to
// GetMessages (self). The "gui" case reads script-manager-gui's own
// override file directly and reconciles it against script-manager-gui's
// defaults the same way GetMessages does for its own file — but only in
// memory: this process isn't script-manager-gui, so writing a fix to its
// override file behind its back would be presumptuous; the reconciled view
// only lands on disk if the user clicks Save. A script-manager-gui that
// has never run yet (and so never wrote its own override file) surfaces a
// clear, actionable error instead of silently producing an empty form.
func (a *App) GetEditableMessages(target string) (map[string]interface{}, error) {
	if target == "configedit" {
		return a.GetMessages()
	}
	filename, err := messages.FilenameFor(target)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(a.exeDir, filename))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("script-manager-gui hasn't generated its message file yet — run it at least once, then reopen this section")
		}
		return nil, err
	}
	var override map[string]interface{}
	if err := json.Unmarshal(data, &override); err != nil {
		return nil, err
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
	return os.WriteFile(filepath.Join(a.exeDir, filename), out, 0o644)
}
