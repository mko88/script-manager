package configedit

import "script-manager/internal/uiprefs"

func (a *App) GetUIPrefs() uiprefs.Prefs {
	return uiprefs.Load(a.appDataDir)
}

func (a *App) SetUIScale(percent int) uiprefs.Prefs {
	p := uiprefs.Load(a.appDataDir)
	p.ScalePercent = percent
	saved, _ := uiprefs.Save(a.appDataDir, p)
	return saved
}

func (a *App) SetUIFonts(fontUI, fontMono string) uiprefs.Prefs {
	p := uiprefs.Load(a.appDataDir)
	p.FontUI = fontUI
	p.FontMono = fontMono
	saved, _ := uiprefs.Save(a.appDataDir, p)
	return saved
}
