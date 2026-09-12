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

### README style: user guide, not engineering notes

The README is a guide for someone *using* the apps, describing the current
state only. Document what the app does and how to use it — not how it's
implemented or why it was built that way.

Belongs in the README:
- How to launch, configure, and operate the apps: keybindings, flags,
  config keys and their effects, search/precedence orders users rely on.
- Behaviour a user acts on or must know to avoid surprises — e.g. the
  "comments are lost on save" warning, the secrets-in-temp-scripts note,
  "changes take effect on next launch".
- Just enough of a feature's mechanics to use it (e.g. "click a preview
  element to filter the field list") — not how the mechanic works inside.

Does NOT belong (put it in CLAUDE.md or a code comment instead, or drop it):
- "We do X because of Y" implementation rationale, and internal
  identifiers users never type (theme tokens, package/component names,
  CSS selectors).
- Change history — "previously this used…", renames, "now uses". The
  CHANGELOG covers that.
- Build internals that don't change how the build is invoked (e.g. that
  build.sh parallelizes its jobs).

## No changelog comments in code

**Rule: don't leave comments that narrate a change's history** — "retired
2026-09-04", "no longer X", "used to be Y", "removed in favor of Z", "just
relocated here", a dated note explaining why a field/case/branch was
deleted. That belongs in the commit message and git history, not the
source. A comment should describe the code as it is now; if something
isn't there anymore, it needs no comment at all, not an epitaph.

This doesn't apply to comments documenting a non-obvious *constraint* the
current code exists to satisfy — why a reactive statement is written a
particular way to avoid a real bug, why a value is read back instead of
assumed. That's about the present code being correct, not about what used
to be there. Write those as the constraint ("re-deriving this mid-drag
corrupts dndzone's tracking"), not as the story of discovering it.

Keep them condensed. A comment earns its length by what the next reader
has to know, not by how much was learned writing it: prefer the two lines
that state the constraint over the ten that reconstruct the investigation.

## Release notes are short

The GitHub release body *is* the matching `## <Version>` section of
`CHANGELOG.md` (`scripts/release.ps1` pulls it out verbatim), so the two
have one standard.

**Rule: `### Changes` and `### Bug fixes` under the version heading, and
nothing else.** No Downloads list — the assets are on the release page
already. No account of what was verified or how it was built.

**One line per entry, saying what changed rather than how it was found or
fixed.** The investigation belongs in the commit message, which still has
it:

    - Request options set as workspace defaults were ignored.

not a paragraph on which layer answered with the wrong defaults. Prefix
the line with which binary it affects (e.g. `` `script-manager-gui`: ``)
when that isn't obvious.

Spend length only where the reader has to *do* something: a breaking
change goes first, marked, and may take a paragraph with the before and
after — everything else is a line.

## Bumping the version

**The version is a git tag, not a source constant — never hand-edit
`internal/version`.** `build.sh` stamps `internal/version.{Version,Commit,
Date}` at link time via ldflags, from `git describe --tags --always
--dirty`. So a build sitting exactly on tag `v1.4.0` reports
`v1.4.0`; four commits later it reports `v1.4.0-4-g94b2835`; with
uncommitted changes it gains `-dirty`; and a binary built outside a git
checkout (or by a bare `go build`/`wails build`) says `dev`. That string
is what `script-manager-gui`'s About panel shows, so a screenshot of it
identifies the exact commit it came from.

Releasing is therefore: merge to `main`, then tag that merge commit.

```
git tag -a v1.4.0 -m "Copy buttons for items, action groups and actions"
```

Tags are `v` + `Major.Minor.Patch` and must match the newest
`CHANGELOG.md` heading. Bump exactly one segment per merge to `main`,
resetting every segment after it to `0`:

- **Major** — by hand only, for a big rewrite or breaking change. Never
  bump this automatically.
- **Minor** — new user-facing functionality (e.g. `1.4.0` → `1.5.0`).
- **Patch** — everything else that reaches a shipped binary: a bug fix, a
  refactor, a dependency bump, anything under `cmd/`, `internal/`
  (excluding `_test.go` files), or a frontend app's `src/`/`frontend-shared`
  (e.g. `1.4.0` → `1.4.1`).

Tags before `v1.4.0` carry a fourth segment, from the scheme this
replaced; they are left as they were.

Skip the tag entirely for a merge that touches nothing compiled into a
binary — a docs-only (`README.md`, `CLAUDE.md`), comment-only, or test-only
change. Those merges just ride along under the previous tag, which
`git describe` reports as `v1.4.0-2-gabc1234`.

Whichever segment is bumped, add a matching `## Major.Minor.Patch`
heading to `CHANGELOG.md` (newest on top, no `v` prefix) in the feature
branch's own commits — not in a separate post-merge commit, so the tag
lands on a commit whose changelog already describes it. Write the entry
per "Release notes are short" above.

Do this before closing the task, same as the README update below. Creating
the tag is the user's call — never tag or push tags automatically; say
which tag the merge is due instead.

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

### Before committing or pushing

Don't commit on the strength of interim checks alone. If this session
hasn't run a full build (`bash build.sh --full` / `.\build-container.ps1
-Full`, both platforms, `--vet`/`--test`/`--check` all included) since the
last code change, say so explicitly before creating the commit.

Before running that (or any) build, always ask first and wait for the
answer — never launch it automatically, even when offering to. The user
may already know it's covered (e.g. just ran it, or the diff is trivial)
and not want to wait on a redundant pass.

Exception: skip asking when the diff is only build scripts, docs, or
comments — no actual code logic changed, so a full build/test pass can't
catch anything the change could have broken.

## Verifying changes: build only, never run the app

**Never launch or manually test the application — in any session, for any
app in this repo (`script-manager`, `script-manager-gui`, `sm-config-edit`).**
Build the Windows binaries and stop there; the user runs them and verifies
the result.

That means none of the following, ever, unless the user explicitly asks for
it in that same request:
- Starting a binary from `bin/` (`./bin/script-manager-gui.exe`, `go run`, …).
- The Xvfb visual-verification loop — virtual display, screenshots,
  simulated clicks via `xdotool`. Don't offer it as an option either.
- Driving the TUI through a pty/terminal harness to exercise keybindings.

So the finish line for a code change is:

```
.\build-container.ps1 -Windows        # plus -Vet/-Test/-Check as the diff warrants
```

then report what changed and hand it over for the user to verify. If
something can only be settled by running the app, say what you'd need
confirmed and let the user check it — don't run it yourself.

Static verification is still expected and unaffected: `go build`, `go vet`,
`go test`, and `npm run check` are not "running the app" — scale them to
what the diff touched, per "Build discipline while iterating" above.

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
