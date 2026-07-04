package config

import (
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
			},
		},
	}
	got := ActionsForItem(nil, item)
	want := []Action{{
		ID: "c1", Title: "Custom", Description: "desc",
		Cmd: "echo", Groups: []string{"g1"}, NoWait: true,
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
