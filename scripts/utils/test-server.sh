#!/bin/bash
set -x

echo "Testing echo-server..."

# Kill any existing servers
pkill -f echo-server || true

# Run server in background
echo "Starting server with config: config/echo-server.yaml"
EPHEMOS_CONFIG=config/echo-server.yaml ./echo-server &
SERVER_PID=$!

echo "Server PID: $SERVER_PID"

# Wait a bit
sleep 2

# Check if server is running
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "Server is running"
    
    # Check what's listening
    echo "Checking ports:"
    ss -tlnp 2>/dev/null | grep -E "50051|$SERVER_PID" || echo "Not found on expected port"
    
    # Check process details
    echo "Process info:"
    ps -p $SERVER_PID -o pid,cmd
else
    echo "Server is NOT running - it crashed"
fi

# Try to see any output
echo "Attempting to connect..."
echo "test" | nc localhost 50051 2>&1 || echo "Connection failed"

# Kill server
kill $SERVER_PID 2>/dev/null || true

echo "Test complete"