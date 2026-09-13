# Changelog

Notable changes across all three binaries, newest first. Versions follow
the `Major.Minor.Patch` scheme described in `CLAUDE.md`. Releases before
1.4.0 carry a fourth segment, from the scheme this replaced.

## 1.4.2

### Changes

- **Breaking: an item's environment variables move into an `env:` section.**
  An item now has exactly six keys — `name`, `display`, `actions`,
  `actionGroups`, `customActions` and `env` — and any other key is an error.

  ```yaml
  # before                  # after
  items:                    items:
    - name: srv1              - name: srv1
      actions: [ssh]            actions: [ssh]
      sshUser: root             env:
      dbHost: db1                 sshUser: root
                                  dbHost: db1
  ```

  A config in the old format is offered for conversion the first time any of
  the three apps opens it, keeping a timestamped backup. Comments in the file
  are lost in the conversion, and declining means the config isn't opened.
  Templates and environment variables are unchanged: `{{.sshUser}}` and
  `$SSHUSER` still work, unprefixed.
- Scripts no longer receive `DISPLAY`, `ACTIONS`, `ACTIONGROUPS` and
  `CUSTOMACTIONS` as environment variables, and `{{.display}}` no longer
  resolves in a details template. A variable may now be named after any of
  those keys.
- Questions both GUI apps ask — converting a config, and every confirmation in
  `sm-config-edit` — are shown in the app's own dialog rather than the
  operating system's.

### Bug fixes

- `script-manager-gui`: opening a different config kept the output of runs
  from the previous one. Opening a config now starts a fresh session, as if
  the app had just been launched; reloading the same config still keeps it.

## 1.4.1

### Changes

- A button beside a script action's path opens the file in the default
  editor, in both GUI apps.
- A script shown on screen is watched, so an edit made outside the app
  refreshes the preview without reselecting the action.
- `script-manager-gui`: the Command pane puts COMMAND above OUTPUT, gives each
  open section a share of the pane's height with its own scrollbar, and
  collapses COMMAND in favour of OUTPUT when an inline run starts.
- `script-manager-gui`: OUTPUT is shown for any action that can run inline.
  Which of the two sections is open now follows the selected action's own run
  state instead of carrying over from the action before it: OUTPUT opens once
  that action has something to show, and COMMAND stays open otherwise —
  including for an action that can't run inline or a run that printed nothing.
- The recent-configs list is styled like the item lists elsewhere in both
  apps, and marks the config in use — which `script-manager-gui` didn't.
- `script-manager-gui`: the Details and Command panes stay as you left them.
  Selecting an item or an action no longer collapses one and expands the other.

### Bug fixes

- `script-manager-gui`: the Copy output button was hidden behind the output
  pane.
- `script-manager-gui`: copying a script action copied the path to the file
  rather than the contents shown beside the button.
- `script-manager-gui`: entries in the recent-configs list had no hover
  highlight and needed a second click to open.
- On Linux, scaling the interface up left empty space below the app.

## 1.4.0

### Changes

- Items, action groups and actions can be copied in `sm-config-edit`; the copy
  lands below the original, named "Server - Copy".
- Environment values can be locked behind a per-config PIN: `config.yaml` keeps
  only the ciphertext, and a locked value never appears on screen.
- Actions opt in with **Requires PIN** to receive those values decrypted;
  others run without a prompt and simply don't have them set.
- `sm-config-edit` has a **PIN** section to set, change or remove the PIN.
- `sm-config-edit` drops its native title bar, like `script-manager-gui`: the
  toolbar is the drag handle and carries the window controls.
- Both GUI apps list the last 10 configs opened, shared between them.
- `sm-config-edit` autosaves, and `script-manager-gui` reloads when the config
  changes on disk.
- Every pane showing code is a CodeMirror editor: highlighting for shell,
  PowerShell, YAML and markdown, folding, and undo.
- The details template completes variables, `#ALL_ENV_LIST#` and
  `#ALL_ENV_TABLE#`, and tints `{{ }}` references.
- The interface scales between 70% and 200% with **Ctrl +** / **-** / **0**.
- The interface and monospace fonts can be set to any font on the machine.
- The version comes from the git tag at build time, shown in the About panel.
- Bold text is drawn rather than synthesized: Nunito ships as a variable font.
- Inputs show a focus ring, and status animations respect reduced motion.

## 1.3.0.0

- `script-manager-gui`: the "Shrink on focus loss" badge is now a tiny
  icon-only nub that pops out to a bigger, fully-opaque badge on hover
  (shrinking back after a moment once the pointer leaves), and fades to
  80% opacity instead of 20% while shrunk.

## 1.2.0.0

- `script-manager-gui`: added a "Shrink on focus loss" checkbox in the
  transparency popover — while the window is pinned always-on-top, losing
  focus fades it to a small badge (icon + item count) parked at the
  vertical center of the screen's right edge; clicking the badge restores
  the window's previous size, position, and opacity.

## 1.1.0.0

- `script-manager-gui`: added an About panel (toolbar "i" button) showing
  the app version, a short description, and a link to the project's
  GitHub page.

## 1.0.0.0

- `script-manager-gui`: added always-on-top and window-opacity toggles to
  the toolbar, both remembered across restarts.
- `script-manager-gui`: removed the native title bar — the toolbar is now
  the window's drag handle, with its own minimize/maximize/close buttons.
- `script-manager-gui`: every pane's whole header (not just the chevron)
  now toggles collapse; selecting an item expands Details and collapses
  Command, and selecting an action does the reverse.
