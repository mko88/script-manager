# Changelog

Notable changes across all three binaries, newest first. Versions follow
the `Major.Minor.Patch.Build` scheme described in `CLAUDE.md`.

## 1.4.0.0

### Changes

- The interface scales with **Ctrl +** / **Ctrl -**, and **Ctrl 0** resets
  it — in both GUI apps, 70% to 200%, spacing included rather than the text
  alone. The setting is shared by both apps.
- The interface and monospace fonts can be set to any font installed on the
  machine, from `sm-config-edit`'s Theme section; empty keeps the bundled
  Nunito.
- Bold text is real bold: Nunito now ships as a variable font, so weights
  are drawn rather than synthesized by the browser.
- Type sizes come from a five-step scale, and every colour comes from a
  theme token — including two new ones, `warning-tint` and `scrim`, which
  the theme editor can now customise.
- Inputs, selects and textareas show a focus ring, and the running-status
  animations respect `prefers-reduced-motion`.

- `sm-config-edit` autosaves, on by default: once a config has a path, edits
  are written about a second after you stop typing. A toolbar button toggles
  it and the choice is remembered. Autosave holds off while a blocking
  validation error is showing.
- `script-manager-gui` watches the config file and reloads when it changes on
  disk, so edits made in the editor appear without pressing F5.

- Environment values can be locked behind a config PIN: `sm-config-edit`
  encrypts the value in place and `config.yaml` stores only the ciphertext.
  One PIN per config, entered once per app run and again after loading a
  different config, in both the GUI and the TUI.
- Actions opt in with **Requires PIN** (`requiresPin: true`). Such an action
  asks for the PIN and receives the decrypted values; every other action
  runs without a prompt and simply doesn't have the locked variables set.
- A locked value never renders on screen: details, list labels and the
  editor's preview show `(locked)` whether or not the PIN has been entered.
  Entering it only feeds the value to a `requiresPin` action.
- `sm-config-edit`: a **PIN** section sets the config's PIN, changes it
  (re-encrypting every locked value in one step), or removes it (decrypting
  them back to plain text, still marked secret). The padlock on a value
  points there when no PIN has been set yet.
- The Load config button (`script-manager-gui`) and Open button
  (`sm-config-edit`) now open a dropdown listing the last 10 configs
  opened, by full path, with Clear recent and Browse… below it. The list
  is shared by both apps, and an entry that no longer loads drops off it.
- `sm-config-edit`: a copy button on the Items, Action Groups and Actions
  toolbars duplicates the selected entry below the original and selects it.
- `sm-config-edit`: copies are named "Server - Copy", then "Server - Copy 2";
  actions and action groups also get a fresh id (`ssh` → `ssh-copy`).
- The version now comes from the git tag at build time, so the About panel
  identifies the exact build (`v1.4.0.0`, `v1.4.0.0-4-g94b2835`, `-dirty`,
  or `dev` outside a checkout).

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
