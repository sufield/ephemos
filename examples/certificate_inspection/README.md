# SPIRE Certificate and Trust Bundle Inspection

This example demonstrates how to inspect SPIRE certificates and trust bundles using SPIRE's built-in CLI tools and go-spiffe/v2 SDK rather than implementing custom inspection logic from scratch.

## Overview

SPIRE provides comprehensive built-in capabilities for certificate and trust bundle inspection:

1. **SPIRE CLI Tools**: Native commands for detailed inspection
2. **go-spiffe/v2 SDK**: Programmatic access through official library
3. **Built-in APIs**: REST and gRPC APIs for automated inspection

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ephemos CLI                                  │
├─────────────────────────────────────────────────────────────────┤
│  inspect command                                                │
│  ├─ svid              (X.509 SVID inspection)                  │
│  ├─ bundle            (Trust bundle inspection)                │
│  └─ authorities       (Local CA inspection)                    │
├─────────────────────────────────────────────────────────────────┤
│         Direct Integration with SPIRE Tools                    │
│  ┌─────────────────────┬─────────────────────────────────────┐ │
│  │   go-spiffe/v2 SDK  │       SPIRE CLI Tools               │ │
│  │  ├─ X509Source      │  ├─ spire-agent api fetch x509      │ │
│  │  ├─ X509Bundle      │  ├─ spire-server bundle show        │ │
│  │  └─ Workload API    │  └─ spire-server localauthority     │ │
│  └─────────────────────┴─────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
│                                                                 │
│                    SPIRE Infrastructure                         │
│              ┌─────────────────────────────────┐                │
│              │          SPIRE Server           │                │
│              │  ├─ Certificate Authority       │                │
│              │  ├─ Trust Bundle Management     │                │
│              │  └─ Registration Database       │                │
│              └─────────────────────────────────┘                │
│                            │                                    │
│              ┌─────────────────────────────────┐                │
│              │          SPIRE Agent            │                │
│              │  ├─ Workload API Socket         │                │
│              │  ├─ X.509 SVID Cache            │                │
│              │  └─ Certificate Rotation        │                │
│              └─────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. **SVID Inspection using Built-in Tools**

#### Using go-spiffe/v2 SDK (Recommended)
```bash
# Inspect current X.509 SVID programmatically
ephemos inspect svid

# JSON output for automation
ephemos inspect svid --format json
```

#### Using SPIRE CLI Tools
```bash
# Use SPIRE's native CLI commands
ephemos inspect svid --use-cli

# Direct SPIRE command (what Ephemos calls internally)
spire-agent api fetch x509 -socketPath /tmp/spire-agent/public/api.sock
```

### 2. **Trust Bundle Inspection using Built-in Tools**

#### Using go-spiffe/v2 SDK
```bash
# Inspect trust bundle for current domain
ephemos inspect bundle

# Inspect specific trust domain
ephemos inspect bundle example.org

# JSON output
ephemos inspect bundle --format json
```

#### Using SPIRE CLI Tools
```bash
# Use SPIRE's native bundle commands
ephemos inspect bundle --use-cli

# Direct SPIRE command
spire-server bundle show -format pem
```

### 3. **Certificate Authority Inspection**

```bash
# Inspect local X.509 authorities
ephemos inspect authorities

# Use SPIRE's native command
spire-server localauthority x509 show
```

## Command Examples

### Basic SVID Inspection

```bash
# Get current SVID information
$ ephemos inspect svid

📋 X.509 SVID Information
SPIFFE ID: spiffe://example.org/my-service
Trust Domain: example.org
Certificate Count: 1
Has Private Key: true
Valid From: 2024-01-15T10:00:00Z
Valid Until: 2024-01-15T11:00:00Z
Serial Number: 123456789
Subject: CN=example.org/my-service
Issuer: CN=example.org SPIRE CA
```

### Trust Bundle Inspection

```bash
# Get trust bundle information
$ ephemos inspect bundle

🔐 Trust Bundle Information
Trust Domain: example.org
Certificate Count: 2

Authority 1:
  Subject: CN=example.org SPIRE CA
  Serial: 987654321
  Valid From: 2024-01-01T00:00:00Z
  Valid Until: 2025-01-01T00:00:00Z
  Is CA: true

Authority 2:
  Subject: CN=example.org SPIRE Intermediate CA
  Serial: 456789123
  Valid From: 2024-01-01T00:00:00Z
  Valid Until: 2024-07-01T00:00:00Z
  Is CA: true
```

