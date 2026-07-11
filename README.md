# script-manager

A tool for organising and running shell scripts across a list of configurable items, driven by a single YAML config. It ships two separate interfaces that both read the same `config.yaml`:

- **TUI** (`cmd/script-manager`) — a terminal UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse items and actions, and run actions directly in the same terminal session.
- **GUI** (`cmd/script-manager-gui`) — a desktop app built with [Wails](https://wails.io). Browse items and actions in a resizable, mouse-driven window; actions run in a separate terminal window instead of inline — a dedicated, reused Windows Terminal window on Windows, or the first available terminal emulator on Linux (see [GUI](#gui) for details and current platform support).
- **Config Editor** (`cmd/sm-config-edit`) — a second Wails desktop app for creating a new `config.yaml` or editing an existing one through forms, instead of hand-writing YAML (see [Config Editor](#config-editor)).

# Disclaimer

This app is 100% vibe-coded by Claude. No human has read most of this code, and Claude doesn't have hands to knock on wood, so there's no guarantee or claim that any of it works perfectly. It exists purely because a developer wanted a nicer way to run his scripts but was too lazy to write one himself, so he made an AI do it instead — truly the pinnacle of engineering effort. Use at your own risk, read the code before trusting it with anything important, and if it deletes your production database, that's between you and the vibes.

*PS: Claude also wrote this disclaimer, so take it with the appropriate grain of salt.*

## Features

- **Config-driven** — define items, display templates, and actions in a YAML file; no code changes needed
- **Free-form items** — each item is a key/value map; any field can be used in templates or passed as an environment variable to actions
- **Template expansion** — action commands are Go templates, so you can interpolate item fields directly into commands
- **Cross-platform** — ships a Linux binary and a Windows binary; on Windows, `config-win.yaml` is preferred when present, falling back to `config.yaml`
- **Scrollable panes** — all four panes are independently scrollable when focused
- **Command preview** — the expanded command for the selected action is shown in a dedicated pane with clipboard copy support
- **State preserved** — returns to the same position after an action completes

## Two-step flow

The UI works in two modes:

**1 — Item selection** (start here)
Navigate the items list and press `Enter` to select an item.

**2 — Action selection**
The items list shrinks to show only the selected item. Navigate and run actions, browse details, or copy the command. Press `Esc` to go back to item selection.

## Layout

**Item selection mode**
```
┌─────────────────┬──────────────────────────────────┐
│ Items           │ Details                          │
│  > Nightly CDM  │  Description: Nightly build CDM  │
│    Staging      │  Cluster Name: test-cluster1     │
│    Production…  │  Cluster IP:  10.20.30.40        │
│    ...          │                                  │
└─────────────────┴──────────────────────────────────┘
  ↑↓/kj Navigate items   Enter Select   Q/Esc Quit
```

**Action selection mode**
```
┌─────────────────┬──────────────────────────────────┐
│  > Nightly CDM  │ Details                          │
├─────────────────┤  Description: Nightly build CDM  │
│ Actions         │  Cluster Name: test-cluster1     │
│  > Test output  │                                  │
│    Test input   ├──────────────────────────────────┤
│    Start k9s    │ Command                          │
│    ...          │  $ cat /etc/hosts                │
└─────────────────┴──────────────────────────────────┘
  ↑↓/kj Navigate   Tab/←→ Focus   [ ] Group   Enter Run   y Copy   Esc Back
```

## Keybindings

### Item selection mode

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` / `Tab` | Select item, enter action mode |
| `F5` | Reload `config.yaml` from disk |
| `Q` / `Esc` / `Ctrl+C` | Quit |

### Action selection mode

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up / scroll up |
| `↓` / `j` | Move down / scroll down |
| `Tab` / `→` | Next pane: Actions → Details → Command |
| `Shift+Tab` / `←` | Previous pane |
| `[` / `]` | Cycle through groups (Actions pane focused) |
| `Enter` | Run the selected action |
| `y` | Copy the expanded command to clipboard |
| `Enter` | Run action (Actions pane) / enter copy-value mode (Details pane) |
| `F5` | Reload `config.yaml` from disk |
| `Esc` | Back to item selection |
| `Q` / `Ctrl+C` | Quit |

`F5` works in either mode. It re-reads the config file, refreshes items/actions/details/titles in place, and preserves your current selection where still valid. If the file fails to read or has a YAML syntax error, the previous config is kept and the status bar shows the error instead of wiping your session.

## Usage

```bash
# Auto-detect config.yaml next to the binary or in the working directory
./bin/script-manager

# Explicit config file
./bin/script-manager -config /path/to/config.yaml
```

## Configuration

Place `config.yaml` in the same directory as the binary (or pass it with `-config`). On Windows, `config-win.yaml` takes precedence when present — next to the binary or in the working directory — and `config.yaml` is used as the fallback. If the preferred file exists but fails to parse (e.g. a YAML syntax error), it's skipped in favor of the next candidate and the load error is shown once at startup — a status bar message in the TUI, a toast in the GUI — so a broken `config-win.yaml` falling back to `config.yaml` doesn't go unnoticed.

Rather than hand-writing this file, you can use the [Config Editor](#config-editor) (`cmd/sm-config-edit`) to create or edit it through forms.

```yaml
shell:
  - bash
  - -c

display:
  - name: default                 # name used by items via display: default
    list: "{{.description}}"      # template for each row in the Items pane
    details: |                    # rendered as Markdown in the Details pane
      **{{.description}}**

      | Field   | Value |
      |---------|-------|
      | Cluster | `{{.clusterName}}` |
      | IP      | `{{.clusterIp}}` |
  - name: compact                 # alternative display — items opt in with display: compact
    list: "{{.name}} ({{.clusterIp}})"
    details: |
      ## {{.name}}
      **IP:** `{{.clusterIp}}`

titles:                           # optional — override pane header labels
  items: Servers
  actions: Tasks
  details: Info
  command: Preview

env:                              # optional — global variables available to all actions
  region: eu-west-1               # can be used in templates: {{.region}}
  sshUser: admin                  # overridden per item if the item defines the same key

items:
  - name: Production
    description: Production cluster
    clusterName: prod-cluster-eu
    clusterIp: 10.0.0.1
    # show only the "safe" group + the ssh action by ID
    actionGroups: [safe]
    actions: [ssh]
    # inline actions available only for this item
    customActions:
      - title: Emergency rollback
        cmd: echo "Rolling back {{.clusterName}}"
  - name: Dev
    description: Dev cluster
    clusterIp: 10.0.0.2
    display: compact              # optional — picks a named display config; omit to use first

actionGroups:                       # optional — a catalog giving groups a friendlier title/color;
  - id: connect                     # the groups: keys above/below still just reference plain
    title: Connectivity             # IDs and work the same with or without a matching entry here
    color: "#7fd4ff"
  - id: safe
    title: Safe to run anytime
    color: "#4caf50"
  - id: diagnostics                 # title/color are both optional — id alone is a valid entry

actions:
  - id: ssh              # optional — used for per-item action filtering
    title: SSH into cluster
    description: Open an interactive SSH session to the cluster node.   # single-line
    groups: [connect]    # optional — one or more groups for per-item filtering
    cmd: ssh admin@{{.clusterIp}}
    interactive: true    # needs terminal input — hides "Run here" in the GUI (see below)
  - id: hosts
    title: Show hosts
    description: |                # multiline — rendered above the command in the Command pane
      Prints the system hosts file.
      Useful for verifying DNS overrides on the node.
    groups: [safe, diagnostics]   # action belongs to multiple groups
    cmd: cat /etc/hosts
  - id: dashboard
    title: Open dashboard
    cmd: xdg-open http://{{.clusterIp}}:8080
    noWait: true         # return to UI immediately, don't wait for a keypress
```

### Action filtering

By default every item sees all global actions. To restrict which actions appear for a specific item, add any combination of these keys:

| Item key | Type | Effect |
|---|---|---|
| `actions` | list of IDs | include global actions whose `id` matches |
| `actionGroups` | list of group names | include global actions whose `group` matches |
| `customActions` | list of action objects | append item-specific actions (same fields as global actions) |

If none of these keys are set the full action list is shown (backward-compatible). When `actions` and `actionGroups` are both set, matches from each are included in that order without duplicates. `customActions` are always appended last.

#### Naming and coloring groups

The top-level `actionGroups:` list is an optional catalog: each entry is `id` (the same string an action's `groups:` or an item's `actionGroups:` already references — required), plus an optional `title` and `color` for tools that want a friendlier label or a color swatch instead of showing the bare ID. Nothing about actually resolving a group (`config.ActionsForItem`) reads this list — it exists purely for editors/UIs to attach a display name and color to a group name, and an existing config with no `actionGroups:` catalog at all keeps working exactly as before.

### Templates

`display.list`, `display.details`, and `actions[*].cmd` are [Go templates](https://pkg.go.dev/text/template). Item fields are available as `{{.fieldName}}`.

`display.details` is rendered as **Markdown** in the Details pane — you can use `**bold**`, `*italic*`, `` `code spans` ``, `## headings`, tables, and bullet lists. Backtick-wrapped values (`` `value` ``) are highlighted in cyan and can be copied to the clipboard.

If the details template references a field an item doesn't have (or the field is null), the field renders as a literal `<nil>` placeholder instead of failing, and a ⚠️ warning at the top of the Details pane lists the missing field names. In the TUI this is a warning line above the rendered details; in the GUI it's a collapsible bar pinned above the details content (like the Actions group filter) that stays visible while the details scroll.

#### Masking sensitive values

Use the built-in `mask` template function to hide passwords or tokens in the Details pane while still making them copyable:

```yaml
details: |
  | Field    | Value                  |
  |----------|------------------------|
  | Password | `{{mask .password}}`   |
  | Token    | `{{mask .apiToken}}`   |
```

The Details pane shows `••••••` instead of the real value. When you enter copy mode and select that row, pressing `Enter` copies the actual secret to the clipboard — it is never displayed.

#### Multi-line values

A backtick span (`` `{{.field}}` ``) is a one-line construct — a literal newline inside it would break whatever line it's sitting on. If the value itself spans multiple lines (e.g. a certificate), it's automatically treated the same as a masked value: the Details pane shows a placeholder like `` `(6-line value)` `` instead of the real content, and pressing `Enter` in copy mode (or clicking it in the GUI) copies the real, full value to the clipboard. In the GUI, hovering over it also shows the real value in a tooltip — a value masked via `{{mask ...}}` never does this, since it's an actual secret rather than merely a long one.

**Always wrap a field in backticks** (`` `{{.field}}` ``) if it might hold a multi-line value — this handling only applies to backtick-wrapped spans. A bare `{{.field}}` reference outside of backticks gets none of it, and a multi-line value there will break the surrounding Markdown exactly as described above.

#### Printing every variable for an item

Rather than listing each field by hand, `display.details` can include either of these literal placeholders (not `{{ }}` template syntax — just the bare text) to have every variable the item exports to its actions' environment rendered automatically:

```yaml
details: |
  ### {{.description}}

  #ALL_ENV_LIST#

  #ALL_ENV_TABLE#
```

- `#ALL_ENV_LIST#` renders a Markdown bullet list: `` - **CLUSTERIP:** `10.0.0.1` ``
- `#ALL_ENV_TABLE#` renders a two-column Markdown table (`Variable` / `Value`)

Both list every merged variable under its exported (uppercase) name, sorted alphabetically — the `display`, `actions`, `actionGroups`, and `customActions` keys are skipped since they configure action filtering rather than holding data worth displaying. Any variable whose name ends in `password`, `passwd`, `pwd`, `secret`, `key`, `token`, `credential`, `credentials`, or `auth` (case-insensitive) is masked automatically, exactly like an explicit `{{mask ...}}` call — no need to name each secret field yourself. A multi-line value (e.g. a certificate) gets the same placeholder-plus-copy-plus-tooltip treatment described in [Multi-line values](#multi-line-values) above.

#### Showing which config file is loaded

The literal placeholder `#CONFIG_FILE#` expands to the full path of the config file actually in use — handy for confirming which of several candidates (`config-win.yaml` vs. `config.yaml`, next to the binary vs. in the working directory) won:

```yaml
details: |
  ### {{.description}}

  _Config: `#CONFIG_FILE#`_
```

### Environment variables

When an action runs, global `env` values and all item fields are exported as uppercase environment variables. Item fields override globals with the same name.

```bash
# For an item with clusterIp: 10.0.0.1 and global env region: eu-west-1
echo $CLUSTERIP   # → 10.0.0.1
echo $REGION      # → eu-west-1
```

### Windows config

Use `config-win.yaml` with a PowerShell shell and Windows-appropriate commands:

```yaml
shell:
  - pwsh
  - -NoProfile
  - -Command

actions:
  - title: Show hosts
    cmd: Get-Content C:\Windows\System32\drivers\etc\hosts
```

## GUI

`cmd/script-manager-gui/` is a desktop GUI built with [Wails](https://wails.io) (Go backend + Svelte frontend) that reads the **same `config.yaml`** as the TUI:

- Items pane → Actions pane → Details pane, exactly as configured in `display.list` / `display.details`
- Actions can be filtered by group with a row of chips below the Actions list header ("All" + one per group found on the item's actions), same grouping the TUI cycles through with `[` / `]`. Unlike the TUI, multiple groups can be selected at once — an action must belong to *all* selected groups to show (AND, not OR); clicking "All" clears the selection back to everything. Each chip shows how many actions would match if it were added to the current filter, e.g. `diagnostics(5)`. A group whose count would drop to 0 disappears from the row entirely rather than sitting there as a dead end — except a group you already have selected, which always stays visible so you have a way to deselect it. Chips are sorted by exactly one key at a time, chosen with the two buttons above them: `A-Z` (by name, the default) or `#` (by that count; equal counts fall back to A-Z). The active button is highlighted; clicking it flips its direction (`A-Z`/`Z-A`, `# ↓`/`# ↑`), clicking the other switches the sort key. The chip row itself is collapsible (▾/▸) — collapsed, it shows the selected groups as text (e.g. "Groups: safe, diagnostics"); this collapsed state persists across restarts like the other panes. A group with a `color` set in the top-level `actionGroups:` catalog (see [Naming and coloring groups](#naming-and-coloring-groups)) shows that color as its chip background instead of the default neutral one, with the text color auto-picked (light or dark) for contrast — selecting a chip still shows the same accent-warm highlight as any other selected chip, regardless of its own color
- Markdown details rendering (tables, `<br>`, bold/italic, etc.) with masked (`{{mask ...}}`) values click-to-copy without ever displaying the secret
- Command preview (expanded template) for the selected action, with a copy button. The action's groups are shown as chips between the description and the command, colored the same way as the Actions group filter chips
- All four panes are collapsible (▾/▸ in each header) and resizable (drag the thin dividers between panes); sizes and collapsed state persist across restarts. Collapsed, the Items and Actions headers show the current selection (e.g. "Actions · Test output"), wrapping onto multiple lines for longer labels, so you don't lose context
- `F5` reloads the config from disk in place — same semantics as the TUI: previous state is kept on a read/parse failure, with the error shown as a toast instead

### Running actions (Windows and Linux)

The **Run** button in the Command pane opens the expanded command in a terminal window. By default it auto-detects the most common terminal for the current OS, tried in this order until one is found on `PATH`:

- **Windows** — `wt` (Windows Terminal; reuses the same dedicated `script-manager`-named window across every run instead of spawning a new one each time) → `wezterm` → `alacritty` → `cmd` (opens a plain console via `cmd`'s `start` — always present on any Windows install, so Run never has nothing to fall back to)
- **Linux** — `x-terminal-emulator` (the Debian-alternatives default, so your configured terminal wins) → `gnome-terminal` → `konsole` → `xfce4-terminal` → `terminator` → `kitty` → `alacritty` → `wezterm` → `foot` → `xterm`

#### Choosing a specific terminal

Auto-detection can be overridden with an optional `terminal:` key in `config.yaml`/`config-win.yaml`:

```yaml
# Pick one specific terminal by name from the built-in list above,
# skipping auto-detection entirely. Errors clearly if it isn't installed.
terminal: alacritty
```

```yaml
# Or give a fully custom launch command for a terminal that isn't built
# in. The first element is the binary; the rest are its flags. "{{title}}"
# and "{{dir}}" are substituted; the resolved shell command (script path
# and any wrapper flags) is always appended as the final arguments.
terminal: [wezterm-gui, start, --title, "{{title}}", --cwd, "{{dir}}", --]
```

Only Windows and Linux are supported; macOS gets a clear "not supported" error rather than a silent no-op.

On both platforms:

- The action's `noWait` flag controls whether the terminal stays open after the command finishes: `false` (default) keeps it open so you can read the output (PowerShell via `-NoExit`; POSIX shells via a "Press Enter to close" prompt appended to the script); `true` closes it automatically, same intent as the TUI's `noWait`
- The starting directory is the GUI executable's folder (not the user's home directory), so relative paths in `cmd:` templates resolve the same way as `config.yaml` auto-detection does

The expanded command is written to a temporary script file (`.ps1` for PowerShell/pwsh, `.bat` for cmd.exe, `.sh` for bash/sh/zsh/dash/ksh) and run as a script argument, rather than inlined on the command line — terminal launchers' reconstruction of the argv doesn't reliably survive multi-line scripts with embedded quotes, so only a plain file path is passed through instead. The script deletes itself once the shell actually starts executing it, rather than on an external timer: PowerShell and POSIX shells self-delete as their first line (both parse/open the whole file before running anything, so this is also the earliest point a secret-bearing script can come off disk); cmd.exe self-deletes as its last line, since deleting a batch file as its first line is a well-known source of quirky behavior in `cmd.exe`. Any leftover from a run that never got this far — e.g. the terminal or shell failed to start at all — is removed unconditionally the next time the GUI starts, regardless of how old it is.

> **Note on secrets:** the temp script contains the *fully expanded* command. If a `cmd:` template interpolates a value you hide with `{{mask ...}}` in the Details pane, that value sits in plain text in the OS temp directory for the brief window before cleanup runs. Avoid putting secrets in `cmd:` templates on shared machines.

The **Run** button's terminal window is independent once launched — no output streams back into the GUI, same trade-off as the TUI's own action execution. macOS is not supported yet and gets a clear "not supported" error instead of a silent no-op.

#### Running a command without a terminal ("Run here")

For a command that isn't expected to need interactive input, the **Run here** button next to **Run** executes it directly and streams the captured output — stdout and stderr interleaved in real execution order — live into the Command pane as it's produced, instead of opening a terminal window. A **Cancel** button appears while it's running, and forcibly terminates the whole process tree (not just the top-level shell); once the command finishes, the exit code stays shown alongside the full output, and **Copy output** copies whatever's been captured so far to the clipboard — even while the command is still running. Like **Run**, the working directory is the GUI executable's folder, and stdin is left disconnected — a command that unexpectedly prompts for input fails fast with an immediate EOF rather than hanging with no terminal for anyone to type into.

An action's `interactive: true` (see the config example above) hides **Run here** entirely for that action — it can still only be run via **Run**, which opens a real terminal — since a script that needs to read from stdin would otherwise just fail fast against the disconnected input **Run here** always uses.

Different actions can run inline at the same time — starting another action while one is still going doesn't stop it, and a small pulsing dot marks which items and actions currently have a run in progress, so you can tell at a glance without leaving the one you're looking at. Switching back to an action that's still running (or has since finished) picks its output back up right where it left off — only starting the same action again while it's already running is rejected.

#### Toolbar

Three buttons above the panes: **Load config** browses for a different YAML file and switches to it (including redirecting **Refresh config**/F5 to reload *that* file from then on); **Refresh config** re-reads the current file, same as pressing F5; **Open config editor** (also **Ctrl+E**) launches `sm-config-edit` — the [Config Editor](#config-editor) below — pointed at whichever config file is currently loaded.

Launch it the same way as the TUI:

```bash
./bin/script-manager-gui
```

It auto-detects `config.yaml` next to the binary or in the working directory, and accepts an explicit path with `-config` — same rules as the TUI:

```bash
./bin/script-manager-gui -config /path/to/config.yaml
```

### GUI build requirements

Building the GUI requires the [Wails CLI](https://wails.io/docs/gettingstarted/installation), Node.js, and (for the Linux target) GTK/WebKit headers, in addition to Go:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Linux target — GTK/WebKit headers and Node.js
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config nodejs npm
```

This devcontainer already has all of this preinstalled — see `.devcontainer/setup.sh`. Nothing needs to be installed on your host machine to build any target, including Windows, from here.

The Windows GUI binary **can be cross-compiled from Linux** — Wails' Windows target only needs a C cross-compiler (`mingw-w64`) at build time, not a running Windows OS. WebView2 itself (needed only at *runtime*) is preinstalled on Windows 10/11.

```bash
# one-time: sudo apt install gcc-mingw-w64-x86-64

cd cmd/script-manager-gui
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
  wails build -platform windows/amd64
```

macOS is the one target that genuinely needs to be built on macOS (Apple's toolchain/frameworks aren't cross-compilable from Linux).

## Config Editor

`cmd/sm-config-edit/` is a second Wails desktop app (same Go backend + Svelte frontend shape as `script-manager-gui`, and the same dark theme — both share `frontend-shared/`) for creating or editing `config.yaml` through forms instead of hand-writing YAML.

- **New / Open / Save / Save As**, same auto-detect rule as the TUI/GUI: on launch it resolves `config-win.yaml`/`config.yaml` (exe dir then working directory) the same way `-config`-less TUI/GUI startup does, with a native file picker to open a different file or choose where to save a new one. Finding nothing on launch isn't treated as an error — it just starts blank, since a first-time user of this tool plausibly has no config yet.
- **Sections**: Items, Action Groups, Actions, Displays, Environment, Shell, Titles, Terminal — one form per top-level `config.yaml` concern. Action Groups edits the `id`/`title`/`color` catalog entries (with a color picker and a swatch next to each entry in the list); an item's reserved keys (`name`, `display`, `actions`, `actionGroups`, `customActions`) get dedicated widgets (text field, a dropdown of configured displays, checkbox lists against the global actions/groups, a repeatable nested action form); anything else is a per-item "Environment" grid (overriding a same-named value from the top-level Environment section), where each value's kind drives which widget edits it: string, multiline (a plain textarea, auto-picked for any existing value with embedded newlines), number, bool, or a raw YAML snippet (auto-picked for anything more complex, like a nested list or map). A lock button next to each field's remove (✕) button marks it secret — independent of kind, so even a multi-line value can be masked — auto-picked when the key ends in "Secret", "Password", or "Key" (case-insensitive, checked live as you type a new field's key too). A secret field shows a fixed placeholder at rest and only reveals its real value (still masked character-for-character while you edit) once focused, so its length isn't given away by how many dots are shown. Kind and secret are both display hints only — the value saves as the same plain string/number/bool either way, and neither is persisted as such; both are re-derived from the value's shape and the key's name every time the field loads.
- **Reordering**: Items, Action Groups, and Actions can each be drag-and-drop reordered — order matters (it's what ends up in `config.yaml`, and for Actions it's the order they show in the TUI/GUI). Dragging is off by default; a third toolbar button (a grip icon, next to Add/Remove) toggles "reorder mode", so a stray click while just selecting a row can't silently reorder the list. The rest of the list animates out of the way live as you drag, showing where the item would land before you release. Turning reorder mode on clears the current selection and blocks selecting a row until it's off again — turning it back off doesn't try to restore what was selected before.
- **Live preview**: with an item selected, its rendered list label and details (against the chosen display) update as you type, along with a preview of any action's expanded command/description against that item — the same template-preview logic (`action.Preview`, missing-field filling) the GUI's Details pane uses, without needing to save first. The Displays section has the same live preview, but since a display isn't tied to one item, pick any item from the "Preview item" dropdown to see how *that* display would render it. Unlike the other sections, Displays has no master-list sidebar — a "Display" combobox picks which one you're editing, and its name plus **Copy display** (duplicates the current one as "*name* - copy") and **Remove display** buttons live in their own small panel above the editor. Four view modes (Edit / Preview / Split ↔ / Split ↕) control how much space editing vs. previewing gets, with a draggable divider between them in the two split modes; the chosen mode and divider position persist across restarts. Above the Details template, an **Insert env…** dropdown lists every env var available to it (the global Environment fields plus the currently-picked preview item's own fields) and inserts `{{.key}}` at the cursor on selection — already wrapped as a masked `` `{{mask .key}}` `` if the key looks like it holds a secret (same "ends in secret/password/key" heuristic the Environment grid uses to auto-lock a new field); **B** / *I* / `` ` `` buttons wrap the current text selection in `**bold**`, `_italic_`, or `` `backtick-highlighted` `` markdown — the last of which also makes that span a copy/mask target, the same as any other backtick-enclosed value in a rendered Details pane. A fourth (padlock) button turns a selected `{{.key}}` reference into a masked `` `{{mask .key}}` `` one directly, for a variable that wasn't auto-masked on insert (e.g. one typed by hand) — it's a no-op with a flashed hint if the selection isn't a bare `{{.key}}`-shaped reference. Every insert/wrap button leaves the affected text selected afterward — the whole inserted token for Insert env and Mask, just the original text (not the added markup) for Bold/Italic/Highlight — so it's clear what changed and easy to keep editing or re-format. Bold/Italic/Highlight toggle off instead of stacking another layer of markers if the current selection is already wrapped (clicking Bold on `**hello**` gives back `hello`, not `****hello****`). All four buttons go through the same undo-compatible edit path ordinary typing does, so Ctrl+Z/Ctrl+Y (or your OS's redo shortcut) undo and redo a button click exactly like a keystroke.
- **Validation**: duplicate global action IDs block Save; duplicate item names and an item referencing a display/action/group that doesn't exist are shown as non-blocking warnings.
- **Messages**: every piece of UI text in *either* GUI app — toasts, tooltips, labels, empty states — lives in a per-app `messages.json`, compiled into that app's frontend as the shipped defaults but also written out at runtime next to its executable (`script-manager-gui.messages.json` / `sm-config-edit.messages.json`), self-seeded from those defaults the first time that app runs. This section edits either file directly: pick a target (**script-manager-gui** or **sm-config-edit (this app)**), edit any value, and **Save** writes the whole file back. If `script-manager-gui` has never been run yet, its file doesn't exist and editing its messages here shows a message asking you to run it once first. **Changes only take effect the next time the edited app is launched** — there's no live-reload while it's already running, the same as a config edit needing Refresh/F5 in the other app.

**Important:** saving always re-serializes the whole file through `yaml.Marshal` — comments and the original file's exact formatting/key order are **not preserved**. Editing a hand-crafted `config.yaml` with inline comments through this tool will lose those comments on save.

Launch the same way as the other two:

```bash
./bin/sm-config-edit
./bin/sm-config-edit -config /path/to/config.yaml
```

It has the same [GUI build requirements](#gui-build-requirements) as `script-manager-gui` (Wails CLI, Node.js, GTK/WebKit headers for the Linux target) and is built by the same `bash build.sh` / `build-container.ps1` workflow described below.

## Building

Requires Go 1.21+.

```bash
bash build.sh
```

Builds both platforms by default. Produces:
- `bin/script-manager` — Linux amd64
- `bin/script-manager.exe` — Windows amd64
- `bin/script-manager-gui` — Linux amd64 GUI (only if the `wails` CLI is installed; skipped otherwise)
- `bin/script-manager-gui.exe` — Windows amd64 GUI, cross-compiled (only if `mingw-w64` is installed; skipped otherwise)
- `bin/sm-config-edit` — Linux amd64 Config Editor (only if the `wails` CLI is installed; skipped otherwise)
- `bin/sm-config-edit.exe` — Windows amd64 Config Editor, cross-compiled (only if `mingw-w64` is installed; skipped otherwise)

Pass `--windows` or `--linux` to build only that platform — `--windows` for routine use on a Windows host (which never runs the Linux binaries), `--linux` when only a Linux binary is needed (e.g. Xvfb-based visual testing of the GUI apps):

```bash
bash build.sh --windows
bash build.sh --linux
```

### Building from a Windows host via a dev container

If you develop on Windows with the Go toolchain only available inside a VS Code dev container (no Go on the host), `build-container.ps1` wraps the steps that otherwise have to be repeated by hand:

```powershell
.\build-container.ps1
```

It stops any running `script-manager*.exe`/`sm-config-edit*.exe` on the host first — a locked binary makes the Windows cross-compile step in `build.sh` fail with "permission denied" — then finds the dev container for this repo (matched by its `devcontainer.local_folder` label, since the container name is auto-generated and changes across recreations) and runs `bash build.sh` inside it. Same default-both / `--windows` / `--linux` split as `build.sh`, via `-Windows`/`-Linux`: `.\build-container.ps1 -Windows`.

To build for a specific target manually:

```bash
GOOS=linux   GOARCH=amd64 go build -o bin/script-manager     ./cmd/script-manager/
GOOS=windows GOARCH=amd64 go build -o bin/script-manager.exe ./cmd/script-manager/

# GUI, Linux (either app — substitute cmd/sm-config-edit for the other one)
(cd cmd/script-manager-gui && wails build)

# GUI, Windows (cross-compiled, see above)
(cd cmd/script-manager-gui && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ wails build -platform windows/amd64)
```

## Dependencies

- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [mko88/bubbletea-tilelayout](https://github.com/mko88/bubbletea-tilelayout) — tile layout manager
- [atotto/clipboard](https://github.com/atotto/clipboard) — clipboard support
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) — config parsing
- [wailsapp/wails](https://wails.io) — GUI shell (Go backend + native webview)
- [yuin/goldmark](https://github.com/yuin/goldmark) — Markdown → HTML rendering for the GUI Details pane
