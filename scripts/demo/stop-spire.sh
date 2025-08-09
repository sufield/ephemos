#!/bin/bash

echo "Stopping SPIRE services..."

SCRIPT_DIR="$(dirname "$0")"

# Read PIDs from files
if [ -f "$SCRIPT_DIR/spire-server.pid" ]; then
    SERVER_PID=$(cat "$SCRIPT_DIR/spire-server.pid")
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo "Stopping SPIRE Server (PID: $SERVER_PID)..."
        sudo kill $SERVER_PID
    fi
    rm -f "$SCRIPT_DIR/spire-server.pid"
fi

if [ -f "$SCRIPT_DIR/spire-agent.pid" ]; then
    AGENT_PID=$(cat "$SCRIPT_DIR/spire-agent.pid")
    if kill -0 $AGENT_PID 2>/dev/null; then
        echo "Stopping SPIRE Agent (PID: $AGENT_PID)..."
        sudo kill $AGENT_PID
    fi
    rm -f "$SCRIPT_DIR/spire-agent.pid"
fi

# Also try systemctl stop as backup
sudo systemctl stop spire-server 2>/dev/null || true
sudo systemctl stop spire-agent 2>/dev/null || true

echo "SPIRE services stopped."