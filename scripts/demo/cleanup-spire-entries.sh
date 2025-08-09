#!/bin/bash

echo "Cleaning up old SPIRE entries..."

# Delete all echo-server entries
echo "Removing echo-server entries..."
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-server 2>/dev/null || true

# Delete all echo-client entries  
echo "Removing echo-client entries..."
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-client 2>/dev/null || true

echo "Cleanup complete. Run setup-demo.sh to re-register with correct selectors."