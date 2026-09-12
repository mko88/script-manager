package configedit

import (
	"encoding/json"
	"os"
	"path/filepath"

	"script-manager/internal/messages"
)

func (a *App) GetMessages() (map[string]interface{}, error) {
	messages.RefreshDefaultsSnapshots(a.appDataDir)
	return messages.LoadOrSync(filepath.Join(a.appDataDir, messages.ConfigEditFilename), messages.ConfigEdit)
}

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
