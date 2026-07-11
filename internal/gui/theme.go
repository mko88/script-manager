package gui

import "script-manager/internal/theme"

// GetTheme returns the theme state persisted by either app (see
// internal/theme), so this app's startup picks up a switch — or a saved
// custom palette — made in the other one.
func (a *App) GetTheme() theme.State {
	return theme.Load(a.exeDir)
}

// SetTheme switches the active theme, keeping whatever custom palette is
// already persisted (this app can select "custom" but not edit it — only
// sm-config-edit's SaveCustomTheme does that).
func (a *App) SetTheme(active string) error {
	s := theme.Load(a.exeDir)
	s.Active = active
	return theme.Save(a.exeDir, s)
}
