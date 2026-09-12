package configedit

import "script-manager/internal/theme"

func (a *App) GetTheme() theme.State {
	return theme.Load(a.appDataDir)
}

func (a *App) SetTheme(active string) error {
	s := theme.Load(a.appDataDir)
	s.Active = active
	return theme.Save(a.appDataDir, s)
}

func (a *App) SaveTheme(name, renamedFrom string, palette map[string]string) error {
	s := theme.Load(a.appDataDir)
	if s.Themes == nil {
		s.Themes = map[string]map[string]string{}
	}
	if renamedFrom != "" && renamedFrom != name {
		delete(s.Themes, renamedFrom)
	}
	s.Themes[name] = palette
	s.Active = name
	return theme.Save(a.appDataDir, s)
}

func (a *App) DeleteTheme(name string) error {
	s := theme.Load(a.appDataDir)
	delete(s.Themes, name)
	if s.Active == name {
		s.Active = "dark"
	}
	return theme.Save(a.appDataDir, s)
}
