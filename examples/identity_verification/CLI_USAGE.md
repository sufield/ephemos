# SPIRE Identity Verification and Diagnostics CLI Examples

This document provides examples of using **SPIRE's built-in CLI commands** for identity verification and diagnostics. Ephemos leverages SPIRE's native capabilities rather than implementing a custom CLI.

> **Note**: Ephemos does not provide its own CLI for SPIRE operations. All identity verification and diagnostics should be done using SPIRE's native `spire-server` and `spire-agent` CLI tools.

## Prerequisites

Ensure you have SPIRE CLI tools installed:
- `spire-server` - SPIRE Server CLI
- `spire-agent` - SPIRE Agent CLI

## Identity Verification Commands

### 1. Verify Current Workload Identity

Get the current workload identity from SPIRE using the Agent API:

```bash
# Show current agent SVID
spire-agent api fetch x509 \
    -socketPath /run/spire/sockets/agent.sock

# Show current agent SVID in JSON format
spire-agent api fetch x509 \
    -socketPath /run/spire/sockets/agent.sock \
    -write /dev/stdout
```

Example output:
```
Received 1 svid after 0.001s

SPIFFE ID:        spiffe://example.org/my-service
SVID Valid After:  2024-01-15 09:30:00 +0000 UTC
SVID Valid Until:  2024-01-15 11:30:00 +0000 UTC
CA #1 Valid After:  2024-01-14 00:00:00 +0000 UTC
CA #1 Valid Until:  2024-01-16 00:00:00 +0000 UTC
```

### 2. Validate JWT SVID

Fetch and validate JWT SVIDs:

```bash
# Fetch JWT SVID for a specific audience
spire-agent api fetch jwt \
    -socketPath /run/spire/sockets/agent.sock \
    -audience backend-service

# Validate a JWT token
spire-agent api validate jwt \
    -socketPath /run/spire/sockets/agent.sock \
    -svid "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
    -audience backend-service
```

### 3. Health Check

Check the health status of SPIRE components:

```bash
# Check agent health
spire-agent healthcheck \
    -socketPath /run/spire/sockets/agent.sock

# Check server health  
spire-server healthcheck \
    -socketPath /run/spire/sockets/registration.sock
```

## SPIRE Server Diagnostics Commands

### 1. List Registration Entries

List all registration entries:

```bash
# List all entries
spire-server entry show \
    -socketPath /run/spire/sockets/registration.sock

# List entries for a specific SPIFFE ID
spire-server entry show \
    -socketPath /run/spire/sockets/registration.sock \
    -spiffeID spiffe://example.org/my-service

# List entries with specific selector
spire-server entry show \
    -socketPath /run/spire/sockets/registration.sock \
    -selector unix:uid:1000

# Output in JSON format
spire-server entry show \
    -socketPath /run/spire/sockets/registration.sock \
    -output json
```

Example output:
```
Found 2 entries
Entry ID         : 12345678-1234-5678-1234-567812345678
SPIFFE ID        : spiffe://example.org/my-service
Parent ID        : spiffe://example.org/spire/agent/k8s_psat/cluster/demo/node/node1
Revision         : 0
TTL              : default
Selector         : k8s:ns:default
Selector         : k8s:sa:my-service
```

### 2. Show Trust Bundle

Display trust bundle information:

```bash
# Show bundle for the trust domain
spire-server bundle show \
    -socketPath /run/spire/sockets/registration.sock

# Show bundle in JSON format
spire-server bundle show \
    -socketPath /run/spire/sockets/registration.sock \
    -format json

# Show federated bundles
spire-server bundle list \
    -socketPath /run/spire/sockets/registration.sock \
    -format spiffe
```

### 3. List Agents

List all registered agents:

```bash
# List all agents
spire-server agent list \
    -socketPath /run/spire/sockets/registration.sock

# Show specific agent
spire-server agent show \
    -socketPath /run/spire/sockets/registration.sock \
    -spiffeID spiffe://example.org/spire/agent/k8s_psat/cluster/demo/node/node1

# Output in JSON format
spire-server agent list \
    -socketPath /run/spire/sockets/registration.sock \
    -output json
```

Example output:
```
Found 3 agents:

SPIFFE ID         : spiffe://example.org/spire/agent/k8s_psat/cluster/demo/node/node1
Attestation type  : k8s_psat
Expiration time   : 2024-01-16 10:30:00 +0000 UTC
Serial number     : 123456789
Can re-attest     : true
```

### 4. Create Registration Entry

Register a new workload:

```bash
# Create a new registration entry
spire-server entry create \
    -socketPath /run/spire/sockets/registration.sock \
    -spiffeID spiffe://example.org/new-service \
    -parentID spiffe://example.org/spire/agent/k8s_psat/cluster/demo/node/node1 \
    -selector k8s:ns:default \
    -selector k8s:sa:new-service \
    -ttl 3600
```

### 5. Delete Registration Entry

Delete a registration entry:

```bash
# Delete by entry ID
spire-server entry delete \
    -socketPath /run/spire/sockets/registration.sock \
    -entryID 12345678-1234-5678-1234-567812345678
```

## SPIRE Agent Diagnostics Commands

### 1. Show Agent SVID

Display the agent's own SVID:

