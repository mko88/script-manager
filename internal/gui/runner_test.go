package gui

import (
	"os"
	"strings"
	"testing"
	"time"

	"script-manager/internal/action"
)

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

func TestCleanupTempScriptsIgnoresAge(t *testing.T) {
	f, err := os.CreateTemp("", action.TempScriptPattern+".ps1")
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
