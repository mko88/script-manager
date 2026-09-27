# Local Secrets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move PIN-locked values and the PIN parameters out of the shared `config.yaml` into a per-user file in app-data, so a config can be committed to git.

**Architecture:** The shared config declares secret env values with a bare `!secret` YAML tag and carries a random `secretsId`. The loader turns each `!secret` into an in-memory sentinel (`secret.Unset`) and then fills it from `<app-data>/secrets/<secretsId>.yaml` (ciphertext plus PIN params). After loading, the in-memory `Config` has the same shape the TUI and GUI already handle, so runtime changes are confined to `internal/secret`'s helpers learning about `Unset`. Saving is the reverse split, done by a new `Config.Save`, which the Config Editor calls.

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v3` (node API), Wails v2, Svelte + TypeScript (sm-config-edit frontend).

**Spec:** `docs/superpowers/specs/2026-09-25-local-secrets-design.md`

## Global Constraints

- Shared config key: `secretsId`, exactly 32 lowercase hex chars (`^[0-9a-f]{32}$`), 128 random bits.
- Local file: `<appdata.Dir()>/secrets/<secretsId>.yaml`, file mode `0600`, directory `0700`.
- Declaration tag: `!secret`, valid only as the value of a key directly under the top-level `env:` or an item's `env:`. Anywhere else it is a load error.
- The shared file never contains a `secrets:` block or an `sm-enc:v1:` value after a save.
- The local file is written before the shared file on save.
- Legacy configs (inline `secrets:` + inline `sm-enc:v1:` values) load unchanged. The first editor save migrates them without re-encrypting.
- Local entries whose item/key the shared config doesn't mention are kept on save only while the PIN params are unchanged (`secret.SameParams`), and ignored on load.
- An `Unset` value is never shown as its sentinel text and never reaches a script's environment.
- No changelog/history comments in code (CLAUDE.md). README is a user guide (CLAUDE.md "README style").
- Never run the apps. Verification is `go test`/`go vet`/`npm run check` plus `.\build-container.ps1 -Windows`.

**Running Go/npm checks.** There's no Go toolchain on the host. Use the dev container:

```bash
C=$(docker ps --filter label=devcontainer.local_folder --format '{{.Names}}' | head -1)
docker exec $C sh -c 'cd /workspaces/script-manager && go test ./internal/secret/'
```

Every "Run:" below means "inside that `docker exec $C sh -c 'cd /workspaces/script-manager && …'`".

**Test isolation of app-data.** `appdata.Dir()` is `os.UserConfigDir()/script-manager`. Tests point it at a temp dir by setting both `AppData` (Windows) and `XDG_CONFIG_HOME` (Linux) with `t.Setenv`, via the `withAppData` helper each test package defines (Task 2, Task 4).

## Review Focus

1. **A fresh clone with no local file.** Every `!secret` is `(not set)` on screen, `requiresPin` actions run without prompting, and the variables are absent from the script env. Pinned by the `TestLoadWithoutLocalFile` (Task 2) and `TestRevealDropsUnset` / `TestHasLockedIgnoresUnset` (Task 1) tests.
2. **A malformed local file.** Load returns the config plus an error naming the local path. A later editor save with no PIN set must not touch that file. Pinned by `TestLoadReportsMalformedLocalFile` (Task 2) and `TestSaveWithoutPINLeavesLocalFileAlone` (Task 3).
3. **A legacy config saved once.** The ciphertext is byte-identical in the local file and the same PIN unlocks it. Pinned by `TestSaveMigratesLegacyInline` (Task 3).
4. **Change PIN with orphaned local entries.** The orphans are dropped rather than kept undecryptable under the new params. Pinned by `TestSaveDropsOrphansWhenParamsChange` (Task 3).
5. **The sentinel leaking.** `secret.Unset` must never be written into the shared file as a string, and never reach an env var or preview. Pinned by `TestMarshalWritesSecretTag` (Task 3) and `TestRedactShowsNotSet` (Task 1).

---

### Task 1: `secret.Unset` sentinel in the runtime helpers

**Files:**
- Modify: `internal/secret/secret.go`
- Test: `internal/secret/secret_test.go`

**Interfaces:**
- Produces: `const Unset = "sm-secret:unset"`, `const UnsetDisplayText = "(not set)"`, and `func IsUnset(v any) bool`. `Redact`, `Strip`, `Reveal` and `HasLocked` get the semantics below. Tasks 2–4 rely on these names.

- [ ] **Step 1: Write the failing tests** (append to `internal/secret/secret_test.go`)

```go
func TestRedactShowsNotSet(t *testing.T) {
	got := Redact(map[string]any{"a": Unset, "b": "plain"})
	if got["a"] != UnsetDisplayText || got["b"] != "plain" {
		t.Errorf("Redact = %v", got)
	}
}

func TestStripDropsUnset(t *testing.T) {
	got := Strip(map[string]any{"a": Unset, "b": "plain"})
	if _, ok := got["a"]; ok || got["b"] != "plain" {
		t.Errorf("Strip = %v", got)
	}
}

