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

actions:
  - id: ssh              # optional — used for per-item action filtering
    title: SSH into cluster
    description: Open an interactive SSH session to the cluster node.   # single-line
    groups: [connect]    # optional — one or more groups for per-item filtering
    cmd: ssh admin@{{.clusterIp}}
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

Both list every merged variable under its exported (uppercase) name, sorted alphabetically — the `display`, `actions`, `actionGroups`, and `customActions` keys are skipped since they configure action filtering rather than holding data worth displaying. Any variable whose name ends in `password`, `passwd`, `pwd`, `secret`, `key`, `token`, `credential`, `credentials`, or `auth` (case-insensitive) is masked automatically, exactly like an explicit `{{mask ...}}` call — no need to name each secret field yourself.

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
- Actions can be filtered by group with a row of chips below the Actions list header ("All" + one per group found on the item's actions), same grouping the TUI cycles through with `[` / `]`. Unlike the TUI, multiple groups can be selected at once — an action must belong to *all* selected groups to show (AND, not OR); clicking "All" clears the selection back to everything. Each chip shows how many actions would match if it were added to the current filter, e.g. `diagnostics(5)`. A group whose count would drop to 0 disappears from the row entirely rather than sitting there as a dead end — except a group you already have selected, which always stays visible so you have a way to deselect it. Chips are sorted by exactly one key at a time, chosen with the two buttons above them: `A-Z` (by name, the default) or `#` (by that count; equal counts fall back to A-Z). The active button is highlighted; clicking it flips its direction (`A-Z`/`Z-A`, `# ↓`/`# ↑`), clicking the other switches the sort key. The chip row itself is collapsible (▾/▸) — collapsed, it shows the selected groups as text (e.g. "Groups: safe, diagnostics"); this collapsed state persists across restarts like the other panes
- Markdown details rendering (tables, `<br>`, bold/italic, etc.) with masked (`{{mask ...}}`) values click-to-copy without ever displaying the secret
- Command preview (expanded template) for the selected action, with a copy button. The action's groups are shown as chips between the description and the command
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

There's no output streamed back into the GUI — the terminal window is independent once launched, same trade-off as the TUI's own action execution. macOS is not supported yet and gets a clear "not supported" error instead of a silent no-op.

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
- **Sections**: Items, Actions, Displays, Environment, Shell, Titles, Terminal — one form per top-level `config.yaml` concern. An item's reserved keys (`name`, `display`, `actions`, `actionGroups`, `customActions`) get dedicated widgets (text field, a dropdown of configured displays, checkbox lists against the global actions/groups, a repeatable nested action form); anything else is a generic "Additional Fields" grid, where each value's kind (string/number/bool, or a raw YAML snippet for anything more complex, like a nested list or map) drives which widget edits it.
- **Live preview**: with an item selected, its rendered list label and details (against the chosen display) update as you type, along with a preview of any action's expanded command/description against that item — the same template-preview logic (`action.Preview`, missing-field filling) the GUI's Details pane uses, without needing to save first.
- **Validation**: duplicate global action IDs block Save; duplicate item names and an item referencing a display/action/group that doesn't exist are shown as non-blocking warnings.

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

Produces:
- `bin/script-manager` — Linux amd64
- `bin/script-manager.exe` — Windows amd64
- `bin/script-manager-gui` — Linux amd64 GUI (only if the `wails` CLI is installed; skipped otherwise)
- `bin/script-manager-gui.exe` — Windows amd64 GUI, cross-compiled (only if `mingw-w64` is installed; skipped otherwise)
- `bin/sm-config-edit` — Linux amd64 Config Editor (only if the `wails` CLI is installed; skipped otherwise)
- `bin/sm-config-edit.exe` — Windows amd64 Config Editor, cross-compiled (only if `mingw-w64` is installed; skipped otherwise)

### Building from a Windows host via a dev container

If you develop on Windows with the Go toolchain only available inside a VS Code dev container (no Go on the host), `build-container.ps1` wraps the steps that otherwise have to be repeated by hand:

```powershell
.\build-container.ps1
```

It stops any running `script-manager*.exe`/`sm-config-edit*.exe` on the host first — a locked binary makes the Windows cross-compile step in `build.sh` fail with "permission denied" — then finds the dev container for this repo (matched by its `devcontainer.local_folder` label, since the container name is auto-generated and changes across recreations) and runs `bash build.sh` inside it.

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
