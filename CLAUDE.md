# Claude Notes — script-manager

## Project structure

```
cmd/script-manager/       ← TUI entry point (thin main: flags + tea.NewProgram)
cmd/script-manager-gui/   ← Wails GUI entry point (thin main + frontend/)
internal/
  config/             ← Config types, YAML loading (config.LoadWithError()),
                        reserved item-key constants (config.KeyName, …)
  action/             ← logic shared by TUI and GUI: Merge, Expand, Preview, Env
  render/             ← mask pipeline (MaskFunc, ProcessMaskSpans) shared by both
  gui/                ← Wails-bound App backend (DTOs, RunAction, temp scripts);
                        bound under the "gui" namespace (window.go.gui.App)
  ui/                 ← all TUI tiles and the App model
    app.go            ← App, NewApp, mode/focus handling
    exec.go           ← actionProcess (tea.Exec wrapper), waitForKey
    list.go           ← ListTile, selectableList
    detail.go         ← DescriptionTile, ActionsTile
    cmdbar.go         ← CmdBarTile
    statusbar.go      ← StatusBarTile
    common.go         ← renderBox, truncateToWidth, scrollableContent
```

Actions in the TUI run via `tea.Exec` — Bubble Tea suspends the UI, hands the
terminal to the subprocess, and resumes the same model. There is no
save/restore state machinery; don't reintroduce it.

When adding new concerns, create a new package under `internal/` rather than adding files to `cmd/` or the root.

## Keeping README.md up to date

After any change that affects user-facing behaviour — keybindings, layout, panes, CLI flags, config format, or build instructions — update README.md before closing the task. Do not wait to be asked.

## Building binaries

Always use `./build.sh` to compile — never run `go build` manually.

```
bash build.sh
```

Produces:
- `bin/script-manager` — Linux amd64
- `bin/script-manager.exe` — Windows amd64

## Verifying GUI changes

After making changes to `cmd/script-manager-gui/`, build the binaries (`bash build.sh`) but stop there — don't automatically launch into a full visual verification pass (Xvfb, screenshots, simulated clicks via xdotool). It's slow and not always necessary. Instead, ask the user to pick one:

1. They'll confirm visually themselves that it looks right.
2. Claude does the full visual verification loop (Xvfb + screenshots + simulated clicks) and reports/fixes what it finds.
3. They'll describe what's wrong and Claude fixes it from that description.

Only go straight to option 2's workflow if the user explicitly asks for visual verification up front.

## .vscode/launch.json

When broadly ignoring `.vscode/` in `.gitignore`, always un-ignore `launch.json` so contributors get the debug config:

```gitignore
.vscode/
!.vscode/launch.json
```

The `program` field must point to `cmd/script-manager/`, not the workspace root:

```json
"program": "${workspaceFolder}/cmd/script-manager"
```
