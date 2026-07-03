# script-manager

A terminal UI for organising and running shell scripts across a list of configurable items. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). A companion desktop GUI (browse-only for now) is also available — see [GUI](#gui).

## Features

- **Config-driven** — define items, display templates, and actions in a YAML file; no code changes needed
- **Free-form items** — each item is a key/value map; any field can be used in templates or passed as an environment variable to actions
- **Template expansion** — action commands are Go templates, so you can interpolate item fields directly into commands
- **Cross-platform** — ships a Linux binary and a Windows binary; automatically loads `config-win.yaml` on Windows if present
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
| `Esc` | Back to item selection |
| `Q` / `Ctrl+C` | Quit |

## Usage

```bash
# Auto-detect config.yaml next to the binary or in the working directory
./bin/script-manager

# Explicit config file
./bin/script-manager -config /path/to/config.yaml
```

## Configuration

Place `config.yaml` in the same directory as the binary (or pass it with `-config`). On Windows, `config-win.yaml` is loaded automatically when present.

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

`cmd/script-manager-gui/` is a desktop GUI built with [Wails](https://wails.io) (Go backend + Svelte frontend) that reads the **same `config.yaml`** as the TUI. It currently covers browsing only:

- Items pane → Actions pane → Details pane, exactly as configured in `display.list` / `display.details`
- Markdown details rendering (tables, `<br>`, bold/italic, etc.) with masked (`{{mask ...}}`) values click-to-copy without ever displaying the secret
- Command preview (expanded template) for the selected action, with a copy button

It does **not** run actions yet — there is no live output streaming or PTY support. Running actions from the GUI is a possible future addition; for now, use the TUI to execute scripts.

Launch it the same way as the TUI:

```bash
./bin/script-manager-gui
```

It auto-detects `config.yaml` next to the binary or in the working directory (same rule as the TUI; no `-config` flag yet).

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

To build for a specific target manually:

```bash
GOOS=linux   GOARCH=amd64 go build -o bin/script-manager     ./cmd/script-manager/
GOOS=windows GOARCH=amd64 go build -o bin/script-manager.exe ./cmd/script-manager/

# GUI, Linux
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
