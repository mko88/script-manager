# Item env schema Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give an item a dedicated `env:` section so environment variables stop
sharing a namespace with the structural keys, and convert old configs once, on
load, with the user's approval.

**Architecture:** `config.Item` becomes a struct with six fields and a strict
unmarshaller, so "unknown key" is a load error rather than a silent env var.
`action.Merge` still hands every consumer the same flat map, so the mask
pipeline, secrets and template expansion are untouched below it. A new
`internal/configmigrate` package detects and rewrites the old shape; each of the
three binaries calls it before loading and asks the user first.

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v3`, Bubble Tea (TUI), Wails v2 (both
GUIs), Svelte 3 frontends (not touched by this plan).

**Spec:** `docs/superpowers/specs/2026-09-13-item-env-schema-design.md`

## Global Constraints

- **Never run the apps.** Build only; the user verifies. No `go run`, no
  launching anything from `bin/`, no Xvfb. (`CLAUDE.md`)
- **Never bump the version or create tags.** Changelog entries go under
  `## Unreleased`. (`CLAUDE.md`)
- **Ask before any `build.sh` / `build-container.ps1` run and wait for the
  answer.** Exception: diffs that are only docs, comments or build scripts.
- **No changelog narration in comments.** A comment describes the code as it is
  now, never what it used to be. Keep them condensed.
- Go work happens inside the dev container. Resolve it once per session:

  ```powershell
  $c = (docker ps --format "{{.Names}}" | Where-Object {
        (docker inspect $_ --format '{{ index .Config.Labels "devcontainer.local_folder" }}').TrimEnd('\') -ieq (Get-Location).Path.TrimEnd('\') })
  ```

  Then run tests as:

  ```powershell
  docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/config/... -v"
  ```

- Branch is `20260913-item-env-schema`, already created. Commit per task.
- The six item keys are exactly: `name`, `display`, `actions`, `actionGroups`,
  `customActions`, `env`.

---

### Task 1: `config.Item`

Additive — nothing changes type yet, so the tree keeps compiling.

**Files:**
- Create: `internal/config/item.go`
- Create: `internal/config/item_test.go`
- Modify: `internal/config/config.go:30-36` (add `KeyEnv` to the const block)

**Interfaces:**
- Consumes: nothing.
- Produces: `config.Item` struct; `(*Item).Values() map[string]any`;
  `config.ItemKey(key string) bool`; `config.KeyEnv = "env"`.

- [ ] **Step 1: Write the failing tests**

`internal/config/item_test.go`:

```go
package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestItemUnmarshalNewShape(t *testing.T) {
	src := `
name: srv1
display: prod
actions: [ssh, logs]
actionGroups: [remote]
customActions:
  - title: Tail
    cmd: tail -f /var/log/app.log
    requiresPin: true
env:
  sshUser: root
  dbHost: db1.internal
`
	var item Item
	if err := yaml.Unmarshal([]byte(src), &item); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if item.Name != "srv1" || item.Display != "prod" {
		t.Errorf("Name/Display = %q/%q", item.Name, item.Display)
	}
	if !reflect.DeepEqual(item.Actions, []string{"ssh", "logs"}) {
		t.Errorf("Actions = %v", item.Actions)
	}
	if !reflect.DeepEqual(item.ActionGroups, []string{"remote"}) {
		t.Errorf("ActionGroups = %v", item.ActionGroups)
	}
	if len(item.CustomActions) != 1 || !item.CustomActions[0].RequiresPIN {
		t.Errorf("CustomActions = %+v", item.CustomActions)
	}
	if item.Env["sshUser"] != "root" || item.Env["dbHost"] != "db1.internal" {
		t.Errorf("Env = %v", item.Env)
	}
}

func TestItemUnmarshalRejectsUnknownKey(t *testing.T) {
	var item Item
	err := yaml.Unmarshal([]byte("name: srv1\nsshUser: root\n"), &item)
	if err == nil {
		t.Fatal("Unmarshal() error = nil, want an error naming the unknown key")
	}
	for _, want := range []string{"srv1", "sshUser", "env"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestItemEnvMayShadowStructuralNames(t *testing.T) {
	src := "name: srv1\nenv:\n  display: screen-0\n  actions: two\n"
	var item Item
	if err := yaml.Unmarshal([]byte(src), &item); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if item.Display != "" || item.Actions != nil {
		t.Errorf("env keys leaked into the struct: Display=%q Actions=%v", item.Display, item.Actions)
	}
	values := item.Values()
	if values["display"] != "screen-0" || values["actions"] != "two" {
		t.Errorf("Values() = %v, want the env values intact", values)
	}
}

func TestItemValues(t *testing.T) {
	item := Item{
		Name:    "srv1",
		Display: "prod",
		Actions: []string{"ssh"},
		Env:     map[string]any{"sshUser": "root"},
	}
	got := item.Values()
	want := map[string]any{"name": "srv1", "sshUser": "root"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Values() = %v, want %v", got, want)
	}
}

func TestItemValuesNameWinsOverEnv(t *testing.T) {
	item := Item{Name: "srv1", Env: map[string]any{"name": "shadowed"}}
	if got := item.Values()["name"]; got != "srv1" {
		t.Errorf("Values()[name] = %v, want srv1", got)
	}
}

func TestItemValuesEmptyNameNotSet(t *testing.T) {
	item := Item{Env: map[string]any{"sshUser": "root"}}
	if _, ok := item.Values()["name"]; ok {
		t.Error("Values() set name for an item with no name")
	}
}

func TestItemKey(t *testing.T) {
	for _, k := range []string{"name", "display", "actions", "actionGroups", "customActions", "env"} {
		if !ItemKey(k) {
			t.Errorf("ItemKey(%q) = false, want true", k)
		}
	}
	if ItemKey("sshUser") {
		t.Error("ItemKey(sshUser) = true, want false")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/config/ -run 'TestItem' -v"
```

