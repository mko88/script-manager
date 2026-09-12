#!/usr/bin/env bash
# Builds both Windows and Linux binaries by default. Pass --windows or
# --linux to build only that platform — --windows for routine use on the
# Windows host (which never runs the Linux binaries), --linux when only a
# Linux binary is needed (e.g. Xvfb-based visual verification of the GUI
# apps in the dev container).
#
# go vet, go test, and svelte-check are opt-in via --vet/--test/--check (or
# --full for all three) — they're not part of the default run since most
# changes don't need all of them; see CLAUDE.md's build discipline notes.
set -e

build_windows=1
build_linux=1
run_vet=0
run_test=0
run_check=0
build_debug=0

for arg in "$@"; do
	case "$arg" in
		--windows) build_linux=0 ;;
		--linux) build_windows=0 ;;
		--vet) run_vet=1 ;;
		--test) run_test=1 ;;
		--check) run_check=1 ;;
		--devtools) build_debug=1 ;;
		--full) run_vet=1; run_test=1; run_check=1 ;;
	esac
done

if [ "$run_vet" = 1 ]; then
	echo "Running go vet..."
	go vet ./...
fi

if [ "$run_test" = 1 ]; then
	echo "Running go test..."
	go test ./...
fi

if [ "$run_check" = 1 ]; then
	for app in script-manager-gui sm-config-edit; do
		echo "Running svelte-check ($app)..."
		(cd "cmd/$app/frontend" && npm run check)
	done
fi

# --devtools ships the GUI apps with WebView2's inspector enabled, so a
# console error in a built app can be read (right-click, Inspect). Not for
# release builds.
wails_extra=""
if [ "$build_debug" = 1 ]; then
	wails_extra="-devtools"
	echo "Building GUI apps with devtools enabled"
fi

mkdir -p bin

# Stamped into internal/version at link time, so the binaries can report
# which build they are instead of carrying a hand-edited constant.
# `git describe` gives "v1.4.0" on a release tag, "v1.4.0-4-g94b2835"
# past one, and a "-dirty" suffix with uncommitted changes. Outside a git
# checkout the defaults in the package stand ("dev").
version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
commit=$(git rev-parse --short HEAD 2>/dev/null || echo "")
date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ldflags="-X script-manager/internal/version.Version=$version"
ldflags="$ldflags -X script-manager/internal/version.Commit=$commit"
ldflags="$ldflags -X script-manager/internal/version.Date=$date"
echo "Version: $version"

# The TUI builds alongside the GUI apps, but the two GUI apps build one
# after the other, in a single job.
#
# `wails build` regenerates the app's frontend/wailsjs from the bound Go
# types. Run concurrently, the two generators race and one has been observed
# writing its bindings into the *other* app's frontend — leaving
# sm-config-edit with the GUI's App.js and a build that fails to resolve its
# own. The two platforms of one app are sequential for the same reason: both
# runs rewrite that app's frontend/dist.
#
# Each job's output is captured to a file and printed as one block when the
# job is collected, so logs don't interleave.
job_dir=$(mktemp -d)
trap 'rm -rf "$job_dir"' EXIT
job_pids=()
job_names=()

start_job() { # start_job <name> <command...>
	local name=$1; shift
	"$@" > "$job_dir/$name.log" 2>&1 &
	job_pids+=($!)
	job_names+=("$name")
}

wait_jobs() {
	local failed=0 i
	for i in "${!job_pids[@]}"; do
		if ! wait "${job_pids[$i]}"; then
			failed=1
		fi
		cat "$job_dir/${job_names[$i]}.log"
	done
	job_pids=()
	job_names=()
	[ "$failed" = 0 ]
}

build_tui() {
	if [ "$build_linux" = 1 ]; then
		echo "Building Linux (amd64)..."
		GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o bin/script-manager ./cmd/script-manager/
	fi

	if [ "$build_windows" = 1 ]; then
		echo "Building Windows (amd64)..."
		GOOS=windows GOARCH=amd64 go build -ldflags "$ldflags" -o bin/script-manager.exe ./cmd/script-manager/
	fi
}

build_all_guis() {
	for app in script-manager-gui sm-config-edit; do
		build_gui "$app"
	done
}

build_gui() { # build_gui <app>
	local app=$1

	if [ "$build_linux" = 1 ]; then
		echo "Building GUI ($app, linux/amd64)..."
		(cd "cmd/$app" && wails build $wails_extra -ldflags "$ldflags")
		cp "cmd/$app/build/bin/$app" bin/
	fi

	if [ "$build_windows" = 1 ]; then
		if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
			echo "Building GUI ($app, windows/amd64)..."
			(cd "cmd/$app" && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
				CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
				wails build $wails_extra -platform windows/amd64 -ldflags "$ldflags")
			cp "cmd/$app/build/bin/$app.exe" bin/
		else
			echo "mingw-w64 not found — skipping GUI Windows cross-compile for $app (see README)"
		fi
	fi
}

start_job tui build_tui

if command -v wails &> /dev/null; then
	start_job gui build_all_guis
else
	echo "wails CLI not found — skipping GUI builds (see README for setup)"
fi

wait_jobs || { echo "Build failed — see output above." >&2; exit 1; }

echo "Done."
ls -lh bin/
