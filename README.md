# script-manager

A tool for organising and running shell scripts across a list of configurable items, driven by a single YAML config. It ships two separate interfaces that both read the same `config.yaml`:

- **TUI** (`script-manager`) — a terminal UI. Browse items and actions, and run actions directly in the same terminal session.
- **GUI** (`script-manager-gui`) — a desktop app. Browse items and actions in a resizable, mouse-driven window; actions run in a separate terminal window (see [GUI](#gui)).
- **Config Editor** (`sm-config-edit`) — a second desktop app for creating a new `config.yaml` or editing an existing one through forms, instead of hand-writing YAML (see [Config Editor](#config-editor)).

# Disclaimer

This app is 100% vibe-coded by Claude. No human has read most of this code, and Claude doesn't have hands to knock on wood, so there's no guarantee or claim that any of it works perfectly. It exists purely because a developer wanted a nicer way to run his scripts but was too lazy to write one himself, so he made an AI do it instead — truly the pinnacle of engineering effort. Use at your own risk, read the code before trusting it with anything important, and if it deletes your production database, that's between you and the vibes.

*PS: Claude also wrote this disclaimer, so take it with the appropriate grain of salt.*

## Features

- **Config-driven** — define items, display templates, and actions in a YAML file; no code changes needed
- **Free-form items** — each item is a key/value map; any field can be used in templates or passed as an environment variable to actions
- **Template expansion** — action commands are Go templates, so you can interpolate item fields directly into commands
- **Cross-platform** — Windows and Linux binaries for all three apps
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
| `Enter` | Run action (Actions pane) / enter copy-value mode (Details pane) |
| `y` | Copy the expanded command to clipboard |
| `F5` | Reload `config.yaml` from disk |
| `Esc` | Back to item selection |
| `Q` / `Ctrl+C` | Quit |

`F5` works in either mode: it re-reads the config file in place and keeps your current selection where still valid. If the file fails to read or parse, your session is untouched and the error is shown in the status bar.

## Usage

```bash
# Auto-detect config.yaml next to the binary, in the working directory, or
# in the app-data directory
./bin/script-manager

# Explicit config file
./bin/script-manager -config /path/to/config.yaml
```

## Configuration

Place `config.yaml` next to the binary, or pass a path with `-config`. On Windows, `config-win.yaml` takes precedence over `config.yaml` when both exist. Without `-config`, the file is looked for next to the binary, then in the working directory, then in the app-data directory (`%AppData%\script-manager` on Windows, `~/.config/script-manager` on Linux). If a candidate file fails to parse, the next one is used and the error is shown once at startup.

On a first-ever run with no config anywhere, a minimal starter config (one example item and action) is created in the app-data directory. All three binaries (`script-manager`, `script-manager-gui`, `sm-config-edit`) resolve the config the same way.

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

actionGroups:                       # optional — gives groups a friendlier title/color
  - id: connect
    title: Connectivity
    color: "#7fd4ff"
  - id: safe
    title: Safe to run anytime
    color: "#4caf50"
  - id: diagnostics                 # title/color are both optional

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
  - id: deploy
    title: Deploy
    script: C:\scripts\deploy-{{.clusterName}}.ps1   # mutually exclusive with cmd — see below
```

`cmd` and `script` are mutually exclusive per action — set one or the other, not both. `cmd` is a command template, expanded and run through the configured `shell:`. `script` is a path to a script file or executable (the path itself supports `{{.field}}` templates) that is invoked directly — a `.ps1` just works on Windows, a POSIX script uses its own shebang, and native executables (`.exe`/`.bat`/`.cmd`, POSIX binaries) run as-is. Either way the action gets the same environment variables (the item's fields plus the global `env:` block, uppercased).

### Action filtering

By default every item sees all global actions. To restrict which actions appear for a specific item, add any combination of these keys:

| Item key | Type | Effect |
|---|---|---|
| `actions` | list of IDs | include global actions whose `id` matches |
| `actionGroups` | list of group names | include global actions whose `group` matches |
| `customActions` | list of action objects | append item-specific actions (same fields as global actions) |

If none of these keys are set the full action list is shown. When several keys are set, their matches combine without duplicates, with `customActions` last.

#### Naming and coloring groups

The top-level `actionGroups:` list is optional: give a group a `title` and/or a `color` and the GUI shows that friendlier label and a colored chip instead of the bare ID.

### Templates

`display.list`, `display.details`, and `actions[*].cmd` are [Go templates](https://pkg.go.dev/text/template). Item fields are available as `{{.fieldName}}`.

`display.details` is rendered as **Markdown** in the Details pane — you can use `**bold**`, `*italic*`, `` `code spans` ``, `## headings`, tables, and bullet lists. Backtick-wrapped values (`` `value` ``) are highlighted in cyan and can be copied to the clipboard.

If the details template references a field an item doesn't have, the field renders as a `<nil>` placeholder instead of failing, and a ⚠️ warning above the details lists the missing field names.

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

If a backtick-wrapped value (`` `{{.field}}` ``) spans multiple lines (e.g. a certificate), the Details pane shows a placeholder like `` `(6-line value)` `` instead of the content. Pressing `Enter` in copy mode (or clicking it in the GUI) copies the real, full value to the clipboard, and in the GUI hovering shows it in a tooltip (a `{{mask ...}}` value never does — it's an actual secret).

**Always wrap a field in backticks** (`` `{{.field}}` ``) if it might hold a multi-line value — a bare `{{.field}}` reference outside backticks gets none of this handling, and a multi-line value there will break the surrounding Markdown.

#### Printing every variable for an item

Rather than listing each field by hand, `display.details` can include either of these literal placeholders (not `{{ }}` template syntax — just the bare text) to render every variable the item exports to its actions' environment:

```yaml
details: |
  ### {{.description}}

  #ALL_ENV_LIST#

  #ALL_ENV_TABLE#
```

- `#ALL_ENV_LIST#` renders a Markdown bullet list: `` - **CLUSTERIP:** `10.0.0.1` ``
- `#ALL_ENV_TABLE#` renders a two-column Markdown table (`Variable` / `Value`)

Both list every merged variable under its exported (uppercase) name, sorted alphabetically (the `display`, `actions`, `actionGroups`, and `customActions` keys are skipped). Any variable whose name ends in `password`, `passwd`, `pwd`, `secret`, `key`, `token`, `credential`, `credentials`, or `auth` (case-insensitive) is masked automatically, exactly like an explicit `{{mask ...}}` call. Multi-line values get the same placeholder-plus-copy treatment described above.

#### Showing which config file is loaded

The literal placeholder `#CONFIG_FILE#` expands to the full path of the config file in use — handy for confirming which of several candidates won:

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

`script-manager-gui` is a desktop app that reads the **same `config.yaml`** as the TUI:

- Items pane → Actions pane → Details pane, exactly as configured in `display.list` / `display.details`
- Actions can be filtered by group with a row of chips below the Actions list header. Multiple groups can be selected at once — an action must belong to *all* selected groups to show; clicking "All" clears the filter
- Markdown details rendering, with masked (`{{mask ...}}`) values click-to-copy without ever displaying the secret
- Command preview (expanded template) for the selected action, with a copy button. An **OUTPUT** section appears once a "Run here" run has started (see below)
- All four panes are collapsible and resizable by dragging the dividers; sizes persist across restarts
- `F5` reloads the config from disk in place — same semantics as the TUI, with errors shown as a toast

### Running actions (Windows and Linux)

The **Run** button in the Command pane opens the expanded command in a terminal window. For a `script:` action, the Command pane shows the path plus the file's own source, and **Run**/**Run here** invoke that file directly — same env vars and buttons as a `cmd:` action. By default the most common terminal for the current OS is auto-detected, tried in this order until one is found on `PATH`:

- **Windows** — `wt` (Windows Terminal; reuses the same dedicated `script-manager`-named window across runs) → `wezterm` → `alacritty` → `cmd` (always present, so Run never has nothing to fall back to)
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

Only Windows and Linux are supported; macOS gets a clear "not supported" error.

On both platforms:

- The action's `noWait` flag controls whether the terminal stays open after the command finishes: `false` (default) keeps it open so you can read the output; `true` closes it automatically — same intent as the TUI's `noWait`
- The starting directory is the app-data directory (`%AppData%\script-manager` on Windows, `~/.config/script-manager` on Linux), so relative paths in a `cmd:` template — or files the script writes with a relative path — land in a reliably writable location

The expanded command runs via a temporary script file that is cleaned up automatically.

> **Note on secrets:** the temp script contains the *fully expanded* command. If a `cmd:` template interpolates a value you hide with `{{mask ...}}` in the Details pane, that value sits in plain text in the OS temp directory for the brief window before cleanup. Avoid putting secrets in `cmd:` templates on shared machines.

The **Run** terminal window is independent once launched — no output streams back into the GUI.

#### Running a command without a terminal ("Run here")

For a command that doesn't need interactive input, the **Run here** button next to **Run** executes it directly and streams the output live into the Command pane's **OUTPUT** section, with **Cancel** and **Copy output** buttons alongside. Like **Run**, the working directory is the app-data directory. Stdin is disconnected, so a command that unexpectedly prompts for input fails fast instead of hanging.

An action's `interactive: true` (see the config example above) hides **Run here** entirely for that action — it can only be run via **Run**, in a real terminal.

Different actions can run inline at the same time. A dot marks items and actions with a run in progress or finished — green for exit code 0, red otherwise — and switching back to an action picks its output back up.

#### Toolbar

Three controls above the panes: **Load config** browses for a different YAML file and switches to it; **Refresh config** re-reads the current file (same as F5); the gearbox icon (**Ctrl+E**) launches the [Config Editor](#config-editor) pointed at the currently loaded config file.

#### Theme

Both GUI apps default to the dark theme. The active theme is shared between the two apps, and themes are managed in the Config Editor's **Theme** section (see below); `script-manager-gui` simply reflects whatever's active.

Launch it the same way as the TUI:

```bash
./bin/script-manager-gui
./bin/script-manager-gui -config /path/to/config.yaml
```

### GUI build requirements

Building the GUI requires the [Wails CLI](https://wails.io/docs/gettingstarted/installation), Node.js, and (for the Linux target) GTK/WebKit headers, in addition to Go:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Linux target — GTK/WebKit headers and Node.js
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config nodejs npm
```

This devcontainer already has all of this preinstalled — nothing needs to be installed on your host machine to build any target, including Windows, from here.

The Windows GUI binaries are cross-compiled from Linux; only the `mingw-w64` C cross-compiler is needed at build time (`sudo apt install gcc-mingw-w64-x86-64`). macOS is the one target that has to be built on macOS.

## Config Editor

`sm-config-edit` is a second desktop app for creating or editing `config.yaml` through forms instead of hand-writing YAML.

- **New / Open / Save / Save As** (Ctrl+N/O/S/Shift+S). On launch it auto-detects the same config file the TUI/GUI would; finding nothing just starts blank. Save also covers the Theme and Messages sections while one of them is open. Toolbar buttons on the far right open the app-data directory in your file manager or the config file in your default editor.
- **Sections**: Items, Action Groups, Actions, Displays, Environment, Shell, Terminal, Theme, Messages — one form per top-level `config.yaml` concern (Theme and Messages live outside `config.yaml` — see below). Item fields get type-appropriate editors, with a lock button to mark a field secret (auto-enabled for key names like "Password"); secret values stay hidden until focused. Actions switch between **Command** and **Script file** mode; Script file mode has a **Browse…** button and a preview of the file's source.
- **Reordering**: Items, Action Groups, and Actions can be drag-and-drop reordered (toggle with the grip icon button) — the order is what ends up in `config.yaml` and what the TUI/GUI display.
- **Live preview**: with an item selected, its rendered list label, details, and any action's expanded command update as you type — no save needed. The Displays section previews templates against any item, with edit/preview/split view modes; an **Insert env…** dropdown inserts any available variable at the cursor, and formatting buttons wrap the selection in bold, italic, highlight, or `{{mask ...}}` markup.
- **Validation**: duplicate global action IDs block Save; duplicate item names and an item referencing a display/action/group that doesn't exist are shown as non-blocking warnings.
- **Themes** *(Theme section)*: pick a theme from the dropdown to apply it immediately, everywhere; **Add** / **Copy** / **Delete** / **Reset** manage custom themes (the built-in Dark/Light are read-only). Every color the apps use is editable, with a live preview panel — click any preview element to filter the field list to just the colors it uses. Saved themes are shared with `script-manager-gui` (see [Theme](#theme) above).
- **Messages**: every piece of UI text in *either* GUI app — toasts, tooltips, labels, empty states — can be customized. A tab per app picks which one you're editing, and a search box filters by key or text. A customized message gets a restore button to reset just that one; **Restore defaults** resets everything. Changes are written by the global Save and take effect the next time the edited app is launched; customizations survive upgrades.

**Important:** saving always re-serializes the whole file — comments and the original file's exact formatting/key order are **not preserved**. Editing a hand-crafted `config.yaml` with inline comments through this tool will lose those comments on save.

Launch the same way as the other two:

```bash
./bin/sm-config-edit
./bin/sm-config-edit -config /path/to/config.yaml
```

It has the same [GUI build requirements](#gui-build-requirements) as `script-manager-gui` and is built by the same `bash build.sh` / `build-container.ps1` workflow described below.

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

Pass `--windows` or `--linux` to build only that platform:

```bash
bash build.sh --windows
bash build.sh --linux
```

### Building from a Windows host via a dev container

If you develop on Windows with the Go toolchain only available inside a VS Code dev container (no Go on the host), `build-container.ps1` finds the dev container for this repo and runs `bash build.sh` inside it, stopping any running `script-manager*.exe`/`sm-config-edit*.exe` on the host first so the cross-compile can overwrite them:

```powershell
.\build-container.ps1
.\build-container.ps1 -Windows   # or -Linux — same split as build.sh
```

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
