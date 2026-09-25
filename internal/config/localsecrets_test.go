package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"script-manager/internal/secret"
)

func withAppData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

const testID = "0123456789abcdef0123456789abcdef"

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func lockedValue(t *testing.T, pin, plain string) (secret.Params, string) {
	t.Helper()
	params, key, err := secret.NewParams(pin)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := secret.Encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	return params, enc
}

const sharedYAML = `
secretsId: ` + testID + `
env:
  region: eu
  token: !secret
items:
  - name: db
    env:
      user: app
      password: !secret
`

func TestLoadFillsSecretsFromLocalFile(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "hunter2")
	local := "secrets:\n  salt: " + params.Salt + "\n  time: 3\n  memory: 65536\n  threads: 4\n  check: " + params.Check +
		"\nenv:\n  token: " + enc + "\nitems:\n  db:\n    password: " + enc + "\n"
	writeFile(t, LocalSecretsPath(testID), local)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, sharedYAML)

	cfg, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SecretsID != testID || cfg.Secrets == nil || cfg.Secrets.Salt != params.Salt {
		t.Fatalf("id/params not loaded: %q %+v", cfg.SecretsID, cfg.Secrets)
	}
	if cfg.Env["token"] != enc || cfg.Items[0].Env["password"] != enc || cfg.Env["region"] != "eu" {
		t.Errorf("values not merged: env=%v item=%v", cfg.Env, cfg.Items[0].Env)
	}
}

func TestLoadWithoutLocalFile(t *testing.T) {
	withAppData(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, sharedYAML)

	cfg, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if !secret.IsUnset(cfg.Env["token"]) || !secret.IsUnset(cfg.Items[0].Env["password"]) {
		t.Errorf("expected Unset, got env=%v item=%v", cfg.Env, cfg.Items[0].Env)
	}
	if cfg.Secrets != nil {
		t.Errorf("Secrets = %+v, want nil", cfg.Secrets)
	}
}

func TestLoadIgnoresUndeclaredLocalEntries(t *testing.T) {
	withAppData(t)
	_, enc := lockedValue(t, "1234", "x")
	writeFile(t, LocalSecretsPath(testID), "env:\n  region: "+enc+"\n  other: "+enc+"\nitems:\n  gone:\n    k: "+enc+"\n")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, sharedYAML)

	cfg, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env["region"] != "eu" {
		t.Errorf("a plain value was overridden by a local entry: %v", cfg.Env["region"])
	}
	if _, ok := cfg.Env["other"]; ok {
		t.Errorf("an undeclared local key was added: %v", cfg.Env)
	}
}

func TestLoadReportsMalformedLocalFile(t *testing.T) {
	withAppData(t)
	writeFile(t, LocalSecretsPath(testID), "env: [unclosed\n")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, sharedYAML)

	cfg, err := LoadFromWithError(path)
	if err == nil || !strings.Contains(err.Error(), LocalSecretsPath(testID)) {
		t.Fatalf("err = %v, want one naming the local file", err)
	}
	if cfg == nil || cfg.SourcePath == "" || len(cfg.Items) != 1 {
		t.Errorf("shared config should still load: %+v", cfg)
	}
}

func TestLoadRejectsSecretTagOutsideEnv(t *testing.T) {
	withAppData(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "shell: !secret\n")
	if _, err := LoadFromWithError(path); err == nil || !strings.Contains(err.Error(), "!secret") {
		t.Errorf("err = %v, want a !secret placement error", err)
	}
}

func TestLoadRejectsBadSecretsID(t *testing.T) {
	withAppData(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "secretsId: ../../evil\n")
	if _, err := LoadFromWithError(path); err == nil || !strings.Contains(err.Error(), "secretsId") {
		t.Errorf("err = %v, want a secretsId error", err)
	}
}

func TestLoadLegacyInlineSecrets(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "hunter2")
	legacy := "secrets:\n  salt: " + params.Salt + "\n  time: 3\n  memory: 65536\n  threads: 4\n  check: " + params.Check +
		"\nenv:\n  token: " + enc + "\n"
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, legacy)

	cfg, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Secrets == nil || cfg.Env["token"] != enc {
		t.Fatalf("legacy config not loaded: %+v %v", cfg.Secrets, cfg.Env)
	}
	if _, err := cfg.Secrets.Unlock("1234"); err != nil {
		t.Errorf("legacy PIN no longer unlocks: %v", err)
	}
}

func TestNewSecretsIDShape(t *testing.T) {
	id, err := NewSecretsID()
	if err != nil || !secretsIDPattern.MatchString(id) {
		t.Errorf("NewSecretsID() = %q, %v", id, err)
	}
}
