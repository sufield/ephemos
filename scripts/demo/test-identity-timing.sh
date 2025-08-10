#!/bin/bash
# Test script for identity timing improvements

set -e

echo "=== IDENTITY TIMING TEST ==="
echo "Testing the timing improvements for SPIRE identity provisioning"
echo ""

# Cleanup any previous test
pkill -f echo-server || true
rm -f scripts/demo/server.log

echo "1. Verifying SPIRE is running..."
if ! ps aux | grep -q "[s]pire-server"; then
    echo "❌ SPIRE server not running. Please start SPIRE first."
    exit 1
fi

if ! ps aux | grep -q "[s]pire-agent"; then
    echo "❌ SPIRE agent not running. Please start SPIRE first."
    exit 1
fi

echo "✅ SPIRE services are running"

echo ""
echo "2. Testing SPIRE entry registration..."

# Get the UID that will run the services
ACTUAL_UID=$(id -u)
ACTUAL_USER=$(whoami)
echo "Services will run as user: $ACTUAL_USER (UID: $ACTUAL_UID)"

# Register echo-server with correct UID
echo "Registering echo-server with unix:uid:$ACTUAL_UID selector..."
if sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>/dev/null; then
    echo "✅ Echo-server entry created successfully"
else
    echo "✅ Echo-server entry already exists or creation failed, continuing..."
fi

echo ""
echo "3. Waiting for SPIRE entries to propagate..."

# Verify entries are actually registered and available
RETRY_COUNT=0
MAX_RETRIES=6  # 6 * 5 seconds = 30 seconds max wait

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock 2>/dev/null | grep -q "echo-server"; then
        echo "✅ SPIRE entries verified and ready"
        break
    else
        echo "⏳ Waiting for SPIRE entries to be ready... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
        sleep 5
        RETRY_COUNT=$((RETRY_COUNT + 1))
    fi
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ TIMEOUT: SPIRE entries not ready after 30 seconds"
    exit 1
fi

echo ""
echo "4. Testing echo-server identity acquisition..."

# Start echo-server in background
cd ../..
echo "Starting echo-server..."
EPHEMOS_CONFIG=config/echo-server.yaml ECHO_SERVER_ADDRESS=:50099 ./bin/echo-server > scripts/demo/server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# Wait for server to start and get SPIFFE identity
echo "Waiting for echo-server to obtain SPIFFE identity..."
SERVER_READY=false
WAIT_COUNT=0
MAX_WAIT=12  # 12 * 5 seconds = 1 minute max wait

while [ $WAIT_COUNT -lt $MAX_WAIT ] && [ "$SERVER_READY" = "false" ]; do
    if [ ! -f scripts/demo/server.log ]; then
        echo "⏳ Waiting for server log file... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
        sleep 5
        WAIT_COUNT=$((WAIT_COUNT + 1))
        continue
    fi
    
    # Check if server is still running
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "❌ Echo-server process died. Check server log:"
        cat scripts/demo/server.log
        exit 1
    fi
    
    # Check for successful identity creation
    if grep -q "Server identity created\|Server ready" scripts/demo/server.log; then
        echo "✅ Echo-server successfully obtained SPIFFE identity!"
        SERVER_READY=true
        break
    fi
    
    # Check for identity-related errors
    if grep -q "failed to get X509 SVID\|No identity issued" scripts/demo/server.log; then
        echo "⏳ Server attempting to get identity... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
    elif grep -q "Failed to create identity server" scripts/demo/server.log; then
        echo "❌ Identity server creation failed"
        cat scripts/demo/server.log
        exit 1
    else
        echo "⏳ Waiting for server to start... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
    fi
    
    sleep 5
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ "$SERVER_READY" = "false" ]; then
    echo "❌ TIMEOUT: Echo-server failed to obtain SPIFFE identity after 1 minute"
    echo ""
    echo "Server log content:"
    echo "==================="
    cat scripts/demo/server.log
    
    echo ""
    echo "Recent SPIRE Agent log entries:"
    echo "=============================="
    tail -10 scripts/demo/spire-agent.log 2>/dev/null || echo "No agent log available"
    
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

# Show success details
echo ""
echo "=========================================="
echo "✅ IDENTITY TIMING TEST SUCCESSFUL!"
echo "=========================================="
echo ""
echo "Server log summary:"
echo "------------------"
grep -E "(Server identity created|Server ready|Service registered)" scripts/demo/server.log || echo "No summary entries found"

# Cleanup
echo ""
echo "5. Cleaning up test..."
kill $SERVER_PID 2>/dev/null || true
sleep 2
rm -f scripts/demo/server.log

echo "✅ Test completed successfully!"
echo ""
echo "The identity timing improvements are working correctly."
echo "The server can now successfully obtain its SPIFFE identity"
echo "after the SPIRE entries have been properly registered and propagated."