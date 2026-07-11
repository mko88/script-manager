package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadDefaultsToDarkWhenFileMissing(t *testing.T) {
	if got := Load(t.TempDir()); got.Active != "dark" || got.Themes != nil || got.Custom != nil {
		t.Errorf("Load() = %+v, want {dark <nil> <nil>}", got)
	}
}

func TestLoadDefaultsToDarkForGarbageContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Active != "dark" {
		t.Errorf("Load() = %+v, want Active dark", got)
	}
}

func TestLoadNormalizesUnknownActive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(`{"active":"sepia"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Active != "dark" {
		t.Errorf("Load() = %+v, want Active dark", got)
	}
}

func TestLoadNormalizesActiveNamingMissingTheme(t *testing.T) {
	dir := t.TempDir()
	body := `{"active":"Ghost","themes":{"Other":{"bg":"#111111"}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Active != "dark" {
		t.Errorf("Load() = %+v, want Active dark (Ghost isn't a saved theme)", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := State{
		Active: "Custom",
		Themes: map[string]map[string]string{
			"Custom": {"bg": "#111111", "accent": "#00ffee"},
		},
	}
	if err := Save(dir, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got := Load(dir)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestMultipleThemesCoexist(t *testing.T) {
	dir := t.TempDir()
	want := State{
		Active: "Ocean",
		Themes: map[string]map[string]string{
			"Ocean":  {"bg": "#001133"},
			"Forest": {"bg": "#113300"},
		},
	}
	if err := Save(dir, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got := Load(dir)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadFallsBackToLegacyPlainTextFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, legacyFilename), []byte("light"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(dir)
	if got.Active != "light" || got.Custom != nil {
		t.Errorf("Load() = %+v, want {light <nil>}", got)
	}
}

func TestLoadPrefersNewFileOverLegacy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, legacyFilename), []byte("light"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(dir, State{Active: "dark"}); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got.Active != "dark" {
		t.Errorf("Load() = %+v, want Active dark (from sm-theme.json, not the legacy file)", got)
	}
}

func TestLoadMigratesOldCustomFormat(t *testing.T) {
	dir := t.TempDir()
	body := `{"active":"custom","custom":{"bg":"#101010","accent":"#ff8800"}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Load(dir)
	if got.Active != "Custom" {
		t.Errorf("Load().Active = %q, want Custom", got.Active)
	}
	if got.Custom != nil {
		t.Errorf("Load().Custom = %v, want nil (migrated away)", got.Custom)
	}
	want := map[string]string{"bg": "#101010", "accent": "#ff8800"}
	if !reflect.DeepEqual(got.Themes["Custom"], want) {
		t.Errorf("Load().Themes[\"Custom\"] = %v, want %v", got.Themes["Custom"], want)
	}

	// A load-then-save cycle finishes the migration for good: the file on
	// disk should never carry the legacy "custom" key again.
	if err := Save(dir, got); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"custom"`) {
		t.Errorf("saved file still contains the legacy \"custom\" key: %s", raw)
	}
}
