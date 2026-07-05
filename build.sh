#!/usr/bin/env bash
set -e

mkdir -p bin

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/script-manager ./cmd/script-manager/

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o bin/script-manager.exe ./cmd/script-manager/

if command -v wails &> /dev/null; then
	for app in script-manager-gui sm-config-edit; do
		echo "Building GUI ($app, linux/amd64)..."
		(cd "cmd/$app" && wails build)
		cp "cmd/$app/build/bin/$app" bin/

		if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
			echo "Building GUI ($app, windows/amd64)..."
			(cd "cmd/$app" && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
				CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
				wails build -platform windows/amd64)
			cp "cmd/$app/build/bin/$app.exe" bin/
		else
			echo "mingw-w64 not found — skipping GUI Windows cross-compile for $app (see README)"
		fi
	done
else
	echo "wails CLI not found — skipping GUI builds (see README for setup)"
fi

echo "Done."
ls -lh bin/
