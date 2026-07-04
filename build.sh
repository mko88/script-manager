#!/usr/bin/env bash
set -e

mkdir -p bin

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/script-manager ./cmd/script-manager/

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o bin/script-manager.exe ./cmd/script-manager/

if command -v wails &> /dev/null; then
	echo "Building GUI (linux/amd64)..."
	(cd cmd/script-manager-gui && wails build)
	cp cmd/script-manager-gui/build/bin/script-manager-gui bin/

	if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
		echo "Building GUI (windows/amd64)..."
		(cd cmd/script-manager-gui && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
			wails build -platform windows/amd64)
		cp cmd/script-manager-gui/build/bin/script-manager-gui.exe bin/
	else
		echo "mingw-w64 not found — skipping GUI Windows cross-compile (see README)"
	fi
else
	echo "wails CLI not found — skipping GUI build (see README for setup)"
fi

echo "Done."
ls -lh bin/
