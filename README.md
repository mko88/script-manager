# script-manager

A terminal UI for organising and running shell scripts across a list of configurable items. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Config-driven** — define items, display templates, and actions in a YAML file; no code changes needed
- **Free-form items** — each item is a key/value map; any field can be used in templates or passed as an environment variable to actions
- **Template expansion** — action commands are Go templates, so you can interpolate item fields directly into commands
- **Cross-platform** — ships a Linux binary and a Windows binary; automatically loads `config-win.yaml` on Windows if present
- **Scrollable panes** — items list, detail view, and actions panel are all independently scrollable
- **State preserved** — returns to the same position after an action completes

## Layout

```
┌─────────────────┬──────────────────────────────────┐
│ Items           │ Details                          │
│  ▶ Nightly CDM │  Description: Nightly build CDM  │
│    Staging      │  Cluster Name: test-cluster1     │
│    Production…  │  ...                             │
│                 │                                  │
├─────────────────│  Command:                        │
│ Actions         │    $ cat /etc/hosts              │
│  ▶ 1  Test out │                                  │
│    2  Test inp  │                                  │
│    3  Start k9s │                                  │
└─────────────────┴──────────────────────────────────┘
  ↑↓/kj Navigate   Tab Switch focus   Enter/1-9 Run   Q Quit
```

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up / scroll up |
| `↓` / `j` | Move down / scroll down |
| `Tab` | Cycle focus: Items → Actions → Details |
| `Enter` | Run the selected action |
| `1`–`9` | Run action by number (works from any pane) |
| `Q` / `Esc` / `Ctrl+C` | Quit |

## Configuration

Place `config.yaml` in the same directory as the binary (or in the working directory as a fallback). On Windows, `config-win.yaml` is loaded automatically if it exists.

```yaml
shell:
  - bash
  - -c

display:
  list: "{{.description}}"        # template for each row in the Items pane
  details: |                      # template for the Details pane
    Description: {{.description}}
    Cluster:     {{.clusterName}}

items:
  - name: Production
    description: Production cluster
    clusterName: prod-cluster-eu
    clusterIp: 10.0.0.1
    # any additional fields you like

actions:
  - title: Show hosts
    cmd: cat /etc/hosts
  - title: SSH into cluster
    cmd: ssh admin@{{.clusterIp}}
  - title: Multi-line script
    cmd: |
      echo "Connecting to {{.clusterName}}"
      kubectl get pods --context {{.clusterName}}
```

### Templates

Both `display.list`, `display.details`, and `actions[*].cmd` are [Go templates](https://pkg.go.dev/text/template). Item fields are available as `{{.fieldName}}`.

### Environment variables

When an action runs, all item fields are exported as uppercase environment variables so scripts can reference them directly:

```bash
# For an item with clusterIp: 10.0.0.1
echo $CLUSTERIP   # → 10.0.0.1
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

## Building

Requires Go 1.21+.

```bash
bash build.sh
```

Produces:
- `script-manager` — Linux amd64
- `script-manager.exe` — Windows amd64

To build for a specific target manually:

```bash
GOOS=linux  GOARCH=amd64 go build -o script-manager .
GOOS=windows GOARCH=amd64 go build -o script-manager.exe .
```

## Dependencies

- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [mko88/bubbletea-tilelayout](https://github.com/mko88/bubbletea-tilelayout) — tile layout manager
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) — config parsing
