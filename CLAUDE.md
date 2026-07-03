# Claude Notes — script-manager

## Project structure

```
cmd/script-manager/   ← entry point (main, runAction, waitForKey)
internal/
  config/             ← Config types and YAML loading (config.Load())
  ui/                 ← all TUI tiles and the App model
    app.go            ← App, State, NewApp, SaveState, RestoreState
    list.go           ← ListTile, selectableList
    detail.go         ← DescriptionTile, ActionsTile
    statusbar.go      ← StatusBarTile
    common.go         ← renderBox, padToLines
```

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
