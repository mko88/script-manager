# Item schema: a separate `env` section

## Problem

An item is `map[string]any`. Five keys are structural — `name`, `display`,
`actions`, `actionGroups`, `customActions` — and every other key is an
environment variable. The two live in the same namespace, with three
consequences:

- `action.Env` exports the structural keys to every script, so a subprocess
  receives `DISPLAY=prod`, `ACTIONS=[ssh logs]`, `CUSTOMACTIONS=...`.
- `internal/render` maintains its own blocklist of structural keys so they
  stay out of `#ALL_ENV_LIST#` / `#ALL_ENV_TABLE#`, duplicating a list that
  `internal/configedit` also keeps.
- An environment variable cannot be named `display`, `actions`,
  `actionGroups` or `customActions`. There is no way to escape the collision.

## Format

An item has exactly six keys:

```yaml
items:
  - name: srv1
    display: prod
    actions: [ssh, logs]
    actionGroups: [remote]
    customActions:
      - title: Tail app log
        cmd: tail -f /var/log/app.log
    env:
      sshUser: root
      dbHost: db1.internal
```

Any other key is a load error naming the item and the key. The old shape is
not accepted at runtime; reaching it is the migration's job, not the
loader's.

## Runtime data is unchanged

Templates and scripts see the same flat map they see today. `action.Merge`
builds it as:

1. the global `env` map,
2. the item's `env` map (overriding 1),
3. `name` (overriding both).

So `{{ .sshUser }}`, `{{ .name }}`, `$SSHUSER` and `$NAME` keep working, and
`secret.Redact` / `Strip` / `Reveal`, the mask pipeline and
`render.ExpandAllEnv` are untouched below `Merge`.

Two deliberate behaviour changes fall out:

- `display`, `actions`, `actionGroups` and `customActions` are no longer in
  the merged map. `{{ .display }}` in a details template stops resolving, and
  `DISPLAY`/`ACTIONS`/`ACTIONGROUPS`/`CUSTOMACTIONS` are no longer exported
  to scripts. This is the fix, not a regression, but it is user-visible and
  belongs in the changelog.
- `render.reservedKeys` is deleted. `allEnvEntries` iterates the item's `env`
  map, which contains nothing structural by construction.

## Types

```go
type Item struct {
    Name          string         `yaml:"name,omitempty"`
    Display       string         `yaml:"display,omitempty"`
    Actions       []string       `yaml:"actions,omitempty"`
    ActionGroups  []string       `yaml:"actionGroups,omitempty"`
    CustomActions []Action       `yaml:"customActions,omitempty"`
    Env           map[string]any `yaml:"env,omitempty"`
}
```

`Config.Items` becomes `[]Item`. A custom `UnmarshalYAML` rejects unknown
keys — struct tags alone do not, since yaml.v3 ignores them silently.

Consequences across the tree:

- `config.FindDisplay(displays, item *Item)` reads `item.Display`.
- `config.ActionsForItem(actions []Action, item *Item)` reads the fields.
  `nil` keeps meaning "no item selected" (returns all actions), matching the
  `itemAt` helpers in `internal/gui` and `internal/ui` that return nil today.
- `config.AsStringSlice` and `config.ParseCustomActions` exist only to dig
  typed values out of `map[string]any`. Delete them once nothing calls them.
- Typed `CustomActions []Action` accepts `requiresPin`, which
  `ParseCustomActions` silently dropped. A widening, not a break.
- `configedit.ToItemDTO` / `FromItemDTO` become field copies, and
  `reservedItemKeys` is deleted. `ItemDTO` already has this exact shape —
  five named fields plus `Fields` — so the DTO layer does not change.

## Migration: `internal/configmigrate`

A package with no dependency on either app.

```go
func Needed(path string) (bool, error)
func Convert(path string) (backupPath string, err error)
```

`Needed` parses the file as loosely-typed YAML and reports whether any item
carries a key outside the six. A file with no items, or whose items are all
already in the new shape, needs nothing. A mixed file needs conversion.

`Convert` backs the file up, then rewrites it: unmarshal loosely, move each
item's unknown keys into `env`, `yaml.Marshal`, write.

**Comments in the file are lost.** `yaml.Marshal` does not preserve them.
This matches what `sm-config-edit` already does on every save, and the README
already warns about it, but the conversion prompt must say so too — this is
the one time it happens to a file the user may never have opened in the
editor.

Backup path is the config path plus `.<YYYYMMDD-HHMM>.bak`, e.g.
`config-win.yaml.20260913-2245.bak`. Timestamped so it never overwrites a
`.bak` taken by hand.

## The gate

`Needed` is checked on **every** load, not only at startup: both GUIs open
other configs from the recent-configs menu and the browse dialog, and those
files can be old too.

**Both GUI apps.** `NewApp` no longer loads in its constructor; it records
the path to load. `Startup(ctx)` checks `Needed`, and for an old config shows
a Wails `MessageDialog` (`QuestionDialog`) naming the file, the backup path,
and the comment loss. Yes converts and loads; No calls `runtime.Quit`.

The same check wraps the runtime open paths (recent menu, browse) and the
reload paths (`ReloadConfig`, and the on-disk config watcher). There,
declining leaves the currently loaded config in place instead of quitting —
the user chose to open or reload a file and backed out of it. The watcher is
the one path that can fire without the user asking for anything, so it must
not prompt repeatedly: one prompt per detected change, and a decline means
the watcher stops offering until the file changes again.

**TUI.** The check goes in `cmd/script-manager/main.go` before
`tea.NewProgram`: print the notice, read `y/N` from stdin, convert or exit.
The alt screen has not been entered yet, so this needs no Bubble Tea model.

## Shipped content to update

- `internal/config/templates/default.yaml` and `default-win.yaml` — neither
  currently has item env keys, so they only need the format's blessing, not
  a rewrite.
- `README.md` — the `items:` example at ~line 136 uses the old flat shape
  (`description`, `clusterName`, `clusterIp` as top-level keys) and must move
  them under `env:`. Any other item example likewise.
- `CHANGELOG.md` — a `## Unreleased` entry covering the new format, the
  one-time conversion prompt, and the dropped `DISPLAY`/`ACTIONS` exports.

## Testing

Round-trip is where the risk is.

- Old shape → `Convert` → new shape → load → merged map equal to what the old
  loader produced for the same file, minus the four structural keys the
  change deliberately drops.
- `Needed` on: an old config, a new config, a mixed config, a config with no
  items, an empty file, a malformed file (error, not `false`).
- `Convert` takes a backup, and the backup is the original bytes.
- Unknown top-level item key is a load error naming item and key.
- An item whose `env` contains a key named `name` or `display` survives the
  round trip and does not collide with the structural field.
- `action.Merge` precedence: global env < item env < name.
- `action.Env` no longer emits `DISPLAY`/`ACTIONS`/`ACTIONGROUPS`/
  `CUSTOMACTIONS`.
- `configedit` round trip: item → DTO → item is stable.
