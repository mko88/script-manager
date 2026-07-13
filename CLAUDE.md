# Claude Notes — script-manager

## Project structure

```
cmd/script-manager/       ← TUI entry point (thin main: flags + tea.NewProgram)
cmd/script-manager-gui/   ← Wails GUI entry point (thin main + frontend/)
cmd/sm-config-edit/       ← Wails config-editor entry point (thin main + frontend/)
frontend-shared/          ← CSS design system + Svelte components/modules shared by
                             both Wails frontends (theme.css, components/{Icon,Toast}.svelte,
                             messages.ts, toast.ts, persist.ts); each frontend's
                             vite.config.ts aliases it as "@shared"
internal/
  config/             ← Config types, YAML loading (config.LoadWithError()) and
                        saving (Config.Marshal()), reserved item-key constants
                        (config.KeyName, …)
  action/             ← logic shared by TUI and GUI: Merge, Expand, Preview, Env
  render/             ← mask pipeline (MaskFunc, ProcessMaskSpans) and the
                        #ALL_ENV_LIST#/#ALL_ENV_TABLE# placeholder expansion
                        (ExpandAllEnv), shared by both frontends
  exepath/            ← executable-directory resolution (Dir()), shared by gui
                        and configedit
  terminal/           ← terminal emulator detection/argv assembly (Launcher,
                        Names(), Resolve()), shared by gui and configedit
  gui/                ← Wails-bound App backend (DTOs, bindings); bound under
                        the "gui" namespace (window.go.gui.App). RunAction,
                        wrapScript, writeTempScript, buildShellArgv, and temp-
                        script cleanup live in runner.go; GetItemDetails and
                        goldmark rendering live in details.go
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

## Branching for new features

When starting work on a new feature, always create a new branch first and do
the work there — don't commit new-feature work directly to `main`. Merge
back to `main` once the feature is done and verified.

Branch name format: `YYYYMMDD-feature`, where `YYYYMMDD` is today's date and
`feature` is a one-or-two-word kebab-case summary of the work (e.g.
`20260713-theme-tokens`).

## Keeping README.md up to date

After any change that affects user-facing behaviour — keybindings, layout, panes, CLI flags, config format, or build instructions — update README.md before closing the task. Do not wait to be asked.

## Building binaries

Always use `./build.sh` to compile — never run `go build` manually.

```
bash build.sh
```

Builds both Windows and Linux binaries by default. Produces:
- `bin/script-manager` / `bin/script-manager.exe` — Linux / Windows amd64
- `bin/script-manager-gui[.exe]`, `bin/sm-config-edit[.exe]` — the two Wails
  GUI apps, built in the same loop in `build.sh` (skipped individually if
  `wails`/`mingw-w64` aren't available)

Pass `--windows` or `--linux` to build only that platform:
`bash build.sh --windows` for routine use (this Windows host never runs the
Linux binaries — always prefer this to save the ~40s the Linux Wails builds
otherwise cost); `bash build.sh --linux` when only a Linux binary is needed,
e.g. Xvfb-based visual verification of the GUI apps in the dev container.

`go vet`, `go test`, and `npm run check` (svelte-check) are opt-in via
`--vet` / `--test` / `--check`, or all three together via `--full`
(`bash build.sh --full`) — not part of the default run. Combine freely with
`--windows`/`--linux`, e.g. `bash build.sh --windows --vet`.

On a Windows host with no local Go toolchain, working through a VS Code dev
container (no host Go install; build via `docker exec` into the container),
run `build-container.ps1` from the repo root instead of the manual
stop-process + `docker exec` dance: it stops any host-side
`script-manager*.exe`/`sm-config-edit*.exe` (a locked binary makes the Windows
cross-compile step fail with "permission denied"), finds the running dev
container for this repo by its `devcontainer.local_folder` label (the
container name is auto-generated and changes across recreations — don't
hardcode one), and runs `bash build.sh` inside it. Same flags, via
`-Windows`/`-Linux`/`-Vet`/`-Test`/`-Check`/`-Full`:
`.\build-container.ps1 -Windows -Full`.

### Build discipline while iterating on a feature

Default to the lightest, fewest builds that actually verify the change in
front of you — this is not the same posture as the final build below.
While iterating:
- Don't run `go vet`/`go test`/`npm run check` (or `--vet`/`--test`/`--check`)
  for every small edit. Run whichever ones the change could plausibly have
  broken, and skip the rest — a pure Svelte/CSS tweak doesn't need `go test`;
  a one-line Go comment change doesn't need `npm run check`.
  Reach for `--full`/`-Full` only when you actually want everything.
- Prefer `--windows`/`-Windows` alone unless a Linux binary specifically
  matters (e.g. Xvfb visual verification) — skip the Linux Wails build's
  ~40s otherwise.
- It's fine to go several edits without invoking `build.sh` at all — a
  `go build ./...`/`go vet ./...` cross-check is enough to catch obvious
  breakage mid-iteration; the real build.sh pass belongs at task boundaries
  (before verifying the change, before a commit), not after every edit.

### Before committing

Don't commit on the strength of interim checks alone. If this session
hasn't run a full build (`bash build.sh --full` / `.\build-container.ps1
-Full`, both platforms, `--vet`/`--test`/`--check` all included) since the
last code change, say so explicitly before creating the commit — offer to
run it rather than assuming the interim checks already covered everything.

Exception: skip this when the diff is only build scripts, docs, or comments
— no actual code logic changed, so a full build/test pass can't catch
anything the change could have broken.

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
