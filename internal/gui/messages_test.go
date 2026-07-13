package gui

import (
	"os"
	"path/filepath"
	"testing"

	"script-manager/internal/messages"
)

func TestGetMessagesWritesOverrideAndDefaultsSnapshot(t *testing.T) {
	dir := t.TempDir()
	a := &App{appDataDir: dir}

	got, err := a.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(got) == 0 {
		t.Error("expected a non-empty message map")
	}

	if _, err := os.Stat(filepath.Join(dir, messages.GUIFilename)); err != nil {
		t.Errorf("expected the override file to be seeded: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, messages.GUIDefaultsFilename)); err != nil {
		t.Errorf("expected script-manager-gui's defaults snapshot to be written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, messages.ConfigEditDefaultsFilename)); err != nil {
		t.Errorf("expected sm-config-edit's defaults snapshot to also be written (regardless of which app started): %v", err)
	}
}
