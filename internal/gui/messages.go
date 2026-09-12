package gui

import (
	"path/filepath"

	"script-manager/internal/messages"
)

func (a *App) GetMessages() (map[string]interface{}, error) {
	messages.RefreshDefaultsSnapshots(a.appDataDir)
	return messages.LoadOrSync(filepath.Join(a.appDataDir, messages.GUIFilename), messages.GUI)
}
