package gui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestShellBasename(t *testing.T) {
	// Backslash paths are deliberately absent: filepath.Base splits them only
	// when the test itself runs on Windows, and these tests run on Linux.
	tests := map[string]string{
		"pwsh.exe":                   "pwsh",
		"cmd.exe":                    "cmd",
		"/usr/bin/bash":              "bash",
		"PowerShell.EXE":             "powershell",
		"C:/Program Files/pwsh/pwsh": "pwsh",
	}
	for in, want := range tests {
		if got := shellBasename(in); got != want {
			t.Errorf("shellBasename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildShellArgv(t *testing.T) {
	tests := []struct {
		name     string
		shell    []string
		stayOpen bool
		want     []string
	}{
		{
			name:     "pwsh strips -Command, adds -NoExit and -File",
			shell:    []string{"pwsh.exe", "-NoLogo", "-Command"},
			stayOpen: true,
			want:     []string{"pwsh.exe", "-NoLogo", "-NoExit", "-File", "s.ps1"},
		},
		{
			name:     "pwsh without stayOpen",
			shell:    []string{"pwsh.exe", "-Command"},
			stayOpen: false,
			want:     []string{"pwsh.exe", "-File", "s.ps1"},
		},
		{
			name:     "cmd stayOpen uses /k",
			shell:    []string{"cmd.exe", "/c"},
			stayOpen: true,
			want:     []string{"cmd.exe", "/k", "s.ps1"},
		},
		{
			name:     "cmd transient uses /c",
			shell:    []string{"cmd.exe"},
			stayOpen: false,
			want:     []string{"cmd.exe", "/c", "s.ps1"},
		},
		{
			name:     "other shells get the script appended",
			shell:    []string{"bash", "-c"},
			stayOpen: true,
			want:     []string{"bash", "-c", "s.ps1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildShellArgv(tt.shell, "s.ps1", tt.stayOpen)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildShellArgv = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteTempScript(t *testing.T) {
	tests := []struct {
		shell   string
		wantExt string
	}{
		{"pwsh.exe", ".ps1"},
		{"powershell.exe", ".ps1"},
		{"cmd.exe", ".bat"},
		{"bash", ".txt"},
	}
	for _, tt := range tests {
		path, err := writeTempScript(tt.shell, "echo hi")
		if err != nil {
			t.Fatalf("writeTempScript(%q): %v", tt.shell, err)
		}
		t.Cleanup(func() { os.Remove(path) })

		if got := filepath.Ext(path); got != tt.wantExt {
			t.Errorf("writeTempScript(%q) ext = %q, want %q", tt.shell, got, tt.wantExt)
		}
		if !strings.Contains(filepath.Base(path), "script-manager-action-") {
			t.Errorf("temp script %q does not match cleanup pattern", path)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "echo hi" {
			t.Errorf("temp script content = %q, err %v", data, err)
		}
	}
}

func TestCleanupTempScriptsIgnoresAge(t *testing.T) {
	f, err := os.CreateTemp("", tempScriptPattern+".ps1")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	// Backdate the file well past the old one-hour cutoff to prove cleanup no
	// longer looks at age at all.
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	cleanupTempScripts()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanupTempScripts left %q behind", path)
	}
}

func TestScheduleTempScriptCleanupRemovesFile(t *testing.T) {
	f, err := os.CreateTemp("", tempScriptPattern+".ps1")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()

	scheduleTempScriptCleanup(path)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	os.Remove(path)
	t.Errorf("scheduleTempScriptCleanup did not remove %q in time", path)
}
