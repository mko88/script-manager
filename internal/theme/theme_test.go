package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsToDarkWhenFileMissing(t *testing.T) {
	if got := Load(t.TempDir()); got != "dark" {
		t.Errorf("Load() = %q, want dark", got)
	}
}

func TestLoadDefaultsToDarkForGarbageContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte("not-a-theme"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got != "dark" {
		t.Errorf("Load() = %q, want dark", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "light"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if got := Load(dir); got != "light" {
		t.Errorf("Load() = %q, want light", got)
	}
}

func TestSaveNormalizesUnknownValueToDark(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "sepia"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if got := Load(dir); got != "dark" {
		t.Errorf("Load() = %q, want dark", got)
	}
}
