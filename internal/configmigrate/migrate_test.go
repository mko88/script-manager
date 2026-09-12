package configmigrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const oldShape = `shell: [bash, -c]
items:
  - name: srv1
    display: prod
    actions: [ssh]
    sshUser: root
    dbHost: db1
  - name: srv2
    env:
      region: eu
actions:
  - id: ssh
    title: SSH
    cmd: ssh {{.sshUser}}@{{.dbHost}}
`

const newShape = `shell: [bash, -c]
items:
  - name: srv1
    display: prod
    actions: [ssh]
    env:
      sshUser: root
actions:
  - id: ssh
    title: SSH
    cmd: ssh
`

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNeeded(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"old shape", oldShape, true},
		{"new shape", newShape, false},
		{"no items", "shell: [bash, -c]\n", false},
		{"empty file", "", false},
		{"empty item list", "items: []\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Needed(write(t, tt.body))
			if err != nil {
				t.Fatalf("Needed() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Needed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeededMalformedIsAnError(t *testing.T) {
	if _, err := Needed(write(t, "items: [\n")); err == nil {
		t.Fatal("Needed() error = nil, want a parse error")
	}
}

func TestNeededMissingFileIsAnError(t *testing.T) {
	if _, err := Needed(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("Needed() error = nil, want a read error")
	}
}

func TestBackupPath(t *testing.T) {
	now := time.Date(2026, 9, 13, 22, 45, 0, 0, time.UTC)
	got := BackupPath("/tmp/config-win.yaml", now)
	want := "/tmp/config-win.yaml.20260913-2245.bak"
	if got != want {
		t.Errorf("BackupPath() = %q, want %q", got, want)
	}
}

func TestConvert(t *testing.T) {
	path := write(t, oldShape)
	backup := path + ".bak"
	if err := Convert(path, backup); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	saved, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("backup not written: %v", err)
	}
	if string(saved) != oldShape {
		t.Error("backup is not the original bytes")
	}

	needed, err := Needed(path)
	if err != nil {
		t.Fatalf("Needed() after Convert error = %v", err)
	}
	if needed {
		t.Error("Needed() after Convert = true, want false")
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	for _, want := range []string{"sshUser: root", "dbHost: db1", "region: eu"} {
		if !strings.Contains(body, want) {
			t.Errorf("converted file lost %q:\n%s", want, body)
		}
	}
}

func TestConvertKeepsExistingEnv(t *testing.T) {
	path := write(t, "items:\n  - name: srv\n    env:\n      a: 1\n    b: 2\n")
	if err := Convert(path, path+".bak"); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	out, _ := os.ReadFile(path)
	body := string(out)
	if !strings.Contains(body, "a: 1") || !strings.Contains(body, "b: 2") {
		t.Errorf("Convert() dropped a key:\n%s", body)
	}
}

func TestEnsureConvertsWhenApproved(t *testing.T) {
	path := write(t, oldShape)
	var gotPath, gotBackup string
	backup, err := Ensure(path, func(p, b string) bool {
		gotPath, gotBackup = p, b
		return true
	})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if gotPath != path {
		t.Errorf("approve got path %q, want %q", gotPath, path)
	}
	if backup != gotBackup || backup == "" {
		t.Errorf("Ensure() = %q, approve saw %q", backup, gotBackup)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Errorf("backup %q not written: %v", backup, err)
	}
}

func TestEnsureDeclined(t *testing.T) {
	path := write(t, oldShape)
	_, err := Ensure(path, func(string, string) bool { return false })
	if !errors.Is(err, ErrDeclined) {
		t.Fatalf("Ensure() error = %v, want ErrDeclined", err)
	}
	out, _ := os.ReadFile(path)
	if string(out) != oldShape {
		t.Error("Ensure() rewrote the file after being declined")
	}
}

func TestEnsureNoopOnNewShape(t *testing.T) {
	path := write(t, newShape)
	called := false
	backup, err := Ensure(path, func(string, string) bool { called = true; return true })
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if called {
		t.Error("Ensure() asked for approval on an already-converted config")
	}
	if backup != "" {
		t.Errorf("Ensure() = %q, want empty", backup)
	}
}
