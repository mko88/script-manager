package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func actionTitles(actions []Action) []string {
	titles := make([]string, len(actions))
	for i, a := range actions {
		titles[i] = a.Title
	}
	return titles
}

func TestActionsForItem(t *testing.T) {
	all := []Action{
		{ID: "ssh", Title: "SSH", Groups: []string{"remote"}},
		{ID: "ping", Title: "Ping", Groups: []string{"net"}},
		{ID: "logs", Title: "Logs", Groups: []string{"remote", "debug"}},
	}

	tests := []struct {
		name string
		item map[string]any
		want []string
	}{
		{
			name: "nil item returns all",
			item: nil,
			want: []string{"SSH", "Ping", "Logs"},
		},
		{
			name: "no filter keys returns all",
			item: map[string]any{"name": "srv"},
			want: []string{"SSH", "Ping", "Logs"},
		},
		{
			name: "ids filter keeps global order",
			item: map[string]any{KeyActions: []any{"logs", "ssh"}},
			want: []string{"SSH", "Logs"},
		},
		{
			name: "unknown id is ignored",
			item: map[string]any{KeyActions: []any{"nope"}},
			want: []string{},
		},
		{
			name: "group filter",
			item: map[string]any{KeyActionGroups: []any{"net"}},
			want: []string{"Ping"},
		},
		{
			name: "ids and groups deduplicate",
			item: map[string]any{
				KeyActions:      []any{"logs"},
				KeyActionGroups: []any{"remote"},
			},
			want: []string{"Logs", "SSH"},
		},
		{
			name: "custom actions appended",
			item: map[string]any{
				KeyActions: []any{"ssh"},
				KeyCustomActions: []any{
					map[string]any{"title": "Custom", "cmd": "echo hi"},
				},
			},
			want: []string{"SSH", "Custom"},
		},
		{
			name: "custom actions only",
			item: map[string]any{
				KeyCustomActions: []any{
					map[string]any{"title": "Only", "cmd": "echo"},
					map[string]any{"description": "no title or cmd — dropped"},
				},
			},
			want: []string{"Only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionTitles(ActionsForItem(all, tt.item))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ActionsForItem = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCustomActionFields(t *testing.T) {
	item := map[string]any{
		KeyCustomActions: []any{
			map[string]any{
				"id":          "c1",
				"title":       "Custom",
				"description": "desc",
				"cmd":         "echo",
				"groups":      []any{"g1"},
				"noWait":      true,
				"interactive": true,
			},
		},
	}
	got := ActionsForItem(nil, item)
	want := []Action{{
		ID: "c1", Title: "Custom", Description: "desc",
		Cmd: "echo", Groups: []string{"g1"}, NoWait: true, Interactive: true,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("custom action = %+v, want %+v", got, want)
	}
}

func TestDisplayListUnmarshal(t *testing.T) {
	t.Run("sequence format", func(t *testing.T) {
		var dl DisplayList
		src := "- name: a\n  list: '{{.name}}'\n- name: b\n  list: '{{.id}}'\n"
		if err := yaml.Unmarshal([]byte(src), &dl); err != nil {
			t.Fatal(err)
		}
		if len(dl) != 2 || dl[0].Name != "a" || dl[1].Name != "b" {
			t.Errorf("got %+v", dl)
		}
	})

	t.Run("legacy single mapping", func(t *testing.T) {
		var dl DisplayList
		src := "name: solo\nlist: '{{.name}}'\ndetails: 'x'\n"
		if err := yaml.Unmarshal([]byte(src), &dl); err != nil {
			t.Fatal(err)
		}
		if len(dl) != 1 || dl[0].Name != "solo" || dl[0].Details != "x" {
			t.Errorf("got %+v", dl)
		}
	})
}

func TestTerminalConfigUnmarshal(t *testing.T) {
	t.Run("scalar names a built-in terminal", func(t *testing.T) {
		var tc TerminalConfig
		if err := yaml.Unmarshal([]byte("gnome-terminal"), &tc); err != nil {
			t.Fatal(err)
		}
		if tc.Name != "gnome-terminal" || tc.Argv != nil {
			t.Errorf("got %+v", tc)
		}
	})

	t.Run("sequence is a custom argv template", func(t *testing.T) {
		var tc TerminalConfig
		src := "- my-term\n- --title\n- '{{title}}'\n- --workdir\n- '{{dir}}'\n"
		if err := yaml.Unmarshal([]byte(src), &tc); err != nil {
			t.Fatal(err)
		}
		want := []string{"my-term", "--title", "{{title}}", "--workdir", "{{dir}}"}
		if tc.Name != "" || !reflect.DeepEqual(tc.Argv, want) {
			t.Errorf("got %+v", tc)
		}
	})

	t.Run("unset field defaults to auto-detect", func(t *testing.T) {
		var cfg Config
		if err := yaml.Unmarshal([]byte("shell: [bash]\n"), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Terminal.Name != "" || cfg.Terminal.Argv != nil {
			t.Errorf("expected zero-value TerminalConfig, got %+v", cfg.Terminal)
		}
	})

	t.Run("mapping is rejected", func(t *testing.T) {
		var tc TerminalConfig
		if err := yaml.Unmarshal([]byte("name: wt\n"), &tc); err == nil {
			t.Error("expected an error for a mapping node")
		}
	})
}

func TestFindDisplay(t *testing.T) {
	displays := DisplayList{{Name: "default"}, {Name: "alt"}}

	if got := FindDisplay(displays, map[string]any{KeyDisplay: "alt"}); got.Name != "alt" {
		t.Errorf("matching display: got %q", got.Name)
	}
	if got := FindDisplay(displays, map[string]any{KeyDisplay: "missing"}); got.Name != "default" {
		t.Errorf("unknown display should fall back to first: got %q", got.Name)
	}
	if got := FindDisplay(displays, nil); got.Name != "default" {
		t.Errorf("nil item should fall back to first: got %q", got.Name)
	}
	if got := FindDisplay(nil, nil); got.Name != "" {
		t.Errorf("empty display list should return zero value: got %q", got.Name)
	}
}

func TestLoadPathsSourcePath(t *testing.T) {
	dir := t.TempDir()
	winPath := filepath.Join(dir, "config-win.yaml")
	basePath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(winPath, []byte("shell: [pwsh]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(basePath, []byte("shell: [bash]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Mirrors LoadWithError's Windows precedence order: config-win.yaml is
	// tried (in both candidate directories) before config.yaml.
	cfg, err := loadPaths([]string{winPath, basePath})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourcePath != winPath {
		t.Errorf("SourcePath = %q, want %q (config-win.yaml should win)", cfg.SourcePath, winPath)
	}
	if len(cfg.Shell) != 1 || cfg.Shell[0] != "pwsh" {
		t.Errorf("expected config-win.yaml's content to win, got shell %v", cfg.Shell)
	}
}

func TestLoadPathsParseErrorFallsBackWithWarning(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "config-win.yaml")
	fallback := filepath.Join(dir, "config.yaml")
	// Duplicate mapping key: a genuine YAML syntax error, not just "missing".
	if err := os.WriteFile(broken, []byte("env:\n  a: 1\n  a: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallback, []byte("shell: [bash]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadPaths([]string{broken, fallback})
	if err == nil {
		t.Fatal("expected a warning error surfacing the broken candidate's parse failure")
	}
	if cfg.SourcePath != fallback {
		t.Errorf("SourcePath = %q, want %q (fallback should still be applied despite the warning)", cfg.SourcePath, fallback)
	}
}

func TestLoadPathsTotalFailureHasNoSourcePath(t *testing.T) {
	cfg, err := loadPaths([]string{filepath.Join(t.TempDir(), "does-not-exist.yaml")})
	if err == nil {
		t.Fatal("expected an error")
	}
	if cfg.SourcePath != "" {
		t.Errorf("SourcePath = %q, want empty on total failure", cfg.SourcePath)
	}
}

func TestLoadFromWithError(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		cfg, err := LoadFromWithError("does-not-exist.yaml")
		if err == nil {
			t.Error("want error for missing file")
		}
		if cfg == nil {
			t.Error("want non-nil empty config even on error")
		}
	})
}

func TestParseCustomActionsScript(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{"title": "Deploy", "script": "./deploy.sh"},
	}
	actions := ParseCustomActions(raw)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Script != "./deploy.sh" {
		t.Errorf("Script = %q, want %q", actions[0].Script, "./deploy.sh")
	}
	if actions[0].Cmd != "" {
		t.Errorf("Cmd = %q, want empty", actions[0].Cmd)
	}
}

func TestParseCustomActionsScriptOnlyStillValid(t *testing.T) {
	// A script-only entry (no title, no cmd) must still survive the
	// validity check ParseCustomActions applies to drop empty entries.
	raw := []interface{}{
		map[string]interface{}{"script": "./deploy.sh"},
	}
	if actions := ParseCustomActions(raw); len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
}
