# Claude Notes — script-manager

## Project structure

```
cmd/script-manager/       ← TUI entry point (thin main: flags + tea.NewProgram)
cmd/script-manager-gui/   ← Wails GUI entry point (thin main + frontend/)
cmd/sm-config-edit/       ← Wails config-editor entry point (thin main + frontend/)
frontend-shared/          ← CSS design system + Svelte components shared by both
                             Wails frontends (theme.css, components/Toast.svelte);
                             each frontend's vite.config.ts aliases it as "@shared"
internal/
  config/             ← Config types, YAML loading (config.LoadWithError()) and
                        saving (Config.Marshal()), reserved item-key constants
                        (config.KeyName, …)
  action/             ← logic shared by TUI and GUI: Merge, Expand, Preview, Env
  render/             ← mask pipeline (MaskFunc, ProcessMaskSpans) and the
                        #ALL_ENV_LIST#/#ALL_ENV_TABLE# placeholder expansion
                        (ExpandAllEnv), shared by both frontends
  gui/                ← Wails-bound App backend (DTOs, RunAction, temp scripts);
                        bound under the "gui" namespace (window.go.gui.App)
  configedit/         ← Wails-bound App backend for sm-config-edit (DTOs,
                        Config<->DTO conversion, validation, live preview);
                        bound under the "configedit" namespace
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
- `bin/script-manager-gui[.exe]`, `bin/sm-config-edit[.exe]` — the two Wails
  GUI apps, built in the same loop in `build.sh` (skipped individually if
  `wails`/`mingw-w64` aren't available)

On a Windows host with no local Go toolchain, working through a VS Code dev
container (no host Go install; build via `docker exec` into the container),
run `build-container.ps1` from the repo root instead of the manual
stop-process + `docker exec` dance: it stops any host-side
`script-manager*.exe`/`sm-config-edit*.exe` (a locked binary makes the Windows
cross-compile step fail with "permission denied"), finds the running dev
container for this repo by its `devcontainer.local_folder` label (the
container name is auto-generated and changes across recreations — don't
hardcode one), and runs `bash build.sh` inside it.

## Verifying GUI changes

After making changes to `cmd/script-manager-gui/` or `cmd/sm-config-edit/`, build the binaries (`bash build.sh`) but stop there — don't automatically launch into a full visual verification pass (Xvfb, screenshots, simulated clicks via xdotool). It's slow and not always necessary. Instead, ask the user to pick one:

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
