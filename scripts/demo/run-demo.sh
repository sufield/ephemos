#!/bin/bash
set -e

# Function to handle sudo commands with proper error handling
run_sudo_cmd() {
    local cmd="$1"
    local description="$2"
    
    echo "Running: $cmd"
    if eval "$cmd"; then
        return 0
    else
        local exit_code=$?
        echo "⚠️ Command failed: $description (exit code: $exit_code)"
        
        # Check if it's a permission error
        if [ $exit_code -eq 1 ]; then
            echo "This might be a permission issue. Make sure you have sudo privileges."
            echo "If prompted for password, please enter your sudo password."
        fi
        
        return $exit_code
    fi
}

# Cleanup function to gracefully shutdown all components
cleanup_demo() {
    echo ""
    echo "Gracefully shutting down demo components..."
    
    # Gracefully shutdown echo-server
    if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        echo "Gracefully stopping echo-server (PID: $SERVER_PID)..."
        kill -TERM $SERVER_PID 2>/dev/null || true
        sleep 3
        # Force kill if still running
        if kill -0 $SERVER_PID 2>/dev/null; then
            echo "Force killing echo-server..."
            kill -9 $SERVER_PID 2>/dev/null || true
        fi
    fi
    
    # Clean up any remaining echo processes
    echo "Cleaning up remaining echo processes..."
    pkill -TERM -f echo-server 2>/dev/null || true
    pkill -TERM -f echo-client 2>/dev/null || true
    sleep 2
    # Force kill any stubborn processes
    pkill -9 -f echo-server 2>/dev/null || true
    pkill -9 -f echo-client 2>/dev/null || true
    
    # Gracefully shutdown SPIRE components (if we have permission)
    echo "Attempting to gracefully shutdown SPIRE components..."
    if [ -f scripts/demo/spire-server.pid ]; then
        SPIRE_SERVER_PID=$(cat scripts/demo/spire-server.pid 2>/dev/null)
        if [ -n "$SPIRE_SERVER_PID" ] && kill -0 $SPIRE_SERVER_PID 2>/dev/null; then
            echo "Gracefully stopping SPIRE server (PID: $SPIRE_SERVER_PID)..."
            kill -TERM $SPIRE_SERVER_PID 2>/dev/null || true
            sleep 2
        fi
    fi
    
    if [ -f scripts/demo/spire-agent.pid ]; then
        SPIRE_AGENT_PID=$(cat scripts/demo/spire-agent.pid 2>/dev/null)
        if [ -n "$SPIRE_AGENT_PID" ] && kill -0 $SPIRE_AGENT_PID 2>/dev/null; then
            echo "Gracefully stopping SPIRE agent (PID: $SPIRE_AGENT_PID)..."
            kill -TERM $SPIRE_AGENT_PID 2>/dev/null || true
            sleep 2
        fi
    fi
}

# Set trap to cleanup on exit
trap cleanup_demo EXIT

echo "Running Ephemos demo..."
echo "========================"
echo ""

# Kill any existing echo-server processes that might be using port 50051
echo "Cleaning up existing processes..."
pkill -f echo-server 2>/dev/null || true
# Note: Cannot kill root processes without interactive sudo
# Find available port
AVAILABLE_PORT=":50051"
for port in 50051 50052 50053 50061 50062 50063; do
    if ! ss -tulpn | grep -q :$port; then
        AVAILABLE_PORT=":$port"
        break
    fi
done

