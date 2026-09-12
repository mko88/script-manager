package filewatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitForChange(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case p := <-ch:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a change notification")
		return ""
	}
}

func TestNotifiesOnWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.ps1")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	changes := make(chan string, 4)
	w, err := New(func(p string) { changes <- p })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()
	w.Follow(path)

	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := waitForChange(t, changes); got != path {
		t.Errorf("notified for %q, want %q", got, path)
	}
}

func TestIgnoresOtherFilesInTheSameDirectory(t *testing.T) {
	dir := t.TempDir()
	watched := filepath.Join(dir, "watched.ps1")
	other := filepath.Join(dir, "other.ps1")
	if err := os.WriteFile(watched, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	changes := make(chan string, 4)
	w, err := New(func(p string) { changes <- p })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()
	w.Follow(watched)

	if err := os.WriteFile(other, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-changes:
		t.Errorf("notified for %q, which is not the watched file", p)
	case <-time.After(700 * time.Millisecond):
	}
}

func TestFollowSwitchesFiles(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.ps1")
	second := filepath.Join(dir, "second.ps1")
	for _, p := range []string{first, second} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	changes := make(chan string, 4)
	w, err := New(func(p string) { changes <- p })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	w.Follow(first)
	w.Follow(second)

	if err := os.WriteFile(second, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := waitForChange(t, changes); got != second {
		t.Errorf("notified for %q, want the file now followed, %q", got, second)
	}
}

func TestFollowEmptyStopsWatching(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.ps1")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	changes := make(chan string, 4)
	w, err := New(func(p string) { changes <- p })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	w.Follow(path)
	w.Follow("")

	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-changes:
		t.Errorf("notified for %q after watching was stopped", p)
	case <-time.After(700 * time.Millisecond):
	}
}
