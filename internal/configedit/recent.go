package configedit

import (
	"script-manager/internal/config"
	"script-manager/internal/recent"
)

func (a *App) RecentConfigs() []string {
	return recent.Load(a.appDataDir)
}

func (a *App) ClearRecentConfigs() []string {
	return recent.Clear(a.appDataDir)
}

func (a *App) OpenRecent(path string) (StateDTO, error) {
	cfg, err := config.LoadFromWithError(path)
	if err != nil {
		recent.Remove(a.appDataDir, path)
		return StateDTO{}, err
	}
	return a.stateFor(cfg), nil
}
