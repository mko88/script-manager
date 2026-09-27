package configedit

import (
	"errors"

	"script-manager/internal/configmigrate"
	"script-manager/internal/recent"
)

func (a *App) RecentConfigs() []string {
	return recent.Load(a.appDataDir)
}

func (a *App) ClearRecentConfigs() []string {
	return recent.Clear(a.appDataDir)
}

func (a *App) OpenRecent(path string) (StateDTO, error) {
	// Declining leaves the config already open in place.
	if err := a.ensureFormat(path); errors.Is(err, configmigrate.ErrDeclined) {
		return a.currentState(), nil
	}

	state, err := a.openPath(path)
	if err != nil {
		recent.Remove(a.appDataDir, path)
		return StateDTO{}, err
	}
	return state, nil
}