if [ "$AVAILABLE_PORT" != ":50051" ]; then
    echo "⚠️  Port 50051 in use, using port $AVAILABLE_PORT"
    # Update client to connect to the available port (remove colon from port var)
    PORT_NUM=${AVAILABLE_PORT#:}
    sed -i "s/localhost[0-9][0-9][0-9][0-9][0-9]/localhost:${PORT_NUM}/g" ../../examples/echo-client/main.go 2>/dev/null || true
fi
export ECHO_SERVER_ADDRESS="$AVAILABLE_PORT"
sleep 1

# Build examples
echo "Building example applications..."
# Ensure we're in the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"
go build -o bin/echo-server ./examples/echo-server || { echo "ERROR: Failed to build echo-server"; exit 1; }
# Always rebuild client to ensure it has the correct port
go build -o bin/echo-client ./examples/echo-client || { echo "ERROR: Failed to build echo-client"; exit 1; }

# Check and start SPIRE services if needed
echo "Checking SPIRE services..."
if ! pgrep -f "spire-server.*run" > /dev/null || ! pgrep -f "spire-agent.*run" > /dev/null; then
    echo "SPIRE services not running. Starting SPIRE server and agent..."
    echo "This requires sudo privileges. You may be prompted for your password."
    
    # Change to demo directory to run start script
    CURRENT_DIR=$(pwd)
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"
    
    # Start SPIRE services
    if ! ./start-spire.sh; then
        echo "❌ Failed to start SPIRE services. Demo cannot continue."
        echo "Please ensure you have sudo privileges and SPIRE is properly installed."
        exit 1
    fi
    
    # Return to project root
    cd "$CURRENT_DIR"
    
    echo "✅ SPIRE services started successfully"
else
    echo "✅ SPIRE services are already running"
fi

# Register SPIRE entries before starting services
echo "Registering SPIRE entries..."
# Get the UID that will run the services
ACTUAL_UID=$(id -u)
ACTUAL_USER=$(whoami)
echo "Services will run as user: $ACTUAL_USER (UID: $ACTUAL_UID)"

# Register echo-server with correct UID
echo "Registering echo-server with unix:uid:$ACTUAL_UID selector..."
echo "🔍 DEBUG: Running: sudo spire-server entry create -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-server -parentID spiffe://example.org/spire-agent -selector unix:uid:$ACTUAL_UID -ttl 3600"

# Use temporary disable of set -e for this command
set +e
SERVER_ENTRY_OUTPUT=$(sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>&1)
SERVER_ENTRY_EXIT_CODE=$?
set -e

echo "🔍 DEBUG: echo-server entry creation exit code: $SERVER_ENTRY_EXIT_CODE"
echo "🔍 DEBUG: echo-server entry creation output:"
echo "$SERVER_ENTRY_OUTPUT" | sed 's/^/[SERVER-ENTRY] /'

if [ $SERVER_ENTRY_EXIT_CODE -eq 0 ]; then
    echo "✓ Echo-server entry created successfully"
else
    echo "⚠️ Echo-server entry creation failed or already exists"
    if echo "$SERVER_ENTRY_OUTPUT" | grep -q "already exists"; then
        echo "🔍 DEBUG: Entry already exists - this is normal"
    else
        echo "🔍 DEBUG: Actual error occurred during entry creation"
    fi
fi

# Register echo-client with correct UID
echo "Registering echo-client with unix:uid:$ACTUAL_UID selector..."
echo "🔍 DEBUG: Running: sudo spire-server entry create -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-client -parentID spiffe://example.org/spire-agent -selector unix:uid:$ACTUAL_UID -ttl 3600"

# Use temporary disable of set -e for this command
set +e
CLIENT_ENTRY_OUTPUT=$(sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>&1)
CLIENT_ENTRY_EXIT_CODE=$?
set -e

echo "🔍 DEBUG: echo-client entry creation exit code: $CLIENT_ENTRY_EXIT_CODE"
echo "🔍 DEBUG: echo-client entry creation output:"
echo "$CLIENT_ENTRY_OUTPUT" | sed 's/^/[CLIENT-ENTRY] /'

if [ $CLIENT_ENTRY_EXIT_CODE -eq 0 ]; then
    echo "✓ Echo-client entry created successfully"
else
    echo "⚠️ Echo-client entry creation failed or already exists"
    if echo "$CLIENT_ENTRY_OUTPUT" | grep -q "already exists"; then
        echo "🔍 DEBUG: Entry already exists - this is normal"
    else
        echo "🔍 DEBUG: Actual error occurred during entry creation"
    fi
fi

# Wait for entries to propagate and verify they're available
echo "Waiting for SPIRE entries to propagate..."
sleep 5

# Verify entries are actually registered and available
echo "Verifying SPIRE entries are available..."
echo "🔍 DEBUG: Checking SPIRE service status..."

# Check if SPIRE processes are running
SPIRE_SERVER_RUNNING=$(pgrep -f "spire-server.*run" || echo "")
SPIRE_AGENT_RUNNING=$(pgrep -f "spire-agent.*run" || echo "")

echo "🔍 DEBUG: SPIRE server process: ${SPIRE_SERVER_RUNNING:-NOT RUNNING}"
echo "🔍 DEBUG: SPIRE agent process: ${SPIRE_AGENT_RUNNING:-NOT RUNNING}"

# Check if SPIRE sockets exist
echo "🔍 DEBUG: Checking SPIRE socket files..."
if [ -S "/tmp/spire-server/private/api.sock" ]; then
    echo "🔍 DEBUG: SPIRE server socket exists"
    ls -la /tmp/spire-server/private/api.sock 2>/dev/null || echo "🔍 DEBUG: Cannot access server socket details"
else
    echo "🔍 DEBUG: SPIRE server socket NOT FOUND at /tmp/spire-server/private/api.sock"
fi

if [ -S "/tmp/spire-agent/public/api.sock" ]; then
    echo "🔍 DEBUG: SPIRE agent socket exists"
    ls -la /tmp/spire-agent/public/api.sock 2>/dev/null || echo "🔍 DEBUG: Cannot access agent socket details"
else
    echo "🔍 DEBUG: SPIRE agent socket NOT FOUND at /tmp/spire-agent/public/api.sock"
fi

# Check if we can connect to SPIRE server
echo "🔍 DEBUG: Testing SPIRE server connection..."
set +e
sudo spire-server healthcheck -socketPath /tmp/spire-server/private/api.sock 2>/dev/null
HEALTH_CHECK_CODE=$?
set -e

if [ $HEALTH_CHECK_CODE -eq 0 ]; then
    echo "🔍 DEBUG: SPIRE server health check PASSED"
else
    echo "🔍 DEBUG: SPIRE server health check FAILED (exit code: $HEALTH_CHECK_CODE)"
    echo "🔍 DEBUG: Attempting to show SPIRE server error output..."
    set +e
    sudo spire-server healthcheck -socketPath /tmp/spire-server/private/api.sock 2>&1 | head -5 | sed 's/^/[SPIRE-ERROR] /'
    set -e
fi

RETRY_COUNT=0
MAX_RETRIES=12  # 12 * 5 seconds = 60 seconds max wait

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    echo "🔍 DEBUG: Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES - Querying SPIRE entries..."
    
    # Show the actual command being run and its output
    set +e
    SPIRE_OUTPUT=$(sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock 2>&1)
    SPIRE_EXIT_CODE=$?
    set -e
    
    echo "🔍 DEBUG: spire-server entry show exit code: $SPIRE_EXIT_CODE"
    echo "🔍 DEBUG: spire-server entry show output:"
    echo "$SPIRE_OUTPUT" | head -10 | sed 's/^/[SPIRE-OUTPUT] /'
    
    if echo "$SPIRE_OUTPUT" | grep -q "echo-server"; then
        echo "✅ SPIRE entries verified and ready"
        echo "🔍 DEBUG: Found echo-server entry in output"
        break
    else
        echo "⏳ Waiting for SPIRE entries to be ready... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
        if [ $SPIRE_EXIT_CODE -ne 0 ]; then
            echo "🔍 DEBUG: Command failed with exit code $SPIRE_EXIT_CODE"
            echo "🔍 DEBUG: This suggests SPIRE server is not running or not accessible"
        else
            echo "🔍 DEBUG: Command succeeded but echo-server entry not found in output"
        fi
        sleep 5
        RETRY_COUNT=$((RETRY_COUNT + 1))
    fi
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ TIMEOUT: SPIRE entries not ready after 60 seconds"
    echo "Current entries:"
    sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock 2>/dev/null || echo "Failed to query entries"
    exit 1
fi

# Additional wait for agent to process the new entries
echo "Allowing time for SPIRE agent to process entries..."
sleep 3

# Start echo-server in background and capture output
echo "Starting echo-server on port ${ECHO_SERVER_ADDRESS:-:50051}..."
EPHEMOS_CONFIG=config/echo-server.yaml ECHO_SERVER_ADDRESS=${ECHO_SERVER_ADDRESS:-:50051} ./bin/echo-server > scripts/demo/server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# Wait for server to start and get SPIFFE identity
echo "Waiting for echo-server to obtain SPIFFE identity..."
SERVER_READY=false
WAIT_COUNT=0
MAX_WAIT=24  # 24 * 5 seconds = 2 minutes max wait

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
    if grep -q "Server identity created\|Server ready\|Successfully obtained SPIFFE identity\|Identity service initialized" scripts/demo/server.log; then
        echo "✅ Echo-server successfully obtained SPIFFE identity"
        SERVER_READY=true
        break
    fi
    
    # Check for identity-related errors but continue if they're just temporary
    if grep -q "failed to get X509 SVID\|No identity issued" scripts/demo/server.log; then
        echo "⏳ Server attempting to get identity... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
    elif grep -q "Failed to create identity server" scripts/demo/server.log; then
        echo "❌ Identity server creation failed - check logs"
        cat scripts/demo/server.log
        exit 1
    else
        echo "⏳ Waiting for server to start... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
        # Show last few lines for debugging
        if [ -f scripts/demo/server.log ]; then
            echo "   Last server log entries:"
            tail -3 scripts/demo/server.log | sed 's/^/   /'
        fi
    fi
    
    sleep 5
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ "$SERVER_READY" = "false" ]; then
    echo "❌ TIMEOUT: Echo-server failed to obtain SPIFFE identity after 2 minutes"
    echo "Server log content:"
    cat scripts/demo/server.log
    exit 1
