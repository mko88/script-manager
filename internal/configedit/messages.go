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

// GetMessages returns this app's own message-text overrides, self-seeding
// its runtime file from the compiled defaults (see SetDefaultMessages) on
// first run — used by sm-config-edit's own startup bootstrap, same as
// gui.App.GetMessages is for script-manager-gui.
func (a *App) GetMessages() (map[string]interface{}, error) {
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
