package configedit

import "testing"

func TestGetThemeDefaultsToDark(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if got := a.GetTheme(); got.Active != "dark" {
		t.Errorf("GetTheme() = %+v, want Active dark", got)
	}
}

func TestSaveCustomThemePersistsAndActivates(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	palette := map[string]string{"bg": "#101010", "accent": "#ff8800"}

	if err := a.SaveCustomTheme(palette); err != nil {
		t.Fatalf("SaveCustomTheme() error = %v", err)
	}

	got := a.GetTheme()
	if got.Active != "custom" {
		t.Errorf("GetTheme().Active = %q, want custom", got.Active)
	}
	if got.Custom["bg"] != "#101010" || got.Custom["accent"] != "#ff8800" {
		t.Errorf("GetTheme().Custom = %v, want the saved palette", got.Custom)
	}
}

func TestSetThemeSwitchesAwayFromCustomWithoutLosingIt(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveCustomTheme(map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if got := a.GetTheme(); got.Active != "light" || got.Custom["bg"] != "#101010" {
		t.Errorf("GetTheme() = %+v, want Active light with the custom palette still saved", got)
	}
}
