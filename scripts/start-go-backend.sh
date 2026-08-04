#!/bin/bash
# Launches the Go backend server and writes the port to a file for discovery

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GO_SERVER_DIR="$PROJECT_ROOT/editor-go"
PORT_FILE="$HOME/.gode/storage/gode-go-server.port"
PID_FILE="$HOME/.gode/storage/gode-go-server.pid"

# Ensure storage directory exists
mkdir -p "$(dirname "$PORT_FILE")"

# Build the Go server
cd "$GO_SERVER_DIR"
echo "Building Go backend..."
go build -o gode-go-server ./cmd/server/

if [ $? -ne 0 ]; then
    echo "Failed to build Go backend"
    exit 1
fi

# Kill any existing instance
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Killing existing Go backend (PID: $OLD_PID)"
        kill "$OLD_PID" 2>/dev/null || true
        sleep 1
    fi
    rm -f "$PID_FILE"
fi

# Find free port
FREE_PORT=$(python3 -c "
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(('', 0))
print(s.getsockname()[1])
s.close()
")

if [ -z "$FREE_PORT" ] || [ "$FREE_PORT" -eq 0 ]; then
    FREE_PORT=18765
fi

echo "Starting Go backend on port $FREE_PORT..."

# Start the Go server
./gode-go-server --port="$FREE_PORT" --root="$PROJECT_ROOT" &
GO_PID=$!

# Save PID
echo "$GO_PID" > "$PID_FILE"

# Wait for server to be ready
MAX_RETRIES=10
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    sleep 0.5
    if [ -f "$PORT_FILE" ]; then
        GO_PORT=$(cat "$PORT_FILE")
        echo "Go backend is ready on port $GO_PORT (PID: $GO_PID)"
        echo "GO_BACKEND_PORT=$GO_PORT"
        echo "GO_BACKEND_PID=$GO_PID"
        exit 0
    fi
    RETRY=$((RETRY + 1))
done

echo "Warning: Go backend may not be ready yet (PID: $GO_PID)"
echo "GO_BACKEND_PORT=$FREE_PORT"
echo "GO_BACKEND_PID=$GO_PID"
exit 0