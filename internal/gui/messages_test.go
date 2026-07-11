package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrSeedMessagesSeedsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	defaults := []byte(`{"toast":{"saved":"Saved"}}`)

	got, err := LoadOrSeedMessages(path, defaults)
	if err != nil {
		t.Fatalf("LoadOrSeedMessages() error = %v", err)
	}
	if toast, _ := got["toast"].(map[string]interface{}); toast["saved"] != "Saved" {
		t.Errorf("got = %v, want toast.saved = Saved", got)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to be seeded on disk: %v", err)
	}
	if string(onDisk) != string(defaults) {
		t.Errorf("on-disk content = %q, want %q", onDisk, defaults)
	}
}

func TestLoadOrSeedMessagesReadsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte(`{"toast":{"saved":"Custom"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadOrSeedMessages(path, []byte(`{"toast":{"saved":"Default"}}`))
	if err != nil {
		t.Fatalf("LoadOrSeedMessages() error = %v", err)
	}
	toast, _ := got["toast"].(map[string]interface{})
	if toast["saved"] != "Custom" {
		t.Errorf("got = %v, want the existing on-disk value to win over defaults", got)
	}
}

func TestLoadOrSeedMessagesInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadOrSeedMessages(path, []byte(`{}`)); err == nil {
		t.Error("expected an error for invalid on-disk JSON")
	}
}

func TestGetMessagesUsesExeDirAndDefaults(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir, defaultMessages: []byte(`{"nav":{"items":"Items"}}`)}

	got, err := a.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	nav, _ := got["nav"].(map[string]interface{})
	if nav["items"] != "Items" {
		t.Errorf("got = %v, want nav.items = Items", got)
	}
	if _, err := os.Stat(filepath.Join(dir, GUIMessagesFilename)); err != nil {
		t.Errorf("expected %s to be seeded in %s: %v", GUIMessagesFilename, dir, err)
	}
}

func TestGetMessagesRefreshesDefaultsSnapshot(t *testing.T) {
	dir := t.TempDir()
	defaultsPath := filepath.Join(dir, GUIMessagesDefaultsFilename)
	if err := os.WriteFile(defaultsPath, []byte(`{"stale":"yes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{exeDir: dir, defaultMessages: []byte(`{"nav":{"items":"Items"}}`)}

	if _, err := a.GetMessages(); err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	got, err := os.ReadFile(defaultsPath)
	if err != nil {
		t.Fatalf("expected defaults snapshot to still exist: %v", err)
	}
	if string(got) != string(a.defaultMessages) {
		t.Errorf("defaults snapshot = %q, want it refreshed to %q", got, a.defaultMessages)
	}
}
