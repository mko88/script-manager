#!/usr/bin/env bash
set -e

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o script-manager .

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o /tmp/script-manager.exe .
cp /tmp/script-manager.exe script-manager.exe

echo "Done."
ls -lh script-manager script-manager.exe
