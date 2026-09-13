package config

import (
	"reflect"
	"strings"
	"testing"

	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

func TestMarshalRoundTrip(t *testing.T) {
	src := []byte(`
shell: [pwsh, -NoLogo]
env:
  region: eu
display:
  - name: default
    list: '{{.name}}'
    details: '**{{.name}}**'
actionGroups:
  - id: remote
    title: Remote access
    color: '#7fd4ff'
actions:
  - id: ssh
    title: SSH
    cmd: ssh {{.host}}
    groups: [remote]
items:
  - name: srv1
    env:
      sshUser: root
`)
	var cfg Config
	if err := yaml.Unmarshal(src, &cfg); err != nil {
		t.Fatal(err)
	}

	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	var roundTripped Config
	if err := yaml.Unmarshal(out, &roundTripped); err != nil {
		t.Fatalf("re-parsing marshaled output: %v\n%s", err, out)
	}

	cfg.SourcePath = ""
	roundTripped.SourcePath = ""
	if !reflect.DeepEqual(cfg, roundTripped) {
		t.Errorf("round trip mismatch:\noriginal:  %+v\nmarshaled: %+v\nyaml:\n%s", cfg, roundTripped, out)
	}
}

func TestMarshalIdempotent(t *testing.T) {
	cfg := &Config{
		Shell: []string{"bash"},
		Items: []Item{{Name: "srv1"}},
	}
	first, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var reloaded Config
	if err := yaml.Unmarshal(first, &reloaded); err != nil {
		t.Fatal(err)
	}
	second, err := reloaded.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("marshal not idempotent:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestActionGroupsMarshal(t *testing.T) {
	t.Run("empty slice omits the key entirely", func(t *testing.T) {
		cfg := &Config{Shell: []string{"bash"}}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "actionGroups") {
			t.Errorf("expected no actionGroups: key for an empty slice, got:\n%s", out)
		}
	})

	t.Run("round trips id/title/color", func(t *testing.T) {
		cfg := &Config{
			ActionGroups: []ActionGroup{
				{ID: "safe", Title: "Safe operations", Color: "#4caf50"},
				{ID: "danger"},
			},
		}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		var reloaded Config
		if err := yaml.Unmarshal(out, &reloaded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cfg.ActionGroups, reloaded.ActionGroups) {
			t.Errorf("round trip = %+v, want %+v\nyaml:\n%s", reloaded.ActionGroups, cfg.ActionGroups, out)
		}
	})
}

func TestSecretsMarshal(t *testing.T) {
	t.Run("nil omits the key entirely", func(t *testing.T) {
		cfg := &Config{Shell: []string{"bash"}}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "secrets") {
			t.Errorf("expected no secrets: key for a config with no PIN, got:\n%s", out)
		}
	})

	t.Run("round trips the KDF parameters and the encrypted value", func(t *testing.T) {
		params, key, err := secret.NewParams("1234")
		if err != nil {
			t.Fatal(err)
		}
		enc, err := secret.Encrypt(key, "hunter2")
		if err != nil {
			t.Fatal(err)
		}
		cfg := &Config{
			Secrets: &params,
			Env:     map[string]any{"password": enc},
		}

		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "hunter2") {
			t.Errorf("the plaintext leaked into the saved YAML:\n%s", out)
		}

		var reloaded Config
		if err := yaml.Unmarshal(out, &reloaded); err != nil {
			t.Fatal(err)
		}
		if reloaded.Secrets == nil || !reflect.DeepEqual(*reloaded.Secrets, params) {
			t.Fatalf("secrets round trip = %+v, want %+v\nyaml:\n%s", reloaded.Secrets, params, out)
		}

		reloadedKey, err := reloaded.Secrets.Unlock("1234")
		if err != nil {
			t.Fatalf("Unlock() after a round trip failed: %v", err)
		}
		got, err := secret.Decrypt(reloadedKey, reloaded.Env["password"].(string))
		if err != nil {
			t.Fatalf("Decrypt() after a round trip failed: %v", err)
		}
		if got != "hunter2" {
			t.Errorf("decrypted value = %q, want %q", got, "hunter2")
		}
	})
}

func TestTerminalConfigMarshal(t *testing.T) {
	t.Run("zero value omits the key entirely", func(t *testing.T) {
		cfg := &Config{Shell: []string{"bash"}}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "terminal") {
			t.Errorf("expected no terminal: key for a zero-value TerminalConfig, got:\n%s", out)
		}
	})

	t.Run("named terminal marshals as a scalar", func(t *testing.T) {
		cfg := &Config{Terminal: TerminalConfig{Name: "wt"}}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "terminal: wt\n") {
			t.Errorf("expected `terminal: wt`, got:\n%s", out)
		}
		var reloaded Config
		if err := yaml.Unmarshal(out, &reloaded); err != nil {
			t.Fatal(err)
		}
		if reloaded.Terminal.Name != "wt" {
			t.Errorf("round trip: got %+v", reloaded.Terminal)
		}
	})

	t.Run("custom argv marshals as a sequence", func(t *testing.T) {
		cfg := &Config{Terminal: TerminalConfig{Argv: []string{"my-term", "--title", "{{title}}"}}}
		out, err := cfg.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		var reloaded Config
		if err := yaml.Unmarshal(out, &reloaded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(reloaded.Terminal.Argv, cfg.Terminal.Argv) || reloaded.Terminal.Name != "" {
			t.Errorf("round trip: got %+v, want %+v", reloaded.Terminal, cfg.Terminal)
		}
	})
}