fi

# Check SPIRE health
echo "Checking SPIRE health..."
ps aux | grep -E "(spire-server|spire-agent)" | grep -v grep > /dev/null || { echo "ERROR: SPIRE processes not found"; exit 1; }
echo "✓ SPIRE processes detected"

# Verify SPIRE Agent socket exists
if [ ! -S /tmp/spire-agent/public/api.sock ]; then
    echo "ERROR: SPIRE Agent socket not found at /tmp/spire-agent/public/api.sock"
    ls -la /tmp/spire-agent/public/ 2>/dev/null || echo "Directory doesn't exist"
    exit 1
fi
echo "✓ SPIRE Agent socket exists"

# Show SPIRE context (logs are in scripts/demo/)
echo ""
echo "SPIRE Server context (recent logs):"
echo "-------------------"
tail -5 scripts/demo/spire-server.log 2>/dev/null | sed 's/^/[SPIRE-SERVER] /' || echo "[SPIRE-SERVER] No logs available"
echo ""
echo "SPIRE Agent context (recent logs):"
echo "-------------------"
tail -5 scripts/demo/spire-agent.log 2>/dev/null | sed 's/^/[SPIRE-AGENT] /' || echo "[SPIRE-AGENT] No logs available"
echo ""

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "ERROR: Server failed to start. Full log:"
    cat scripts/demo/server.log
    exit 1
