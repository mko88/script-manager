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

## Building binaries

Always use `./build.sh` to compile — never run `go build` manually.

```
bash build.sh
```

Produces:
- `script-manager` — Linux amd64
- `script-manager.exe` — Windows amd64

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
