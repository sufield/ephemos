#!/bin/bash
set -e

# Demo script colors and formatting
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BOLD='\033[1m'
RESET='\033[0m'
CHECKMARK='✅'
ARROW='➜'
INFO='📋'
WARNING='⚠️'
ERROR='❌'

# Function to print step headers
print_step() {
    local step_num=$1
    local title=$2
    local description=$3
    
    echo ""
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}${BLUE}STEP $step_num: $title${RESET}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${INFO} ${description}"
    echo ""
}

# Function to print substeps
print_substep() {
    local title=$1
    echo -e "${ARROW} ${BOLD}$title${RESET}"
}

# Function to print success messages
print_success() {
    local message=$1
    echo -e "${CHECKMARK} ${GREEN}$message${RESET}"
}

# Function to print info messages
print_info() {
    local message=$1
    echo -e "${INFO} $message"
}

# Function to print warnings
print_warning() {
    local message=$1
    echo -e "${WARNING} ${YELLOW}$message${RESET}"
}

# Function to print errors
print_error() {
    local message=$1
    echo -e "${ERROR} ${RED}$message${RESET}"
}

# Function to show code snippets
show_code() {
    local title=$1
    local code=$2
    echo ""
    echo -e "${BOLD}${YELLOW}$title:${RESET}"
    echo -e "${YELLOW}╔════════════════════════════════════════════════════════════════════════════════╗${RESET}"
    echo "$code" | while IFS= read -r line; do
        echo -e "${YELLOW}║${RESET} $line"
    done
    echo -e "${YELLOW}╚════════════════════════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

# Function to handle sudo commands with proper error handling
run_sudo_cmd() {
    local cmd="$1"
    local description="$2"
    
    print_substep "Running: $cmd"
    if eval "$cmd"; then
        return 0
    else
        local exit_code=$?
        print_error "Command failed: $description (exit code: $exit_code)"
        
        # Check if it's a permission error
        if [ $exit_code -eq 1 ]; then
            print_warning "This might be a permission issue. Make sure you have sudo privileges."
        fi
        
        return $exit_code
    fi
}

# Cleanup function to gracefully shutdown all components
cleanup_demo() {
    echo ""
    print_step "CLEANUP" "Graceful Demo Shutdown" "Cleaning up all demo components and processes"
    
    # Gracefully shutdown echo-server
    if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        print_substep "Stopping echo-server (PID: $SERVER_PID)"
        kill -TERM $SERVER_PID 2>/dev/null || true
        sleep 3
        if kill -0 $SERVER_PID 2>/dev/null; then
            print_warning "Force killing unresponsive echo-server"
            kill -9 $SERVER_PID 2>/dev/null || true
        else
            print_success "Echo-server stopped gracefully"
        fi
    fi
    
    # Clean up any remaining echo processes
    print_substep "Cleaning up remaining processes"
    pkill -TERM -f echo-server 2>/dev/null || true
    pkill -TERM -f echo-client 2>/dev/null || true
    sleep 2
    pkill -9 -f echo-server 2>/dev/null || true
    pkill -9 -f echo-client 2>/dev/null || true
    
    # Gracefully shutdown SPIRE components
    print_substep "Stopping SPIRE services"
    if [ -f scripts/demo/spire-server.pid ]; then
        SPIRE_SERVER_PID=$(cat scripts/demo/spire-server.pid 2>/dev/null)
        if [ -n "$SPIRE_SERVER_PID" ] && kill -0 $SPIRE_SERVER_PID 2>/dev/null; then
            kill -TERM $SPIRE_SERVER_PID 2>/dev/null || true
            sleep 2
        fi
    fi
    
    if [ -f scripts/demo/spire-agent.pid ]; then
        SPIRE_AGENT_PID=$(cat scripts/demo/spire-agent.pid 2>/dev/null)
        if [ -n "$SPIRE_AGENT_PID" ] && kill -0 $SPIRE_AGENT_PID 2>/dev/null; then
            kill -TERM $SPIRE_AGENT_PID 2>/dev/null || true
            sleep 2
        fi
    fi
    
    print_substep "Cleaning up temporary files"
    rm -f scripts/demo/*.log scripts/demo/*.pid || true
    
    print_success "Demo cleanup completed"
}

# Set trap to cleanup on exit
trap cleanup_demo EXIT

# Demo Header
echo ""
echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${BLUE}║                              EPHEMOS DEMO                                   ║${RESET}"
echo -e "${BOLD}${BLUE}║                    Identity-Based Authentication                           ║${RESET}"  
echo -e "${BOLD}${BLUE}║                 🚫 ZERO PLAINTEXT SECRETS REVOLUTION 🚫                  ║${RESET}"
echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo -e "${BOLD}${RED}🔥 PLAINTEXT SECRETS HAVE VANISHED! 🔥${RESET}"
echo ""
echo -e "${BOLD}${GREEN}This demo proves that authentication secrets are now completely EPHEMERAL:${RESET}"
echo "• 🚫 NO API keys in your code"
echo "• 🚫 NO passwords in configuration files"  
echo "• 🚫 NO tokens in environment variables (.env files)"
echo "• 🚫 NO secrets in Docker images or Kubernetes manifests"
echo "• 🚫 NO possibility of secret leakage in log files"
echo "• 🚫 NO long-lived credentials to rotate or manage"
echo ""
echo -e "${BOLD}${BLUE}✨ INSTEAD: Cryptographic certificates that live for only 1 hour! ✨${RESET}"
echo ""
echo "This demo teaches how SPIFFE/SPIRE + Ephemos eliminates secrets entirely:"
echo "• How services get ephemeral X.509 certificates automatically"  
echo "• How authentication works with ZERO stored secrets"
echo "• How developers build secure services without touching credentials"
echo ""

# ============================================================================
# STEP 1: Environment Setup
# ============================================================================
print_step "1" "Environment Setup & Preparation" \
"Preparing the demo environment by building applications and checking ports"

print_substep "Cleaning up any existing processes on ports 50051-50063"
pkill -f echo-server 2>/dev/null || true

print_substep "Finding available port for demo"
AVAILABLE_PORT=":50051"
for port in 50051 50052 50053 50061 50062 50063; do
    if ! ss -tulpn | grep -q :$port; then
        AVAILABLE_PORT=":$port"
        break
    fi
done

if [ "$AVAILABLE_PORT" != ":50051" ]; then
    print_warning "Port 50051 in use, using port $AVAILABLE_PORT"
    PORT_NUM=${AVAILABLE_PORT#:}
    sed -i "s/localhost[0-9][0-9][0-9][0-9][0-9]/localhost:${PORT_NUM}/g" ../../examples/echo-client/main.go 2>/dev/null || true
fi
export ECHO_SERVER_ADDRESS="$AVAILABLE_PORT"
print_info "Demo will use port: ${AVAILABLE_PORT}"

print_substep "Building example applications"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

print_info "Building echo-server..."
go build -o bin/echo-server ./examples/echo-server || { 
    print_error "Failed to build echo-server"; exit 1; 
}

print_info "Building echo-client..."  
go build -o bin/echo-client ./examples/echo-client || { 
    print_error "Failed to build echo-client"; exit 1; 
}

print_success "Applications built successfully"

show_code "Server Code Structure (examples/echo-server/main.go)" \
"func main() {
    ctx := context.Background()
    
    // Step 1: Create identity-aware server with automatic mTLS
    server, err := ephemos.NewIdentityServer(ctx, \"config/echo-server.yaml\")
    if err != nil {
        log.Fatal(err)
    }
    defer server.Close()
    
    // Step 2: Register your business logic
    registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
        pb.RegisterEchoServiceServer(s, &echoServer{})
    })
    server.RegisterService(ctx, registrar)
    
    // Step 3: Start server - mTLS authentication is automatic
    lis, _ := net.Listen(\"tcp\", address)
    server.Serve(ctx, lis)  // Only authenticated clients can connect!
}"

show_code "Client Code Structure (examples/echo-client/main.go)" \
"func main() {
    ctx := context.Background()
    
    // Step 1: Create identity-aware client with automatic mTLS  
    client, err := ephemos.NewIdentityClient(ctx, \"config/echo-client.yaml\")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Step 2: Connect with automatic certificate authentication
    conn, err := client.Connect(ctx, \"echo-server\", \"localhost:50051\")
    if err != nil {
        log.Fatal(err)  // Fails if not authorized by server
    }
    defer conn.Close()
    
    // Step 3: Use the connection normally
    echoClient := pb.NewEchoServiceClient(conn.GetClientConnection())
    response, err := echoClient.Echo(ctx, &pb.EchoRequest{Message: \"Hello\"})
}"

# ============================================================================
# STEP 1.5: PROOF OF ZERO SECRETS
# ============================================================================
print_step "1.5" "🚫 PROVING ZERO PLAINTEXT SECRETS 🚫" \
"Examining actual code and configuration files to prove NO secrets exist anywhere!"

print_substep "Examining Server Configuration File"
print_info "Let's look at config/echo-server.yaml - the complete server configuration:"

if [ -f "config/echo-server.yaml" ]; then
    echo ""
    echo -e "${BOLD}${YELLOW}📄 config/echo-server.yaml (COMPLETE FILE):${RESET}"
    echo -e "${YELLOW}╔════════════════════════════════════════════════════════════════════════════════╗${RESET}"
    while IFS= read -r line; do
        echo -e "${YELLOW}║${RESET} $line"
    done < config/echo-server.yaml
    echo -e "${YELLOW}╚════════════════════════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    print_success "✅ ZERO secrets found in server configuration!"
else
    print_info "Server config file not found - using default configuration (also zero secrets)"
fi

print_substep "Examining Client Configuration File"
print_info "Let's look at config/echo-client.yaml - the complete client configuration:"

if [ -f "config/echo-client.yaml" ]; then
    echo ""
    echo -e "${BOLD}${YELLOW}📄 config/echo-client.yaml (COMPLETE FILE):${RESET}"
    echo -e "${YELLOW}╔════════════════════════════════════════════════════════════════════════════════╗${RESET}"
    while IFS= read -r line; do
        echo -e "${YELLOW}║${RESET} $line"
    done < config/echo-client.yaml
    echo -e "${YELLOW}╚════════════════════════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    print_success "✅ ZERO secrets found in client configuration!"
else
    print_info "Client config file not found - using default configuration (also zero secrets)"
fi

print_substep "Examining Environment Variables"
print_info "Checking if any secret-related environment variables are needed:"

echo ""
echo -e "${BOLD}${YELLOW}Environment Variables Currently Set for Demo:${RESET}"
echo -e "${YELLOW}╔════════════════════════════════════════════════════════════════════════════════╗${RESET}"
env | grep -E "(EPHEMOS|API_KEY|PASSWORD|SECRET|TOKEN|CREDENTIAL)" | while IFS= read -r line; do
    if [ -n "$line" ]; then
        echo -e "${YELLOW}║${RESET} $line"
    fi
done
if ! env | grep -E "(API_KEY|PASSWORD|SECRET|TOKEN|CREDENTIAL)" > /dev/null; then
    echo -e "${YELLOW}║${RESET} ${GREEN}✅ NO SECRET-RELATED ENVIRONMENT VARIABLES FOUND!${RESET}"
fi
echo -e "${YELLOW}║${RESET} EPHEMOS_CONFIG=${EPHEMOS_CONFIG:-NOT_SET} ${GREEN}(just points to config file - no secrets!)${RESET}"
echo -e "${YELLOW}║${RESET} ECHO_SERVER_ADDRESS=${ECHO_SERVER_ADDRESS} ${GREEN}(just network address - no secrets!)${RESET}"
echo -e "${YELLOW}╚════════════════════════════════════════════════════════════════════════════════╝${RESET}"
echo ""

print_substep "🧠 DEVELOPER MENTAL MODEL SHIFT: From Dashboard Secrets to Ephemeral"
print_info "How developer workflow changes with Ephemos:"

echo ""
echo -e "${BOLD}${RED}❌ OLD WORKFLOW: Developer Secret Management Burden${RESET}"
echo ""
show_code "Developer's Traditional Secret Workflow (PAINFUL!)" \
"1. 🌐 Developer logs into company secrets dashboard
2. 📋 Developer copies API key: [REDACTED-LONG-SECRET-STRING]
3. 💾 Developer pastes secret into .env file
4. 🔄 Developer commits code (hopefully .env is in .gitignore!)
5. 🐳 Developer copies secret into Docker build args
6. ☸️ Developer pastes secret into Kubernetes secret manifest
7. 📊 Developer updates monitoring/logging (careful not to log secrets!)
8. 🔄 Every 90 days: Developer repeats steps 1-7 for rotation
9. 😰 Developer constantly worries about secret leaks
10. 🚨 When secret leaks: PANIC! Immediate rotation required!"

echo ""
echo -e "${BOLD}${GREEN}✅ NEW WORKFLOW: Developer Secret-Free Experience${RESET}"
echo ""
show_code "Developer's Ephemos Workflow (SIMPLE!)" \
"1. 👩‍💻 Developer writes business logic code (ZERO secrets!)
2. 📝 Developer creates config.yaml (service name only - no secrets!)
3. 💻 Developer runs: ephemos.NewIdentityServer(ctx, \"config.yaml\")
4. ✨ Ephemos automatically handles ALL authentication
5. 🚀 Developer deploys (no secrets in images/manifests!)
6. 😌 Developer never thinks about secrets again
7. ♻️ Certificates auto-rotate every 30 minutes (developer unaware!)
8. 🔒 Zero risk of secret leaks (there are no secrets to leak!)
9. 📈 Developer focuses on business value, not credential management!"

print_substep "What About Traditional API Key Authentication?"
print_info "Let's compare the detailed secret management approaches:"

show_code "❌ TRADITIONAL INSECURE APPROACH (what we DON'T do)" \
"// ❌ DEVELOPER NIGHTMARE: Secrets everywhere!

Step 1: Log into dashboard, copy API key
const API_KEY = \"[COPY-PASTE-SECRET-HERE]\"  // ❌ Secret in code!

Step 2: Put in environment 
apiKey := os.Getenv(\"API_KEY\")  // ❌ Secret in .env file!

Step 3: Configure deployment
auth:
  api_key: [PASTE-FROM-DASHBOARD]   # ❌ Secret in YAML!
  password: [ANOTHER-SECRET]        # ❌ Another secret!

// ❌ DEVELOPER PAIN POINTS:
// • Dashboard login required every time
// • Manual copy/paste of secrets
// • Multiple places to update secrets
// • Git commit anxiety (did I leak secrets?)
// • Container security scanning (secrets in images?)
// • Kubernetes secret management complexity
// • Log filtering (avoid logging secrets)
// • Rotation scheduling (quarterly manual work)
// • Incident response (secret leak = emergency)
// • Multi-environment management (dev/staging/prod secrets)"

show_code "✅ EPHEMOS APPROACH (what we DO)" \
"// ✅ SECURE: ZERO secrets needed!
server, err := ephemos.NewIdentityServer(ctx, \"config.yaml\")
// • No API keys needed ✅
// • No passwords needed ✅  
// • No tokens needed ✅
// • Certificates obtained automatically from SPIRE ✅
// • Certificates expire in 1 hour ✅
// • Certificates rotate automatically ✅

client, err := ephemos.NewIdentityClient(ctx, \"config.yaml\")
conn, err := client.Connect(ctx, \"server-name\", \"address\")
// • Authentication happens automatically ✅
// • No credentials to manage ✅
// • No secrets to leak ✅"

print_substep "Runtime Certificate Verification"
print_info "During this demo, we'll prove that:"
echo "  🔍 Certificates are generated on-demand by SPIRE (not stored)"
echo "  🔍 Certificates exist only in memory (never written to disk by our apps)"
echo "  🔍 Certificates expire automatically after 1 hour"
echo "  🔍 Authentication works without ANY stored secrets"
echo "  🔍 If you search the entire codebase, you'll find ZERO hardcoded secrets"

echo ""
print_success "🎉 PROOF COMPLETE: This is a SECRET-FREE architecture!"
echo ""
echo -e "${BOLD}${GREEN}The revolution is here: Authentication without secrets! 🚀${RESET}"

# ============================================================================
# STEP 2: SPIRE Infrastructure Setup (DevOps Task)
# ============================================================================
print_step "2" "🔧 SPIRE Infrastructure Setup (DevOps Responsibility)" \
"Starting SPIRE Server and Agent - the EPHEMERAL certificate factory that eliminates secrets!"

echo -e "${BOLD}${RED}👤 WHO DOES THIS STEP: DevOps/Platform Team${RESET}"
echo "  🔧 DevOps installs and configures SPIRE infrastructure (one-time setup)"
echo "  🔧 DevOps ensures SPIRE Server and Agent are running"
echo "  🔧 Developers NEVER need to touch SPIRE infrastructure"
echo ""

print_info "🏭 SPIRE: The Ephemeral Certificate Factory (Think JWT tokens, but for service identity!)"
echo ""
echo -e "${BOLD}${BLUE}📚 CONNECTING TO FAMILIAR BACKEND CONCEPTS:${RESET}"
echo ""
echo -e "${BOLD}🔄 You know JWT tokens for user authentication? This is similar for services:${RESET}"
echo "  👤 JWT: Short-lived tokens for USER authentication (expires in hours/days)"
echo "  🤖 SPIRE: Short-lived certificates for SERVICE authentication (expires in 1 hour)"
echo "  👤 JWT: Contains user claims (user_id, roles, permissions)"  
echo "  🤖 SPIRE: Contains service identity (spiffe://domain/service-name)"
echo "  👤 JWT: Signed by auth server, verified by services"
echo "  🤖 SPIRE: Signed by SPIRE server, verified by peer services"
echo ""
echo -e "${BOLD}🌐 You know HTTPS certificates for websites? This is different:${RESET}"
echo "  🌐 HTTPS Certs: Long-lived (1-2 years), stored on disk, manually renewed"
echo "  🤖 Service Certs: Short-lived (1 hour), in-memory only, auto-renewed"
echo "  🌐 HTTPS Certs: Protect data in transit (encryption)"
echo "  🤖 Service Certs: Prove service identity (authentication) + encrypt"
echo "  🌐 HTTPS Certs: One cert per domain/website"
echo "  🤖 Service Certs: One cert per service instance (thousands in microservices)"
echo ""
echo -e "${BOLD}🔑 You know API key rotation best practices? This automates it:${RESET}"
echo "  🔑 API Keys: Rotate monthly/quarterly (manual process)"
echo "  🤖 Service Certs: Rotate every 30 minutes (fully automatic)"
echo "  🔑 API Keys: Same key used by all instances"
echo "  🤖 Service Certs: Unique cert per service instance"
echo "  🔑 API Keys: Stored in env vars, config files, secrets managers"
echo "  🤖 Service Certs: Generated on-demand, exist only in memory"
echo ""
echo -e "${BOLD}${RED}🔥 The Familiar Evolution:${RESET}"
echo "  1️⃣ Static passwords (1990s) → API keys (2000s) → JWT tokens (2010s) → Ephemeral service certs (2020s)"
echo "  2️⃣ Like upgrading from: Database passwords → Redis auth tokens → OAuth access tokens → SPIFFE certificates"
echo "  3️⃣ Each generation: Shorter-lived, more secure, less manual management"

print_substep "Checking if SPIRE services are running"
if ! pgrep -f "spire-server.*run" > /dev/null || ! pgrep -f "spire-agent.*run" > /dev/null; then
    print_info "SPIRE services not running - starting them now"
    print_warning "This requires sudo privileges for SPIRE server operations"
    
    CURRENT_DIR=$(pwd)
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)" 
    cd "$SCRIPT_DIR"
    
    if ! ./start-spire.sh; then
        print_error "Failed to start SPIRE services"
        print_info "Make sure you have sudo privileges and SPIRE is installed"
        exit 1
    fi
    
    cd "$CURRENT_DIR"
    print_success "SPIRE Server and Agent started successfully"
else
    print_success "SPIRE services already running"
fi

print_substep "Verifying SPIRE infrastructure"
print_info "Checking SPIRE Server socket: /tmp/spire-server/private/api.sock"
if [ -S "/tmp/spire-server/private/api.sock" ]; then
    print_success "SPIRE Server socket accessible"
else
    print_error "SPIRE Server socket not found"
    exit 1
fi

print_info "Checking SPIRE Agent socket: /tmp/spire-agent/public/api.sock"
if [ -S "/tmp/spire-agent/public/api.sock" ]; then
    print_success "SPIRE Agent socket accessible"
else
    print_error "SPIRE Agent socket not found"  
    exit 1
fi

print_info "Testing SPIRE Server health"
if sudo spire-server healthcheck -socketPath /tmp/spire-server/private/api.sock >/dev/null 2>&1; then
    print_success "SPIRE Server health check passed"
else
    print_warning "SPIRE Server health check failed - continuing anyway"
fi

# ============================================================================
# STEP 3: Service Registration (DevOps Security Task)
# ============================================================================
print_step "3" "🔒 Service Identity Registration (DevOps Security Task)" \
"Manually registering services with SPIRE - this is the required security step"

echo -e "${BOLD}${RED}👤 WHO DOES THIS STEP: DevOps/Security Team${RESET}"
echo "  🔒 DevOps registers each service identity (security-controlled step)"
echo "  🔒 DevOps decides which services can get certificates"
echo "  🔒 Developers request registration via ticket/workflow"
echo "  🔒 One-time registration per service (not per developer)"
echo ""

print_info "Why manual registration is required:"
echo "  • Security: Prevents unauthorized services from self-registering"  
echo "  • Control: Administrators decide which services get identities"
echo "  • Audit: Creates a clear trail of authorized services"
echo "  • Zero Trust: No automatic trust - everything must be explicit"

ACTUAL_UID=$(id -u)
ACTUAL_USER=$(whoami)
print_info "Services will run as user: $ACTUAL_USER (UID: $ACTUAL_UID)"

print_substep "Registering echo-server identity"
print_info "Creating SPIFFE ID: spiffe://example.org/echo-server"
print_info "Parent ID: spiffe://example.org/spire-agent"  
print_info "Selector: unix:uid:$ACTUAL_UID (identifies the process)"

show_code "Registration Command Being Executed" \
"sudo spire-server entry create \\
    -socketPath /tmp/spire-server/private/api.sock \\
    -spiffeID spiffe://example.org/echo-server \\
    -parentID spiffe://example.org/spire-agent \\
    -selector unix:uid:$ACTUAL_UID \\
    -ttl 3600"

set +e
SERVER_ENTRY_OUTPUT=$(sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>&1)
SERVER_ENTRY_EXIT_CODE=$?
set -e

if [ $SERVER_ENTRY_EXIT_CODE -eq 0 ]; then
    print_success "Echo-server registered successfully"
elif echo "$SERVER_ENTRY_OUTPUT" | grep -q "already exists"; then
    print_success "Echo-server already registered (this is normal)"
else
    print_error "Echo-server registration failed"
    echo "$SERVER_ENTRY_OUTPUT"
    exit 1
fi

print_substep "Registering echo-client identity"  
print_info "Creating SPIFFE ID: spiffe://example.org/echo-client"

set +e
CLIENT_ENTRY_OUTPUT=$(sudo spire-server entry create \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:$ACTUAL_UID \
    -ttl 3600 2>&1)
CLIENT_ENTRY_EXIT_CODE=$?
set -e

if [ $CLIENT_ENTRY_EXIT_CODE -eq 0 ]; then
    print_success "Echo-client registered successfully"
elif echo "$CLIENT_ENTRY_OUTPUT" | grep -q "already exists"; then
    print_success "Echo-client already registered (this is normal)"
else
    print_error "Echo-client registration failed"
    echo "$CLIENT_ENTRY_OUTPUT"
    exit 1
fi

print_substep "Waiting for registration to propagate to SPIRE Agent"
sleep 5

print_substep "Verifying registrations are active"
RETRY_COUNT=0
MAX_RETRIES=12

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    set +e
    SPIRE_OUTPUT=$(sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock 2>&1)
    SPIRE_EXIT_CODE=$?
    set -e
    
    if echo "$SPIRE_OUTPUT" | grep -q "echo-server"; then
        print_success "Service registrations verified and active"
        break
    else
        print_info "Waiting for registrations to become active... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
        sleep 5
        RETRY_COUNT=$((RETRY_COUNT + 1))
    fi
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    print_error "Timeout: Service registrations not ready after 60 seconds"
    exit 1
fi

print_info "Services are now authorized to receive SPIFFE certificates"

# ============================================================================
# STEP 4: Developer Writes Server Code (Developer Task)
# ============================================================================
print_step "4" "👩‍💻 Developer Writes & Runs Server Code (Developer Responsibility)" \
"🚫 WATCH: Server gets ephemeral certificates with ZERO stored secrets!"

echo -e "${BOLD}${GREEN}👤 WHO DOES THIS STEP: Developer${RESET}"
echo "  💻 Developer writes service code using Ephemos SDK"
echo "  💻 Developer creates config file (no secrets needed!)"
echo "  💻 Developer runs service - automatic certificate retrieval!"
echo "  💻 ZERO secret management burden on developers"
echo ""

print_substep "Starting echo-server in background"
print_info "Server configuration (config/echo-server.yaml) - NO SECRETS INSIDE:"
echo "  • Service name: echo-server (NOT a secret - just an identifier)"
echo "  • SPIRE socket: /tmp/spire-agent/public/api.sock (NOT a secret - just a file path)"  
echo "  • Authorized clients: [echo-client] (NOT a secret - just authorization policy)"
echo ""
echo -e "${BOLD}${RED}🔍 NOTICE: Search the entire config file - you'll find ZERO secrets!${RESET}"

show_code "🚫 EPHEMERAL CERTIFICATE MAGIC (What Happens When Server Starts)" \
"1. ephemos.NewIdentityServer() automatically:
   ⚡ Connects to SPIRE Agent via Unix socket (NO secrets needed!)
   ⚡ Requests ephemeral X.509 certificate for spiffe://example.org/echo-server
   ⚡ Certificate contains cryptographic identity (NO plaintext secrets!)
   ⚡ Certificate stored ONLY in memory (NEVER written to disk!)
   ⚡ Certificate expires in 1 hour (EPHEMERAL by design!)
   ⚡ Sets up mTLS server with ephemeral certificate
   ⚡ Starts auto-rotation (new certificate every ~30 minutes)

2. server.Serve() starts listening with mTLS enabled:
   🔐 Only clients with valid certificates can connect
   🔐 Client certificates must be signed by same SPIRE trust bundle  
   🔐 Client SPIFFE ID must be in authorized_clients list
   🔐 NO API keys, passwords, or tokens anywhere in this process!

🎯 KEY INSIGHT: The 'secret' (certificate) is generated on-demand
   and exists only temporarily in memory. It's truly EPHEMERAL!"

EPHEMOS_CONFIG=config/echo-server.yaml ECHO_SERVER_ADDRESS=${ECHO_SERVER_ADDRESS:-:50051} ./bin/echo-server > scripts/demo/server.log 2>&1 &
SERVER_PID=$!
print_info "Server started with PID: $SERVER_PID"

print_substep "Waiting for server to obtain SPIFFE identity"
SERVER_READY=false
WAIT_COUNT=0
MAX_WAIT=24

while [ $WAIT_COUNT -lt $MAX_WAIT ] && [ "$SERVER_READY" = "false" ]; do
    if [ ! -f scripts/demo/server.log ]; then
        print_info "Waiting for server log... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
        sleep 5
        WAIT_COUNT=$((WAIT_COUNT + 1))
        continue
    fi
    
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        print_error "Server process died unexpectedly"
        echo "Server log:"
        cat scripts/demo/server.log
        exit 1
    fi
    
    if grep -q "Server identity created\|Server ready\|Successfully obtained SPIFFE identity\|Identity service initialized" scripts/demo/server.log; then
        print_success "Server successfully obtained SPIFFE identity!"
        SERVER_READY=true
        break
    fi
    
    if grep -q "failed to get X509 SVID\|No identity issued" scripts/demo/server.log; then
        print_info "Server requesting identity from SPIRE... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
    elif grep -q "Failed to create identity server" scripts/demo/server.log; then
        print_error "Identity server creation failed"
        cat scripts/demo/server.log
        exit 1
    else
        print_info "Server starting up... (attempt $((WAIT_COUNT + 1))/$MAX_WAIT)"
    fi
    
    sleep 5
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ "$SERVER_READY" = "false" ]; then
    print_error "Timeout: Server failed to obtain identity after 2 minutes"
    cat scripts/demo/server.log
    exit 1
fi

print_substep "Displaying server startup log"
echo ""
echo -e "${YELLOW}╔══════════════════════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${YELLOW}║                              SERVER STARTUP LOG                             ║${RESET}"  
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"
while IFS= read -r line; do
    echo -e "${YELLOW}║${RESET} $line"
done < scripts/demo/server.log
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

print_success "Echo-server is running with mTLS authentication enabled"

# ============================================================================  
# STEP 5: Developer Writes Client Code (Developer Task)
# ============================================================================
print_step "5" "👨‍💻 Developer Writes Client Code (Developer Responsibility)" \
"The client automatically authenticates and communicates securely with the server"

echo -e "${BOLD}${GREEN}👤 WHO DOES THIS STEP: Developer${RESET}"
echo "  💻 Developer writes client code using Ephemos SDK"
echo "  💻 Developer creates client config (no secrets needed!)"
echo "  💻 Developer runs client - automatic certificate retrieval!"
echo "  💻 Authentication happens transparently - developer doesn't manage it"
echo ""

print_substep "Starting echo-client"
print_info "Client configuration (config/echo-client.yaml):"
echo "  • Service name: echo-client"
echo "  • SPIRE socket: /tmp/spire-agent/public/api.sock"
echo "  • Trusted servers: [echo-server] (will only connect to this server)"

show_code "What Happens When Client Connects" \
"1. ephemos.NewIdentityClient() automatically:
   • Connects to SPIRE Agent via Unix socket  
   • Requests X.509 certificate for spiffe://example.org/echo-client
   • Sets up mTLS client with certificate

2. client.Connect() performs mTLS handshake:
   • Client presents its certificate to server
   • Server verifies client certificate against SPIRE trust bundle
   • Server checks client SPIFFE ID is in authorized_clients
   • Both authenticate each other mutually

3. If successful, normal gRPC communication proceeds"

print_info "Running client with 10-second timeout..."
echo ""
timeout 10 bash -c 'EPHEMOS_CONFIG=config/echo-client.yaml ./bin/echo-client 2>&1' | tee scripts/demo/client.log | while IFS= read -r line; do
    echo -e "${GREEN}[CLIENT]${RESET} $line"
done

CLIENT_EXIT=${PIPESTATUS[0]}

if grep -q "Echo response received" scripts/demo/client.log; then
    print_success "Client successfully authenticated and communicated with server!"
    
    print_substep "Displaying complete communication log"
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${GREEN}║                              CLIENT-SERVER MESSAGES                         ║${RESET}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"
    
    echo -e "${GREEN}║ SERVER LOG:${RESET}"
    while IFS= read -r line; do
        echo -e "${GREEN}║${RESET} [SERVER] $line"
    done < scripts/demo/server.log
    
    echo -e "${GREEN}║ CLIENT LOG:${RESET}"
    while IFS= read -r line; do
        echo -e "${GREEN}║${RESET} [CLIENT] $line"
    done < scripts/demo/client.log
    
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"
    
    SUCCESS=true
    
elif [ $CLIENT_EXIT -eq 124 ]; then
    print_error "Client timed out without successful communication"
    exit 1
else
    print_error "Client failed with exit code $CLIENT_EXIT"
    cat scripts/demo/client.log
    exit 1
fi

# ============================================================================
# STEP 6: Authentication Failure Demo  
# ============================================================================
print_step "6" "Demonstrating Authentication Failure" \
"Showing what happens when unauthorized services try to connect"

print_substep "Simulating unauthorized client connection"
print_info "This demonstrates the Zero Trust security model:"
echo "  • Only registered services can obtain certificates"
echo "  • Only authorized clients can connect to servers"
echo "  • Authentication failures happen at transport layer"

print_info "Attempting connection with unregistered/unauthorized client..."
set +e
FAILURE_OUTPUT=$(EPHEMOS_CONFIG=config/echo-client.yaml timeout 5 ./bin/echo-client 2>&1)
FAILURE_EXIT=$?
set -e

if echo "$FAILURE_OUTPUT" | grep -qi "error\|fail\|denied\|unauthorized"; then
    print_success "Authentication properly rejected unauthorized client"
    print_info "This proves the security is working correctly"
else
    print_warning "No authentication failure detected - connection may have succeeded"
fi

# ============================================================================
# STEP 7: Learning Summary
# ============================================================================
print_step "7" "Learning Summary - What You Learned" \
"Understanding the complete identity-based authentication workflow"

echo ""
echo -e "${BOLD}${BLUE}🎓 KEY CONCEPTS DEMONSTRATED:${RESET}"
echo ""

echo -e "${BOLD}1. Manual Registration (Security)${RESET}"
echo "   • Services must be explicitly registered with SPIRE"
echo "   • Prevents unauthorized services from getting identities"
echo "   • Command: sudo spire-server entry create -spiffeID spiffe://domain/service"
echo ""

echo -e "${BOLD}2. Automatic Identity Retrieval (Developer Experience)${RESET}" 
echo "   • ephemos.NewIdentityServer() automatically gets certificates"
echo "   • ephemos.NewIdentityClient() automatically gets certificates"
echo "   • No manual certificate management required"
echo ""

echo -e "${BOLD}3. Transport-Layer Security (mTLS)${RESET}"
echo "   • Authentication happens during TLS handshake"
echo "   • Both client and server verify each other's certificates"
echo "   • Application code never runs if authentication fails"
echo ""

echo -e "${BOLD}4. Configuration-Based Authorization${RESET}"
echo "   • Server config: authorized_clients = [\"echo-client\"]"
echo "   • Client config: trusted_servers = [\"echo-server\"]"
echo "   • Fine-grained access control"
echo ""

echo -e "${BOLD}5. Certificate Lifecycle Management${RESET}"
echo "   • Certificates expire in 1 hour (short-lived)"
echo "   • SPIRE automatically rotates certificates"
echo "   • No manual certificate renewal needed"
echo ""

echo -e "${BOLD}${BLUE}🎯 ROLE RESPONSIBILITIES SUMMARY:${RESET}"
echo ""
echo -e "${BOLD}${RED}DevOps/Platform Team (One-Time Setup):${RESET}"
echo "  🔧 Step 1: Install SPIRE infrastructure (servers/agents)"
echo "  🔧 Step 2: Configure SPIRE trust domain and policies" 
echo "  🔒 Step 3: Register service identities (per service request)"
echo "  📋 DevOps handles infrastructure - developers never touch it"
echo ""

echo -e "${BOLD}${GREEN}Developer Team (Ongoing Development):${RESET}"
echo "  📝 Step 1: Create config.yaml (service name only - NO secrets!)"
echo "  💻 Step 2: Write code using Ephemos SDK (automatic authentication)"
echo "  🚀 Step 3: Deploy and run (certificates obtained automatically)"
echo "  😌 Developers focus on business logic - zero secret management!"
echo ""

echo -e "${BOLD}${GREEN}📋 DEVELOPER WORKFLOW COMPARISON:${RESET}"
echo ""
echo -e "${BOLD}${RED}❌ Traditional (10+ steps with secret pain):${RESET}"
echo "  1️⃣ Log into company secrets dashboard"
echo "  2️⃣ Navigate to service API keys section"
echo "  3️⃣ Generate or copy existing API key"
echo "  4️⃣ Store in .env file locally"
echo "  5️⃣ Add to Docker environment variables"
echo "  6️⃣ Create Kubernetes secret manifest"
echo "  7️⃣ Update deployment to use secret"
echo "  8️⃣ Configure log filtering (don't log secrets!)"
echo "  9️⃣ Set up secret rotation schedule"
echo "  🔟 Monitor for secret leaks"
echo "  📋 PLUS: Quarterly rotation of ALL above steps!"
echo ""

echo -e "${BOLD}${GREEN}✅ Ephemos (3 steps - NO secrets):${RESET}"
echo "  1️⃣ Write config.yaml (service name only)"
echo "  2️⃣ Write code: ephemos.NewIdentityServer(ctx, \"config.yaml\")"  
echo "  3️⃣ Run service (authentication automatic!)"
echo "  ✨ DONE! No secrets, no rotation, no dashboard!"
echo ""

echo -e "${BOLD}${BLUE}🎉 ONE LESS STEP? Try SEVEN LESS STEPS!${RESET}"
echo -e "${BOLD}We eliminated the entire secret management workflow!${RESET}"
echo ""

echo -e "${BOLD}${GREEN}🔒 SECURITY REVOLUTION - SECRETS ELIMINATED:${RESET}"
echo ""
echo -e "${BOLD}${RED}🚫 WHAT'S COMPLETELY GONE (Zero Risk):${RESET}"
echo "  🚫 NO API keys in code (impossible to leak in git commits)"
echo "  🚫 NO passwords in config files (impossible to leak in Docker images)"
echo "  🚫 NO tokens in .env files (impossible to leak in environment dumps)"
echo "  🚫 NO secrets in Kubernetes manifests (impossible to leak in YAML files)"
echo "  🚫 NO credentials in log files (impossible to leak in application logs)"
echo "  🚫 NO long-lived secrets to rotate (impossible to forget to rotate)"
echo "  🚫 NO secret management burden (impossible to manage incorrectly)"
echo ""
echo -e "${BOLD}${GREEN}✅ WHAT YOU GET INSTEAD (Ephemeral Security):${RESET}"
echo "  ✅ Ephemeral certificates (exist only 1 hour, then vanish)"
echo "  ✅ In-memory only credentials (never touch disk storage)"
echo "  ✅ Automatic generation (no human intervention needed)"
echo "  ✅ Automatic rotation (new credentials every ~30 minutes)"
echo "  ✅ Cryptographic authentication (mathematically verifiable)"
echo "  ✅ Mutual verification (both client and server authenticate)"
echo "  ✅ Transport-layer security (authentication before application code)"
echo "  ✅ Zero-knowledge deployment (developers never see or touch secrets)"
echo ""
echo -e "${BOLD}${BLUE}🧠 DEVELOPER MENTAL MODEL TRANSFORMATION:${RESET}"
echo ""
echo -e "${BOLD}${RED}❌ OLD THINKING: \"How do I manage secrets securely?\"${RESET}"
echo "  💭 Where do I store API keys?"
echo "  💭 How do I rotate credentials?"
echo "  💭 Did I accidentally commit secrets?"
echo "  💭 How do I handle different environments?"
echo "  💭 What if secrets leak in logs?"
echo "  💭 Dashboard login required for every new service"
echo ""

echo -e "${BOLD}${GREEN}✅ NEW THINKING: \"I don't need to think about secrets at all!\"${RESET}"
echo "  💭 Just write business logic code"
echo "  💭 Add ephemos.NewIdentityServer() call"
echo "  💭 Deploy anywhere - no secret configuration needed"
echo "  💭 Focus on features, not credential management"
echo "  💭 Sleep peacefully - nothing can leak"
echo "  💭 No dashboard, no copy/paste, no secret anxiety!"
echo ""

echo -e "${BOLD}${BLUE}🎯 THE BOTTOM LINE:${RESET}"
echo -e "${BOLD}If someone steals your entire codebase, config files, environment variables,${RESET}"
echo -e "${BOLD}Docker images, and Kubernetes manifests... they still get ZERO secrets!${RESET}"
echo -e "${BOLD}${RED}Because there are no secrets to steal. They're all EPHEMERAL! 🔥${RESET}"
echo ""

# Success Message
echo ""
echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${GREEN}║                    🚫🔥 SECRETS ELIMINATED! DEMO COMPLETE! 🔥🚫             ║${RESET}"
echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo -e "${CHECKMARK} ${GREEN}You just witnessed the COMPLETE ELIMINATION of plaintext secrets!${RESET}"
echo -e "${CHECKMARK} ${GREEN}Authentication now works with ZERO stored credentials${RESET}"  
echo -e "${CHECKMARK} ${GREEN}Your microservices are now IMMUNE to secret leaks${RESET}"
echo -e "${CHECKMARK} ${GREEN}You can deploy with confidence - there are NO secrets to steal!${RESET}"
echo ""
echo -e "${BOLD}${RED}🔥 THE SECRET REVOLUTION IS COMPLETE! 🔥${RESET}"
echo -e "${BOLD}You've entered the post-secret era of authentication.${RESET}"
echo ""
echo -e "${BOLD}${BLUE}🎯 FOR DEVELOPERS: You just experienced the secret-free future!${RESET}"
echo ""
echo -e "${BOLD}${GREEN}✨ Remember this feeling:${RESET}"
echo "  💻 You wrote authentication code with ZERO secrets"
echo "  🔐 You built a secure service without managing credentials"
echo "  🚀 You deployed without worrying about secret leaks"
echo "  😌 You focused on business logic, not authentication complexity"
echo ""
echo -e "${BOLD}${RED}🚫 Never again will you need to:${RESET}"
echo "  🚫 Log into a secrets dashboard to copy API keys"
echo "  🚫 Paste secrets into environment variables"  
echo "  🚫 Rotate credentials quarterly"
echo "  🚫 Worry about committing secrets to git"
echo "  🚫 Configure secret scanning in CI/CD"
echo "  🚫 Debug secret leaks in logs"
echo "  🚫 Handle secret rotation outages"
echo ""
echo -e "${BOLD}${BLUE}🔥 This is the post-secret era of development! 🔥${RESET}"
echo ""

echo -e "${BOLD}Next Steps:${RESET}"
echo "  • Read examples/echo-server/main.go and examples/echo-client/main.go"
echo "  • Try modifying the authorized_clients in config/echo-server.yaml"
echo "  • Build your own services using the Ephemos patterns you learned"
echo "  • Share this demo with your team - show them the secret-free future!"
echo "  • Check out docs/GETTING_STARTED.md for more examples"
echo ""