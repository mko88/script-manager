#!/usr/bin/env bash
set -e

mkdir -p bin

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/script-manager ./cmd/script-manager/

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o /tmp/script-manager.exe ./cmd/script-manager/
cp /tmp/script-manager.exe bin/script-manager.exe

echo "Done."
ls -lh bin/script-manager bin/script-manager.exe
