#!/bin/bash

echo "Starting oven process..."
./oven &
PID=$!

echo "Oven PID: $PID"
echo "Waiting 3 seconds for startup..."
sleep 3

echo "Sending SIGINT (Ctrl+C) to process..."
kill -INT $PID

echo "Waiting for process to exit..."
COUNTER=0
while kill -0 $PID 2>/dev/null; do
    echo "Process still running... ($COUNTER seconds)"
    sleep 1
    COUNTER=$((COUNTER + 1))
    if [ $COUNTER -ge 10 ]; then
        echo "Process did not exit after 10 seconds, force killing..."
        kill -9 $PID
        exit 1
    fi
done

echo "Process exited successfully!"
exit 0