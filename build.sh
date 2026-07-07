#!/usr/bin/env bash
# Builds both Windows and Linux binaries by default. Pass --windows or
# --linux to build only that platform — --windows for routine use on the
# Windows host (which never runs the Linux binaries), --linux when only a
# Linux binary is needed (e.g. Xvfb-based visual verification of the GUI
# apps in the dev container).
set -e

build_windows=1
build_linux=1
case "$1" in
	--windows) build_linux=0 ;;
	--linux) build_windows=0 ;;
esac

mkdir -p bin

if [ "$build_linux" = 1 ]; then
	echo "Building Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o bin/script-manager ./cmd/script-manager/
fi

if [ "$build_windows" = 1 ]; then
	echo "Building Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -o bin/script-manager.exe ./cmd/script-manager/
fi

if command -v wails &> /dev/null; then
	for app in script-manager-gui sm-config-edit; do
		if [ "$build_linux" = 1 ]; then
			echo "Building GUI ($app, linux/amd64)..."
			(cd "cmd/$app" && wails build)
			cp "cmd/$app/build/bin/$app" bin/
		fi

		if [ "$build_windows" = 1 ]; then
			if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
				echo "Building GUI ($app, windows/amd64)..."
				(cd "cmd/$app" && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
					CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
					wails build -platform windows/amd64)
				cp "cmd/$app/build/bin/$app.exe" bin/
			else
				echo "mingw-w64 not found — skipping GUI Windows cross-compile for $app (see README)"
			fi
		fi
	done
else
	echo "wails CLI not found — skipping GUI builds (see README for setup)"
fi

echo "Done."
ls -lh bin/
