#!/bin/bash
set -e

echo "Testing SPIFFE registration for demo services..."
echo "Current user: $(whoami), UID: $(id -u)"
echo ""

# Clean up old entries
echo "Cleaning up old entries..."
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-server 2>/dev/null || true
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-client 2>/dev/null || true

# Get the UID of the user who invoked sudo (not root)
if [ -n "$SUDO_UID" ]; then
    ACTUAL_UID=$SUDO_UID
    ACTUAL_USER=$SUDO_USER
else
    ACTUAL_UID=$(id -u)
    ACTUAL_USER=$(whoami)
fi

echo "Services will run as user: $ACTUAL_USER (UID: $ACTUAL_UID)"
echo "Registering with UID: $ACTUAL_UID"

# Register echo-server
echo "Registering echo-server..."
sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600

# Register echo-client
echo "Registering echo-client..."
sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600

echo ""
echo "Registration complete. Current entries:"
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock

echo ""
echo "Testing identity fetch..."
echo "Starting echo-server for 3 seconds to test SVID fetch..."
timeout 3 bash -c 'EPHEMOS_CONFIG=config/echo-server.yaml ./bin/echo-server 2>&1' || true