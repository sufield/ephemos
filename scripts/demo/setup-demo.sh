#!/bin/bash

echo "Setting up Ephemos demo..."

# Build the CLI first
echo "Building Ephemos CLI..."
cd ../..
go build -o ephemos cmd/ephemos-cli/main.go

# Check for existing entries instead of deleting them
echo "Checking for existing SPIRE entries..."

# Get the UID that will run the services (the user who invoked sudo, not root)
if [ -n "$SUDO_UID" ]; then
    ACTUAL_UID=$SUDO_UID
    ACTUAL_USER=$SUDO_USER
else
    ACTUAL_UID=$(id -u)
    ACTUAL_USER=$(whoami)
fi

echo "Services will run as user: $ACTUAL_USER (UID: $ACTUAL_UID)"

# Register echo-server with correct UID (or use existing)
echo "Registering echo-server with unix:uid:$ACTUAL_UID selector..."
if sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>/dev/null; then
    echo "✓ Echo-server entry created successfully"
else
    echo "✓ Echo-server entry already exists, using existing entry"
fi

# Register echo-client with correct UID (or use existing)
echo "Registering echo-client with unix:uid:$ACTUAL_UID selector..."
if sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>/dev/null; then
    echo "✓ Echo-client entry created successfully"
else
    echo "✓ Echo-client entry already exists, using existing entry"
fi

echo ""
echo "Demo setup completed!"
echo ""
echo "Services registered:"
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock
echo ""
echo "To run the demo:"
echo "  ./run-demo.sh"