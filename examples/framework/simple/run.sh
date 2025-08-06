#!/bin/bash

# Kill any existing processes connected to IoT platform
echo "Checking for existing connections..."
EXISTING_PID=$(lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | head -1)

if [ ! -z "$EXISTING_PID" ]; then
    echo "Found existing connection with PID: $EXISTING_PID"
    echo "Killing existing process..."
    kill -9 $EXISTING_PID
    sleep 1
fi

# Run the application
echo "Starting IoT framework demo..."
echo "Press Ctrl+C to stop..."

# Run in background and capture PID
go run main.go &
PID=$!

# Function to cleanup on exit
cleanup() {
    echo -e "\nStopping demo..."
    kill $PID 2>/dev/null || true
    wait $PID 2>/dev/null || true
    echo "Demo stopped."
    exit 0
}

# Trap Ctrl+C
trap cleanup INT

# Wait for the process
wait $PID