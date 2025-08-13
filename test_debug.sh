#!/bin/bash
cd examples/framework/simple
./oven 2>&1 | head -100 &
PID=$!
sleep 2
kill -INT $PID
sleep 1
ps aux | grep $PID | grep -v grep