func TestRevealDropsUnset(t *testing.T) {
	// No locked values and no key: Unset must still be removed, not passed through.
	got, err := Reveal(map[string]any{"a": Unset, "b": "plain"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["a"]; ok || got["b"] != "plain" {
		t.Errorf("Reveal = %v", got)
	}
}

func TestHasLockedIgnoresUnset(t *testing.T) {
	if HasLocked(map[string]any{"a": Unset}) {
		t.Error("HasLocked counted an unset value as locked")
	}
	if IsLocked(Unset) {
		t.Error("IsLocked(Unset) = true")
	}
	if !IsUnset(Unset) || IsUnset("x") || IsUnset(nil) {
		t.Error("IsUnset wrong")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/secret/`
Expected: compile failure, `undefined: Unset`.

- [ ] **Step 3: Implement** in `internal/secret/secret.go`

Add next to `LockedDisplayText`:

```go
// Unset stands in for a value the shared config declares with !secret but
// this machine's local secrets file doesn't provide. It is never shown or
// exported as-is.
const Unset = "sm-secret:unset"

const UnsetDisplayText = "(not set)"

func IsUnset(v any) bool {
	s, ok := v.(string)
	return ok && s == Unset
}

func hasHidden(values map[string]any) bool {
	for _, v := range values {
		if IsLockedValue(v) || IsUnset(v) {
			return true
		}
	}
	return false
}
```

Replace `Reveal`, `Redact` and `Strip` with:

```go
func Reveal(values map[string]any, key []byte) (map[string]any, error) {
	if !hasHidden(values) {
		return values, nil
	}
	if HasLocked(values) && len(key) == 0 {
		return nil, ErrLocked
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		if IsUnset(v) {
			continue
		}
		if s, ok := v.(string); ok && IsLocked(s) {
			plain, err := Decrypt(key, s)
			if err != nil {
				return nil, err
			}
			out[k] = plain
			continue
		}
		out[k] = v
	}
	return out, nil
}

func Redact(values map[string]any) map[string]any {
	if !hasHidden(values) {
		return values
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		switch {
		case IsUnset(v):
			out[k] = UnsetDisplayText
		case IsLockedValue(v):
			out[k] = LockedDisplayText
		default:
			out[k] = v
		}
	}
	return out
}

func Strip(values map[string]any) map[string]any {
	if !hasHidden(values) {
		return values
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		if IsLockedValue(v) || IsUnset(v) {
			continue
		}
		out[k] = v
	}
	return out
}
```

`HasLocked` stays as it is (it already only matches `Prefix`).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/secret/ ./internal/gui/ ./internal/ui/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/secret
git commit -m "Add an Unset sentinel for secrets this machine doesn't have"
```

---

### Task 2: Loading `!secret`, `secretsId` and the local secrets file

**Files:**
- Create: `internal/config/localsecrets.go`
- Create: `internal/config/localsecrets_test.go`
- Modify: `internal/config/config.go` (the `Config` struct and `loadPaths`)

**Interfaces:**
- Consumes: `secret.Unset`, `secret.IsUnset`, `secret.IsLocked`, `secret.Params` (Task 1).
- Produces: `Config.SecretsID string` (`yaml:"secretsId,omitempty"`), with `Config.Secrets` becoming `yaml:"-"`. Also `func LocalSecretsPath(id string) string`, `func NewSecretsID() (string, error)`, the unexported `type localSecrets`, `func parseConfig(data []byte) (Config, error)` and `func (c *Config) applyLocalSecrets() error`.

- [ ] **Step 1: Write the failing tests** (`internal/config/localsecrets_test.go`)

```go
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
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/config/`
Expected: compile failure, `undefined: LocalSecretsPath` / `cfg.SecretsID undefined`.

- [ ] **Step 3: Change the `Config` struct** in `internal/config/config.go`

```go
type Config struct {
	SecretsID    string         `yaml:"secretsId,omitempty"`
	Shell        []string       `yaml:"shell,omitempty"`
	Display      DisplayList    `yaml:"display,omitempty"`
	Terminal     TerminalConfig `yaml:"terminal,omitempty"`
	Env          map[string]any `yaml:"env,omitempty"`
	Items        []Item         `yaml:"items,omitempty"`
	ActionGroups []ActionGroup  `yaml:"actionGroups,omitempty"`
	Actions      []Action       `yaml:"actions,omitempty"`

	// Secrets comes from the local secrets file (or, in a config not yet
	// saved by this version, an inline secrets: block).
	Secrets    *secret.Params `yaml:"-"`
	SourcePath string         `yaml:"-"`
}
```

- [ ] **Step 4: Create `internal/config/localsecrets.go`**

```go
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"script-manager/internal/appdata"
	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

const secretTag = "!secret"

var secretsIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// localSecrets is the per-user file holding a config's PIN params and the
// ciphertext for every value the shared config declares with !secret.
type localSecrets struct {
	Secrets *secret.Params               `yaml:"secrets,omitempty"`
	Env     map[string]string            `yaml:"env,omitempty"`
	Items   map[string]map[string]string `yaml:"items,omitempty"`
}

func NewSecretsID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LocalSecretsPath is where this user's secrets for the config with the given
// ID live, or "" when there is no ID or no app-data directory.
func LocalSecretsPath(id string) string {
	dir := appdata.Dir()
	if dir == "" || id == "" {
		return ""
	}
	return filepath.Join(dir, "secrets", id+".yaml")
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func envMappings(root *yaml.Node) []*yaml.Node {
	var out []*yaml.Node
	if env := mappingValue(root, KeyEnv); env != nil && env.Kind == yaml.MappingNode {
		out = append(out, env)
	}
	if items := mappingValue(root, "items"); items != nil && items.Kind == yaml.SequenceNode {
		for _, item := range items.Content {
			if env := mappingValue(item, KeyEnv); env != nil && env.Kind == yaml.MappingNode {
				out = append(out, env)
			}
		}
	}
	return out
}

// claimSecretTags turns each env-level !secret into the Unset sentinel, then
// rejects any !secret left anywhere else.
func claimSecretTags(root *yaml.Node) error {
	for _, env := range envMappings(root) {
		for i := 1; i < len(env.Content); i += 2 {
			v := env.Content[i]
			if v.Tag != secretTag {
				continue
			}
			if v.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: %s takes no value", v.Line, secretTag)
			}
			v.Tag = "!!str"
			v.Value = secret.Unset
			v.Style = 0
		}
	}
	return findStraySecretTag(root)
}

func findStraySecretTag(n *yaml.Node) error {
	if n.Tag == secretTag {
		return fmt.Errorf("line %d: %s is only allowed as a value under env:", n.Line, secretTag)
	}
	for _, c := range n.Content {
		if err := findStraySecretTag(c); err != nil {
			return err
		}
	}
	return nil
}

func parseConfig(data []byte) (Config, error) {
	var cfg Config
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, err
	}
	if len(doc.Content) == 0 {
		return cfg, nil
	}
	root := doc.Content[0]
	if err := claimSecretTags(root); err != nil {
		return cfg, err
	}
	if err := root.Decode(&cfg); err != nil {
		return cfg, err
	}
	if inline := mappingValue(root, "secrets"); inline != nil {
		var p secret.Params
		if err := inline.Decode(&p); err != nil {
			return cfg, fmt.Errorf("secrets: %w", err)
		}
		cfg.Secrets = &p
	}
	if cfg.SecretsID != "" && !secretsIDPattern.MatchString(cfg.SecretsID) {
		return cfg, fmt.Errorf("secretsId %q must be 32 lowercase hex characters", cfg.SecretsID)
	}
	return cfg, nil
}

func readLocalSecrets(path string) (*localSecrets, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var local localSecrets
	if err := yaml.Unmarshal(data, &local); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &local, nil
}

// applyLocalSecrets fills each Unset slot from this user's local secrets file.
// Local entries the shared config doesn't declare are ignored.
func (c *Config) applyLocalSecrets() error {
	path := LocalSecretsPath(c.SecretsID)
	if path == "" {
		return nil
	}
	local, err := readLocalSecrets(path)
	if err != nil || local == nil {
		return err
	}
	if local.Secrets != nil {
		c.Secrets = local.Secrets
	}
	fillUnset(c.Env, local.Env)
	for i := range c.Items {
		fillUnset(c.Items[i].Env, local.Items[c.Items[i].Name])
	}
	return nil
}

func fillUnset(env map[string]any, values map[string]string) {
	for k, v := range env {
		if s, ok := values[k]; ok && secret.IsUnset(v) && secret.IsLocked(s) {
			env[k] = s
		}
	}
}
```

- [ ] **Step 5: Use it in `loadPaths`** (`internal/config/config.go`)

Replace the loop body's parse and return:

```go
		cfg, err := parseConfig(data)
		if err != nil {
			wrapped := fmt.Errorf("%s: %w", p, err)
			if parseErr == nil {
				parseErr = wrapped
			}
			lastErr = wrapped
			continue
		}
		if abs, err := filepath.Abs(p); err == nil {
			cfg.SourcePath = abs
		} else {
			cfg.SourcePath = p
		}
		if err := cfg.applyLocalSecrets(); err != nil {
			return &cfg, err
		}
		return &cfg, parseErr
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/config/`
Expected: the new tests PASS. `TestSecretsMarshal` in `marshal_test.go` now FAILS (it expects a `secrets:` block in the shared output); Task 3 rewrites it. Any other failure is a bug here.

- [ ] **Step 7: Commit**

```bash
git add internal/config
git commit -m "Load !secret declarations and fill them from a per-user secrets file"
```

---

### Task 3: Saving the shared/local split

**Files:**
- Modify: `internal/config/marshal.go`
- Modify: `internal/config/marshal_test.go` (replace `TestSecretsMarshal`)
- Test: `internal/config/localsecrets_test.go` (append)

**Interfaces:**
- Consumes: Task 2's `localSecrets`, `envMappings`, `LocalSecretsPath`, `NewSecretsID`, `readLocalSecrets`, `secretTag`.
- Produces: `func (c *Config) Marshal() ([]byte, error)` (now writes `!secret`), `func (c *Config) Save(path string) error`, and `func RemoveLocalSecrets(id string) error`. Task 4 relies on `Save` and `RemoveLocalSecrets`.

- [ ] **Step 1: Write the failing tests**

In `internal/config/marshal_test.go`, delete `TestSecretsMarshal` and add:

```go
func TestMarshalWritesSecretTag(t *testing.T) {
	params, enc := func() (secret.Params, string) {
		p, key, err := secret.NewParams("1234")
		if err != nil {
			t.Fatal(err)
		}
		e, err := secret.Encrypt(key, "hunter2")
		if err != nil {
			t.Fatal(err)
		}
		return p, e
	}()
	cfg := &Config{
		SecretsID: "0123456789abcdef0123456789abcdef",
		Secrets:   &params,
		Env:       map[string]any{"token": enc, "missing": secret.Unset, "region": "eu"},
		Items:     []Item{{Name: "db", Env: map[string]any{"password": enc}}},
	}
	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"sm-enc:", secret.Unset, "salt", "secrets:"} {
		if strings.Contains(s, bad) {
			t.Errorf("shared output contains %q:\n%s", bad, s)
		}
	}
	if strings.Count(s, "!secret") != 3 {
		t.Errorf("want 3 !secret tags:\n%s", s)
	}
	reparsed, err := parseConfig(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, s)
	}
	if !secret.IsUnset(reparsed.Env["token"]) || !secret.IsUnset(reparsed.Items[0].Env["password"]) || reparsed.Env["region"] != "eu" {
		t.Errorf("reparsed = %v / %v", reparsed.Env, reparsed.Items[0].Env)
	}
}
```

Append to `internal/config/localsecrets_test.go`:

```go
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
```

Add `"runtime"` to that test file's imports.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/config/`
Expected: compile failure, `cfg.Save undefined` / `RemoveLocalSecrets undefined`.

- [ ] **Step 3: Implement** `internal/config/marshal.go`

```go
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

func isHiddenValue(v *yaml.Node) bool {
	return v.Kind == yaml.ScalarNode && (secret.IsLocked(v.Value) || v.Value == secret.Unset)
}

// Marshal renders the shared config: every locked or unset env value is
// written as a bare !secret, so the file carries no ciphertext.
func (c *Config) Marshal() ([]byte, error) {
	var root yaml.Node
	if err := root.Encode(c); err != nil {
		return nil, err
	}
	for _, env := range envMappings(&root) {
		for i := 1; i < len(env.Content); i += 2 {
			if isHiddenValue(env.Content[i]) {
				env.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: secretTag}
			}
		}
	}
	return yaml.Marshal(&root)
}

// Save writes the local secrets file first, when there's a PIN, so a watcher
// on path never reloads a config whose secrets aren't on disk yet.
func (c *Config) Save(path string) error {
	if c.Secrets != nil {
		if c.SecretsID == "" {
			id, err := NewSecretsID()
			if err != nil {
				return err
			}
			c.SecretsID = id
		}
		if err := c.writeLocalSecrets(); err != nil {
			return err
		}
	}
	out, err := c.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func (c *Config) writeLocalSecrets() error {
	path := LocalSecretsPath(c.SecretsID)
	if path == "" {
		return errors.New("no per-user data directory to store the secrets in")
	}
	existing, _ := readLocalSecrets(path)
	out, err := yaml.Marshal(c.buildLocalSecrets(existing))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

// buildLocalSecrets collects every locked value. Entries in existing for items
// or keys this config doesn't mention are kept — they may belong to another
// branch of a shared config — but only while they're under the same PIN
// params, since nothing re-encrypts them.
func (c *Config) buildLocalSecrets(existing *localSecrets) localSecrets {
	out := localSecrets{Secrets: c.Secrets, Env: lockedOnly(c.Env), Items: map[string]map[string]string{}}
	for _, item := range c.Items {
		if vals := lockedOnly(item.Env); len(vals) > 0 {
			out.Items[item.Name] = vals
		}
	}
	if existing != nil && secret.SameParams(existing.Secrets, c.Secrets) {
		keepOrphans(out.Env, existing.Env, c.Env)
		known := make(map[string]map[string]any, len(c.Items))
		for _, item := range c.Items {
			known[item.Name] = item.Env
		}
		for name, vals := range existing.Items {
			env, ok := known[name]
			if !ok {
				out.Items[name] = vals
				continue
			}
			if out.Items[name] == nil {
				out.Items[name] = map[string]string{}
			}
			keepOrphans(out.Items[name], vals, env)
			if len(out.Items[name]) == 0 {
				delete(out.Items, name)
			}
		}
	}
	if len(out.Items) == 0 {
		out.Items = nil
	}
	return out
}

func lockedOnly(env map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range env {
		if s, ok := v.(string); ok && secret.IsLocked(s) {
			out[k] = s
		}
	}
	return out
}

func keepOrphans(dst, existing map[string]string, env map[string]any) {
	for k, v := range existing {
		if _, declared := env[k]; !declared {
			dst[k] = v
		}
	}
}

func RemoveLocalSecrets(id string) error {
	path := LocalSecretsPath(id)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
```

`TestSaveKeepsOrphansUnderSameParams` depends on `keepOrphans` being given `nil` env for an item that no longer exists. That's the `!ok` branch, which copies the whole map.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/ && go vet ./internal/config/`
Expected: PASS. If `TestMarshalWritesSecretTag` fails on the tag count because yaml.v3 emits `!secret ""`, that's fine as long as the count is 3 and the reparse passes. If the count is wrong, inspect the output before changing anything.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "Save locked values to the per-user secrets file and !secret to the config"
```

---

### Task 4: Config Editor backend

**Files:**
- Modify: `internal/configedit/dto.go`, `internal/configedit/convert.go`, `internal/configedit/app.go`, `internal/configedit/secrets.go`
- Test: `internal/configedit/secrets_test.go`, `internal/configedit/convert_test.go`

**Interfaces:**
- Consumes: `config.Config.Save`, `config.RemoveLocalSecrets`, `config.NewSecretsID`, `config.LocalSecretsPath` (Task 3/2), and `secret.Unset` / `secret.IsUnset` (Task 1).
- Produces for the frontend (Wails bindings): `FieldDTO.Unset bool` (json `unset`), `ConfigDTO.SecretsID string` (json `secretsId`), and the bound methods `NewSecretsID() (string, error)` and `SecretsFilePath(id string) string`. `RemoveSecretsPIN(pin, dto)` now returns a DTO whose locked fields are `{Value:"", Locked:true, Secret:true, Unset:true}`.

- [ ] **Step 1: Write the failing tests**

In `internal/configedit/secrets_test.go`, replace `TestRemovePINDecryptsEverythingAndKeepsValuesMasked` with:

```go
func TestRemovePINLeavesDeclarationsUnset(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	updated, err := a.RemoveSecretsPIN("demo1234", dto)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Secrets != nil {
		t.Error("params still set")
	}
	for _, f := range []FieldDTO{updated.EnvFields[0], updated.Items[0].Fields[0]} {
		if !f.Unset || !f.Locked || f.Value != "" {
			t.Errorf("field %q = %+v, want an unset declaration", f.Key, f)
		}
	}
	if updated.Items[0].Fields[1].Unset || updated.Items[0].Fields[1].Value != "example" {
		t.Errorf("plain field changed: %+v", updated.Items[0].Fields[1])
	}
}

func TestSaveAfterRemovePINDeletesLocalFile(t *testing.T) {
	withAppData(t)
	a, dto := lockedDraft(t, "demo1234")
	path := filepath.Join(t.TempDir(), "config.yaml")
	if _, err := a.Save(dto, path); err != nil {
		t.Fatal(err)
	}
	id := a.cfg.SecretsID
	if _, err := os.Stat(config.LocalSecretsPath(id)); err != nil {
		t.Fatalf("local file not written: %v", err)
	}
	dto.SecretsID = id
	updated, err := a.RemoveSecretsPIN("demo1234", dto)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Save(updated, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.LocalSecretsPath(id)); !os.IsNotExist(err) {
		t.Errorf("local file still there after Remove PIN + save: %v", err)
	}
}
```

Add at the top of `secrets_test.go` (imports `os`, `path/filepath`, `script-manager/internal/config`):

```go
func withAppData(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
}
```

Append to `internal/configedit/convert_test.go`:

```go
func TestFieldsUnsetRoundTrip(t *testing.T) {
	fields := FieldsFromMap(map[string]any{"pw": secret.Unset})
	if len(fields) != 1 || !fields[0].Unset || !fields[0].Locked || fields[0].Value != "" {
		t.Fatalf("FieldsFromMap = %+v", fields)
	}
	back, err := FieldsToMap(fields)
	if err != nil || !secret.IsUnset(back["pw"]) {
		t.Errorf("FieldsToMap = %v, %v", back, err)
	}
}
```

(Add the `script-manager/internal/secret` import there if it's missing.)

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/configedit/`
Expected: compile failure, `unknown field Unset` / `dto.SecretsID undefined`.

- [ ] **Step 3: DTOs** (`internal/configedit/dto.go`)

```go
type FieldDTO struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
	Locked bool   `json:"locked"`
	Unset  bool   `json:"unset"`
}
```

Add `SecretsID string \`json:"secretsId"\`` as the first field of `ConfigDTO`.

- [ ] **Step 4: Conversion** (`internal/configedit/convert.go`)

In `FieldsFromMap`, at the top of the loop body:

```go
		if secret.IsUnset(m[k]) {
			fields = append(fields, FieldDTO{Key: k, Kind: "string", Secret: true, Locked: true, Unset: true})
			continue
		}
```

In `FieldsToMap`, after the empty-key check:

```go
		if f.Unset {
			out[f.Key] = secret.Unset
			continue
		}
```

In `ToConfigDTO` add `SecretsID: cfg.SecretsID,`. In `FromConfigDTO` add `SecretsID: dto.SecretsID,`.

- [ ] **Step 5: Save and Remove PIN** (`internal/configedit/app.go`, `internal/configedit/secrets.go`)

Add the field `dropLocalSecrets bool` to `App`. Clear it (`a.dropLocalSecrets = false`) in `stateFor` and `NewBlank`, next to where `secretKey` is reset.

Replace the body of `Save` after `FromConfigDTO`:

```go
	cfg, err := FromConfigDTO(state)
	if err != nil {
		return SaveResultDTO{}, err
	}
	if err := cfg.Save(path); err != nil {
		return SaveResultDTO{}, err
	}
	if a.dropLocalSecrets && cfg.Secrets == nil {
		if err := config.RemoveLocalSecrets(cfg.SecretsID); err != nil {
			return SaveResultDTO{}, err
		}
	}
	a.dropLocalSecrets = false
	cfg.SourcePath = path
	a.cfg = cfg
	a.path = path
	recent.Add(a.appDataDir, path)
	return SaveResultDTO{Path: path}, nil
```

Remove the now-unused `os` import if the compiler says so.

Replace `RemoveSecretsPIN` in `secrets.go`:

```go
// RemoveSecretsPIN keeps every locked field declared as a secret but empties
// it; the next save deletes this user's local secrets file.
func (a *App) RemoveSecretsPIN(pin string, dto ConfigDTO) (ConfigDTO, error) {
	current := secretsFromDTO(dto.Secrets)
	if current == nil {
		return dto, fmt.Errorf("this config has no PIN set")
	}
	if _, err := current.Unlock(pin); err != nil {
		return dto, err
	}
	clear := func(fields []FieldDTO) {
		for i := range fields {
			if fields[i].Locked {
				fields[i] = FieldDTO{Key: fields[i].Key, Kind: "string", Secret: true, Locked: true, Unset: true}
			}
		}
	}
	clear(dto.EnvFields)
	for i := range dto.Items {
		clear(dto.Items[i].Fields)
	}
	dto.Secrets = nil
	a.secretKey = nil
	a.secretParams = nil
	a.dropLocalSecrets = true
	return dto, nil
}

func (a *App) NewSecretsID() (string, error) {
	return config.NewSecretsID()
}

func (a *App) SecretsFilePath(id string) string {
	return config.LocalSecretsPath(id)
}
```

Add the `script-manager/internal/config` import to `secrets.go`. `ChangeSecretsPIN` needs no change: an unset field's `Value` is `""`, which `IsLocked` skips.

- [ ] **Step 6: Run tests**

Run: `go test ./internal/configedit/ ./internal/config/ ./internal/secret/ && go vet ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/configedit
git commit -m "Config Editor: save through the shared/local split, Remove PIN keeps declarations"
```

---

### Task 5: Config Editor frontend

**Files:**
- Modify: `cmd/sm-config-edit/frontend/src/components/FieldGrid.svelte`
- Modify: `cmd/sm-config-edit/frontend/src/components/ItemsEditor.svelte`
- Modify: `cmd/sm-config-edit/frontend/src/components/SecretsEditor.svelte`
- Modify: `cmd/sm-config-edit/frontend/src/App.svelte`
- Modify: `internal/messages/configedit.json`
- Regenerated: `cmd/sm-config-edit/frontend/wailsjs/go/configedit/App.{js,d.ts}`, `wailsjs/go/models.ts`

**Interfaces:**
- Consumes: the bindings `NewSecretsID`, `SecretsFilePath`, `LockValue`, and the DTO fields `unset` / `secretsId` (Task 4).
- Produces: a new FieldGrid prop, `onFillSecret: ((plain: string) => Promise<string | null>) | null`.

- [ ] **Step 1: Regenerate the Wails bindings**

Run (PowerShell, repo root): `.\build-container.ps1 -Windows`
Expected: build succeeds, and `git status` shows `wailsjs/go/configedit/App.*` and `models.ts` changed, with `NewSecretsID`, `SecretsFilePath`, `unset` and `secretsId` present.

- [ ] **Step 2: Messages** (`internal/messages/configedit.json`)

In `"secrets"`, replace the four entries named here, and add the new keys:

```json
    "setHint": "Set a PIN to start locking environment values. Locked values are encrypted with it and kept in a secrets file on this computer only; config.yaml just marks them as secret, so it's safe to share.",
    "removeHint": "Removing the PIN deletes this computer's secrets file when you save. The values stay marked secret in config.yaml, but show as not set until you enter them again.",
    "removeConfirm": "Remove the PIN? All {count} locked value(s) will be deleted from this computer when you save, and have to be entered again.",
    "localPath": "Secrets file on this computer:",
    "localPathNone": "Not created yet. It is written on the next save after a PIN is set.",
    "newIdButton": "Give this config its own secrets",
    "newIdHint": "Use this on a config copied from another one, so the two stop sharing a secrets file. The values you have now move to the new file on the next save.",
    "newIdConfirm": "Give this config its own secrets file? The current values are written to the new file on the next save; the old file is left in place for the config it came from."
```

In `"toast"`, replace `pinRemoved`:

```json
    "pinRemoved": "PIN removed. Save the config to delete this computer's secrets file.",
```

In `"text"`, add:

```json
    "secretNotSet": "Not set on this computer — type a value to lock it",
```

In `"tooltip"`, replace `pinSetMessage`:

```json
    "pinSetMessage": "Values you lock are encrypted with this PIN and kept in a secrets file on this computer, not in config.yaml. There is no way to recover a forgotten PIN — the locked values would have to be re-entered.",
```

- [ ] **Step 3: FieldGrid unset fields** (`FieldGrid.svelte`)

Change the `fields` prop type to include `unset?: boolean`, and add a prop:

```ts
  export let fields: { key: string; kind: string; value: string; secret: boolean; locked?: boolean; unset?: boolean }[] = []
  export let onFillSecret: ((plain: string) => Promise<string | null>) | null = null
```

Add below `toggleLock`:

```ts
  async function fillSecret(i: number, e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const plain = input.value
    if (!plain || !onFillSecret) return
    const locked = await onFillSecret(plain)
    input.value = ''
    if (locked === null) return
    fields[i] = { ...fields[i], value: locked, locked: true, unset: false, secret: true }
    fields = fields
  }
```

In the markup, add an `unset` branch as the **first** branch of the value `{#if}` chain, so it wins over the kind-based branches:

```svelte
      {#if fields[i].unset}
        <input
          class="field-value field-value-unset"
          type="password"
          autocomplete="off"
          placeholder={t('text.secretNotSet')}
          on:change={(e) => fillSecret(i, e)}
        />
      {:else if fields[i].kind === 'bool'}
```

(The existing `{#if fields[i].kind === 'bool'}` becomes `{:else if …}`.) Disable the padlock for unset fields, since there's nothing to reveal. Add `disabled={fields[i].unset}` to the PIN-protected `IconButton`. If `IconButton` doesn't forward `disabled`, wrap `toggleLock` instead: `on:click={() => !fields[i].unset && toggleLock(i)}`.

Add to `<style>`:

```css
  .field-value-unset::placeholder {
    color: var(--sm-warning, var(--sm-muted));
    font-style: italic;
  }
```

Check `frontend-shared/theme.css` for the actual warning/muted token names before using these two. Use whichever exist.

- [ ] **Step 4: Pass `onFillSecret` through ItemsEditor** (`ItemsEditor.svelte`)

Add the prop next to `onToggleLock`:

```ts
  export let onFillSecret: ((plain: string) => Promise<string | null>) | null = null
```

Then change line 229's FieldGrid to `<FieldGrid bind:fields={items[selectedItem].fields} {validateField} {onToggleLock} {onFillSecret} />`.

- [ ] **Step 5: SecretsEditor** (`SecretsEditor.svelte`)

Add props:

```ts
  export let secretsPath = ''
  export let onNewSecretsId: () => Promise<string>
```

Add a function:

```ts
  async function newId() {
    if (busy) return
    if (!(await confirmAsk(t('secrets.newIdConfirm')))) return
    busy = true
    error = await onNewSecretsId()
    busy = false
  }
```

Directly after the `.secrets-status` div, add:

```svelte
  <div class="secrets-path">
    <span>{t('secrets.localPath')}</span>
    {#if secretsPath}<code>{secretsPath}</code>{:else}<em>{t('secrets.localPathNone')}</em>{/if}
  </div>
```

Inside the `{#if configured}` remove block, before `.secrets-remove`, add a sibling block:

```svelte
  {#if configured}
    <div class="secrets-remove">
      <p class="hint">{t('secrets.newIdHint')}</p>
      <div class="secrets-actions">
        <button class="btn" type="button" disabled={busy} on:click={newId}>{t('secrets.newIdButton')}</button>
      </div>
    </div>
  {/if}
```

Style:

```css
  .secrets-path {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    font-size: var(--sm-type-small, var(--sm-type-base));
    word-break: break-all;
  }
```

- [ ] **Step 6: App wiring** (`App.svelte`)

Import `NewSecretsID` and `SecretsFilePath` from the same `wailsjs/go/configedit/App` import that already brings in `LockValue`. Add:

```ts
  async function fillSecret(plain: string): Promise<string | null> {
    if (!(await ensureUnlocked())) return null
    try {
      return await LockValue(plain)
    } catch (err) {
      flash(t('toast.lockFailed', { error: String(err) }))
      return null
    }
  }

  let secretsPath = ''
  $: SecretsFilePath(cfg?.secretsId ?? '').then((p) => (secretsPath = p))

  async function newSecretsId(): Promise<string> {
    try {
      cfg.secretsId = await NewSecretsID()
      cfg = cfg
      return ''
    } catch (err) {
      return String(err)
    }
  }
```

Change `lockedValueCount` to count only filled values, `(f) => f.locked && !f.unset`, in both places.

Pass `onFillSecret={fillSecret}` to the env `FieldGrid` (line ~587) and to `ItemsEditor` (line ~632). Pass `{secretsPath}` and `onNewSecretsId={newSecretsId}` to `SecretsEditor`.

`secretsPath` shows a path even before the file exists. That's fine: `localPathNone` only shows when there's no ID at all.

- [ ] **Step 7: Type-check**

Run: `cd cmd/sm-config-edit/frontend && npm run check`
Expected: 0 errors.

- [ ] **Step 8: Commit**

```bash
git add cmd/sm-config-edit/frontend internal/messages/configedit.json
git commit -m "Config Editor: fill unset secrets, show the secrets file, give a config its own secrets"
```

---

### Task 6: Examples

**Files:**
- Modify: `examples/pin-secrets.yaml`, `examples/pin-secrets-win.yaml`, `examples/README.md` (if it describes the PIN example)
- Create: `examples/pin-secrets.local.yaml` (the local half, for users to copy)
- Modify: `internal/config/examples_test.go`

**Interfaces:**
- Consumes: `LoadFromWithError`, `LocalSecretsPath`, `secret.IsUnset` (Tasks 1–2).

- [ ] **Step 1: Convert the examples**

In both example configs:
- Delete the `secrets:` block.
- Add `secretsId: 5e1ec7ed5e1ec7ed5e1ec7ed5e1ec7ed` at the top.
- Replace every `sm-enc:v1:…` value with `!secret`.

Move the removed `secrets:` block and the ciphertext into `examples/pin-secrets.local.yaml`, in the local format:

```yaml
secrets:
  salt: e5XgjyUaMO5ZKugmtCQhYA==
  time: 3
  memory: 65536
  threads: 4
  check: sm-enc:v1:h+Yu+4Ynn0gs+4lPMAMZS7JMW+k7mL4g3kZCz/ETZRx8rsIQry/AbCUc
env:
  vault_token: <the vault_token ciphertext from pin-secrets.yaml>
items:
  demo-database:
    db_password: <demo-database's ciphertext>
  demo-api:
    db_password: <demo-api's ciphertext>
```

Copy the ciphertext strings verbatim from the current `pin-secrets.yaml`. If `pin-secrets-win.yaml` uses different ciphertext or salt, check with `git diff --no-index`. If it does, give the Windows one its own ID (`5e1ec7ed5e1ec7ed5e1ec7ed5e1ec7ee`) and its own `pin-secrets-win.local.yaml`. Update `examples/README.md` to say where to copy the local file: `<app-data>/script-manager/secrets/<secretsId>.yaml`.

- [ ] **Step 2: Extend the example test** (`examples_test.go`)

After the existing loop, add:

```go
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
```

Add the imports `os`, `path/filepath` and `script-manager/internal/secret`. Also make the existing `TestExampleConfigsAreCurrent` set the two env vars, so it never reads a real user's app-data.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/config/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add examples internal/config/examples_test.go
git commit -m "Convert the PIN examples to a shared config plus a local secrets file"
```

---

### Task 7: README and CHANGELOG

**Files:**
- Modify: `README.md` ("PIN-locked values" section, ~line 229, and the Config Editor "Sections" bullet if it mentions PIN storage)
- Modify: `CHANGELOG.md`

- [ ] **Step 1: README.** Rewrite "#### PIN-locked values" as a user guide. Cover:
  - Locked values are kept in a secrets file on your computer, not in `config.yaml`. The config only marks them `name: !secret`, so it's safe to commit and share.
  - Where the file lives: `%AppData%\script-manager\secrets\` on Windows, `~/.config/script-manager/secrets/` on Linux, named after the `secretsId` the config gets when a PIN is first set.
  - Each person who uses a shared config sets their own PIN and types in their own values. Until then those values show as `(not set)` and scripts don't receive them.
  - A config copied from another one: use **Give this config its own secrets** in the PIN section.
  - **Remove PIN** now deletes this computer's secrets file on save. The values stay declared as secret.
  - A config saved by an earlier version has its locked values moved into the secrets file automatically the next time the Config Editor saves it, with the same PIN.
  - Keep the existing Requires PIN / "never rendered" / threat-model paragraphs. Change "someone reading your `config.yaml`" to also mention that the secrets file itself holds only encrypted values.

- [ ] **Step 2: CHANGELOG.** Add at the top, under the intro:

```markdown
## Unreleased

### Changes

- PIN-locked values now live in a per-user secrets file instead of the config, which only marks them `!secret` — a config with locked values is safe to commit and share, and each person sets their own PIN and values.
- A config saved by an earlier version has its locked values moved into the secrets file the next time `sm-config-edit` saves it; the PIN stays the same.
- `sm-config-edit`: Remove PIN deletes this computer's secrets file instead of writing the values into the config as plain text.
```

- [ ] **Step 3: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "Document per-user secrets files"
```

---

### Task 8: Final verification

- [ ] **Step 1:** Ask the user before running the full build (CLAUDE.md "Before committing or pushing"). With a yes, run `.\build-container.ps1 -Windows -Full`.
  Expected: vet, tests and svelte-check are clean, and the Windows binaries are built.
- [ ] **Step 2:** Hand over for manual verification. Don't run the apps. Suggest checking these:
  - open `examples/pin-secrets.yaml` in the GUI with no local file: values show `(not set)`
  - copy the local example into app-data: the PIN prompt works
  - migrate a legacy config in the editor, then `git diff` it
  - Remove PIN + save deletes the file
  - the new-ID button
