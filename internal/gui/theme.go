package gui

import "script-manager/internal/theme"

func (a *App) GetTheme() theme.State {
	return theme.Load(a.appDataDir)
}

func (a *App) SetTheme(active string) error {
	s := theme.Load(a.appDataDir)
	s.Active = active
	return theme.Save(a.appDataDir, s)
}
