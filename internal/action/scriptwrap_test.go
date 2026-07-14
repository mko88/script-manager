package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrapScriptFile(t *testing.T) {
	t.Run("posix shells self-delete then invoke the script path", func(t *testing.T) {
		got := WrapScriptFile("bash", "/opt/deploy.sh", false)
		if !strings.HasPrefix(got, "rm -f -- \"$0\"\n") {
			t.Errorf("missing self-delete prologue: %q", got)
		}
		if !strings.Contains(got, "/opt/deploy.sh") {
			t.Errorf("missing script invocation: %q", got)
		}
		if strings.Contains(got, "read -r") {
			t.Errorf("noWait script must not wait for a key: %q", got)
		}
	})
	t.Run("posix stayOpen waits for Enter after the script", func(t *testing.T) {
		got := WrapScriptFile("bash", "/opt/deploy.sh", true)
		if !strings.Contains(got, "read -r") {
			t.Errorf("stayOpen script must wait for a key: %q", got)
		}
		if idx := strings.Index(got, "deploy.sh"); idx > strings.Index(got, "read -r") {
			t.Errorf("wait epilogue must come after the script invocation: %q", got)
		}
	})
	t.Run("pwsh self-deletes via PSCommandPath and uses the call operator", func(t *testing.T) {
		got := WrapScriptFile("pwsh", "C:\\scripts\\deploy.ps1", true)
		if !strings.HasPrefix(got, "Remove-Item -LiteralPath $PSCommandPath") {
			t.Errorf("missing self-delete prologue: %q", got)
		}
		if !strings.Contains(got, "& '") {
			t.Errorf("missing call-operator invocation: %q", got)
		}
		if strings.Contains(got, "read -r") {
			t.Errorf("pwsh stays open via -NoExit, not a read epilogue: %q", got)
		}
	})
	t.Run("cmd calls the script then self-deletes after", func(t *testing.T) {
		got := WrapScriptFile("cmd", "C:\\scripts\\deploy.bat", true)
		if !strings.HasPrefix(got, `call "C:\scripts\deploy.bat"`) {
			t.Errorf("cmd script must call the script first: %q", got)
		}
		delIdx := strings.Index(got, `del "%~f0"`)
		if delIdx < 0 {
			t.Fatalf("missing self-delete: %q", got)
		}
		if delIdx < strings.Index(got, "call") {
			t.Errorf("self-delete must come after the call: %q", got)
		}
	})
}

func TestPsQuoteShQuote(t *testing.T) {
	if got := psQuote(`C:\a b\c'd.ps1`); got != `'C:\a b\c''d.ps1'` {
		t.Errorf("psQuote = %q", got)
	}
	if got := shQuote(`/a b/c'd.sh`); got != `'/a b/c'\''d.sh'` {
		t.Errorf("shQuote = %q", got)
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
		{"/usr/bin/bash", ".sh"},
		{"zsh", ".sh"},
		{"fish", ".txt"},
	}
	for _, tt := range tests {
		path, err := WriteTempScript(tt.shell, "echo hi")
		if err != nil {
			t.Fatalf("WriteTempScript(%q): %v", tt.shell, err)
		}
		t.Cleanup(func() { os.Remove(path) })

		if got := filepath.Ext(path); got != tt.wantExt {
			t.Errorf("WriteTempScript(%q) ext = %q, want %q", tt.shell, got, tt.wantExt)
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