### JSON Output for Automation

```bash
# SVID inspection with JSON output
$ ephemos inspect svid --format json
{
  "spiffe_id": "spiffe://example.org/my-service",
  "trust_domain": "example.org",
  "certificate_count": 1,
  "has_private_key": true,
  "not_before": "2024-01-15T10:00:00Z",
  "not_after": "2024-01-15T11:00:00Z",
  "serial_number": "123456789",
  "subject": "CN=example.org/my-service",
  "issuer": "CN=example.org SPIRE CA"
}
```

## Integration with Existing SPIRE Tools

### 1. **SPIRE Agent API Commands**

Ephemos leverages these built-in SPIRE commands:

```bash
# Fetch X.509 SVID (what Ephemos uses internally)
spire-agent api fetch x509 -socketPath /tmp/spire-agent/public/api.sock

# Watch for SVID updates
spire-agent api watch -socketPath /tmp/spire-agent/public/api.sock

# Fetch JWT SVID
spire-agent api fetch jwt -audience my-service -socketPath /tmp/spire-agent/public/api.sock
```

### 2. **SPIRE Server Bundle Commands**

```bash
# Show trust bundle (what Ephemos uses internally)
spire-server bundle show -format pem

# List federated bundles
spire-server bundle list

# Count bundles
spire-server bundle count
```

### 3. **SPIRE Server Authority Commands**

```bash
# Show local authorities (what Ephemos uses internally)
spire-server localauthority x509 show

# Show JWT authorities
spire-server localauthority jwt show
```

## Advanced Usage

### 1. **Certificate Chain Analysis**

```bash
# For detailed certificate analysis, use SPIRE's fetch command to save files
spire-agent api fetch x509 -write /tmp/certs/ -socketPath /tmp/spire-agent/public/api.sock

# Then analyze with standard tools
openssl x509 -in /tmp/certs/svid.0.pem -text -noout
openssl verify -CAfile /tmp/certs/bundle.0.pem /tmp/certs/svid.0.pem
```

### 2. **Bundle Verification**

```bash
# Verify trust bundle integrity
spire-server bundle show -format pem | openssl x509 -noout -text

# Check federated bundles
spire-server bundle list | jq '.[]'
```

### 3. **Monitoring and Automation**

```bash
# Monitor SVID expiration
ephemos inspect svid --format json | jq -r '.not_after'

# Check bundle health
ephemos inspect bundle --format json | jq '.certificate_count'

# Automated health checks
#!/bin/bash
EXPIRES=$(ephemos inspect svid --format json | jq -r '.not_after')
if [[ $(date -d "$EXPIRES" +%s) -lt $(date -d "+1 hour" +%s) ]]; then
    echo "SVID expires soon: $EXPIRES"
fi
```

## Configuration Options

### Command Line Flags

```bash
# Socket configuration
--socket string           Workload API socket path (default: unix:///tmp/spire-agent/public/api.sock)
--server-socket string    SPIRE server socket path (default: unix:///tmp/spire-server/private/api.sock)

# Output options
--format string           Output format (text|json) (default: text)
--quiet                   Suppress non-essential output
--no-emoji               Disable emoji in output

# Operation options
--timeout duration        Operation timeout (default: 30s)
--use-cli                Use SPIRE CLI commands instead of SDK
```

### Environment Variables

```bash
export SPIRE_AGENT_SOCKET="/tmp/spire-agent/public/api.sock"
export SPIRE_SERVER_SOCKET="/tmp/spire-server/private/api.sock"
```

## Comparison: Custom vs Built-in Approaches

### ❌ **Custom Implementation (Removed)**

Previously, Ephemos implemented custom certificate parsing:

```go
// REMOVED: Custom key usage extraction
func extractKeyUsage(cert *x509.Certificate) []string { ... }

// REMOVED: Custom TLS version parsing  
func tlsVersionString(version uint16) string { ... }

// REMOVED: Custom cipher suite mapping
func tlsCipherSuite(suite uint16) string { ... }
```

**Problems with custom approach:**
- Duplicates SPIRE's existing functionality
- Risk of inconsistencies with SPIRE's interpretation
- Maintenance burden for certificate parsing logic
- Missing features available in SPIRE CLI tools

### ✅ **Built-in Approach (Current)**

Now Ephemos leverages SPIRE's native capabilities:

