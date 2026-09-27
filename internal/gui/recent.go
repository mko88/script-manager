package gui

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

// LoadRecentConfig returns a warning when the config opened but not cleanly,
// e.g. its local secrets file couldn't be read.
func (a *App) LoadRecentConfig(path string) (string, error) {
	// Declining leaves the config already open in place.
	if err := a.ensureFormat(path); errors.Is(err, configmigrate.ErrDeclined) {
		return "", nil
	}

	warning, err := a.openConfigFile(path)
	if err != nil {
		recent.Remove(a.appDataDir, path)
		return "", err
	}
	return warning, nil
}

// openConfigFile switches to the config at path. A config that loads with an
// error but still has a source (a broken local secrets file) opens anyway,
// with the error as a warning.
func (a *App) openConfigFile(path string) (string, error) {
	cfg, err := a.loadFrom(path)
	if cfg.SourcePath == "" {
		return "", err
	}
	a.configPath = path
	a.setConfig(cfg)
	recent.Add(a.appDataDir, cfg.SourcePath)
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}
