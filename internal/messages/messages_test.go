package messages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsFor(t *testing.T) {
	if got, err := DefaultsFor("gui"); err != nil || string(got) != string(GUI) {
		t.Errorf("DefaultsFor(gui) = (%v, %v)", got != nil, err)
	}
	if got, err := DefaultsFor("configedit"); err != nil || string(got) != string(ConfigEdit) {
		t.Errorf("DefaultsFor(configedit) = (%v, %v)", got != nil, err)
	}
	if _, err := DefaultsFor("bogus"); err == nil {
		t.Error("expected an error for an unknown target")
	}
}

func TestFilenameFor(t *testing.T) {
	if got, err := FilenameFor("gui"); err != nil || got != GUIFilename {
		t.Errorf("FilenameFor(gui) = (%q, %v)", got, err)
	}
	if got, err := FilenameFor("configedit"); err != nil || got != ConfigEditFilename {
		t.Errorf("FilenameFor(configedit) = (%q, %v)", got, err)
	}
	if _, err := FilenameFor("bogus"); err == nil {
		t.Error("expected an error for an unknown target")
	}
}

func TestRefreshDefaultsSnapshotsWritesBoth(t *testing.T) {
	dir := t.TempDir()
	RefreshDefaultsSnapshots(dir)

	got, err := os.ReadFile(filepath.Join(dir, GUIDefaultsFilename))
	if err != nil || string(got) != string(GUI) {
		t.Errorf("gui defaults snapshot = (%v, %v), want a copy of GUI", got != nil, err)
	}
	got, err = os.ReadFile(filepath.Join(dir, ConfigEditDefaultsFilename))
	if err != nil || string(got) != string(ConfigEdit) {
		t.Errorf("configedit defaults snapshot = (%v, %v), want a copy of ConfigEdit", got != nil, err)
	}
}

func TestSyncKeysBackfillsMissingKeys(t *testing.T) {
	override := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Custom saved"},
	}
	defaults := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved", "failed": "Failed"},
	}

	synced, changed := SyncKeys(override, defaults)
	if !changed {
		t.Error("expected changed = true when a key is backfilled")
	}
	toast := synced["toast"].(map[string]interface{})
	if toast["saved"] != "Custom saved" {
		t.Errorf("existing override value overwritten: %v", toast["saved"])
	}
	if toast["failed"] != "Failed" {
		t.Errorf("missing key not backfilled: %v", toast)
	}
}

func TestSyncKeysRemovesStaleKeys(t *testing.T) {
	override := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved", "stale": "gone in newer version"},
	}
	defaults := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved"},
	}

	synced, changed := SyncKeys(override, defaults)
	if !changed {
		t.Error("expected changed = true when a stale key is removed")
	}
	toast := synced["toast"].(map[string]interface{})
	if _, exists := toast["stale"]; exists {
		t.Error("stale key should have been removed")
	}
}

func TestSyncKeysRemovesStaleCategory(t *testing.T) {
	override := map[string]interface{}{
		"toast":       map[string]interface{}{"saved": "Saved"},
		"oldCategory": map[string]interface{}{"x": "y"},
	}
	defaults := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved"},
	}

	synced, changed := SyncKeys(override, defaults)
	if !changed {
		t.Error("expected changed = true when a whole category is removed")
	}
	if _, exists := synced["oldCategory"]; exists {
		t.Error("stale category should have been removed")
	}
}

func TestSyncKeysNoOpWhenAlreadyInSync(t *testing.T) {
	override := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved"},
	}
	defaults := map[string]interface{}{
		"toast": map[string]interface{}{"saved": "Saved"},
	}

	_, changed := SyncKeys(override, defaults)
	if changed {
		t.Error("expected changed = false when override already matches defaults' key set")
	}
}

func TestLoadOrSyncSeedsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	defaults := []byte(`{"toast":{"saved":"Saved"}}`)

	got, err := LoadOrSync(path, defaults)
	if err != nil {
		t.Fatalf("LoadOrSync() error = %v", err)
	}
	if toast, _ := got["toast"].(map[string]interface{}); toast["saved"] != "Saved" {
		t.Errorf("got = %v, want toast.saved = Saved", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to be seeded on disk: %v", err)
	}
}

func TestLoadOrSyncReconcilesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte(`{"toast":{"saved":"Custom","stale":"drop me"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	defaults := []byte(`{"toast":{"saved":"Saved","failed":"Failed"}}`)

	got, err := LoadOrSync(path, defaults)
	if err != nil {
		t.Fatalf("LoadOrSync() error = %v", err)
	}
	toast := got["toast"].(map[string]interface{})
	if toast["saved"] != "Custom" {
		t.Errorf("existing value overwritten: %v", toast["saved"])
	}
	if toast["failed"] != "Failed" {
		t.Errorf("missing key not backfilled: %v", toast)
	}
	if _, exists := toast["stale"]; exists {
		t.Errorf("stale key not removed: %v", toast)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var onDiskMap map[string]interface{}
	if err := json.Unmarshal(onDisk, &onDiskMap); err != nil {
		t.Fatalf("on-disk file isn't valid JSON: %v", err)
	}
}

func TestLoadOrSyncSkipsRewriteWhenAlreadyInSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	content := []byte(`{"toast":{"saved":"Saved"}}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	modTimeBefore := info.ModTime()

	if _, err := LoadOrSync(path, content); err != nil {
		t.Fatalf("LoadOrSync() error = %v", err)
	}

	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modTimeBefore) {
		t.Error("file was rewritten even though nothing changed")
	}
}

func TestLoadOrSyncInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrSync(path, []byte(`{}`)); err == nil {
		t.Error("expected an error for invalid on-disk JSON")
	}
}
