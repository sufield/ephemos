#!/bin/bash
set -e

echo "Setting up Ephemos demo..."

# Build the CLI first
echo "Building Ephemos CLI..."
cd ../..
go build -o ephemos cmd/ephemos/main.go

# Register echo-server
echo "Registering echo-server..."
./ephemos register --name echo-server --domain example.org

# Register echo-client
echo "Registering echo-client..."
./ephemos register --name echo-client --domain example.org

echo ""
echo "Demo setup completed!"
echo ""
echo "Services registered:"
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock
echo ""
echo "To run the demo:"
echo "  ./run-demo.sh"