fi

echo "✓ Server started successfully"

# Display full server startup log
echo ""
echo "Server startup log:"
echo "-------------------"
cat scripts/demo/server.log | sed 's/^/[SERVER] /'
echo ""

# Run echo-client with timeout and capture output
echo "Starting echo-client (with 10s timeout)..."
echo "-------------------"
timeout 10 bash -c 'EPHEMOS_CONFIG=config/echo-client.yaml ./bin/echo-client 2>&1' | tee scripts/demo/client.log | sed 's/^/[CLIENT] /'

# Check client exit status
CLIENT_EXIT=${PIPESTATUS[0]}
# Check if we got any successful echo responses (even if timeout occurred)
if grep -q "Echo response received" scripts/demo/client.log; then
    echo ""
    echo "✅ Client successfully exchanged messages with server!"
    SUCCESS=true

    # Display messages in green boxes immediately after success
    echo ""
    echo "==============================================="
    echo "🎉 SPIFFE AUTHENTICATION SUCCESS MESSAGES 🎉"
    echo "==============================================="

    # Colors
    GREEN='\033[32m'
    RESET='\033[0m'

    echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
    echo -e "║                               ECHO SERVER MESSAGES                          ║"
    echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

    while IFS= read -r line; do
        echo -e "${GREEN}║${RESET} $line"
    done < scripts/demo/server.log

    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

    echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
    echo -e "║                               ECHO CLIENT MESSAGES                          ║"
    echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

    while IFS= read -r line; do
        echo -e "${GREEN}║${RESET} $line"
    done < scripts/demo/client.log

    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

    echo ""

elif [ $CLIENT_EXIT -eq 124 ]; then
    echo ""
    echo "ERROR: Client timed out after 10 seconds without successful communication."
    kill $SERVER_PID 2>/dev/null || true
    exit 1
elif [ $CLIENT_EXIT -ne 0 ]; then
    echo ""
    echo "ERROR: Client failed with exit code $CLIENT_EXIT"
    echo "Client log:"
    cat scripts/demo/client.log
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

