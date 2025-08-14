# Ephemos Interactive Demo Guide

This guide walks you through the Ephemos demo step-by-step, allowing you to run each component in separate terminals to see the real-time output and understand how identity-based authentication works.

## Prerequisites

Open 4 terminal windows/tabs and navigate to the project root (`~/work/ephemos`) in each.

## Terminal 1: SPIRE Setup and Monitoring

### Step 1: Install SPIRE (if needed)
```bash
# Check if SPIRE is installed
spire-server version 2>/dev/null || echo "Not installed"

# Install SPIRE (skip if already installed)
cd scripts/demo
./install-spire.sh

# For forced reinstall:
# ./install-spire.sh --force

# For specific version:
# ./install-spire.sh --version 1.9.0
```

### Step 2: Start SPIRE Services
```bash
# Start SPIRE server and agent
./start-spire.sh

# Verify services are running
sudo systemctl status spire-server --no-pager
sudo systemctl status spire-agent --no-pager
```

### Step 3: Keep monitoring SPIRE logs
```bash
# Monitor SPIRE server logs
sudo journalctl -u spire-server -f
```

## Terminal 2: Service Registration

### Step 1: Build the Ephemos CLI
```bash
# Build the CLI tool
make build
```

### Step 2: Register Services with SPIRE
```bash
# Register echo-server
sudo bin/ephemos register --name echo-server --domain example.org

# Register echo-client  
sudo bin/ephemos register --name echo-client --domain example.org

# Verify registrations
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock
```

You should see entries like:
```
Entry ID         : <UUID>
SPIFFE ID        : spiffe://example.org/echo-server
Parent ID        : spiffe://example.org/spire-agent
Revision         : 0
X509-SVID TTL    : default
JWT-SVID TTL     : default
Selector         : unix:uid:0

Entry ID         : <UUID>
SPIFFE ID        : spiffe://example.org/echo-client
Parent ID        : spiffe://example.org/spire-agent
Revision         : 0
X509-SVID TTL    : default
JWT-SVID TTL     : default
Selector         : unix:uid:0
```

## Terminal 3: Echo Server

### Step 1: Build the Echo Server
```bash
# Generate protobuf code
make build

# Build example applications
make examples
```

### Step 2: Start the Echo Server
```bash
# Run the server with identity-based authentication
EPHEMOS_CONFIG=config/echo-server.yaml bin/echo-server
```

Expected output:
```
time=2025-08-08T23:44:55.036-04:00 level=INFO msg="Echo server starting" address=:50051 service=echo-server
```

The server will:
1. Connect to SPIRE agent at `/tmp/spire-agent/public/api.sock`
2. Obtain its SPIFFE identity (`spiffe://example.org/echo-server`)
3. Start listening on port 50051 with mTLS
4. Only accept connections from authorized clients (`spiffe://example.org/echo-client`)

Keep this terminal open to see incoming requests.

## Terminal 4: Echo Client

### Step 1: Run the Echo Client
```bash
# Run the client
EPHEMOS_CONFIG=config/echo-client.yaml bin/echo-client
```

Expected output:
```
time=2025-08-08T23:45:00.123-04:00 level=INFO msg="Connected to echo server" address=localhost:50051
time=2025-08-08T23:45:00.234-04:00 level=INFO msg="Echo response received" request=1 message="Hello from echo-client! Request #1" from=echo-server
time=2025-08-08T23:45:02.345-04:00 level=INFO msg="Echo response received" request=2 message="Hello from echo-client! Request #2" from=echo-server
...
time=2025-08-08T23:45:10.567-04:00 level=INFO msg="Echo client completed successfully"
```

In Terminal 3 (server), you should see:
```
time=2025-08-08T23:45:00.234-04:00 level=INFO msg="Processing echo request" message="Hello from echo-client! Request #1"
time=2025-08-08T23:45:02.345-04:00 level=INFO msg="Processing echo request" message="Hello from echo-client! Request #2"
...
```

### Step 2: Test Authentication Failure

Remove the client's SPIFFE registration:
```bash
# Remove client registration
sudo spire-server entry delete \
    -socketPath /tmp/spire-server/private/api.sock \
    -spiffeID spiffe://example.org/echo-client

# Try to run client again (should fail)
EPHEMOS_CONFIG=config/echo-client.yaml bin/echo-client
```

Expected error:
```
time=2025-08-08T23:46:00.123-04:00 level=ERROR msg="Failed to create identity client" error="failed to obtain SVID: no SVID available"
```

## Understanding the Flow

### What's Happening Behind the Scenes

1. **Identity Provisioning**:
   - Each service gets a SPIFFE ID (e.g., `spiffe://example.org/echo-server`)
   - SPIRE automatically issues and rotates X.509 certificates
   - Certificates are used for mTLS authentication

2. **Server Side** (`ephemos.NewIdentityServer`):
   - Fetches its SVID from SPIRE
   - Creates a gRPC server with TLS using the SVID
   - Validates client certificates against authorized list

3. **Client Side** (`ephemos.NewIdentityClient`):
   - Fetches its SVID from SPIRE  
   - Creates a gRPC client with TLS using the SVID
   - Presents certificate when connecting to servers

4. **Zero-Trust Security**:
   - No hardcoded credentials or API keys
   - Identity verified cryptographically via mTLS
   - Authorization based on SPIFFE IDs

## Configuration Files

### Server Config (`config/echo-server.yaml`)
```yaml
service:
  name: "echo-server"
  domain: "example.org"
  
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"
  
authorized_clients:
  - "spiffe://example.org/echo-client"
```

### Client Config (`config/echo-client.yaml`)
```yaml
service:
  name: "echo-client" 
  domain: "example.org"
  
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"
  
trusted_servers:
  - "spiffe://example.org/echo-server"
```

## Troubleshooting

### Server not starting?
```bash
# Check SPIRE agent is running
sudo systemctl status spire-agent

# Check socket exists
ls -la /tmp/spire-agent/public/api.sock

# Check server registration
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock | grep echo-server
```

### Client can't connect?
```bash
# Check server is listening
ss -tln | grep 50051

# Check client registration
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock | grep echo-client

# Test basic connectivity
nc -zv localhost 50051
```

### SPIRE issues?
```bash
# Restart SPIRE services
sudo systemctl restart spire-server
sudo systemctl restart spire-agent

# Check logs
sudo journalctl -u spire-server -n 50
sudo journalctl -u spire-agent -n 50
```

## Cleanup

When done with the demo:
```bash
# Stop services
pkill -f echo-server
pkill -f echo-client

# Stop SPIRE (optional)
sudo systemctl stop spire-agent
sudo systemctl stop spire-server

# Remove SPIRE entries (optional)
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-server
sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -spiffeID spiffe://example.org/echo-client
```

## Key Takeaways

1. **Simple API**: Developers only need to call `ephemos.NewIdentityServer()` and `ephemos.NewIdentityClient()`
2. **Automatic Security**: mTLS, certificate rotation, and identity verification handled transparently
3. **Zero Configuration**: No need to manage certificates, keys, or credentials manually
4. **Production Ready**: Based on SPIFFE/SPIRE, used by major companies for workload identity

## Next Steps

- Explore the code in `examples/echo-server/` and `examples/echo-client/`
- Read about SPIFFE: https://spiffe.io/
- Learn about SPIRE: https://spire.io/
- Customize the authorization policies in your config files