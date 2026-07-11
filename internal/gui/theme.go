package gui

import "script-manager/internal/theme"

// GetTheme returns the theme persisted by either app (see internal/theme),
// so this app's startup picks up a switch made in the other one.
func (a *App) GetTheme() string {
	return theme.Load(a.exeDir)
}

// SetTheme persists t for both apps to pick up on their next startup.
func (a *App) SetTheme(t string) error {
	return theme.Save(a.exeDir, t)
}
