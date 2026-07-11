package configedit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"script-manager/internal/gui"
)

// configEditMessagesFilename is the runtime message-override file this app
// self-seeds next to its own executable — mirrors gui.GUIMessagesFilename.
const configEditMessagesFilename = "sm-config-edit.messages.json"

// configEditMessagesDefaultsFilename mirrors gui.GUIMessagesDefaultsFilename
// — a read-only snapshot of this app's own compiled defaults, refreshed on
// every startup, used for "Restore defaults" on the "configedit" target.
const configEditMessagesDefaultsFilename = "sm-config-edit.messages.defaults.json"

// messagesPathFor resolves the runtime messages file for either this app
// ("configedit") or its sibling ("gui"), both expected next to this
// executable (build.sh puts both binaries in the same bin/ directory).
func messagesPathFor(exeDir, target string) (path string, err error) {
	switch target {
	case "gui":
		return filepath.Join(exeDir, gui.GUIMessagesFilename), nil
	case "configedit":
		return filepath.Join(exeDir, configEditMessagesFilename), nil
	default:
		return "", fmt.Errorf("unknown messages target %q", target)
	}
}

// defaultsPathFor is messagesPathFor's counterpart for the read-only
// compiled-defaults snapshot rather than the user's editable override.
func defaultsPathFor(exeDir, target string) (path string, err error) {
	switch target {
	case "gui":
		return filepath.Join(exeDir, gui.GUIMessagesDefaultsFilename), nil
	case "configedit":
		return filepath.Join(exeDir, configEditMessagesDefaultsFilename), nil
	default:
		return "", fmt.Errorf("unknown messages target %q", target)
	}
}

// readMessagesFile reads and parses a messages JSON file, translating a
// missing file into the same "hasn't run yet" hint regardless of whether
// it's the editable override or the defaults snapshot being read — both are
// only ever missing because script-manager-gui (the one target this process
// doesn't self-seed for) has never run.
func readMessagesFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("script-manager-gui hasn't generated its message file yet — run it at least once, then reopen this section")
		}
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetMessages returns this app's own message-text overrides, self-seeding
// its runtime file from the compiled defaults (see SetDefaultMessages) on
// first run — used by sm-config-edit's own startup bootstrap, same as
// gui.App.GetMessages is for script-manager-gui. It also refreshes
// configEditMessagesDefaultsFilename on every call; see GetMessages' own
// doc comment in internal/gui for why.
func (a *App) GetMessages() (map[string]interface{}, error) {
	_ = os.WriteFile(filepath.Join(a.exeDir, configEditMessagesDefaultsFilename), a.defaultMessages, 0o644)
	return gui.LoadOrSeedMessages(filepath.Join(a.exeDir, configEditMessagesFilename), a.defaultMessages)
}

// GetEditableMessages returns the current message text for the Messages
// section's editor, for either target. Unlike GetMessages, the "gui" case
// never self-seeds — this process has no compiled defaults for
// script-manager-gui's text, only its own — so a script-manager-gui that
// has never run yet (and so never wrote its own messages file) surfaces a
// clear, actionable error instead of silently producing an empty form.
func (a *App) GetEditableMessages(target string) (map[string]interface{}, error) {
	if target == "configedit" {
		return a.GetMessages()
	}
	path, err := messagesPathFor(a.exeDir, target)
	if err != nil {
		return nil, err
	}
	return readMessagesFile(path)
}

// GetDefaultMessages returns the target's compiled-default message text
// (ignoring any edits already saved to its override file), for the
// Messages section's "Restore defaults" button. Populates the in-memory
// editor form only — the caller must still Save to persist it.
func (a *App) GetDefaultMessages(target string) (map[string]interface{}, error) {
	path, err := defaultsPathFor(a.exeDir, target)
	if err != nil {
		return nil, err
	}
	return readMessagesFile(path)
}

// SaveMessages writes the full message set back to disk for either target.
func (a *App) SaveMessages(target string, data map[string]interface{}) error {
	path, err := messagesPathFor(a.exeDir, target)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
