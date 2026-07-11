package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultsToDarkWhenFileMissing(t *testing.T) {
	if got := Load(t.TempDir()); got.Active != "dark" || got.Custom != nil {
		t.Errorf("Load() = %+v, want {dark <nil>}", got)
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

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := State{Active: "custom", Custom: map[string]string{"bg": "#111111", "accent": "#00ffee"}}
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
