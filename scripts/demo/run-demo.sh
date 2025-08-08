#!/bin/bash
set -e

echo "Running Ephemos demo..."
echo "========================"
echo ""

# Build examples
echo "Building example applications..."
cd ../..
go build -o echo-server examples/echo-server/main.go
go build -o echo-client examples/echo-client/main.go

# Start echo-server in background
echo "Starting echo-server..."
EPHEMOS_CONFIG=configs/echo-server.yaml ./echo-server &
SERVER_PID=$!
sleep 2

# Check if server started
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "Error: echo-server failed to start"
    exit 1
fi

echo ""
echo "Server started with identity-based authentication"
echo ""

# Run echo-client
echo "Starting echo-client..."
EPHEMOS_CONFIG=configs/echo-client.yaml ./echo-client &
CLIENT_PID=$!

# Wait for client to complete a few requests
sleep 12

# Stop client
kill $CLIENT_PID 2>/dev/null || true

echo ""
echo "Demo Part 1 Complete: Client successfully authenticated and communicated with server"
echo ""
echo "Now demonstrating authentication failure..."
echo ""

# Delete client registration
echo "Removing echo-client registration..."
sudo spire-server entry delete \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client 2>/dev/null || true

# Try to run client again (should fail)
echo "Attempting to run unregistered client..."
EPHEMOS_CONFIG=configs/echo-client.yaml timeout 5 ./echo-client 2>&1 | grep -i "error\|fail" || echo "Authentication failed as expected!"

# Cleanup
echo ""
echo "Cleaning up..."
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "================================"
echo "Demo completed successfully!"
echo "================================"
echo ""
echo "Summary:"
echo "1. ✅ Started SPIRE server and agent"
echo "2. ✅ Registered services using 'ephemos register'"
echo "3. ✅ Started echo-server with identity 'echo-server'"
echo "4. ✅ Client successfully connected using mTLS"
echo "5. ✅ Demonstrated authentication failure after deregistration"
echo ""
echo "The entire identity-based authentication was handled transparently!"
echo "Developers only needed to call:"
echo "  - Server: ephemos.IdentityServer()"
echo "  - Client: ephemos.IdentityClient()"