Expected: FAIL — `undefined: Item`.

- [ ] **Step 3: Add `KeyEnv`**

In `internal/config/config.go`, extend the existing const block:

```go
const (
	KeyName          = "name"
	KeyDisplay       = "display"
	KeyActions       = "actions"
	KeyActionGroups  = "actionGroups"
	KeyCustomActions = "customActions"
	KeyEnv           = "env"
)
```

- [ ] **Step 4: Create `internal/config/item.go`**

```go
package config

import (
	"fmt"
	"maps"

	"gopkg.in/yaml.v3"
)

type Item struct {
	Name          string         `yaml:"name,omitempty"`
	Display       string         `yaml:"display,omitempty"`
	Actions       []string       `yaml:"actions,omitempty"`
	ActionGroups  []string       `yaml:"actionGroups,omitempty"`
	CustomActions []Action       `yaml:"customActions,omitempty"`
	Env           map[string]any `yaml:"env,omitempty"`
}

var itemKeys = map[string]bool{
	KeyName:          true,
	KeyDisplay:       true,
	KeyActions:       true,
	KeyActionGroups:  true,
	KeyCustomActions: true,
	KeyEnv:           true,
}

// ItemKey reports whether key is one of an item's six structural keys.
func ItemKey(key string) bool {
	return itemKeys[key]
}

// rawItem drops the UnmarshalYAML method so Decode doesn't recurse into it.
type rawItem Item

func (i *Item) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("item: expected a mapping, got YAML node kind %v", value.Kind)
	}
	var raw rawItem
	if err := value.Decode(&raw); err != nil {
		return err
	}
	for n := 0; n+1 < len(value.Content); n += 2 {
		key := value.Content[n].Value
		if !itemKeys[key] {
			name := raw.Name
			if name == "" {
				name = "(unnamed)"
			}
			return fmt.Errorf("item %q: unknown key %q — environment variables belong under %q", name, key, KeyEnv)
		}
	}
	*i = Item(raw)
	return nil
}

// Values is the flat map templates and scripts see: the item's env, with name
// on top so a stray env key called "name" can't shadow the item's own.
func (i *Item) Values() map[string]any {
	out := make(map[string]any, len(i.Env)+1)
	maps.Copy(out, i.Env)
	if i.Name != "" {
		out[KeyName] = i.Name
	}
	return out
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/config/ -run 'TestItem' -v"
```

Expected: PASS, all seven.

- [ ] **Step 6: Commit**

```bash
git add internal/config/item.go internal/config/item_test.go internal/config/config.go
git commit -m "Add a typed config.Item with a separate env section"
```

---

### Task 2: `internal/configmigrate`

Still additive. Detection and conversion work on loosely-typed YAML, so they are
independent of Task 4's type flip and can be written and tested now.

**Files:**
- Create: `internal/configmigrate/migrate.go`
- Create: `internal/configmigrate/migrate_test.go`
- Modify: `internal/config/config.go:250-279` (extract `searchPaths`, add
  `ResolvePath`)

**Interfaces:**
- Consumes: `config.ItemKey`, `config.KeyEnv` (Task 1).
- Produces:
  - `config.ResolvePath() (string, error)`
  - `configmigrate.Needed(path string) (bool, error)`
  - `configmigrate.BackupPath(path string, now time.Time) string`
  - `configmigrate.Convert(path, backupPath string) error`
  - `configmigrate.Ensure(path string, approve func(path, backupPath string) bool) (string, error)`
  - `configmigrate.ErrDeclined`

- [ ] **Step 1: Write the failing tests**

`internal/configmigrate/migrate_test.go`:

```go
package configmigrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const oldShape = `shell: [bash, -c]
items:
  - name: srv1
    display: prod
    actions: [ssh]
    sshUser: root
    dbHost: db1
  - name: srv2
    env:
      region: eu
actions:
  - id: ssh
    title: SSH
    cmd: ssh {{.sshUser}}@{{.dbHost}}
`

const newShape = `shell: [bash, -c]
items:
  - name: srv1
    display: prod
    actions: [ssh]
    env:
      sshUser: root
actions:
  - id: ssh
    title: SSH
    cmd: ssh
