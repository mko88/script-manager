package gui

import (
	"testing"

	"script-manager/internal/theme"
)

func TestGetThemeDefaultsToDark(t *testing.T) {
	a := &App{appDataDir: t.TempDir()}
	if got := a.GetTheme(); got.Active != "dark" {
		t.Errorf("GetTheme() = %+v, want Active dark", got)
	}
}

func TestSetThemeRoundTrip(t *testing.T) {
	a := &App{appDataDir: t.TempDir()}
	if err := a.SetTheme("light"); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}
	if got := a.GetTheme(); got.Active != "light" {
		t.Errorf("GetTheme() = %+v, want Active light", got)
	}
}

func TestSetThemePreservesExistingCustomPalette(t *testing.T) {
	dir := t.TempDir()
	// Simulate sm-config-edit having saved a named custom theme out from
	// under this process — this app can't edit one, only switch to/from
	// it, and SetTheme must not clobber it when switching away and back.
	themes := map[string]map[string]string{"Custom": {"bg": "#123456"}}
	if err := theme.Save(dir, theme.State{Active: "Custom", Themes: themes}); err != nil {
		t.Fatal(err)
	}

	a := &App{appDataDir: dir}
	if err := a.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetTheme("Custom"); err != nil {
		t.Fatal(err)
	}
	got := a.GetTheme()
	if got.Active != "Custom" || got.Themes["Custom"]["bg"] != "#123456" {
		t.Errorf("GetTheme() = %+v, want Active Custom with bg #123456 preserved", got)
	}
}
