package terminal

import (
	"reflect"
	"testing"

	"script-manager/internal/config"
)

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
		if got := lt.Args("t", "/opt/app", shellArgv); !reflect.DeepEqual(got, tt.want) {
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
		got := lt.Args("", "", []string{"bash", "/tmp/s.sh"})
		want := []string{"-title", "t", "-dir", "/opt/app", "bash", "/tmp/s.sh"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("args = %v, want %v", got, want)
		}
	})
}

func TestResolve(t *testing.T) {
	t.Run("custom argv template takes precedence", func(t *testing.T) {
		cfg := config.TerminalConfig{Name: "wt", Argv: []string{"bash"}}
		lt, err := Resolve(cfg, "linux", "t", "")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if lt.bin != "bash" {
			t.Errorf("bin = %q, want %q", lt.bin, "bash")
		}
	})
	t.Run("named override skips auto-detection", func(t *testing.T) {
		cfg := config.TerminalConfig{Name: "not-a-real-terminal"}
		if _, err := Resolve(cfg, "linux", "t", ""); err == nil {
			t.Error("expected the unknown name to surface as an error, not fall back to auto-detect")
		}
	})
}
