package gui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"script-manager/internal/config"
)

func TestLoadError(t *testing.T) {
	t.Run("surfaces the initial load error", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, errors.New("boom") })
		if got := a.LoadError(); got != "boom" {
			t.Errorf("LoadError() = %q, want %q", got, "boom")
		}
	})
	t.Run("empty when the initial load succeeds", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, nil })
		if got := a.LoadError(); got != "" {
			t.Errorf("LoadError() = %q, want empty", got)
		}
	})
}

func TestGetActionGroups(t *testing.T) {
	t.Run("converts the catalog", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) {
			return &config.Config{ActionGroups: []config.ActionGroup{
				{ID: "safe", Title: "Safe", Color: "#2ca02c"},
				{ID: "diagnostics"},
			}}, nil
		})
		got := a.GetActionGroups()
		want := []ActionGroupDTO{
			{ID: "safe", Title: "Safe", Color: "#2ca02c"},
			{ID: "diagnostics"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetActionGroups() = %+v, want %+v", got, want)
		}
	})
	t.Run("empty catalog returns an empty (non-nil) slice", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, nil })
		got := a.GetActionGroups()
		if got == nil || len(got) != 0 {
			t.Errorf("GetActionGroups() = %#v, want a non-nil empty slice", got)
		}
	})
}

func TestReloadConfig(t *testing.T) {
	t.Run("total failure keeps the previous config and returns the error", func(t *testing.T) {
		calls := 0
		a := NewApp(func() (*config.Config, error) {
			calls++
			if calls == 1 {
				return &config.Config{SourcePath: "/ok.yaml", Shell: []string{"bash"}}, nil
			}
			return &config.Config{}, errors.New("boom")
		})
		warning, err := a.ReloadConfig()
		if err == nil {
			t.Fatal("expected an error")
		}
		if warning != "" {
			t.Errorf("warning = %q, want empty", warning)
		}
		if a.cfg.SourcePath != "/ok.yaml" {
			t.Errorf("previous config should be kept, got SourcePath %q", a.cfg.SourcePath)
		}
	})
	t.Run("fallback success surfaces a warning instead of an error", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) {
			return &config.Config{SourcePath: "/fallback.yaml"}, errors.New("config-win.yaml: boom")
		})
		warning, err := a.ReloadConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if warning == "" {
			t.Error("expected a non-empty warning")
		}
		if a.cfg.SourcePath != "/fallback.yaml" {
			t.Errorf("SourcePath = %q, want /fallback.yaml", a.cfg.SourcePath)
		}
	})
}

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

func TestKnownTerminalArgs(t *testing.T) {
	shellArgv := []string{"bash", "/tmp/s.sh"}
	tests := []struct {
		name string
		want []string
	}{
		{"wt", []string{"-w", wtWindowName, "new-tab", "--title", "t", "-d", "/opt/app", "--", "bash", "/tmp/s.sh"}},
		{"cmd", []string{"/c", "start", "t", "/D", "/opt/app", "bash", "/tmp/s.sh"}},
		{"x-terminal-emulator", []string{"-T", "t", "-e", "bash", "/tmp/s.sh"}},
		{"gnome-terminal", []string{"--title", "t", "--working-directory", "/opt/app", "--", "bash", "/tmp/s.sh"}},
		{"konsole", []string{"--workdir", "/opt/app", "-e", "bash", "/tmp/s.sh"}},
		{"xfce4-terminal", []string{"-T", "t", "--working-directory", "/opt/app", "-x", "bash", "/tmp/s.sh"}},
		{"terminator", []string{"-T", "t", "--working-directory", "/opt/app", "-x", "bash", "/tmp/s.sh"}},
		{"foot", []string{"-T", "t", "-D", "/opt/app", "bash", "/tmp/s.sh"}},
		{"xterm", []string{"-T", "t", "-e", "bash", "/tmp/s.sh"}},
		{"kitty", []string{"--title", "t", "--directory", "/opt/app", "bash", "/tmp/s.sh"}},
		{"alacritty", []string{"-T", "t", "--working-directory", "/opt/app", "-e", "bash", "/tmp/s.sh"}},
		{"wezterm", []string{"start", "--cwd", "/opt/app", "--", "bash", "/tmp/s.sh"}},
	}
	for _, tt := range tests {
		lt, ok := knownTerminals[tt.name]
		if !ok {
			t.Errorf("%s missing from knownTerminals", tt.name)
			continue
		}
		if got := lt.args("t", "/opt/app", shellArgv); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s args = %v, want %v", tt.name, got, tt.want)
		}
	}
	if len(knownTerminals) != len(tests) {
		t.Errorf("test covers %d terminals, knownTerminals has %d", len(tests), len(knownTerminals))
	}
}

func TestNamedTerminal(t *testing.T) {
	t.Run("unknown name", func(t *testing.T) {
		if _, err := namedTerminal("not-a-real-terminal"); err == nil {
			t.Error("expected an error for an unknown terminal name")
		}
	})
	t.Run("known name not on PATH", func(t *testing.T) {
		// konsole is unlikely to be installed in a headless test
		// environment; skip gracefully if it happens to be present rather
		// than asserting a specific environment.
		if _, err := namedTerminal("konsole"); err == nil {
			t.Skip("konsole happens to be on PATH in this environment")
		}
	})
}

