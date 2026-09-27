# script-manager

Organise and run shell scripts against a list of items (servers, clusters, environments — anything), all driven by one YAML config. Three apps read the same config:

- **`script-manager-gui`**: a desktop app for browsing items and running their actions.
- **`sm-config-edit`**: a desktop app for creating and editing the config through forms.
- **`script-manager`**: a terminal UI with the same items and actions, driven by the keyboard.

All three run on Windows and Linux.

# Disclaimer

This app is 100% vibe-coded by Claude. No human has read most of this code, and Claude doesn't have hands to knock on wood, so there's no guarantee or claim that any of it works perfectly. It exists purely because a developer wanted a nicer way to run his scripts but was too lazy to write one himself, so he made an AI do it instead — truly the pinnacle of engineering effort. Use at your own risk, read the code before trusting it with anything important, and if it deletes your production database, that's between you and the vibes.

*PS: Claude also wrote this disclaimer, so take it with the appropriate grain of salt.*

## Getting started

```bash
./bin/script-manager-gui                        # find the config automatically
./bin/script-manager-gui -config /path/to/config.yaml
```

`sm-config-edit` and `script-manager` take the same `-config` flag. Without it, all three apps look for the config in this order:

1. next to the binary,
2. in the working directory,
3. in the app-data directory: `%AppData%\script-manager` on Windows, `~/.config/script-manager` on Linux.

On Windows, `config-win.yaml` wins over `config.yaml` at each location. If a file fails to load, the next one is used and the error is shown once. If there is no config anywhere, a small starter config is created in the app-data directory.

