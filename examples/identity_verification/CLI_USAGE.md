# SPIRE Identity Verification and Diagnostics CLI Examples

This document provides examples of using Ephemos CLI commands for SPIRE identity verification and diagnostics. These commands leverage SPIRE's built-in capabilities rather than implementing custom verification logic.

## Identity Verification Commands

### 1. Verify Current Workload Identity

Get the current workload identity from SPIRE using the Workload API:

```bash
# Get current identity with text output
ephemos verify current

# Get current identity with JSON output
ephemos verify current --format json

# Use custom Workload API socket
ephemos verify current --socket unix:///custom/spire-agent/api.sock
```

Example output:
```
🆔 Current Identity
SPIFFE ID: spiffe://example.org/my-service
Trust Domain: example.org
Source: workload-api
Fetched At: 2024-01-15T10:30:00Z
Has SVID: true
Has Trust Bundle: true
Certificate Expires: 2024-01-15T11:30:00Z
Certificate Serial: 123456789
```

### 2. Verify Specific SPIFFE Identity

Verify that the current workload identity matches an expected SPIFFE ID:

```bash
# Verify identity matches expected SPIFFE ID
ephemos verify identity spiffe://example.org/my-service

# Verify with custom trust domain validation
ephemos verify identity spiffe://example.org/my-service \
    --trust-domain example.org

# Verify with allowed SPIFFE IDs restriction
ephemos verify identity spiffe://example.org/my-service \
    --allowed-ids spiffe://example.org/my-service,spiffe://example.org/backup-service

# JSON output for automation
ephemos verify identity spiffe://example.org/my-service --format json
```

Example output:
```
✅ Identity Verification
Identity: spiffe://example.org/my-service
Trust Domain: example.org
Valid: true
Message: Identity verification successful
Verified At: 2024-01-15T10:30:00Z
Not Before: 2024-01-15T09:30:00Z
Not After: 2024-01-15T11:30:00Z
Serial Number: 123456789
Key Usage: [DigitalSignature KeyEncipherment]
```

### 3. Validate mTLS Connection

Validate a mutual TLS connection to another service with SPIFFE identity verification:

```bash
# Validate connection to another service
ephemos verify connection spiffe://example.org/backend-service localhost:8080

# Validate connection with custom timeout
ephemos verify connection spiffe://example.org/backend-service localhost:8080 \
    --timeout 10s

# Validate with custom socket
ephemos verify connection spiffe://example.org/backend-service localhost:8080 \
    --socket unix:///custom/spire-agent/api.sock
```

Example output:
```
✅ Identity Verification
Identity: spiffe://example.org/backend-service
Trust Domain: example.org
Valid: true
Message: Successfully validated connection to spiffe://example.org/backend-service
Verified At: 2024-01-15T10:30:00Z
Certificate Expires: 2024-01-15T11:30:00Z
TLS Version: TLS 1.3
Cipher Suite: TLS_AES_256_GCM_SHA384
```

### 4. Refresh Workload Identity

Force a refresh of the workload identity from SPIRE:

```bash
# Refresh identity
ephemos verify refresh

# Refresh with JSON output
ephemos verify refresh --format json
```

## SPIRE Diagnostics Commands

### 1. Server Diagnostics

Get comprehensive SPIRE server diagnostic information:

```bash
# Get server diagnostics using default socket
ephemos diagnose server

# Get server diagnostics with custom socket
ephemos diagnose server --server-socket unix:///custom/spire-server/api.sock

# Use server API instead of CLI commands
ephemos diagnose server --use-api --server-address https://spire-server:8081 \
    --api-token "bearer-token"

# JSON output for monitoring systems
ephemos diagnose server --format json
```

Example output:
```
🔍 spire-server Diagnostics
Version: 1.8.7
Status: running
Trust Domain: example.org
Collected At: 2024-01-15T10:30:00Z

Registration Entries:
  Total: 25
  Recent: 3
  By Selector:
    unix: 15
    docker: 8
    k8s: 2

Agents:
  Total: 5
  Active: 4
  Inactive: 1
  Banned: 0

Details:
  healthcheck: Server is healthy
```

### 2. Agent Diagnostics

Get comprehensive SPIRE agent diagnostic information:

```bash
# Get agent diagnostics
ephemos diagnose agent

# Get agent diagnostics with custom sockets
ephemos diagnose agent \
    --agent-socket unix:///custom/spire-agent/api.sock

# JSON output
ephemos diagnose agent --format json
```

Example output:
```
🔍 spire-agent Diagnostics
Version: 1.8.7
Status: running
Trust Domain: example.org
Collected At: 2024-01-15T10:30:00Z

Details:
  healthcheck: Agent is healthy
  workload_spiffe_id: spiffe://example.org/my-service
  certificate_expires_at: 2024-01-15T11:30:00Z
  certificate_serial: 123456789
```

### 3. List Registration Entries

List all registration entries using SPIRE CLI:

```bash
# List all registration entries
ephemos diagnose entries

# List with custom server socket
ephemos diagnose entries --server-socket unix:///custom/spire-server/api.sock

# JSON output for processing
ephemos diagnose entries --format json
```

