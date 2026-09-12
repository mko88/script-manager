package configedit

import "script-manager/internal/openfile"

func (a *App) OpenInEditor() error {
	return openfile.Open(a.path)
}

func (a *App) OpenDataFolder() error {
	return openfile.Open(a.appDataDir)
}
