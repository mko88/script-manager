package configedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"script-manager/internal/messages"
)

func TestGetMessagesWritesOverrideAndDefaultsSnapshot(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir}

	got, err := a.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(got) == 0 {
		t.Error("expected a non-empty message map")
	}

	if _, err := os.Stat(filepath.Join(dir, messages.ConfigEditFilename)); err != nil {
		t.Errorf("expected the override file to be seeded: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, messages.GUIDefaultsFilename)); err != nil {
		t.Errorf("expected script-manager-gui's defaults snapshot to also be written (regardless of which app started): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, messages.ConfigEditDefaultsFilename)); err != nil {
		t.Errorf("expected sm-config-edit's own defaults snapshot to be written: %v", err)
	}
}

func TestGetEditableMessagesSelf(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir}

	got, err := a.GetEditableMessages("configedit")
	if err != nil {
		t.Fatalf("GetEditableMessages(configedit) error = %v", err)
	}
	nav, _ := got["nav"].(map[string]interface{})
	if nav["messages"] != "Messages" {
		t.Errorf("got = %v, want nav.messages = Messages", got)
	}
}

func TestGetEditableMessagesGuiMissingFile(t *testing.T) {
	a := &App{exeDir: t.TempDir()}

	_, err := a.GetEditableMessages("gui")
	if err == nil {
		t.Fatal("expected an error when script-manager-gui has never run")
	}
	if !strings.Contains(err.Error(), "run it at least once") {
		t.Errorf("error = %q, want a hint to run script-manager-gui first", err.Error())
	}
}

func TestGetEditableMessagesGuiReconcilesAgainstDefaults(t *testing.T) {
	dir := t.TempDir()
	guiPath := filepath.Join(dir, messages.GUIFilename)
	// A stale override missing keys defaults has, and carrying one defaults
	// no longer has — GetEditableMessages should reconcile this in memory
	// (without writing it back — that's the "in memory only" contract).
	if err := os.WriteFile(guiPath, []byte(`{"panel":{"stale":"drop me"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{exeDir: dir}

	got, err := a.GetEditableMessages("gui")
	if err != nil {
		t.Fatalf("GetEditableMessages(gui) error = %v", err)
	}
	panel, _ := got["panel"].(map[string]interface{})
	if panel["items"] != "Items" {
		t.Errorf("got = %v, want panel.items backfilled from defaults", got)
	}
	if _, exists := panel["stale"]; exists {
		t.Errorf("got = %v, want the stale key removed", got)
	}

	onDisk, err := os.ReadFile(guiPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "stale") {
		t.Error("on-disk file should be untouched by GetEditableMessages (only Save should persist changes)")
	}
}

func TestSaveMessagesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir}

	data := map[string]interface{}{"nav": map[string]interface{}{"items": "Edited"}}
	if err := a.SaveMessages("gui", data); err != nil {
		t.Fatalf("SaveMessages() error = %v", err)
	}

	onDisk, err := os.ReadFile(filepath.Join(dir, messages.GUIFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "Edited") {
		t.Errorf("on-disk content = %q, want it to contain the saved edit", onDisk)
	}
}

func TestSaveMessagesUnknownTarget(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveMessages("bogus", map[string]interface{}{}); err == nil {
		t.Error("expected an error for an unknown target")
	}
}

func TestGetDefaultMessagesWorksWithoutAnyFileOnDisk(t *testing.T) {
	// GetDefaultMessages reads compiled bytes, not a file — it must work
	// even in a directory where neither app has ever run.
	a := &App{exeDir: t.TempDir()}

	got, err := a.GetDefaultMessages("gui")
	if err != nil {
		t.Fatalf("GetDefaultMessages(gui) error = %v", err)
	}
	panel, _ := got["panel"].(map[string]interface{})
	if panel["items"] != "Items" {
		t.Errorf("got = %v, want panel.items = Items", got)
	}
}

func TestGetDefaultMessagesUnknownTarget(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if _, err := a.GetDefaultMessages("bogus"); err == nil {
		t.Error("expected an error for an unknown target")
	}
}
