package gui

import (
	"testing"

	"script-manager/internal/theme"
)

func TestGetThemeDefaultsToDark(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if got := a.GetTheme(); got.Active != "dark" {
		t.Errorf("GetTheme() = %+v, want Active dark", got)
	}
}

func TestSetThemeRoundTrip(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SetTheme("light"); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}
	if got := a.GetTheme(); got.Active != "light" {
		t.Errorf("GetTheme() = %+v, want Active light", got)
	}
}

func TestSetThemePreservesExistingCustomPalette(t *testing.T) {
	dir := t.TempDir()
	// Simulate sm-config-edit having saved a custom palette out from under
	// this process — this app can't edit one, only switch to/from it, and
	// SetTheme must not clobber it when switching away and back.
	if err := theme.Save(dir, theme.State{Active: "custom", Custom: map[string]string{"bg": "#123456"}}); err != nil {
		t.Fatal(err)
	}

	a := &App{exeDir: dir}
	if err := a.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetTheme("custom"); err != nil {
		t.Fatal(err)
	}
	got := a.GetTheme()
	if got.Active != "custom" || got.Custom["bg"] != "#123456" {
		t.Errorf("GetTheme() = %+v, want Active custom with bg #123456 preserved", got)
	}
}