func TestCustomTerminal(t *testing.T) {
	t.Run("empty template", func(t *testing.T) {
		if _, err := customTerminal(nil, "t", "/opt/app"); err == nil {
			t.Error("expected an error for an empty argv template")
		}
	})
	t.Run("binary not on PATH", func(t *testing.T) {
		if _, err := customTerminal([]string{"not-a-real-binary"}, "t", "/opt/app"); err == nil {
			t.Error("expected an error for a binary not on PATH")
		}
	})
	t.Run("substitutes placeholders and appends shellArgv", func(t *testing.T) {
		lt, err := customTerminal([]string{"bash", "-title", "{{title}}", "-dir", "{{dir}}"}, "t", "/opt/app")
		if err != nil {
			t.Fatalf("customTerminal: %v", err)
		}
		got := lt.args("", "", []string{"bash", "/tmp/s.sh"})
		want := []string{"-title", "t", "-dir", "/opt/app", "bash", "/tmp/s.sh"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("args = %v, want %v", got, want)
		}
	})
}

func TestResolveTerminal(t *testing.T) {
	t.Run("custom argv template takes precedence", func(t *testing.T) {
		cfg := config.TerminalConfig{Name: "wt", Argv: []string{"bash"}}
		lt, err := resolveTerminal(cfg, "linux", "t", "")
		if err != nil {
			t.Fatalf("resolveTerminal: %v", err)
		}
		if lt.bin != "bash" {
			t.Errorf("bin = %q, want %q", lt.bin, "bash")
		}
	})
	t.Run("named override skips auto-detection", func(t *testing.T) {
		cfg := config.TerminalConfig{Name: "not-a-real-terminal"}
		if _, err := resolveTerminal(cfg, "linux", "t", ""); err == nil {
			t.Error("expected the unknown name to surface as an error, not fall back to auto-detect")
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

func inlineTestApp(action config.Action) *App {
	return NewApp(func() (*config.Config, error) {
		return &config.Config{
			Shell:   []string{"bash", "-c"},
			Items:   []map[string]any{{"name": "test"}},
			Actions: []config.Action{action},
		}, nil
	})
}

func TestRunActionInline(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hello-inline"})

	result, err := a.RunActionInline(0, 0)
	if err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	if result.ExitCode != 0 || !strings.Contains(result.Output, "hello-inline") {
		t.Errorf("RunActionInline() = %+v, want exit 0 and output containing %q", result, "hello-inline")
	}
}

func TestRunActionInlineCapturesStderrAndNonZeroExit(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Fail", Cmd: "echo oops >&2; exit 3"})

	result, err := a.RunActionInline(0, 0)
	if err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	if result.ExitCode != 3 || !strings.Contains(result.Output, "oops") {
		t.Errorf("RunActionInline() = %+v, want exit 3 and output containing %q", result, "oops")
	}
}

func TestRunActionInlineInvalidItemOrAction(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})

	if _, err := a.RunActionInline(5, 0); err == nil {
		t.Error("expected an error for an out-of-range item")
	}
	if _, err := a.RunActionInline(0, 5); err == nil {
		t.Error("expected an error for an out-of-range action")
	}
}

// waitForInlineRunning blocks until a's inlineCmd is set — used by tests
// that need RunActionInline (a single blocking call) running on a
// background goroutine so the test's own goroutine can act concurrently
// against it (CancelInlineAction, or a second RunActionInline expected to be
// rejected).
func waitForInlineRunning(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.inlineMu.Lock()
		running := a.inlineCmd != nil
		a.inlineMu.Unlock()
		if running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("inline action never started running within the deadline")
}

func TestRunActionInlineRejectsConcurrentRuns(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Sleep", Cmd: "sleep 2"})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := a.RunActionInline(0, 0); err != nil {
			t.Errorf("first RunActionInline() error = %v", err)
		}
	}()
	waitForInlineRunning(t, a)

	if _, err := a.RunActionInline(0, 0); err == nil {
		t.Error("expected an error running a second inline action while the first is still running")
	}

	if err := a.CancelInlineAction(); err != nil {
		t.Fatalf("CancelInlineAction() error = %v", err)
	}
	<-done
}

func TestCancelInlineActionKillsProcessTree(t *testing.T) {
	// A child process the shell spawns and waits on, so cancel only truly
	// works if it kills the whole process group/tree — killing just the
	// shell would silently orphan this sleep, leaving it running.
	a := inlineTestApp(config.Action{Title: "Sleep", Cmd: "sleep 30"})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := a.RunActionInline(0, 0); err != nil {
			t.Errorf("RunActionInline() error = %v", err)
		}
	}()
	waitForInlineRunning(t, a)

	if err := a.CancelInlineAction(); err != nil {
		t.Fatalf("CancelInlineAction() error = %v", err)
	}
	<-done
}

func TestCancelInlineActionNoneRunning(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})
	if err := a.CancelInlineAction(); err == nil {
		t.Error("expected an error when no inline action is running")
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
