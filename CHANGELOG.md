# Changelog

Notable changes across all three binaries, newest first. Versions follow
the `Major.Minor.Patch.Build` scheme described in `CLAUDE.md`.

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
