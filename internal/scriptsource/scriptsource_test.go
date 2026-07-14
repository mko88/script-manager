package scriptsource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "does-not-exist.sh"))
	if err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestReadDirectory(t *testing.T) {
	_, err := Read(t.TempDir())
	if err == nil {
		t.Error("expected an error for a directory")
	}
}

func TestReadTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.sh")
	if err := os.WriteFile(path, make([]byte, MaxBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("expected an error for a file over the size cap")
	}
}

func TestReadBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.sh")
	if err := os.WriteFile(path, []byte{0x00, 0xff, 0xfe, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("expected an error for non-UTF8 content")
	}
}

func TestReadReturnsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy.sh")
	content := "#!/bin/bash\necho \"hello\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("Read = %q, want %q", got, content)
	}
}
