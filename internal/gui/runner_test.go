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
			name:     "posix shells strip -c and get the script appended",
			shell:    []string{"bash", "-c"},
			stayOpen: true,
			want:     []string{"bash", "s.sh"},
		},
		{
			name:     "posix shells keep other flags",
			shell:    []string{"/usr/bin/zsh", "--no-rcs"},
			stayOpen: false,
			want:     []string{"/usr/bin/zsh", "--no-rcs", "s.sh"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := "s.ps1"
			if strings.HasSuffix(tt.want[len(tt.want)-1], ".sh") {
				script = "s.sh"
			}
			got := buildShellArgv(tt.shell, script, tt.stayOpen)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildShellArgv = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapScript(t *testing.T) {
	t.Run("posix shells self-delete then run", func(t *testing.T) {
		got := wrapScript("bash", "echo hi", false)
		if !strings.HasPrefix(got, "rm -f -- \"$0\"\n") {
			t.Errorf("missing self-delete prologue: %q", got)
		}
		if !strings.Contains(got, "echo hi") {
			t.Errorf("missing command: %q", got)
		}
		if strings.Contains(got, "read -r") {
			t.Errorf("noWait script must not wait for a key: %q", got)
		}
	})
	t.Run("posix stayOpen waits for Enter", func(t *testing.T) {
		got := wrapScript("bash", "echo hi", true)
		if !strings.Contains(got, "read -r") {
			t.Errorf("stayOpen script must wait for a key: %q", got)
		}
		if idx := strings.Index(got, "echo hi"); idx > strings.Index(got, "read -r") {
			t.Errorf("wait epilogue must come after the command: %q", got)
		}
	})
	t.Run("pwsh self-deletes via PSCommandPath, no read epilogue", func(t *testing.T) {
		got := wrapScript("pwsh", "Get-Date", true)
		if !strings.HasPrefix(got, "Remove-Item -LiteralPath $PSCommandPath") {
			t.Errorf("missing self-delete prologue: %q", got)
		}
		if strings.Contains(got, "read -r") {
			t.Errorf("pwsh stays open via -NoExit, not a read epilogue: %q", got)
		}
	})
	t.Run("cmd self-deletes after the command, not before", func(t *testing.T) {
		got := wrapScript("cmd", "echo hi", true)
		if !strings.HasPrefix(got, "echo hi") {
			t.Errorf("cmd script must run the command first: %q", got)
		}
		delIdx := strings.Index(got, `del "%~f0"`)
		if delIdx < 0 {
			t.Fatalf("missing self-delete: %q", got)
		}
		if delIdx < strings.Index(got, "echo hi") {
			t.Errorf("self-delete must come after the command: %q", got)
		}
	})
}

func TestWriteTempScript(t *testing.T) {
	tests := []struct {
		shell   string
		wantExt string
	}{
		{"pwsh.exe", ".ps1"},
		{"powershell.exe", ".ps1"},
		{"cmd.exe", ".bat"},
		{"/usr/bin/bash", ".sh"},
		{"zsh", ".sh"},
		{"fish", ".txt"},
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
