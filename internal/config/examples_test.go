package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"script-manager/internal/config"
	"script-manager/internal/configmigrate"
	"script-manager/internal/secret"
)

// Guards the shipped example configs: they must load under the strict loader
// and must not ask the user to convert anything.
func TestExampleConfigsAreCurrent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	for _, path := range []string{
		"../../examples/pin-secrets.yaml",
		"../../examples/pin-secrets-win.yaml",
	} {
		needed, err := configmigrate.Needed(path)
		if err != nil {
			t.Fatalf("%s: Needed() error = %v", path, err)
		}
		if needed {
			t.Errorf("%s: ships in the old item format", path)
		}
		cfg, err := config.LoadFromWithError(path)
		if err != nil {
			t.Fatalf("%s: load error = %v", path, err)
		}
		if len(cfg.Items) == 0 {
			t.Fatalf("%s: no items loaded", path)
		}
		if len(cfg.Items[0].Env) == 0 {
			t.Errorf("%s: first item has no env values", path)
		}
	}
}

// Guards examples/pin-secrets.local.yaml: it must unlock the shared config's
// !secret values with the documented PIN.
func TestExampleSecretsLoadFromLocalFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := config.LoadFromWithError("../../examples/pin-secrets.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !secret.IsUnset(cfg.Env["vault_token"]) {
		t.Fatalf("without the local file vault_token = %v, want unset", cfg.Env["vault_token"])
	}
	local, err := os.ReadFile("../../examples/pin-secrets.local.yaml")
	if err != nil {
		t.Fatal(err)
	}
	target := config.LocalSecretsPath(cfg.SecretsID)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, local, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.LoadFromWithError("../../examples/pin-secrets.yaml")
	if err != nil {
		t.Fatal(err)
	}
	key, err := cfg.Secrets.Unlock("demo1234")
	if err != nil {
		t.Fatalf("example PIN: %v", err)
	}
	if _, err := secret.Reveal(cfg.Env, key); err != nil {
		t.Errorf("example values don't decrypt: %v", err)
	}
}
