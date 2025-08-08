#!/bin/bash
set -e

echo "Starting SPIRE services..."

# Generate upstream CA if it doesn't exist
if [ ! -f /opt/spire/data/upstream_ca.key ]; then
    echo "Generating upstream CA..."
    sudo openssl req -x509 -newkey rsa:4096 -keyout /opt/spire/data/upstream_ca.key \
        -out /opt/spire/data/upstream_ca.crt -days 365 -nodes \
        -subj "/C=US/ST=CA/L=SF/O=Example/CN=upstream-ca" 2>/dev/null
fi

# Start SPIRE server
echo "Starting SPIRE server..."
sudo systemctl start spire-server
sleep 3

# Check if server is running
if ! sudo systemctl is-active --quiet spire-server; then
    echo "Error: SPIRE server failed to start"
    sudo journalctl -u spire-server -n 20
    exit 1
fi

# Generate join token for agent
echo "Generating join token..."
TOKEN=$(sudo spire-server token generate -spiffeID spiffe://example.org/spire-agent -socketPath /tmp/spire-server/private/api.sock | awk '{print $2}')

# Configure agent with join token
echo "Configuring agent with join token..."
sudo spire-agent run -config /opt/spire/conf/agent.conf -joinToken $TOKEN &
AGENT_PID=$!
sleep 3

# Kill the temporary agent process
kill $AGENT_PID 2>/dev/null || true

# Start SPIRE agent via systemd
echo "Starting SPIRE agent..."
sudo systemctl start spire-agent
sleep 2

# Check if agent is running
if ! sudo systemctl is-active --quiet spire-agent; then
    echo "Error: SPIRE agent failed to start"
    sudo journalctl -u spire-agent -n 20
    exit 1
fi

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
echo "SPIRE Server Status:"
sudo systemctl status spire-server --no-pager | head -n 3
echo ""
echo "SPIRE Agent Status:"
sudo systemctl status spire-agent --no-pager | head -n 3
echo ""
echo "To register services, run:"
echo "  ./setup-demo.sh"