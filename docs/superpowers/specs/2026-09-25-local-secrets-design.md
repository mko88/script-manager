# Local secrets — design

## Goal

A config that uses PIN-locked values must be safe to share (e.g. commit to
git). Today the `secrets:` block (salt, KDF params, check) and every
`sm-enc:v1:` ciphertext sit inline in `config.yaml`, so sharing the file
shares the ciphertext and forces everyone onto one PIN.

After this change the shared config only *declares* which variables are
secret. Each user keeps their own values, under their own PIN, in a local
file outside the repo.

Decisions agreed:

- **Per-user values.** Teammates each have their own values for the same
  declared variable.
- **Local file lives in app-data**, never beside the config.
- **Matched by a stable ID** stored in the shared config, not by path.
- **Declared with a YAML tag**, `!secret`.

## 1. Format

### Shared config

```yaml
secretsId: 3f9c2a7e5b1d4c8f9a0e6b2d7c4f1a93

env:
  region: eu-central-1
  vault_token: !secret

items:
  - name: demo-database
    env:
      db_user: appuser
      db_password: !secret
```

- `secretsId`: optional top-level key, 32 lowercase hex chars (128 random
  bits). Present only once a PIN has been set on the config.
- `!secret`: allowed as the value of any key in the global `env:` or an
  item's `env:`. Its node value is ignored (write it bare). Anywhere
  else it is a load error.
- The shared config no longer writes a `secrets:` block.

### Local file

Path: `<appdata.Dir()>/secrets/<secretsId>.yaml`, written with mode `0600`
(directory `0700`).

```yaml
secrets:
  salt: …
  time: 3
  memory: 65536
  threads: 4
  check: sm-enc:v1:…
env:
  vault_token: sm-enc:v1:…
items:
  demo-database:
    db_password: sm-enc:v1:…
```

- Item entries are keyed by item `name`. Item names are already validated
  unique by the Config Editor (`configedit/validate.go`).
- Values are always `sm-enc:v1:` ciphertext under the local `secrets:`
  params.
- Entries for keys or items that the shared config doesn't declare are
  kept on save (so switching branches doesn't destroy another branch's
  values) but ignored at load.

## 2. Loading (`internal/config`)

1. Parse the shared config. Each `!secret` becomes the in-memory sentinel
   `secret.Unset` (a distinct exported string constant, e.g.
   `"sm-secret:unset"`, that `IsLocked` does not match).
2. If `secretsId` is set, read the local file (missing → no error, all
   slots stay `Unset`; malformed → load error naming the local path).
3. Fill each `Unset` slot that has a local value with that ciphertext; set
   `cfg.Secrets` from the local `secrets:` block.
4. Record on the `Config` (non-YAML fields) the `secretsId`, the local file
   path, and the set of declared-secret keys (global and per item), so the
   saver knows which keys to write as `!secret` even when unfilled.

After this, the in-memory `Config` has the same shape the runtime already
handles: locked values are `sm-enc:v1:` strings and `cfg.Secrets` holds the
params. The TUI and GUI's `Reveal`/`Redact`/`HasLocked`/PIN prompt code is
unchanged except for handling `Unset`.

A config is **legacy** when it has an inline `secrets:` block or inline
`sm-enc:v1:` values. It loads exactly as it does today (those values are
treated as local).

## 3. Runtime handling of `Unset` (`internal/secret`, `ui`, `gui`)

- `Redact`: `Unset` displays as `(not set)` everywhere a locked value
  displays `(locked)` today (Details pane, list label, editor preview).
- `Reveal` / `Strip`: an `Unset` variable is removed from the environment,
  so the script sees it as not set, the same as a locked value under
  `requiresPin: false` today.
- `HasLocked` ignores `Unset`, so an action whose only secrets are unset
  doesn't prompt for a PIN it can't use.

## 4. Saving (`internal/config`, `internal/configedit`)

- `Config.Marshal` returns the shared YAML. A new `Config.MarshalLocal`
  (or a combined `MarshalSplit`) returns the local YAML. Every key in the
  declared-secret set, and every value that `IsLocked`, is written as
  `!secret` in the shared file. Its ciphertext (if any) goes into the
  local file.
- The editor writes the local file **first**, then the shared file, so the
  GUI's config watcher (which watches the shared file) never reloads a
  shared file whose local half isn't on disk yet.
- The editor's DTOs keep representing a locked field as `Locked: true` with
  the ciphertext. An unfilled declared secret is `Locked: true` with an
  empty value plus a new `Unset: true` flag.
- **Item rename** in the editor carries that item's local entries to the
  new name. It falls out naturally: the local file is rebuilt from the
  DTO on save, keyed by current names.

### Config Editor, PIN section

- **Set PIN** on a config without `secretsId` generates one.
- Shows the local secrets file path.
- **"Give this config its own secrets"** issues a new `secretsId`. The
  current values are re-written under the new ID on the next save; the old
  file is left in place. For a config copied from another one.
- **Change PIN** is unchanged (re-encrypts every locked value).
- **Remove PIN** asks for confirmation (the app's own dialog, not a browser
  confirm). It then clears every locked value to `Unset` and clears the
  params. On save the local file is deleted and the shared file keeps its
  `!secret` declarations. It no longer writes plaintext into the config.

### Fields

- An `Unset` field shows a "not set on this machine" padlock state.
  Entering a value encrypts it under the session PIN (the PIN section opens
  first if no PIN exists yet, as clicking a padlock does today).
- Locking a plain field turns it into a declared secret. Unlocking it (after
  entering the PIN) turns it back into a plain value in the shared file,
  as today.

## 5. Migration

When the editor saves a legacy config it moves the values out automatically.
It generates a `secretsId` and writes the local file with the existing
params and ciphertext (no re-encryption, so the same PIN keeps working).
The shared file is rewritten with `!secret` and without `secrets:`. The
TUI and GUI never write, so the migration happens on the first editor save.

## 6. Errors

- Local file unreadable or malformed: load error naming the local path.
  The apps already surface config load errors.
- Local file write fails: save fails before the shared file is touched.
- `!secret` outside an `env:` mapping: load error naming the location.
- `secretsId` not 32 hex chars: load error (it becomes a file name).

## 7. Testing

`internal/config` unit tests, with `appdata` pointed at a temp dir via an
overridable var:

- round trip: split save → load gives the same in-memory values and params
- unfilled `!secret` loads as `Unset`; local file missing entirely
- local entries for undeclared keys are kept on save, ignored on load
- item rename moves entries
- legacy inline config loads; saving it produces shared + local split with
  unchanged ciphertext
- `!secret` in an invalid position, and a bad `secretsId`, are rejected

`internal/secret`: `Redact`/`Reveal`/`Strip`/`HasLocked` with `Unset`.

`internal/configedit`: Remove PIN clears to `Unset`; the new-ID action.

`examples/pin-secrets*.yaml` convert to the new format. `examples_test.go`
loads them against a fixture local file written into the temp app-data dir.

## 8. Docs

- README "PIN-locked values": rewrite around sharing. The shared file is
  safe to commit; values live per user in app-data (give the path per OS);
  each user sets their own PIN and fills in their own values; the copied-
  config button; the changed Remove PIN; the automatic move on first save.
- CHANGELOG `## Unreleased`: one Changes line for the feature, and one
  for the automatic move of existing locked values (the user has to know
  that the values now live in app-data).