`

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNeeded(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"old shape", oldShape, true},
		{"new shape", newShape, false},
		{"no items", "shell: [bash, -c]\n", false},
		{"empty file", "", false},
		{"empty item list", "items: []\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Needed(write(t, tt.body))
			if err != nil {
				t.Fatalf("Needed() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Needed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeededMalformedIsAnError(t *testing.T) {
	if _, err := Needed(write(t, "items: [\n")); err == nil {
		t.Fatal("Needed() error = nil, want a parse error")
	}
}

func TestNeededMissingFileIsAnError(t *testing.T) {
	if _, err := Needed(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("Needed() error = nil, want a read error")
	}
}

func TestBackupPath(t *testing.T) {
	now := time.Date(2026, 9, 13, 22, 45, 0, 0, time.UTC)
	got := BackupPath("/tmp/config-win.yaml", now)
	want := "/tmp/config-win.yaml.20260913-2245.bak"
	if got != want {
		t.Errorf("BackupPath() = %q, want %q", got, want)
	}
}

func TestConvert(t *testing.T) {
	path := write(t, oldShape)
	backup := path + ".bak"
	if err := Convert(path, backup); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	saved, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("backup not written: %v", err)
	}
	if string(saved) != oldShape {
		t.Error("backup is not the original bytes")
	}

	needed, err := Needed(path)
	if err != nil {
		t.Fatalf("Needed() after Convert error = %v", err)
	}
	if needed {
		t.Error("Needed() after Convert = true, want false")
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	for _, want := range []string{"sshUser: root", "dbHost: db1", "region: eu"} {
		if !strings.Contains(body, want) {
			t.Errorf("converted file lost %q:\n%s", want, body)
		}
	}
}

func TestConvertKeepsExistingEnv(t *testing.T) {
	path := write(t, "items:\n  - name: srv\n    env:\n      a: 1\n    b: 2\n")
	if err := Convert(path, path+".bak"); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	out, _ := os.ReadFile(path)
	body := string(out)
	if !strings.Contains(body, "a: 1") || !strings.Contains(body, "b: 2") {
		t.Errorf("Convert() dropped a key:\n%s", body)
	}
}

func TestEnsureConvertsWhenApproved(t *testing.T) {
	path := write(t, oldShape)
	var gotPath, gotBackup string
	backup, err := Ensure(path, func(p, b string) bool {
		gotPath, gotBackup = p, b
		return true
	})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if gotPath != path {
		t.Errorf("approve got path %q, want %q", gotPath, path)
	}
	if backup != gotBackup || backup == "" {
		t.Errorf("Ensure() = %q, approve saw %q", backup, gotBackup)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Errorf("backup %q not written: %v", backup, err)
	}
}

func TestEnsureDeclined(t *testing.T) {
	path := write(t, oldShape)
	_, err := Ensure(path, func(string, string) bool { return false })
	if !errors.Is(err, ErrDeclined) {
		t.Fatalf("Ensure() error = %v, want ErrDeclined", err)
	}
	out, _ := os.ReadFile(path)
	if string(out) != oldShape {
		t.Error("Ensure() rewrote the file after being declined")
	}
}

func TestEnsureNoopOnNewShape(t *testing.T) {
	path := write(t, newShape)
	called := false
	backup, err := Ensure(path, func(string, string) bool { called = true; return true })
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if called {
		t.Error("Ensure() asked for approval on an already-converted config")
	}
	if backup != "" {
		t.Errorf("Ensure() = %q, want empty", backup)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/configmigrate/ -v"
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Create `internal/configmigrate/migrate.go`**

```go
// Package configmigrate converts configs written before items had their own
// env section, where any key that wasn't structural was an environment
// variable.
package configmigrate

import (
	"errors"
	"fmt"
	"os"
	"time"

	"script-manager/internal/config"

	"gopkg.in/yaml.v3"
)

// ErrDeclined reports that the user was asked to convert and said no.
var ErrDeclined = errors.New("config conversion declined")

// Needed reports whether any item in the file carries a key outside the six.
func Needed(path string) (bool, error) {
	doc, err := read(path)
	if err != nil {
		return false, err
	}
	for _, item := range items(doc) {
		for key := range item {
			if !config.ItemKey(key) {
				return true, nil
			}
		}
	}
	return false, nil
}

// BackupPath is where Convert saves the original, timestamped so it never
// overwrites a backup taken by hand.
func BackupPath(path string, now time.Time) string {
	return fmt.Sprintf("%s.%s.bak", path, now.Format("20060102-1504"))
}