Example output:
```
📋 Registration Entries (25 total)

1. ID: entry-12345
   SPIFFE ID: spiffe://example.org/web-service
   Parent ID: spiffe://example.org/spire/agent/node
   Selectors: [unix:uid:1000 docker:label:app:web]
   TTL: 3600s
   Admin: false
   Created: 2024-01-15T09:00:00Z

2. ID: entry-23456
   SPIFFE ID: spiffe://example.org/database
   Parent ID: spiffe://example.org/spire/agent/node
   Selectors: [unix:uid:1001 k8s:sa:database]
   TTL: 7200s
   Admin: false
   Created: 2024-01-15T08:30:00Z
...
```

### 4. Show Trust Bundle Information

Display trust bundle information using SPIRE CLI:

```bash
# Show trust bundle for default domain
ephemos diagnose bundles

# Show trust bundle for specific domain
ephemos diagnose bundles example.org

# JSON output
ephemos diagnose bundles example.org --format json
```

Example output:
```
🔐 Trust Bundle Information

Local Bundle:
  Trust Domain: example.org
  Certificate Count: 2
  Last Updated: 2024-01-15T10:00:00Z
  Expires At: 2024-01-16T10:00:00Z

Federated Bundles:
  partner.org:
    Certificate Count: 1
    Last Updated: 2024-01-15T09:30:00Z
    Expires At: 2024-01-16T09:30:00Z
```

### 5. List Connected Agents

List all connected SPIRE agents:

```bash
# List all agents
ephemos diagnose agents

# List with custom server socket
ephemos diagnose agents --server-socket unix:///custom/spire-server/api.sock

# JSON output
ephemos diagnose agents --format json
```

Example output:
```
🤖 SPIRE Agents (5 total)

1. ID: spiffe://example.org/spire/agent/node-1
   Attestation Type: node
   Serial Number: agent-cert-123
   Expires At: 2024-01-16T10:30:00Z
   Banned: false
   Can Reattest: true
   Selectors: [node:hostname:node-1]

2. ID: spiffe://example.org/spire/agent/node-2
   Attestation Type: node
   Serial Number: agent-cert-456
   Expires At: 2024-01-16T10:45:00Z
   Banned: false
   Can Reattest: true
   Selectors: [node:hostname:node-2]
...
```

### 6. Get Component Versions

Get the version of SPIRE components:

```bash
# Get server version
ephemos diagnose version spire-server

# Get agent version
ephemos diagnose version spire-agent

# JSON output
ephemos diagnose version spire-server --format json
```

Example output:
```
spire-server version: 1.8.7
```

## Global Options

All commands support these global options:

- `--format`: Output format (`text` or `json`)
- `--quiet`: Suppress non-essential output
- `--no-emoji`: Disable emoji in output (useful for logs)
- `--timeout`: Global timeout for operations (default: 30s)

## Integration with Monitoring Systems

### Prometheus Integration

Use JSON output with monitoring systems:

```bash
# Script for Prometheus metrics collection
#!/bin/bash

# Get server health
ephemos diagnose server --format json --quiet > /tmp/spire-server.json

# Get agent health
ephemos diagnose agent --format json --quiet > /tmp/spire-agent.json

# Process JSON and expose metrics
python3 process_spire_metrics.py
```

### Automation Scripts

Example automation script:

```bash
#!/bin/bash

echo "Running SPIRE infrastructure health check..."

# Check server
if ephemos diagnose server --quiet; then
    echo "✅ SPIRE server is healthy"
else
    echo "❌ SPIRE server check failed"
    exit 1
fi

# Check agent
if ephemos diagnose agent --quiet; then
    echo "✅ SPIRE agent is healthy"
else
    echo "❌ SPIRE agent check failed"
    exit 1
fi

# Verify our identity
if ephemos verify current --quiet; then
    echo "✅ Workload identity is available"
else
    echo "❌ Workload identity check failed"
    exit 1
fi

echo "🎉 All SPIRE infrastructure checks passed!"
```

## Security Considerations

1. **Socket Permissions**: Ensure proper file permissions on SPIRE sockets
2. **Network Access**: Validate network connectivity for connection tests
3. **Timeouts**: Set appropriate timeouts for production environments
4. **Authentication**: Use API tokens when accessing SPIRE server APIs
5. **Logging**: Be careful not to log sensitive identity information

## Troubleshooting

### Common Issues

1. **Socket Not Found**: Check SPIRE agent/server is running and socket path is correct
2. **Permission Denied**: Ensure process has access to SPIRE sockets
3. **Connection Timeout**: Verify network connectivity and increase timeout
4. **Identity Not Available**: Confirm workload is registered in SPIRE
5. **CLI Command Failed**: Check SPIRE CLI tools are installed and accessible

### Debug Mode

Enable verbose output for troubleshooting:

```bash
# Run with verbose output
ephemos --timeout 60s verify current --format json

# Check logs for additional information
journalctl -u spire-agent -f
journalctl -u spire-server -f
```