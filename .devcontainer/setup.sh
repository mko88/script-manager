#!/usr/bin/env bash
# Installs everything needed to build and smoke-test both the TUI
# (cmd/script-manager) and the GUI (cmd/script-manager-gui), including
# cross-compiling the GUI's Windows binary, all inside this devcontainer.
set -e

echo "Installing build dependencies..."
sudo apt-get update
sudo apt-get install -y \
	build-essential pkg-config nodejs npm \
	libgtk-3-dev libwebkit2gtk-4.0-dev \
	gcc-mingw-w64-x86-64 nsis \
	xclip xvfb imagemagick x11-apps xdotool
# libgtk/libwebkit2gtk: native Linux GUI target (webview)
# gcc-mingw-w64/nsis:   cross-compile + package the GUI's Windows target
# xclip:                clipboard support (used by both TUI and GUI on Linux)
# xvfb/imagemagick/x11-apps/xdotool: headless GUI smoke-testing (screenshot + simulated clicks)

echo "Installing Wails CLI..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest

echo "Downloading Go modules..."
go mod download

echo "Done. 'bash build.sh' now also produces bin/script-manager-gui and bin/script-manager-gui.exe."
