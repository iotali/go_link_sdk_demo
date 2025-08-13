#!/bin/bash

echo "Testing Ctrl+C signal handling..."

# Start the oven program in background
./oven &
OVEN_PID=$!

echo "Started oven with PID: $OVEN_PID"

# Wait a bit for the program to fully start
sleep 3

echo "Sending SIGINT (Ctrl+C equivalent) to process..."
kill -INT $OVEN_PID

# Wait for graceful shutdown
sleep 5

# Check if process still exists
if kill -0 $OVEN_PID 2>/dev/null; then
    echo "WARNING: Process still running after 5 seconds, forcing kill..."
    kill -9 $OVEN_PID
    exit 1
else
    echo "SUCCESS: Process terminated gracefully"
    exit 0
fi