// Convert saves the file to backupPath, then rewrites each item's non-
// structural keys under env. Comments do not survive the rewrite.
func Convert(path, backupPath string) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(original, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	for _, item := range items(doc) {
		env, _ := item[config.KeyEnv].(map[string]any)
		for key, value := range item {
			if config.ItemKey(key) {
				continue
			}
			if env == nil {
				env = make(map[string]any)
			}
			env[key] = value
			delete(item, key)
		}
		if len(env) > 0 {
			item[config.KeyEnv] = env
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(backupPath, original, 0o644); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// Ensure converts path if it needs it, asking approve first. It returns the
// backup path when a conversion happened, "" when none was needed, and
// ErrDeclined when approve said no.
func Ensure(path string, approve func(path, backupPath string) bool) (string, error) {
	needed, err := Needed(path)
	if err != nil || !needed {
		return "", err
	}
	backup := BackupPath(path, time.Now())
	if !approve(path, backup) {
		return "", ErrDeclined
	}
	if err := Convert(path, backup); err != nil {
		return "", err
	}
	return backup, nil
}

func read(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}

func items(doc map[string]any) []map[string]any {
	raw, _ := doc["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, elem := range raw {
		if item, ok := elem.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/configmigrate/ -v"
```

Expected: PASS.

- [ ] **Step 5: Add `config.ResolvePath`**

The gate runs before loading, so it needs the path `LoadWithError` would pick.
In `internal/config/config.go`, extract the path list out of `LoadWithError`
and add a resolver beside it. Replace `LoadWithError` (currently lines 250-279)
with:

```go
func configNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"config-win.yaml", "config.yaml"}
	}
	return []string{"config.yaml"}
}

func searchPaths() []string {
	names := configNames()

	var exeDir string
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	dataDir := appdata.Dir()

	var paths []string
	for _, name := range names {
		if exeDir != "" {
			paths = append(paths, filepath.Join(exeDir, name))
		}
		paths = append(paths, name)
	}
	if dataDir != "" {
		for _, name := range names {
			paths = append(paths, filepath.Join(dataDir, name))
		}
	}
	return paths
}

// ResolvePath is the file LoadWithError would read, creating the starter
// config if nothing exists yet. Callers that must inspect the file before
// loading it — the format check on startup — use this to find it.
func ResolvePath() (string, error) {
	paths := searchPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	dataDir := appdata.Dir()
	if dataDir == "" {
		return "", fmt.Errorf("no config file found (tried %s)", strings.Join(paths, ", "))
	}
	defaultPath := filepath.Join(dataDir, configNames()[0])
	if err := os.WriteFile(defaultPath, []byte(defaultConfigYAML()), 0o644); err != nil {
		return "", err
	}
	return defaultPath, nil
}

func LoadWithError() (*Config, error) {
	paths := searchPaths()
	dataDir := appdata.Dir()
	if dataDir == "" {
		return loadPaths(paths)
	}
	return loadOrCreate(paths, filepath.Join(dataDir, configNames()[0]), defaultConfigYAML())
}
```

- [ ] **Step 6: Verify the config package still passes**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/config/... && go build ./..."
```

Expected: PASS and a clean build — this step is a pure refactor.

- [ ] **Step 7: Commit**

```bash
git add internal/configmigrate internal/config/config.go
git commit -m "Add configmigrate: detect and convert pre-env item configs"
```

---

### Task 3: Flip the item type across the tree

The breaking, atomic one. `Config.Items` changes type, so **the tree does not
compile until the last step of this task.** Work through the steps in order and
let the compiler drive; run `go build ./...` after each package.

**Files:**
- Modify: `internal/config/config.go` (`Config.Items`, `FindDisplay`,
  `ActionsForItem`; delete `AsStringSlice`, `ParseCustomActions`)
- Modify: `internal/action/action.go:12-17` (`Merge`)
- Modify: `internal/render/allenv.go:28-33,67-88` (delete `reservedKeys`)
- Modify: `internal/gui/app.go`, `internal/gui/details.go:30`,
  `internal/gui/secrets.go:51`
- Modify: `internal/ui/app.go`, `internal/ui/list.go`, `internal/ui/detail.go`,
  `internal/ui/secrets.go`
- Modify: `internal/configedit/convert.go`, `internal/configedit/preview.go`
- Test: every `*_test.go` that builds an item literal — `internal/config`,
  `internal/render`, `internal/action`, `internal/gui`, `internal/configedit`

**Interfaces:**
- Consumes: `config.Item`, `(*Item).Values()` (Task 1).
- Produces:
  - `config.Config.Items []Item`
  - `config.FindDisplay(displays DisplayList, item *Item) DisplayConfig`
  - `config.ActionsForItem(allActions []Action, item *Item) []Action`
  - `action.Merge(env map[string]any, item *config.Item) map[string]any`
  - `(*ui.DescriptionTile).SetItem(item *config.Item, merged map[string]any)`

- [ ] **Step 1: Write the failing tests for the new signatures**

Add to `internal/action/action_test.go`:

```go
func TestMergePrecedence(t *testing.T) {
	global := map[string]any{"region": "eu", "sshUser": "global"}
	item := &config.Item{
		Name:    "srv1",
		Display: "prod",
		Actions: []string{"ssh"},
		Env:     map[string]any{"sshUser": "root"},
	}
	got := Merge(global, item)
	want := map[string]any{"region": "eu", "sshUser": "root", "name": "srv1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Merge() = %v, want %v", got, want)
	}
}

func TestMergeNilItem(t *testing.T) {
	got := Merge(map[string]any{"region": "eu"}, nil)
	if !reflect.DeepEqual(got, map[string]any{"region": "eu"}) {
		t.Errorf("Merge(nil item) = %v", got)
	}
}

func TestEnvOmitsStructuralKeys(t *testing.T) {
	merged := Merge(nil, &config.Item{
		Name:          "srv1",
		Display:       "prod",
		Actions:       []string{"ssh"},
		ActionGroups:  []string{"remote"},
		CustomActions: []config.Action{{Title: "x"}},
		Env:           map[string]any{"sshUser": "root"},
	})
	for _, line := range Env(merged) {
		for _, banned := range []string{"DISPLAY=", "ACTIONS=", "ACTIONGROUPS=", "CUSTOMACTIONS="} {
			if strings.HasPrefix(line, banned) {
				t.Errorf("Env() exported %q", line)
			}
		}
	}
}
```

Add `"reflect"`, `"strings"` and `"script-manager/internal/config"` to that
file's imports.

- [ ] **Step 2: Run them to verify they fail**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go test ./internal/action/ -run 'TestMerge|TestEnvOmits' -v"
```

Expected: FAIL to compile — `Merge` does not take `*config.Item`.

- [ ] **Step 3: `internal/config`**

In `config.go`: change the field, retype the two helpers, delete the two
map-digging helpers.

```go
type Config struct {
	Shell        []string       `yaml:"shell,omitempty"`
	Display      DisplayList    `yaml:"display,omitempty"`
	Terminal     TerminalConfig `yaml:"terminal,omitempty"`
	Secrets      *secret.Params `yaml:"secrets,omitempty"`
	Env          map[string]any `yaml:"env,omitempty"`
	Items        []Item         `yaml:"items,omitempty"`
	ActionGroups []ActionGroup  `yaml:"actionGroups,omitempty"`
	Actions      []Action       `yaml:"actions,omitempty"`

	SourcePath string `yaml:"-"`
}

func FindDisplay(displays DisplayList, item *Item) DisplayConfig {
	if len(displays) == 0 {
		return DisplayConfig{}
	}
	if item != nil && item.Display != "" {
		for _, d := range displays {
			if d.Name == item.Display {
				return d
			}
		}
	}
	return displays[0]
}

func ActionsForItem(allActions []Action, item *Item) []Action {
	if item == nil {
		return allActions
	}
	if len(item.Actions) == 0 && len(item.ActionGroups) == 0 && len(item.CustomActions) == 0 {
		return allActions
	}

	seen := make(map[int]bool)
	var result []Action

	if len(item.Actions) > 0 {
		idSet := make(map[string]bool, len(item.Actions))
		for _, id := range item.Actions {
			idSet[id] = true
		}
		for i, a := range allActions {
			if a.ID != "" && idSet[a.ID] && !seen[i] {
				result = append(result, a)
				seen[i] = true
			}
		}
	}

	if len(item.ActionGroups) > 0 {
		groupSet := make(map[string]bool, len(item.ActionGroups))
		for _, g := range item.ActionGroups {
			groupSet[g] = true
		}
		for i, a := range allActions {
			if seen[i] {
				continue
			}
			for _, g := range a.Groups {
				if groupSet[g] {
					result = append(result, a)
					seen[i] = true
					break
				}
			}
		}
	}

	return append(result, item.CustomActions...)
}
```

Delete `AsStringSlice` and `ParseCustomActions` (config.go:188-240). Keep
`StrVal` — `configedit` still uses it elsewhere; delete it too if the compiler
says it is unused at the end of this task.

Rewrite the item literals in `internal/config/config_test.go` and
`marshal_test.go` to `Item{...}` values, e.g. `map[string]any{KeyActions:
[]any{"logs", "ssh"}}` becomes `&Item{Actions: []string{"logs", "ssh"}}`.

- [ ] **Step 4: `internal/action`**

`item` is nil whenever nothing is selected, so it must not be dereferenced for
the capacity hint:

```go
func Merge(env map[string]any, item *config.Item) map[string]any {
	merged := make(map[string]any, len(env)+1)
	maps.Copy(merged, env)
	if item != nil {
		maps.Copy(merged, item.Values())
	}
	return merged
}
```

Add `"script-manager/internal/config"` to the imports. `Expand`, `Preview` and
`Env` keep their `map[string]any` signatures — they operate on the merged map.

- [ ] **Step 5: `internal/render`**

Delete the `reservedKeys` var (allenv.go:28-33) and the `if reservedKeys[k] {
continue }` guard in `allEnvEntries` (allenv.go:71-73). Drop the now-unused
`"script-manager/internal/config"` import. `allEnvEntries` still takes the
merged `map[string]any`; nothing structural reaches it any more.

In `internal/render/allenv_test.go`, delete the structural keys from the test
item (lines 90-93) and the assertions that they were filtered out; replace with
an assertion that the env keys are present.

- [ ] **Step 6: `internal/gui`**

- `app.go:132-137` — `GetItems` ranges `a.cfg.Items` ( `[]config.Item`); pass
  `&a.cfg.Items[i]` to `renderListLabel`.
- `app.go:139-146` — `renderListLabel(item *config.Item)`:

```go
func (a *App) renderListLabel(item *config.Item) string {
	d := config.FindDisplay(a.cfg.Display, item)
	out, err := action.Expand(d.List, item.Values())
	if err != nil {
		return item.Name
	}
	return out
}
```

- `app.go:156-170` — `mergedItem(item *config.Item)` and
  `mergedItemForRun(item *config.Item, act config.Action)`; bodies unchanged
  apart from the parameter type.
- `app.go:172-177` — `itemAt(index int) *config.Item` returns
  `&a.cfg.Items[index]`.
- `details.go:30` — **behaviour bug if missed.** It currently resolves the
  display from the *merged* map, which no longer carries `display`. Change
  `config.FindDisplay(a.cfg.Display, merged)` to
  `config.FindDisplay(a.cfg.Display, item)`.
- `secrets.go:51` — `secret.HasLocked(item)` becomes
  `secret.HasLocked(item.Env)`; the loop variable is a `config.Item` value, so
  `for i := range a.cfg.Items` and index it, or range by value and use
  `item.Env` directly (no pointer needed for a map read).

Update `internal/gui/secrets_test.go` and `inline_test.go` and
`actiondetail_test.go` item literals to `config.Item{Name: ..., Env:
map[string]any{...}}`. Line 168 of `secrets_test.go`
(`append(cfg.Items, map[string]any{"name": "plain", "password": "not-locked"})`)
becomes `append(cfg.Items, config.Item{Name: "plain", Env: map[string]any{"password": "not-locked"}})`.

- [ ] **Step 7: `internal/ui`**

- `list.go` — `ListTile.items` becomes `[]config.Item`; `SetItems(items
  []config.Item, displays config.DisplayList)`; `Selected() *config.Item`
  returns `&t.items[t.selected]` (nil when the list is empty);
  `renderLabel(item *config.Item)` mirrors the GUI version above, executing the
  template against `item.Values()` and falling back to `item.Name`.
- `detail.go:138,180` — `DescriptionTile` needs the item for the display lookup
  *and* the merged map for rendering, because the merged map no longer carries
  `display`:

```go
type DescriptionTile struct {
	// ...
	item   *config.Item
	merged map[string]any
}

func (t *DescriptionTile) SetItem(item *config.Item, merged map[string]any) {
	t.item = item
	t.merged = merged
}
```

  `renderItem` keeps `config.FindDisplay(t.displays, t.item)` and switches every
  other use of `t.item` (lines 219, 224, 268, and the `FillMissingFields` call)
  to `t.merged`.
- `app.go:95,114,238` — `a.description.SetItem(a.list.Selected(),
  a.mergedItem(a.list.Selected()))`.
- `app.go:245-284` — `mergedItem(item *config.Item)` and
  `mergedItemForRun(item *config.Item, act config.Action)`.
- `app.go:53,106` — `newListTile(cfg.Items, cfg.Display)` and
  `SetItems(cfg.Items, cfg.Display)` now pass `[]config.Item`; no call-site
  change beyond the type.
- `secrets.go:21` — the parameter becomes `*config.Item`; the body is unchanged.

- [ ] **Step 8: `internal/configedit`**

- `convert.go:190-196` — delete `reservedItemKeys`.
- `convert.go:198-217` — `ToItemDTO(item config.Item) ItemDTO`:

```go
func ToItemDTO(item config.Item) ItemDTO {
	dto := ItemDTO{
		Name:          item.Name,
		Display:       item.Display,
		Actions:       item.Actions,
		ActionGroups:  item.ActionGroups,
		CustomActions: []ActionDTO{},
	}
	if dto.Actions == nil {
		dto.Actions = []string{}
	}
	if dto.ActionGroups == nil {
		dto.ActionGroups = []string{}
	}
	for _, a := range item.CustomActions {
		dto.CustomActions = append(dto.CustomActions, actionToDTO(a))
	}
	dto.Fields = FieldsFromMap(item.Env, nil)
	return dto
}
```

- `convert.go:219-252` — `FromItemDTO(dto ItemDTO) (config.Item, error)`:

```go
func FromItemDTO(dto ItemDTO) (config.Item, error) {
	item := config.Item{
		Name:         dto.Name,
		Display:      dto.Display,
		Actions:      dto.Actions,
		ActionGroups: dto.ActionGroups,
	}
	for _, a := range dto.CustomActions {
		item.CustomActions = append(item.CustomActions, actionFromDTO(a))
	}
	env, err := FieldsToMap(dto.Fields)
	if err != nil {
		name := dto.Name
		if name == "" {
			name = "(unnamed)"
		}
		return config.Item{}, fmt.Errorf("item %q: %w", name, err)
	}
	if len(env) > 0 {
		item.Env = env
	}
	return item, nil
}
```

  Note this replaces `actionDTOToMap` for item custom actions — check whether
  `actionDTOToMap` (convert.go:164-188) still has a caller; delete it if not.
- `preview.go:24-32` and `78-86` — `itemMap` becomes a `config.Item`; the
  `action.Merge(env, itemMap)` calls become `action.Merge(env, &itemMap)`.
- `convert.go:296-297,324-329` — the `ConfigDTO` round trip needs no change
  beyond the element type.
- `FieldsFromMap(m map[string]any, exclude map[string]bool)` now only ever gets
  `nil` for `exclude`. Drop the parameter and update its other callers (the
  global env fields), or leave it — the compiler will say which callers exist.

Add a round-trip test to `internal/configedit/convert_test.go`, replacing the
map-literal fixture at lines 146-149:

```go
func TestItemDTORoundTrip(t *testing.T) {
	item := config.Item{
		Name:          "srv1",
		Display:       "prod",
		Actions:       []string{"deploy", "logs"},
		ActionGroups:  []string{"remote"},
		CustomActions: []config.Action{{Title: "Rollback", Cmd: "echo rollback"}},
		Env:           map[string]any{"sshUser": "root", "port": 22},
	}
	got, err := FromItemDTO(ToItemDTO(item))
	if err != nil {
		t.Fatalf("FromItemDTO() error = %v", err)
	}
	if !reflect.DeepEqual(got, item) {
		t.Errorf("round trip = %+v, want %+v", got, item)
	}
}
```

If `FieldsToMap` normalises scalars (e.g. `22` coming back as `"22"`), assert
the normalised form rather than changing the conversion — the editor has always
round-tripped values through its field strings.

- [ ] **Step 9: Build and run the whole suite**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go build ./... && go vet ./... && go test ./..."
```

Expected: clean build, clean vet, all packages PASS. Fix whatever the compiler
found in test fixtures not listed above — they are all the same mechanical
change from an item map literal to a `config.Item` literal.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "Make items a typed struct with a separate env section"
```

---

### Task 4: The gate in `script-manager-gui`

**Files:**
- Create: `internal/gui/migrate.go`
- Modify: `internal/gui/app.go:26-68` (`NewApp`, `Startup`), `:100-110`
  (`ReloadConfig`)
- Modify: `internal/gui/browse.go:24-29`, `internal/gui/recent.go:16-23`
- Modify: `internal/gui/configwatch.go` (the reload the watcher triggers)
- Modify: `cmd/script-manager-gui/main.go:23-30`

**Interfaces:**
- Consumes: `configmigrate.Ensure`, `configmigrate.ErrDeclined`,
  `config.ResolvePath` (Task 2).
- Produces: `(*gui.App).ensureFormat(path string) error`.

- [ ] **Step 1: Simplify how the app knows its config path**

`NewApp` currently takes a `load func() (*config.Config, error)` closure that
`browse.go` and `recent.go` reassign. The gate needs the *path*, so make the
path the thing the app holds — matching `configedit.NewApp(cfgPath string)`:

In `cmd/script-manager-gui/main.go`, replace the closure with the flag value:

```go
cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
flag.Parse()

app := gui.NewApp(*cfgPath)
```

In `internal/gui/app.go`, replace the `load` field with `configPath string`, and
move the loading out of `NewApp` into `Startup` — `NewApp` runs before there is
a `ctx` to show a dialog on:

```go
func NewApp(cfgPath string) *App {
	go cleanupTempScripts()
	return &App{
		configPath: cfgPath,
		exeDir:     exepath.Dir(),
		appDataDir: appdata.Dir(),
		inlineRuns: make(map[inlineKey]*inlineRun),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

func (a *App) loadFrom(path string) (*config.Config, error) {
	if path != "" {
		return config.LoadFromWithError(path)
	}
	return config.LoadWithError()
}
```

- [ ] **Step 2: Write `internal/gui/migrate.go`**

```go
package gui

import (
	"fmt"

	"script-manager/internal/configmigrate"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ensureFormat converts a pre-env config after asking, and reports
// configmigrate.ErrDeclined when the user says no.
func (a *App) ensureFormat(path string) error {
	_, err := configmigrate.Ensure(path, a.approveConversion)
	return err
}

func (a *App) approveConversion(path, backupPath string) bool {
	choice, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:  wailsruntime.QuestionDialog,
		Title: "Convert config?",
		Message: fmt.Sprintf(
			"%s stores item environment variables in the old format and has to be converted before it can be opened.\n\n"+
				"The original is saved as:\n%s\n\nComments in the file will be lost.",
			path, backupPath),
		Buttons:       []string{"Convert", "Cancel"},
		DefaultButton: "Convert",
		CancelButton:  "Cancel",
	})
	return err == nil && choice == "Convert"
}
```

- [ ] **Step 3: Gate the startup load**

In `Startup`, resolve the path, run the gate, quit on decline, then load:

```go
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	applog.Init(a.appDataDir)

	path := a.configPath
	if path == "" {
		resolved, err := config.ResolvePath()
		if err == nil {
			path = resolved
		}
	}
	if path != "" {
		if err := a.ensureFormat(path); err != nil {
			applog.Printf("startup: %v", err)
			wailsruntime.Quit(ctx)
			return
		}
	}

	cfg, err := a.loadFrom(a.configPath)
	a.cfg, a.loadErr = cfg, err
	if cfg != nil && cfg.SourcePath != "" {
		recent.Add(a.appDataDir, cfg.SourcePath)
	}

	applog.Printf("startup version=%s config=%s", version.Version, a.configSourcePath())
	a.watchTheme()
	a.watchConfig()
}
```

`Quit` on a decline covers both the declined case and a conversion error —
either way the config cannot be opened.

- [ ] **Step 4: Gate the runtime open paths**

`browse.go` and `recent.go` both do `config.LoadFromWithError(path)` then
reassign the loader. Both become:

```go
if err := a.ensureFormat(path); err != nil {
	return err   // ErrDeclined leaves the current config loaded
}
cfg, err := config.LoadFromWithError(path)
if err != nil {
	return err
}
a.configPath = path
a.setConfig(cfg)
```

Do **not** call `Quit` here — declining a second config means staying on the
first.

- [ ] **Step 5: Gate reload, once per change**

`ReloadConfig` (F5) and the config watcher both reload the current file. Gate
both through `ensureFormat`, and give the watcher a latch so a file being edited
in another window cannot prompt on every save:

Add the latch to `App` in `app.go`:

```go
	declinedMu   sync.Mutex
	declinedAt   time.Time // mtime of the config the user declined to convert
```

and in `migrate.go`, a wrapper the watcher uses instead of `ensureFormat`:

```go
// ensureFormatOnChange is ensureFormat for reloads the user didn't ask for.
// A file the user already declined must not prompt again until it changes.
func (a *App) ensureFormatOnChange(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	a.declinedMu.Lock()
	declined := a.declinedAt.Equal(info.ModTime())
	a.declinedMu.Unlock()
	if declined {
		return configmigrate.ErrDeclined
	}

	err = a.ensureFormat(path)
	if errors.Is(err, configmigrate.ErrDeclined) {
		a.declinedMu.Lock()
		a.declinedAt = info.ModTime()
		a.declinedMu.Unlock()
	}
	return err
}
```

Imports for `migrate.go`: add `errors`, `os`. The watcher calls
`ensureFormatOnChange` and skips the reload on any error, keeping the loaded
config. `ReloadConfig` (F5) is an explicit user action, so it calls plain
`ensureFormat` and prompts every time.

- [ ] **Step 6: Build and test**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go build ./... && go test ./internal/gui/..."
```

Expected: clean build; `internal/gui` tests PASS. `app_test.go:52-80`
(`TestReloadConfig`) constructs an `App` directly — update it for the
`configPath` field.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "Prompt before converting an old-format config in script-manager-gui"
```

---

### Task 5: The gate in `sm-config-edit`

**Files:**
- Create: `internal/configedit/migrate.go`
- Modify: `internal/configedit/app.go:34-60` (`Startup`), `:93` (open)
- Modify: `internal/configedit/recent.go:17`

**Interfaces:**
- Consumes: `configmigrate.Ensure`, `config.ResolvePath`.
- Produces: `(*configedit.App).ensureFormat(path string) error`.

- [ ] **Step 1: Write `internal/configedit/migrate.go`**

`configedit` imports the Wails runtime as plain `runtime` (`app.go:16`), so
follow that here. The wording is identical to the other app's on purpose:

```go
package configedit

import (
	"fmt"

	"script-manager/internal/configmigrate"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ensureFormat converts a pre-env config after asking, and reports
// configmigrate.ErrDeclined when the user says no.
func (a *App) ensureFormat(path string) error {
	_, err := configmigrate.Ensure(path, a.approveConversion)
	return err
}

func (a *App) approveConversion(path, backupPath string) bool {
	choice, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:  runtime.QuestionDialog,
		Title: "Convert config?",
		Message: fmt.Sprintf(
			"%s stores item environment variables in the old format and has to be converted before it can be opened.\n\n"+
				"The original is saved as:\n%s\n\nComments in the file will be lost.",
			path, backupPath),
		Buttons:       []string{"Convert", "Cancel"},
		DefaultButton: "Convert",
		CancelButton:  "Cancel",
	})
	return err == nil && choice == "Convert"
}
```

`sm-config-edit` has no config watcher of its own — it autosaves rather than
reloading — so it needs no equivalent of the GUI's decline latch.

- [ ] **Step 2: Gate `Startup`**

`app.go:50-56` picks between `LoadFromWithError(a.cfgPath)` and
`LoadWithError()`. Resolve the path first (`config.ResolvePath()` when
`a.cfgPath` is empty), call `a.ensureFormat(path)`, and on error log it and
`runtime.Quit(a.ctx)` before loading.

- [ ] **Step 3: Gate the open paths**

`app.go:93` and `recent.go:17` both `LoadFromWithError(path)`. Call
`a.ensureFormat(path)` first and return the error — declining leaves the current
config open, as in the GUI.

- [ ] **Step 4: Build and test**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go build ./... && go test ./internal/configedit/..."
```

Expected: clean build, tests PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "Prompt before converting an old-format config in sm-config-edit"
```

---

### Task 6: The gate in the TUI

**Files:**
- Modify: `cmd/script-manager/main.go`

**Interfaces:**
- Consumes: `configmigrate.Ensure`, `configmigrate.ErrDeclined`,
  `config.ResolvePath`.
- Produces: nothing other tasks use.

- [ ] **Step 1: Add the prompt before `tea.NewProgram`**

The alt screen has not been entered yet, so this needs no Bubble Tea model:

```go
func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	path := *cfgPath
	if path == "" {
		resolved, err := config.ResolvePath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		path = resolved
	}
	if err := ensureConfigFormat(path); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	loadConfig := func() (*config.Config, error) {
		if *cfgPath != "" {
			return config.LoadFromWithError(*cfgPath)
		}
		return config.LoadWithError()
	}
	cfg, err := loadConfig()

	p := tea.NewProgram(ui.NewApp(cfg, loadConfig, err), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func ensureConfigFormat(path string) error {
	backup, err := configmigrate.Ensure(path, func(p, backupPath string) bool {
		fmt.Printf("%s stores item environment variables in the old format and\n", p)
		fmt.Println("has to be converted before it can be opened.")
		fmt.Printf("The original will be saved as %s.\n", backupPath)
		fmt.Println("Comments in the file will be lost.")
		fmt.Print("Convert now? [y/N] ")
		line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil {
			return false
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		return answer == "y" || answer == "yes"
	})
	if backup != "" {
		fmt.Printf("Converted. Backup: %s\n", backup)
	}
	return err
}
```

Imports to add: `bufio`, `strings`, `script-manager/internal/configmigrate`.

- [ ] **Step 2: Build**

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && go build ./... && go vet ./..."
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add cmd/script-manager/main.go
git commit -m "Prompt before converting an old-format config in the TUI"
```

---

### Task 7: Docs

Docs-only, so no build approval is needed for this task.

**Files:**
- Modify: `README.md:136-150` (the `items:` example) and any other item example
- Modify: `CHANGELOG.md` (`## Unreleased`)
- Check only: `internal/config/templates/default.yaml`, `default-win.yaml`

- [ ] **Step 1: Confirm the starter templates are already valid**

Both templates ship a single `- name: Example item` with no env keys, so they
need no change — but a starter config that triggered the conversion prompt on
first launch would be an embarrassing bug, so verify rather than assume:

```powershell
docker exec $c bash -c "cd /workspaces/script-manager && grep -A6 '^items:' internal/config/templates/default.yaml internal/config/templates/default-win.yaml"
```

Expected: only `name:` under each item. If either template has gained env keys,
move them under `env:`.

- [ ] **Step 2: Update the README example**

Move the non-structural keys under `env:`:

```yaml
items:
  - name: Production
    # show only the "safe" group + the ssh action by ID
    actionGroups: [safe]
    actions: [ssh]
    # inline actions available only for this item
    customActions:
      - title: Emergency rollback
        cmd: echo "Rolling back {{.clusterName}}"
    env:
      description: Production cluster
      clusterName: prod-cluster-eu
      clusterIp: 10.0.0.1
  - name: Dev
    env:
      description: Dev cluster
      clusterIp: 10.0.0.2
```

Then search the rest of the README (`grep -n "items:" README.md` and the config
reference section) for any other item sample and any prose saying that unknown
item keys become environment variables. Document the six keys, that `env:` holds
the variables, that templates still reference them unprefixed (`{{.clusterName}}`),
and that an old config is converted once on first launch with a backup.

Per `CLAUDE.md`, the README is a user guide: state the format as it is now. No
"previously this was flat", no rationale.

- [ ] **Step 3: Add the changelog entry**

Under `## Unreleased` in `CHANGELOG.md` — create the heading if the last release
consumed it:

```markdown
## Unreleased

### Changes

- An item's environment variables now live in their own `env:` section,
  alongside `name`, `display`, `actions`, `actionGroups` and `customActions`.
  A config in the old format is converted once, after asking, keeping a
  timestamped backup — comments in the file are lost in the conversion.
- Scripts no longer receive `DISPLAY`, `ACTIONS`, `ACTIONGROUPS` and
  `CUSTOMACTIONS` as environment variables, and `{{.display}}` no longer
  resolves in a details template.
```

One line each, per the release-notes rule. No account of how it was built.

- [ ] **Step 4: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "Document the item env section and the one-time conversion"
```

---

### Task 8: Full build and hand-off

- [ ] **Step 1: Ask before building**

`CLAUDE.md` requires asking and waiting before any build. Ask the user for
approval to run the full pass, then:

```powershell
.\build-container.ps1 -Full
```

Expected: `go vet` clean, every package PASS, both frontends
`svelte-check found 0 errors, 0 warnings, and 0 hints`, six binaries in `bin/`.

- [ ] **Step 2: Hand over for verification**

Do not run the apps. Report what changed and name what the user has to check,
which is everything that cannot be settled statically:

- An old config on first launch of each of the three binaries: the prompt
  appears, converting writes the timestamped backup, declining quits the GUIs
  and exits the TUI.
- A config already in the new format starts with no prompt.
- Opening an old config from the recent menu or Browse prompts, and declining
  leaves the current config loaded rather than quitting.
- Editing the open config in another window does not re-prompt on every save.
- Item env values still reach scripts and templates; `sm-config-edit` still
  shows them in the item's field list and saves them back under `env:`.
