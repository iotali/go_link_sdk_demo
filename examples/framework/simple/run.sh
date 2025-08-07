#!/bin/bash

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the SDK root directory (3 levels up from examples/framework/simple)
SDK_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Kill any existing processes connected to IoT platform
echo "Checking for existing connections..."
EXISTING_PIDS=$(lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | sort -u)

if [ ! -z "$EXISTING_PIDS" ]; then
    echo "Found existing connections with PIDs: $EXISTING_PIDS"
    echo "Killing existing processes..."
    echo "$EXISTING_PIDS" | xargs -r kill -9 2>/dev/null || true
    sleep 2
fi

# Change to SDK root directory
cd "$SDK_ROOT"

# Run the application with both source files
echo "Starting IoT framework demo..."
echo "Press Ctrl+C to stop..."

# Run the application directly (not in background for better signal handling)
exec go run examples/framework/simple/main.go examples/framework/simple/electric_oven.go