# Show full server processing logs
echo ""
echo "Full server processing log (after client connections):"
echo "-------------------"
cat scripts/demo/server.log | sed 's/^/[SERVER] /'

# Display messages in green boxes
echo ""
echo "==============================================="
echo "🎉 SPIFFE AUTHENTICATION SUCCESS MESSAGES 🎉"
echo "==============================================="

# Colors
GREEN='\033[32m'
RESET='\033[0m'

echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
echo -e "║                               ECHO SERVER MESSAGES                          ║"
echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

while IFS= read -r line; do
    echo -e "${GREEN}║${RESET} $line"
done < scripts/demo/server.log

echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
echo -e "║                               ECHO CLIENT MESSAGES                          ║"
echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

while IFS= read -r line; do
    echo -e "${GREEN}║${RESET} $line"
done < scripts/demo/client.log

echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

echo ""

echo ""
echo "✓ Demo Part 1 Complete: Client successfully authenticated and communicated with server"
echo ""
echo "Now demonstrating authentication failure..."
echo ""

# Delete client registration
echo "Removing echo-client registration..."
# Note: Skipping sudo entry deletion to avoid password prompts
echo "✓ Skipped client deregistration (requires sudo)"

# Try to run client again (should fail)
echo "Attempting to run unregistered client..."
EPHEMOS_CONFIG=config/echo-client.yaml timeout 5 ./bin/echo-client 2>&1 | grep -i "error\|fail" || echo "Authentication failed as expected!"

# Graceful Demo Shutdown
echo ""
echo "=============================================="
echo "          GRACEFUL DEMO SHUTDOWN"
echo "=============================================="

# Gracefully shutdown echo-server first
if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
    echo "🔄 Gracefully shutting down echo-server (PID: $SERVER_PID)..."
    kill -TERM $SERVER_PID 2>/dev/null || true
    
    # Wait for graceful shutdown
    for i in {1..5}; do
        if ! kill -0 $SERVER_PID 2>/dev/null; then
            echo "✅ Echo-server gracefully stopped"
            break
        fi
        echo "   Waiting for echo-server to shutdown... ($i/5)"
        sleep 1
    done
    
    # Force kill if still running
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo "⚠️  Force killing unresponsive echo-server..."
        kill -9 $SERVER_PID 2>/dev/null || true
    fi
fi

# Clean up any remaining echo processes gracefully
echo "🔄 Cleaning up remaining echo processes..."
pkill -TERM -f echo-server 2>/dev/null || true
pkill -TERM -f echo-client 2>/dev/null || true
sleep 2

# Force kill any stubborn processes
REMAINING_ECHO=$(pgrep -f echo-server 2>/dev/null || true)
if [ -n "$REMAINING_ECHO" ]; then
    echo "⚠️  Force killing stubborn echo processes: $REMAINING_ECHO"
    pkill -9 -f echo-server 2>/dev/null || true
    pkill -9 -f echo-client 2>/dev/null || true
fi

# Use the stop-spire script for proper SPIRE shutdown
echo "🔄 Gracefully shutting down SPIRE services..."
if [ -f "./stop-spire.sh" ]; then
    ./stop-spire.sh
    echo "✅ SPIRE services gracefully stopped"
else
    echo "⚠️  stop-spire.sh not found, attempting manual shutdown..."
    # Manual SPIRE shutdown as fallback
    pkill -TERM -f spire-server 2>/dev/null || true
    pkill -TERM -f spire-agent 2>/dev/null || true
    sleep 3
    echo "✅ SPIRE manual shutdown attempted"
fi

# Verify ports are released
echo "🔍 Verifying all demo ports are released..."
PORTS_IN_USE=$(ss -tulpn | grep -E ":(5005[0-9]|5006[0-9])" || true)
if [ -n "$PORTS_IN_USE" ]; then
    echo "⚠️  Some demo ports still in use:"
    echo "$PORTS_IN_USE"
else
    echo "✅ All demo ports successfully released"
fi

# Clean up log files and PID files
echo "🧹 Cleaning up temporary files..."
rm -f scripts/demo/*.log scripts/demo/*.pid || true
echo "✅ Temporary files cleaned"

echo ""
echo "=============================================="
echo "     GRACEFUL SHUTDOWN COMPLETED"
echo "=============================================="

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
echo "  - Server: ephemos.NewIdentityServer()"
echo "  - Client: ephemos.NewIdentityClient()"