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

// SaveTheme creates or updates a named custom theme and makes it active.
// renamedFrom is the theme's previous name if it was renamed alongside
// this save ("" otherwise) — the old entry is removed so a rename and a
// color edit save as one atomic step instead of two round trips.
func (a *App) SaveTheme(name, renamedFrom string, palette map[string]string) error {
	s := theme.Load(a.exeDir)
	if s.Themes == nil {
		s.Themes = map[string]map[string]string{}
	}
	if renamedFrom != "" && renamedFrom != name {
		delete(s.Themes, renamedFrom)
	}
	s.Themes[name] = palette
	s.Active = name
	return theme.Save(a.exeDir, s)
}

// DeleteTheme removes a named custom theme, falling back to "dark" if it
// was the active one.
func (a *App) DeleteTheme(name string) error {
	s := theme.Load(a.exeDir)
	delete(s.Themes, name)
	if s.Active == name {
		s.Active = "dark"
	}
	return theme.Save(a.exeDir, s)
}