```go
// Use go-spiffe/v2 SDK directly
source, err := workloadapi.NewX509Source(ctx, clientOptions)
svid, err := source.GetX509SVID()
bundle, err := source.GetX509BundleForTrustDomain(trustDomain)

// Use SPIRE CLI commands for detailed inspection
exec.Command("spire-agent", "api", "fetch", "x509")
exec.Command("spire-server", "bundle", "show")
```

**Benefits of built-in approach:**
- ✅ Consistent with SPIRE's canonical interpretation
- ✅ Automatically updated with new SPIRE features
- ✅ Reduced code complexity and maintenance
- ✅ Access to full SPIRE CLI feature set
- ✅ Better integration with SPIRE ecosystem

## Best Practices

### 1. **Use SDK for Programmatic Access**

```go
// Recommended: Use go-spiffe/v2 for programmatic access
source, err := workloadapi.NewX509Source(ctx, 
    workloadapi.WithClientOptions(
        workloadapi.WithAddr("unix:///tmp/spire-agent/public/api.sock"),
    ),
)
defer source.Close()

svid, err := source.GetX509SVID()
```

### 2. **Use CLI for Detailed Analysis**

```bash
# For detailed certificate inspection, use SPIRE CLI
spire-agent api fetch x509 -write /tmp/ -socketPath /tmp/spire-agent/public/api.sock

# Then use standard tools for analysis
openssl x509 -in /tmp/svid.0.pem -text -noout
```

### 3. **Combine Both Approaches**

```bash
# Use Ephemos for structured output
ephemos inspect svid --format json > svid-info.json

# Use SPIRE CLI for detailed analysis
ephemos inspect svid --use-cli > svid-details.txt
```

### 4. **Monitor Certificate Health**

```bash
# Regular SVID monitoring
*/5 * * * * ephemos inspect svid --format json | jq -r '.not_after' | xargs -I {} date -d {} +%s | awk '{if ($1 < systime() + 3600) print "SVID expires in 1 hour"}'

# Bundle health monitoring
0 6 * * * ephemos inspect bundle --format json | jq '.certificate_count' | awk '{if ($1 < 1) print "No trust bundle certificates found"}'
```

## Security Considerations

1. **Socket Permissions**: Ensure proper permissions on SPIRE sockets
2. **Certificate Validation**: Trust SPIRE's built-in validation over custom logic
3. **Automation Security**: Use JSON output for parsing in scripts
4. **Monitoring**: Regular inspection of certificate expiration
5. **Federation**: Verify federated trust bundles using SPIRE's tools

## Troubleshooting

### Common Issues

1. **Socket Access Errors**
   ```bash
   # Check socket permissions
   ls -la /tmp/spire-agent/public/api.sock
   sudo chmod 666 /tmp/spire-agent/public/api.sock
   ```

2. **SPIRE CLI Not Found**
   ```bash
   # Ensure SPIRE binaries are in PATH
   which spire-agent spire-server
   export PATH="/opt/spire/bin:$PATH"
   ```

3. **Connection Timeouts**
   ```bash
   # Increase timeout for slow environments
   ephemos inspect svid --timeout 60s
   ```

4. **Empty Results**
   ```bash
   # Check SPIRE agent status
   systemctl status spire-agent
   
   # Check Workload API connectivity
   spire-agent api fetch x509 -socketPath /tmp/spire-agent/public/api.sock
   ```

## Migration Guide

If upgrading from custom certificate inspection:

1. **Replace custom parsing calls**:
   ```bash
   # Old: Custom parsing in application code
   # New: Use Ephemos inspect commands
   ephemos inspect svid --format json
   ```

2. **Update monitoring scripts**:
   ```bash
   # Old: Custom certificate parsing
   # New: SPIRE native tools
   ephemos inspect svid --use-cli
   ```

3. **Leverage SPIRE CLI features**:
   ```bash
   # Access features not available in custom implementation
   spire-agent api watch
   spire-server bundle count
   ```

## Related Documentation

- [SPIRE Agent API Documentation](https://spiffe.io/docs/latest/spire/developing/getting-started-with-spire-agent-apis/)
- [SPIRE Server CLI Reference](https://spiffe.io/docs/latest/spire/using/cli-reference/)
- [go-spiffe/v2 Library Documentation](https://pkg.go.dev/github.com/spiffe/go-spiffe/v2)
- [Ephemos Identity Verification](../identity_verification/) - Related identity verification features