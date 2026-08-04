#!/bin/bash
# Stops the Go backend server

PID_FILE="$HOME/.gode/storage/gode-go-server.pid"
PORT_FILE="$HOME/.gode/storage/gode-go-server.port"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
        echo "Stopping Go backend (PID: $PID)..."
        kill "$PID"
        sleep 1
        # Force kill if still running
        if kill -0 "$PID" 2>/dev/null; then
            kill -9 "$PID"
        fi
        echo "Go backend stopped"
    else
        echo "Go backend not running (stale PID file)"
    fi
    rm -f "$PID_FILE"
else
    echo "No Go backend PID file found"
fi

# Clean up port file
rm -f "$PORT_FILE"