```bash
# Show agent SVID
spire-agent api fetch x509 \
    -socketPath /run/spire/sockets/agent.sock \
    -silent
```

### 2. List Cached Entries

Show entries cached by the agent:

```bash
# This information is available through debug endpoints
# when the agent is started with appropriate configuration
curl http://localhost:8080/debug/entries
```

## Automation Scripts

### Health Check Script

Example script for monitoring SPIRE health:

```bash
#!/bin/bash

# check_spire_health.sh - Monitor SPIRE infrastructure health

AGENT_SOCKET="${SPIRE_AGENT_SOCKET:-/run/spire/sockets/agent.sock}"
SERVER_SOCKET="${SPIRE_SERVER_SOCKET:-/run/spire/sockets/registration.sock}"

echo "Checking SPIRE infrastructure health..."

# Check server health
if spire-server healthcheck -socketPath "$SERVER_SOCKET" >/dev/null 2>&1; then
    echo "✅ SPIRE server is healthy"
else
    echo "❌ SPIRE server check failed"
    exit 1
fi

# Check agent health
if spire-agent healthcheck -socketPath "$AGENT_SOCKET" >/dev/null 2>&1; then
    echo "✅ SPIRE agent is healthy"
else
    echo "❌ SPIRE agent check failed"
    exit 1
fi

# Verify workload API is accessible
if spire-agent api fetch x509 -socketPath "$AGENT_SOCKET" -silent >/dev/null 2>&1; then
    echo "✅ Workload API is accessible"
else
    echo "❌ Workload API check failed"
    exit 1
fi

echo "🎉 All SPIRE health checks passed!"
```

### Identity Verification Script

Example script to verify workload identity:

```bash
#!/bin/bash

# verify_identity.sh - Verify workload has expected SPIFFE ID

EXPECTED_SPIFFE_ID="${1:-spiffe://example.org/my-service}"
AGENT_SOCKET="${SPIRE_AGENT_SOCKET:-/run/spire/sockets/agent.sock}"

echo "Verifying workload identity..."

# Fetch current SVID
CURRENT_ID=$(spire-agent api fetch x509 -socketPath "$AGENT_SOCKET" -silent | \
    grep "SPIFFE ID:" | awk '{print $3}')

if [ "$CURRENT_ID" = "$EXPECTED_SPIFFE_ID" ]; then
    echo "✅ Identity verified: $CURRENT_ID"
    exit 0
else
    echo "❌ Identity mismatch!"
    echo "   Expected: $EXPECTED_SPIFFE_ID"
    echo "   Got: $CURRENT_ID"
    exit 1
fi
```

## Integration with Monitoring Systems

### Prometheus Metrics Collection

SPIRE Server and Agent expose Prometheus metrics:

```yaml
# prometheus.yml configuration
scrape_configs:
  - job_name: 'spire-server'
    static_configs:
      - targets: ['localhost:8088']
    
  - job_name: 'spire-agent'
    static_configs:
      - targets: ['localhost:8089']
```

### JSON Output for Automation

Most SPIRE commands support JSON output for automation:

```bash
# Get entries in JSON for processing
spire-server entry show \
    -socketPath /run/spire/sockets/registration.sock \
    -output json | jq '.entries[] | select(.spiffe_id | contains("my-service"))'

# Get agent list in JSON
spire-server agent list \
    -socketPath /run/spire/sockets/registration.sock \
    -output json | jq '.agents[] | {id: .id.trust_domain, expires: .x509svid_expires_at}'
```

## Ephemos Configuration Validation

While Ephemos doesn't provide identity verification CLI, it does provide a configuration validator:

```bash
# Validate Ephemos configuration
./config-validator --config ephemos.yaml --production

# Validate using environment variables only
./config-validator --env-only --production

# Output in JSON format
./config-validator --config ephemos.yaml --format json
```

## Security Considerations

1. **Socket Permissions**: Ensure proper file permissions on SPIRE sockets (typically 0660)
2. **Socket Paths**: Use absolute paths for socket locations
3. **mTLS**: All SPIRE communications use mTLS by default
4. **Logging**: Avoid logging sensitive information like private keys
5. **Access Control**: Restrict access to SPIRE server admin socket

## Troubleshooting

### Common Issues

1. **Permission Denied on Socket**
   ```bash
   # Check socket permissions
   ls -la /run/spire/sockets/
   # Add user to spire group if needed
   sudo usermod -a -G spire $USER
   ```

2. **Socket Not Found**
   ```bash
   # Verify SPIRE is running
   systemctl status spire-agent
   systemctl status spire-server
   ```

3. **No SVID Available**
   ```bash
   # Check if workload is registered
   spire-server entry show -socketPath /run/spire/sockets/registration.sock
   ```

4. **Connection Refused**
   ```bash
   # Check if services are listening
   ss -tlnp | grep spire
   netstat -tlnp | grep spire
   ```

## Additional Resources

- [SPIRE Documentation](https://spiffe.io/docs/latest/spire/)
- [SPIRE CLI Reference](https://spiffe.io/docs/latest/spire/using/spire_cli/)
- [SPIFFE Specifications](https://spiffe.io/docs/latest/spiffe/)
- [Ephemos Configuration Guide](../../docs/configuration.md)