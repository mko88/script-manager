package configedit

import "script-manager/internal/theme"

// GetTheme returns the theme state persisted by either app (see
// internal/theme), so this app's startup picks up a switch — or a saved
// custom palette — made in the other one.
func (a *App) GetTheme() theme.State {
	return theme.Load(a.exeDir)
}

// SetTheme switches the active theme, keeping whatever custom palette is
// already persisted.
func (a *App) SetTheme(active string) error {
	s := theme.Load(a.exeDir)
	s.Active = active
	return theme.Save(a.exeDir, s)
}

// SaveCustomTheme persists palette as the shared custom theme and makes it
// the active theme for both apps to pick up — editing a custom palette is
// config-editor-only, unlike switching to/from one.
func (a *App) SaveCustomTheme(palette map[string]string) error {
	return theme.Save(a.exeDir, theme.State{Active: "custom", Custom: palette})
}
