#!/bin/bash
set -e

SCRIPT_DIR="$(dirname "$0")"

# Clean up PIDs on exit
trap 'rm -f "$SCRIPT_DIR/spire-server.pid" "$SCRIPT_DIR/spire-agent.pid" || true' EXIT

echo "Starting SPIRE services..."

# Kill any existing SPIRE processes to free up ports
echo "Cleaning up any existing SPIRE processes..."
sudo pkill -f spire-server 2>/dev/null || true
sudo pkill -f spire-agent 2>/dev/null || true
sudo pkill -f echo-server 2>/dev/null || true
pkill -f echo-server 2>/dev/null || true
sleep 2

# Generate upstream CA if it doesn't exist
if [ ! -f /opt/spire/data/upstream_ca.key ]; then
    echo "Generating upstream CA..."
    sudo openssl req -x509 -newkey rsa:4096 -keyout /opt/spire/data/upstream_ca.key \
        -out /opt/spire/data/upstream_ca.crt -days 365 -nodes \
        -subj "/C=US/ST=CA/L=SF/O=Example/CN=upstream-ca" 2>/dev/null
fi

# Start SPIRE server and capture logs
echo "Starting SPIRE server..."
sudo systemctl stop spire-server 2>/dev/null || true
sudo spire-server run -config /opt/spire/conf/server.conf > "$SCRIPT_DIR/spire-server.log" 2>&1 &
SERVER_PID=$!
echo "SPIRE Server PID: $SERVER_PID"
sleep 5  # Give more time to start

# Check if server process exists (use pgrep to find the actual spire-server process)
if ! pgrep -f "spire-server.*run" > /dev/null; then
    echo "Error: SPIRE server process not found. Log:"
    cat "$SCRIPT_DIR/spire-server.log"
    exit 1
fi

# Show startup log
echo "SPIRE Server startup log:"
cat "$SCRIPT_DIR/spire-server.log" | head -5 | sed 's/^/[SPIRE-SERVER] /'

# Run health check and append to log
if ! sudo spire-server healthcheck -socketPath /tmp/spire-server/private/api.sock >> "$SCRIPT_DIR/spire-server.log" 2>&1; then
    echo "Error: SPIRE server health check failed"
    cat "$SCRIPT_DIR/spire-server.log"
    exit 1
fi
echo "✓ SPIRE Server health check passed"

# Generate join token for agent
echo "Generating join token..."
TOKEN=$(sudo spire-server token generate -spiffeID spiffe://example.org/spire-agent -socketPath /tmp/spire-server/private/api.sock | awk '{print $2}')

# Configure agent with join token
echo "Configuring agent with join token..."
sudo spire-agent run -config /opt/spire/conf/agent.conf -joinToken $TOKEN > /dev/null 2>&1 &
TEMP_AGENT_PID=$!
sleep 3

# Kill the temporary agent process
kill $TEMP_AGENT_PID 2>/dev/null || true

# Start SPIRE agent and capture logs
echo "Starting SPIRE agent..."
sudo systemctl stop spire-agent 2>/dev/null || true
sudo spire-agent run -config /opt/spire/conf/agent.conf > "$SCRIPT_DIR/spire-agent.log" 2>&1 &
AGENT_PID=$!
echo "SPIRE Agent PID: $AGENT_PID"
sleep 3  # Give more time to start

# Check if agent process exists (use pgrep to find the actual spire-agent process)
if ! pgrep -f "spire-agent.*run" > /dev/null; then
    echo "Error: SPIRE agent process not found. Log:"
    cat "$SCRIPT_DIR/spire-agent.log"
    exit 1
fi

# Show startup log
echo "SPIRE Agent startup log:"
cat "$SCRIPT_DIR/spire-agent.log" | head -5 | sed 's/^/[SPIRE-AGENT] /'

# Run health check and append to log
if ! sudo spire-agent healthcheck -socketPath /tmp/spire-agent/public/api.sock >> "$SCRIPT_DIR/spire-agent.log" 2>&1; then
    echo "Error: SPIRE agent health check failed"
    cat "$SCRIPT_DIR/spire-agent.log"
    exit 1
fi
echo "✓ SPIRE Agent health check passed"

# Configure socket permissions for non-root access
echo "Configuring socket permissions..."
# Make socket accessible to all users for CI compatibility
sudo chmod 777 /tmp/spire-agent/public/api.sock
echo "✓ Socket permissions configured for all users (CI mode)"

# Create initial node entry
echo "Creating node entry..."
sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/spire-agent \
    -selector unix:uid:0 \
    -node 2>/dev/null || true

echo ""
echo "SPIRE services started successfully!"
echo ""
echo "SPIRE Server PID: $SERVER_PID (running in background)"
echo "SPIRE Agent PID: $AGENT_PID (running in background)"
echo ""
echo "Log files created:"
echo "  - spire-server.log"
echo "  - spire-agent.log"
echo ""
echo "To register services, run:"
echo "  ./setup-demo.sh"

# Export PIDs for cleanup later
echo "$SERVER_PID" > "$SCRIPT_DIR/spire-server.pid"
echo "$AGENT_PID" > "$SCRIPT_DIR/spire-agent.pid"