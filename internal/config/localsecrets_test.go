package config

import (
	"os"
	"path/filepath"
	"runtime"
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

func readLocal(t *testing.T, id string) *localSecrets {
	t.Helper()
	l, err := readLocalSecrets(LocalSecretsPath(id))
	if err != nil || l == nil {
		t.Fatalf("local file: %v %v", l, err)
	}
	return l
}

func TestSaveRoundTrip(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "hunter2")
	cfg := &Config{
		Secrets: &params,
		Env:     map[string]any{"token": enc, "region": "eu"},
		Items:   []Item{{Name: "db", Env: map[string]any{"password": enc, "user": "app"}}},
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if !secretsIDPattern.MatchString(cfg.SecretsID) {
		t.Fatalf("Save did not assign a secretsId: %q", cfg.SecretsID)
	}
	if info, err := os.Stat(LocalSecretsPath(cfg.SecretsID)); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("local file mode = %v, want 0600", info.Mode().Perm())
	}
	loaded, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Env["token"] != enc || loaded.Items[0].Env["password"] != enc || loaded.Env["region"] != "eu" {
		t.Errorf("round trip lost values: %v %v", loaded.Env, loaded.Items[0].Env)
	}
	if !secret.SameParams(loaded.Secrets, &params) {
		t.Errorf("params = %+v", loaded.Secrets)
	}
}

func TestSaveFollowsItemRename(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "x")
	cfg := &Config{SecretsID: testID, Secrets: &params, Items: []Item{{Name: "old", Env: map[string]any{"pw": enc}}}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	cfg.Items[0].Name = "new"
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Items[0].Env["pw"] != enc {
		t.Errorf("renamed item lost its secret: %v", loaded.Items[0].Env)
	}
}

func TestSaveKeepsOrphansUnderSameParams(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "x")
	cfg := &Config{SecretsID: testID, Secrets: &params, Items: []Item{
		{Name: "a", Env: map[string]any{"pw": enc}},
		{Name: "b", Env: map[string]any{"pw": enc}},
	}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	cfg.Items = cfg.Items[:1] // "b" exists only on another branch now
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if readLocal(t, testID).Items["b"]["pw"] != enc {
		t.Error("orphaned entry for item b was dropped")
	}
}

func TestSaveDropsOrphansWhenParamsChange(t *testing.T) {
	withAppData(t)
	params, enc := lockedValue(t, "1234", "x")
	cfg := &Config{SecretsID: testID, Secrets: &params, Items: []Item{
		{Name: "a", Env: map[string]any{"pw": enc}},
		{Name: "b", Env: map[string]any{"pw": enc}},
	}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	newParams, newEnc := lockedValue(t, "5678", "x")
	cfg.Secrets = &newParams
	cfg.Items = []Item{{Name: "a", Env: map[string]any{"pw": newEnc}}}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := readLocal(t, testID).Items["b"]; ok {
		t.Error("orphan under the old PIN params was kept")
	}
}

func TestSaveMigratesLegacyInline(t *testing.T) {
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
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	shared, _ := os.ReadFile(path)
	if strings.Contains(string(shared), "sm-enc:") || strings.Contains(string(shared), "salt") {
		t.Errorf("shared file still holds secrets:\n%s", shared)
	}
	if got := readLocal(t, cfg.SecretsID).Env["token"]; got != enc {
		t.Errorf("ciphertext changed in migration: %q", got)
	}
	reloaded, err := LoadFromWithError(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.Secrets.Unlock("1234"); err != nil {
		t.Errorf("PIN no longer unlocks after migration: %v", err)
	}
}

func TestSaveWithoutPINLeavesLocalFileAlone(t *testing.T) {
	withAppData(t)
	writeFile(t, LocalSecretsPath(testID), "env: [unclosed\n")
	cfg := &Config{SecretsID: testID, Env: map[string]any{"token": secret.Unset}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(LocalSecretsPath(testID))
	if string(data) != "env: [unclosed\n" {
		t.Errorf("local file was touched: %q", data)
	}
	shared, _ := os.ReadFile(path)
	if !strings.Contains(string(shared), "!secret") || !strings.Contains(string(shared), testID) {
		t.Errorf("declaration or id lost:\n%s", shared)
	}
}

func TestSaveRejectsDuplicateNamesWithSecrets(t *testing.T) {
	withAppData(t)
	_, enc := lockedValue(t, "1234", "x")
	cfg := &Config{Items: []Item{
		{Name: "dup", Env: map[string]any{"pw": enc}},
		{Name: "dup", Env: map[string]any{"other": enc}},
	}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err == nil {
		t.Fatal("want error for duplicate item names sharing a locked value")
	}
	if cfg.SecretsID != "" {
		t.Error("a rejected save should not assign a secretsId")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("shared file should not have been written")
	}
}

func TestSaveRejectsEmptyNameWithSecret(t *testing.T) {
	withAppData(t)
	_, enc := lockedValue(t, "1234", "x")
	cfg := &Config{Items: []Item{{Name: "", Env: map[string]any{"pw": enc}}}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err == nil {
		t.Fatal("want error for an unnamed item holding a locked value")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("shared file should not have been written")
	}
}

func TestSaveAllowsDuplicateNamesWithoutSecrets(t *testing.T) {
	withAppData(t)
	cfg := &Config{Items: []Item{
		{Name: "dup", Env: map[string]any{"a": "1"}},
		{Name: "dup", Env: map[string]any{"b": "2"}},
	}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save() = %v, want nil for duplicate names without secrets", err)
	}
}

func TestSaveRejectsSecretItemThenPlainItemSharingAName(t *testing.T) {
	withAppData(t)
	params, encA1 := lockedValue(t, "1234", "a1")
	cfg := &Config{Secrets: &params, Items: []Item{{Name: "dup", Env: map[string]any{"pw": encA1}}}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	id := cfg.SecretsID
	sharedBefore, _ := os.ReadFile(path)

	_, encA2 := lockedValue(t, "1234", "a2")
	cfg.Items = []Item{
		{Name: "dup", Env: map[string]any{"pw": encA2}},
		{Name: "dup", Env: map[string]any{"foo": "bar"}},
	}
	if err := cfg.Save(path); err == nil {
		t.Fatal("want error when a secret item shares its name with a later plain item")
	}

	sharedAfter, _ := os.ReadFile(path)
	if string(sharedBefore) != string(sharedAfter) {
		t.Error("shared file was modified by a rejected save")
	}
	if got := readLocal(t, id).Items["dup"]["pw"]; got != encA1 {
		t.Errorf("local file was modified by a rejected save: %q, want %q", got, encA1)
	}
}

func TestSaveRejectsPlainItemThenSecretItemSharingAName(t *testing.T) {
	withAppData(t)
	_, enc := lockedValue(t, "1234", "x")
	cfg := &Config{Items: []Item{
		{Name: "dup", Env: map[string]any{"foo": "bar"}},
		{Name: "dup", Env: map[string]any{"pw": enc}},
	}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := cfg.Save(path); err == nil {
		t.Fatal("want error when a plain item shares its name with a secret item")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("shared file should not have been written")
	}
}

func TestRemoveLocalSecrets(t *testing.T) {
	withAppData(t)
	writeFile(t, LocalSecretsPath(testID), "env: {}\n")
	if err := RemoveLocalSecrets(testID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(LocalSecretsPath(testID)); !os.IsNotExist(err) {
		t.Errorf("file still there: %v", err)
	}
	if err := RemoveLocalSecrets(testID); err != nil {
		t.Errorf("removing a missing file: %v", err)
	}
}
