#!/bin/bash

# 电烤炉运行脚本 - 确保没有ClientID冲突

echo "Cleaning up any existing connections..."
# Kill any existing oven processes
pkill -f "oven" 2>/dev/null || true

# Kill any connections to IoT platform (macOS compatible)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS version
    lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | while read pid; do
        kill -9 $pid 2>/dev/null || true
    done
else
    # Linux version
    lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true
fi

# Wait a moment for cleanup
sleep 1

echo "Starting electric oven with OTA support..."
./oven