The easiest way to build a config is the [Config Editor](#config-editor). The [Configuration](#configuration) section describes every setting.

## GUI

`script-manager-gui` shows four panes:

- **Items**: rows labelled by your `display.list` template.
- **Actions**: the actions available for the selected item.
- **Details**: the item, rendered as Markdown from your `display.details` template.
- **Command**: the selected action's expanded command.

To use it:

- **Pick an item, then an action.** Chips under the Actions header filter by group. Selecting several chips shows only actions in *all* of them; **All** clears the filter.
- **Copy values.** Click any `` `backticked` `` value in Details to copy it. Masked values (`••••••`) copy the real value without ever showing it. Multi-line values show as a placeholder such as `(6-line value)`; hover it to see the value, click it to copy.
- **Arrange the panes.** Click a pane header to collapse it, and drag the dividers to resize. Sizes and collapsed state are remembered.
- **Reload the config** with `F5` or the **Refresh config** button. The GUI also picks up every save from the Config Editor on its own.

### Running actions

The Command pane has two ways to run the selected action:

- **Run** opens it in a new terminal window. The window is independent — its output does not come back into the app. With `noWait: true` the window closes when the command finishes; otherwise it stays open so you can read the output.
- **Run here** runs it inside the app and streams the output into the pane's **OUTPUT** section, with **Cancel** and **Copy output** buttons. The command gets no keyboard input, so a command that asks for input fails straight away. Actions marked `interactive: true` don't offer **Run here**.

Several actions can run inline at once. A dot next to an item or action shows a run in progress or finished: green for success, red for failure. Selecting the action again brings its output back.

Both kinds of run start in the app-data directory, so relative paths in a command resolve there. For a `script:` action, the Command pane shows the file's source, and a pencil button opens it in your default editor. The preview refreshes when the file changes.

A PIN-protected action asks for the PIN before it runs (see [PIN-locked values](#pin-locked-values)).

The terminal used by **Run** is picked automatically, or set with [`terminal:`](#terminal).

### Toolbar

The toolbar is also the window's title bar: drag an empty part of it to move the window. From left to right:

- **Load config**: the last 10 configs you opened, **Browse…** to pick another file, and **Clear recent**. The list is shared with the Config Editor.
- **Refresh config**: same as `F5`.
- **Pin**: keeps the window on top of other windows.
- **Opacity** (checkerboard icon): a slider from 20% to 100%. It only has an effect on Windows.
  - **Shrink on focus loss** is available while the window is pinned. When you click away, the window shrinks to a small nub on the right edge of the screen. Hover the nub to enlarge it, and click it to restore the window.
- **About** ("i" icon): version and project link.
- **Config Editor** (gear icon, `Ctrl+E`): opens the loaded config in the Config Editor.
- Minimize, maximize and close.

`Ctrl +` / `Ctrl -` scale the interface from 70% to 200%, and `Ctrl 0` resets it. The pin and opacity are remembered. The scale and the theme are shared with the Config Editor, where themes are chosen.

## Config Editor

`sm-config-edit` edits the config through forms. With an item selected, a live preview shows its list label, its details and every action's expanded command.

> **Saving rewrites the whole file: comments and custom formatting in `config.yaml` are lost.** Autosave is on by default and writes about a second after you stop typing, so turn autosave off before opening a hand-written config whose comments you want to keep.

- **File**: **New**, **Open**, **Save** and **Save As** (`Ctrl+N` / `Ctrl+O` / `Ctrl+S` / `Ctrl+Shift+S`).
  - **Open** lists your recent configs, the same list the GUI shows.
  - On launch the editor opens the same config the GUI would.
  - Autosave is toggled with the circular-arrow button. It waits while a blocking error is shown, and a new config is only written once you have saved it with a name.
  - Buttons on the right open the app-data folder, or the config in your default editor.
- **Sections**: one form per part of the config: Items, Action Groups, Actions, Displays, Environment, Shell, Terminal and PIN. Two more sections hold app-wide settings rather than the config: Theme and Messages.
- **Fields**: each variable has a type: string, multiline, number, bool or yaml.
  - The **Mark as secret** button (open lock) marks a field **secret**, so its value is hidden until you click into it. Names ending in "password", "secret" or "key" are marked secret automatically.
  - The **PIN-protected** button (padlock) makes a field PIN-protected (see [PIN-locked values](#pin-locked-values)). A PIN-protected field is always secret: turning PIN protection on switches secret on, and the secret button stays disabled until PIN protection is turned off again.
- **Actions**: switch between **Command** (an inline command) and **Script file** (a path with **Browse…** and a preview of the file).
- **Reorder** items, action groups and actions by drag and drop, using the grip button. The order is what the other apps show.
- **Duplicate** the selected item, action group, action, display or theme with the copy button. The copy gets a " - Copy" name and, for actions and groups, a new id.
- **Code editors** highlight shell, PowerShell and YAML. In a template, typing `{{` suggests the available variables (`Ctrl+Space` shows the list anywhere inside `{{ }}`). The Displays section adds edit/preview/split views, an **Insert env…** menu, and buttons for bold, italic, highlight and mask.
- **Validation**: these block saving:
  - duplicate action ids;
  - an item with PIN-protected values that has no name, or shares its name with another item.

  Other problems, such as a reference to a display, action or group that doesn't exist, are shown as warnings.
- **Theme**: pick a theme to apply it in both apps at once. Add, copy, delete or reset custom themes; the built-in Dark and Light themes can't be changed. Every color can be edited with a live preview, and clicking an element in the preview shows only the colors it uses.
- **Fonts** (in the Theme section): set the interface font and the code font to any installed font, or leave them empty for the built-in font. The other app picks the change up the next time it starts.
- **Messages**: change any text either GUI app shows (toasts, tooltips, labels). Search by key or text, restore one message or all of them. Changes take effect the next time that app starts.

## Configuration

A config has these top-level keys, all optional:

| Key | What it sets |
|---|---|
| `shell` | how commands are run, e.g. `[bash, -c]` or `[pwsh, -NoProfile, -Command]` |
| `terminal` | which terminal **Run** opens (see [Terminal](#terminal)) |
| `display` | how items look in the list and the Details pane |
| `env` | variables shared by all items |
| `items` | the things you run actions against |
| `actionGroups` | titles and colors for action groups |
| `actions` | the commands you can run |

### Example

```yaml
shell: [bash, -c]

display:
  - name: default                 # items use the first display unless they pick another
    list: "{{.description}}"      # label in the Items list
    details: |                    # Markdown in the Details pane
      **{{.description}}**

      | Field   | Value |
      |---------|-------|
      | Cluster | `{{.clusterName}}` |
      | IP      | `{{.clusterIp}}` |
  - name: compact
    list: "{{.name}} ({{.clusterIp}})"
    details: "## {{.name}}"

env:                              # available to every item; an item's own value wins
  region: eu-west-1
  sshUser: admin

items:
  - name: Production
    actionGroups: [safe]          # show only these groups…
    actions: [ssh]                # …plus these actions
    customActions:                # actions only this item has
      - title: Emergency rollback
        cmd: echo "Rolling back {{.clusterName}}"
    env:
      description: Production cluster
      clusterName: prod-cluster-eu
      clusterIp: 10.0.0.1
  - name: Dev
    display: compact
    env:
      description: Dev cluster
      clusterIp: 10.0.0.2

actionGroups:
  - id: connect
    title: Connectivity
    color: "#7fd4ff"
  - id: safe
    title: Safe to run anytime
    color: "#4caf50"

actions:
  - id: ssh
    title: SSH into cluster
    description: Open an SSH session to the node.
    groups: [connect]
    cmd: ssh {{.sshUser}}@{{.clusterIp}}
    interactive: true
  - id: hosts
    title: Show hosts
    groups: [safe]
    cmd: cat /etc/hosts
  - id: dashboard
    title: Open dashboard
    cmd: xdg-open http://{{.clusterIp}}:8080
    noWait: true
  - id: deploy
    title: Deploy
    script: /opt/scripts/deploy-{{.clusterName}}.sh
```

On Windows, use a PowerShell `shell:` and Windows commands. Keeping them in `config-win.yaml` lets one folder hold both configs.

### Items

An item has these keys, and only these:

| Key | Meaning |
|---|---|
| `name` | the item's name, available as `{{.name}}` |
| `display` | which named display to use (default: the first one) |
| `actions` | ids of actions to show |
| `actionGroups` | groups whose actions to show |
| `customActions` | extra actions for this item only, with the same fields as `actions` |
| `env` | the item's variables |

Filtering: with none of `actions`, `actionGroups` or `customActions` set, an item shows every action. With any of them set, it shows only the matching actions, with custom actions last.

A config from an older version, with variables directly under the item instead of under `env:`, is offered for conversion when opened. The conversion keeps a timestamped `.bak` copy of the file, and comments are lost. Declining means the config isn't opened.

### Actions

| Field | Meaning |
|---|---|
| `title` | what the Actions list shows |
| `id` | lets items pick this action with `actions:` |
| `description` | shown above the command (one line or several) |
| `groups` | groups the action belongs to |
| `cmd` | a command template, run through `shell:` |
| `script` | a path to a script or program, run directly (the path can use templates) |
| `interactive` | `true` for commands that need keyboard input: hides **Run here** in the GUI |
| `noWait` | `true` to close the terminal (GUI) or return straight away (TUI) when the command ends |
| `requiresPin` | `true` to ask for the PIN and pass PIN-locked values to the command |

Use either `cmd` or `script`, not both. A `script` can be a `.ps1`, a shell script with its own shebang, or an executable.

Action groups need no declaration. Listing one under `actionGroups:` just gives it a nicer `title` and a `color` for the GUI's filter chips.

### Variables

Every action receives the global `env:`, the item's `env:` and the item's `name` as **uppercase** environment variables. An item's value overrides a global one with the same name.

```bash
# item env clusterIp: 10.0.0.1, global env region: eu-west-1
echo $CLUSTERIP   # 10.0.0.1
echo $REGION      # eu-west-1
```

The same values are available in templates, unprefixed and in their original case: `{{.clusterIp}}`.

> **Secrets in commands:** **Run** writes the fully expanded command to a temporary file for a moment. A secret written into a `cmd:` template sits there in plain text until the file is cleaned up. Prefer reading secrets from environment variables in the script.

### Displays and templates

`display.list`, `display.details`, `cmd` and `script` are [Go templates](https://pkg.go.dev/text/template). `display.details` is rendered as Markdown: headings, bold, italic, tables and lists.

- **Copyable values**: wrap a value in backticks (`` `{{.clusterIp}}` ``) to highlight it and make it copyable. Always use backticks for values that may span several lines, such as certificates. They then show as a copyable placeholder instead of breaking the layout.
- **Masking**: `` `{{mask .password}}` `` shows `••••••` and still copies the real value. Masking only hides the value on screen; it is still plain text in the config. For real protection, see [PIN-locked values](#pin-locked-values).
- **Missing fields** render as `<nil>`, and a warning above the details names them.
- **Placeholders** (plain text, not `{{ }}`):
  - `#ALL_ENV_LIST#` — a bullet list of every variable the item passes to its actions.
  - `#ALL_ENV_TABLE#` — the same as a two-column table.
  - `#CONFIG_FILE#` — the path of the loaded config.

  Variables named like secrets (ending in password, pwd, secret, key, token, credential or auth) are masked automatically in the two lists.

### Terminal

By default, **Run** uses the first terminal it finds:

- **Windows**: Windows Terminal (`wt`, reusing one window), then `wezterm`, `alacritty`, `cmd`.
- **Linux**: `x-terminal-emulator`, then `gnome-terminal`, `konsole`, `xfce4-terminal`, `terminator`, `kitty`, `alacritty`, `wezterm`, `foot`, `xterm`.

To choose one, name it:

```yaml
terminal: alacritty
```

Or give a full launch command. `{{title}}` and `{{dir}}` are filled in, and the command to run is appended at the end:

```yaml
terminal: [wezterm-gui, start, --title, "{{title}}", --cwd, "{{dir}}", --]
```

### PIN-locked values

A PIN-locked value is stored encrypted **on your computer**, not in the config. The config only marks the variable as secret (`password: !secret`), so it is safe to commit and share. Everyone who uses a shared config sets their own PIN and enters their own values.

**Setting it up (Config Editor):**

1. In the **PIN** section, set a PIN. Choose something like a password, not a 4-digit code.
2. Click the padlock next to any variable, global or per item, to lock it. For a field that has no value on this computer yet, type the value and it is stored encrypted.
3. On each action that needs the locked values, tick **Requires PIN**.

**How it behaves:**

- A locked value is **never shown**. It displays as `(locked)` everywhere, even after the PIN is entered, and it can't be copied.
- A value that nobody has entered on this computer shows as `(not set)`.
- Running a **Requires PIN** action asks for the PIN once per app session, then passes the values to the command as normal environment variables.
- Actions without **Requires PIN** run with no prompt, and the locked variables are simply not set.
- The PIN section also shows where your secrets file is and lets you:
  - **Change PIN**;
  - **Remove PIN**: after a confirmation, the next save deletes your secrets file, and the values go back to `(not set)`;
  - **Give this config its own secrets**: use this after copying a config, so the copy stops sharing the original's secrets.
- Your secrets file is `%AppData%\script-manager\secrets\<id>.yaml` on Windows or `~/.config/script-manager/secrets/<id>.yaml` on Linux.
- **There is no recovery**: a forgotten PIN means entering every value again.
- A config with PIN-locked values from an older version keeps working. The next save in the Config Editor moves its values into your secrets file, with the same PIN. For a shared config, the other people then set their own PIN and enter their values again.

The PIN protects the values in the file, in backups, sync folders and repositories. It does not protect against other programs running as you while the app is unlocked.

A demo config with locked values (PIN `demo1234`) is in [`examples/`](examples/).

## Terminal UI

`script-manager` runs in a terminal. It uses the same config and supports PIN prompts.

Compared with the GUI, it has these limitations:

- An action runs in the same terminal, taking over the screen until it finishes. Only one action runs at a time, and there is no **Run here** or captured output.
- There is no mouse support, toolbar or recent-configs list. Choose the config with `-config` when launching.
- Group titles and colors are not shown; groups are cycled with `[` and `]`.
- There are no themes, interface scaling or window options.

It works in two modes: pick an item with `Enter`, then pick and run an action; `Esc` goes back.

**Choosing an item**

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | move |
| `Enter` / `Tab` | select the item |
| `F5` | reload the config |
| `Q` / `Esc` / `Ctrl+C` | quit |

**Choosing an action**

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | move / scroll |
| `Tab` / `→`, `Shift+Tab` / `←` | next / previous pane: Actions, Details, Command |
| `[` / `]` | previous / next action group |
| `Enter` | run the action (Actions pane) / copy a value (Details pane) |
| `y` | copy the expanded command |
| `F5` | reload the config |
| `Esc` | back to items |
| `Q` / `Ctrl+C` | quit |

## Building

```bash
bash build.sh             # Windows and Linux
bash build.sh --windows   # or --linux
```

This produces `bin/script-manager`, `bin/script-manager-gui` and `bin/sm-config-edit`, each for Linux and as `.exe` for Windows.

Requirements:
- Go 1.21 or newer.
- For the two GUI apps: the [Wails CLI](https://wails.io/docs/gettingstarted/installation) and Node.js.
  - For Linux builds: GTK/WebKit headers.
  - For Windows builds from Linux: `mingw-w64`.
- The dev container has all of this installed.

A GUI app whose tools are missing is skipped.

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config nodejs npm gcc-mingw-w64-x86-64
```

The version shown in the GUI's About panel comes from the git tag, for example `v1.4.0` or `v1.4.0-4-g94b2835`. Builds made outside `build.sh` or outside a git checkout say `dev`.

On a Windows host that has Go only inside the VS Code dev container, run `build-container.ps1` instead. It closes running copies of the apps and builds inside the container:

```powershell
.\build-container.ps1
.\build-container.ps1 -Windows   # or -Linux
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

Go:

- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [charmbracelet/glamour](https://github.com/charmbracelet/glamour) — Markdown rendering for the TUI Details pane
- [mko88/bubbletea-tilelayout](https://github.com/mko88/bubbletea-tilelayout) — tile layout manager
- [atotto/clipboard](https://github.com/atotto/clipboard) — clipboard support
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) — config parsing
- [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) — watching the config and theme files for changes
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) — Argon2id key derivation for [PIN-locked values](#pin-locked-values)
- [wailsapp/wails](https://wails.io) — GUI shell (Go backend + native webview)
- [yuin/goldmark](https://github.com/yuin/goldmark) — Markdown → HTML rendering for the GUI Details pane

Frontend, in both GUI apps:

- [Svelte](https://svelte.dev) — UI framework, built with [Vite](https://vitejs.dev)
- [CodeMirror 6](https://codemirror.net) — every pane that shows or edits code: script and command previews, the inline command, YAML and multi-line values, the Displays templates
- [svelte-dnd-action](https://github.com/isaacHagoel/svelte-dnd-action) — drag-and-drop reordering in the Config Editor's lists
- [Nunito](https://fonts.google.com/specimen/Nunito) — the bundled interface font (SIL Open Font License, see `frontend-shared/fonts/OFL.txt`)
