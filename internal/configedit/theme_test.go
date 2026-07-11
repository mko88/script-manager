package configedit

import "testing"

func TestGetThemeDefaultsToDark(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if got := a.GetTheme(); got.Active != "dark" {
		t.Errorf("GetTheme() = %+v, want Active dark", got)
	}
}

func TestSaveThemeCreatesAndActivates(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	palette := map[string]string{"bg": "#101010", "accent": "#ff8800"}

	if err := a.SaveTheme("Custom", "", palette); err != nil {
		t.Fatalf("SaveTheme() error = %v", err)
	}

	got := a.GetTheme()
	if got.Active != "Custom" {
		t.Errorf("GetTheme().Active = %q, want Custom", got.Active)
	}
	if got.Themes["Custom"]["bg"] != "#101010" || got.Themes["Custom"]["accent"] != "#ff8800" {
		t.Errorf("GetTheme().Themes[\"Custom\"] = %v, want the saved palette", got.Themes["Custom"])
	}
}

func TestSaveThemeUpdatesInPlace(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveTheme("Custom", "", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveTheme("Custom", "Custom", map[string]string{"bg": "#202020"}); err != nil {
		t.Fatal(err)
	}

	got := a.GetTheme()
	if len(got.Themes) != 1 || got.Themes["Custom"]["bg"] != "#202020" {
		t.Errorf("GetTheme().Themes = %v, want a single updated Custom entry", got.Themes)
	}
}

func TestSaveThemeRenames(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveTheme("Custom", "", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveTheme("Ocean", "Custom", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}

	got := a.GetTheme()
	if got.Active != "Ocean" {
		t.Errorf("GetTheme().Active = %q, want Ocean", got.Active)
	}
	if _, exists := got.Themes["Custom"]; exists {
		t.Errorf("GetTheme().Themes still has the old \"Custom\" key: %v", got.Themes)
	}
	if got.Themes["Ocean"]["bg"] != "#101010" {
		t.Errorf("GetTheme().Themes[\"Ocean\"] = %v, want the renamed palette", got.Themes["Ocean"])
	}
}

func TestSetThemeSwitchesAwayFromCustomWithoutLosingIt(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveTheme("Custom", "", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if got := a.GetTheme(); got.Active != "light" || got.Themes["Custom"]["bg"] != "#101010" {
		t.Errorf("GetTheme() = %+v, want Active light with the custom palette still saved", got)
	}
}

func TestDeleteThemeFallsBackToDarkWhenActive(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveTheme("Custom", "", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteTheme("Custom"); err != nil {
		t.Fatalf("DeleteTheme() error = %v", err)
	}

	got := a.GetTheme()
	if got.Active != "dark" {
		t.Errorf("GetTheme().Active = %q, want dark", got.Active)
	}
	if _, exists := got.Themes["Custom"]; exists {
		t.Errorf("GetTheme().Themes still has the deleted theme: %v", got.Themes)
	}
}

func TestDeleteThemeKeepsActiveWhenUnrelated(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveTheme("Custom", "", map[string]string{"bg": "#101010"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveTheme("Ocean", "", map[string]string{"bg": "#001133"}); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteTheme("Custom"); err != nil {
		t.Fatalf("DeleteTheme() error = %v", err)
	}

	got := a.GetTheme()
	if got.Active != "Ocean" {
		t.Errorf("GetTheme().Active = %q, want Ocean (untouched by deleting Custom)", got.Active)
	}
	if _, exists := got.Themes["Custom"]; exists {
		t.Errorf("GetTheme().Themes still has the deleted theme: %v", got.Themes)
	}
}
