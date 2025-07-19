#!/bin/bash

# Development script for TUI applications
# This works better than Air for interactive terminals

build_and_run() {
    echo "Building..."
    if go build -o ./tmp/media-ripper ./cmd/media-ripper; then
        echo "Starting media-ripper..."
        ./tmp/media-ripper
    else
        echo "Build failed!"
        return 1
    fi
}

# Initial build and run
build_and_run

# Watch for changes (requires inotify-tools)
if command -v inotifywait >/dev/null 2>&1; then
    echo "Watching for changes... (Ctrl+C to stop)"
    while inotifywait -r -e modify,create,delete --include='\.go$' cmd/ internal/ 2>/dev/null; do
        echo "Changes detected, rebuilding..."
        build_and_run
    done
else
    echo "Install inotify-tools for auto-rebuild: sudo apt install inotify-tools"
fi