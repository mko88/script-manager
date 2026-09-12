package gui

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

func (a *App) LoadRecentConfig(path string) error {
	cfg, err := config.LoadFromWithError(path)
	if err != nil {
		recent.Remove(a.appDataDir, path)
		return err
	}
	a.cfg = cfg
	a.load = func() (*config.Config, error) { return config.LoadFromWithError(path) }
	recent.Add(a.appDataDir, cfg.SourcePath)
	return nil
}
