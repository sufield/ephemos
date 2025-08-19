# SPIRE Identity Verification and Diagnostics

This example demonstrates how to use Ephemos' SPIRE identity verification and diagnostics capabilities that leverage SPIRE's built-in mechanisms rather than implementing custom verification logic from scratch.

## Overview

Ephemos provides two main capabilities for SPIRE integration:

1. **Identity Verification**: Uses go-spiffe/v2 library to verify SPIFFE identities through the Workload API
2. **Diagnostics**: Uses SPIRE's built-in CLI tools to gather diagnostic information about the SPIRE infrastructure

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ephemos CLI                                  │
├─────────────────────────────────────────────────────────────────┤
│  verify command          │         diagnose command            │
│  ├─ identity            │         ├─ server                   │
│  ├─ current             │         ├─ agent                    │
│  ├─ connection          │         ├─ entries                  │
│  └─ refresh             │         ├─ bundles                  │
│                         │         ├─ agents                   │
│                         │         └─ version                  │
├─────────────────────────────────────────────────────────────────┤
│         Identity Verification    │    Diagnostics Provider     │
│         (go-spiffe/v2)           │    (SPIRE CLI Integration)  │
├─────────────────────────────────────────────────────────────────┤
│              SPIRE Workload API  │    SPIRE CLI Tools          │
│              ├─ GetX509SVID      │    ├─ spire-server          │
│              ├─ GetX509Bundle    │    │   ├─ entry show        │
│              └─ mTLS validation  │    │   ├─ bundle show       │
│                                  │    │   ├─ agent list       │
│                                  │    │   └─ healthcheck      │
│                                  │    └─ spire-agent          │
│                                  │        └─ healthcheck      │
└─────────────────────────────────────────────────────────────────┘
│                                                                 │
│                    SPIRE Infrastructure                         │
│              ┌─────────────────────────────────┐                │
│              │          SPIRE Server           │                │
│              │  ├─ Registration Entries        │                │
│              │  ├─ Trust Bundles               │                │
│              │  └─ Agent Management            │                │
│              └─────────────────────────────────┘                │
│                            │                                    │
│              ┌─────────────────────────────────┐                │
│              │          SPIRE Agent            │                │
│              │  ├─ Workload API Socket         │                │
│              │  ├─ Certificate Management      │                │
│              │  └─ Identity Attestation        │                │
│              └─────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

## Key Features

### Identity Verification
- **Workload API Integration**: Uses go-spiffe/v2 for native SPIRE integration
- **Identity Validation**: Verify current workload identity against expected SPIFFE ID
- **mTLS Connection Testing**: Validate connections with SPIFFE identity verification
- **Trust Domain Validation**: Ensure identities belong to expected trust domains
- **Certificate Information**: Extract and display certificate details and key usage

### Diagnostics
- **SPIRE CLI Integration**: Leverages spire-server and spire-agent CLI commands
- **Server Health**: Monitor SPIRE server status, version, and configuration
- **Agent Health**: Monitor SPIRE agent status and workload connectivity
- **Registration Entries**: List and analyze workload registration entries
- **Trust Bundles**: Display trust bundle information and federation status
- **Agent Management**: List connected agents and their attestation status

## Running the Example

### Prerequisites

1. **SPIRE Infrastructure**: A running SPIRE server and agent
2. **Go Environment**: Go 1.24 or later
3. **Socket Access**: Permissions to access SPIRE sockets

### Basic Usage

```bash
# Run the comprehensive example
go run main.go

# Or use the CLI commands directly
go build ../../cmd/ephemos-cli
./ephemos-cli verify current
./ephemos-cli diagnose server
```

### Configuration

The example can be configured through environment variables or command-line flags:

```bash
# Configure socket paths
export SPIRE_AGENT_SOCKET="/tmp/spire-agent/public/api.sock"
export SPIRE_SERVER_SOCKET="/tmp/spire-server/private/api.sock"

# Configure trust domain
export SPIRE_TRUST_DOMAIN="example.org"

# Run with custom configuration
./ephemos-cli verify current --socket "$SPIRE_AGENT_SOCKET"
./ephemos-cli diagnose server --server-socket "$SPIRE_SERVER_SOCKET"
```

## Example Scenarios

### 1. Service Identity Verification

Verify that your service is running with the correct SPIFFE identity:

```go
// Create verifier with production configuration
config := &ports.VerificationConfig{
    WorkloadAPISocket: "unix:///tmp/spire-agent/public/api.sock",
    TrustDomain:       spiffeid.RequireTrustDomainFromString("prod.company.com"),
    AllowedSPIFFEIDs: []spiffeid.ID{
        spiffeid.RequireFromString("spiffe://prod.company.com/payment-service"),
    },
    Timeout: 30 * time.Second,
}

verifier, err := verification.NewSpireIdentityVerifier(config)
if err != nil {
    return fmt.Errorf("failed to create verifier: %w", err)
}
defer verifier.Close()

// Verify current identity
expectedID := spiffeid.RequireFromString("spiffe://prod.company.com/payment-service")
result, err := verifier.VerifyIdentity(ctx, expectedID)
if err != nil || !result.Valid {
    return fmt.Errorf("identity verification failed: %w", err)
}
```

### 2. Service-to-Service Connection Validation

Validate mTLS connections between services:

```go
// Validate connection to backend service
targetID := spiffeid.RequireFromString("spiffe://prod.company.com/database")
result, err := verifier.ValidateConnection(ctx, targetID, "database:5432")
if err != nil || !result.Valid {
    return fmt.Errorf("connection validation failed: %w", err)
}

log.Printf("Successfully validated connection to %s", targetID)
log.Printf("TLS version: %s", result.Details["tls_version"])
log.Printf("Cipher suite: %s", result.Details["cipher_suite"])
```

### 3. Infrastructure Health Monitoring

Monitor SPIRE infrastructure health:

```go
// Create diagnostics provider
provider := verification.NewSpireDiagnosticsProvider(&ports.DiagnosticsConfig{
    ServerSocketPath: "unix:///tmp/spire-server/private/api.sock",
    AgentSocketPath:  "unix:///tmp/spire-agent/public/api.sock",
    Timeout:          30 * time.Second,
})

// Check server health
serverDiag, err := provider.GetServerDiagnostics(ctx)
if err != nil {
    return fmt.Errorf("server health check failed: %w", err)
}

if serverDiag.Status != "running" {
    return fmt.Errorf("server is not healthy: %s", serverDiag.Status)
}

// Check registration entries
entries, err := provider.ListRegistrationEntries(ctx)
if err != nil {
    return fmt.Errorf("failed to list entries: %w", err)
}

log.Printf("SPIRE infrastructure healthy: %d registration entries", len(entries))
```

### 4. Trust Bundle Validation

Validate trust relationships:

```go
// Check trust bundle for federated domain
trustDomain := spiffeid.RequireTrustDomainFromString("partner.company.com")
bundleInfo, err := provider.ShowTrustBundle(ctx, trustDomain)
if err != nil {
    return fmt.Errorf("trust bundle check failed: %w", err)
}

if bundleInfo.Local == nil {
    return fmt.Errorf("no local trust bundle found")
}

log.Printf("Trust bundle validated: %d certificates, expires %s",
    bundleInfo.Local.CertificateCount, bundleInfo.Local.ExpiresAt)
```

## Integration Patterns

### Health Check Endpoints

```go
// HTTP health check endpoint
func healthCheckHandler(verifier *verification.SpireIdentityVerifier, 
                       provider *verification.SpireDiagnosticsProvider) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Check workload identity
        _, err := verifier.GetCurrentIdentity(ctx)
        if err != nil {
            http.Error(w, "Identity unavailable", http.StatusServiceUnavailable)
            return
        }
        
        // Check SPIRE agent
        _, err = provider.GetAgentDiagnostics(ctx)
        if err != nil {
            http.Error(w, "Agent unhealthy", http.StatusServiceUnavailable)
            return
        }
        
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    }
}
```

### Prometheus Metrics

```go
// Prometheus metrics collector
type SpireMetricsCollector struct {
    provider *verification.SpireDiagnosticsProvider
    // ... metric definitions
}

func (c *SpireMetricsCollector) Collect(ch chan<- prometheus.Metric) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Collect server metrics
    serverDiag, err := c.provider.GetServerDiagnostics(ctx)
    if err == nil && serverDiag.Entries != nil {
        ch <- prometheus.MustNewConstMetric(
            c.registrationEntriesTotal,
            prometheus.GaugeValue,
            float64(serverDiag.Entries.Total),
        )
    }
    
    // ... collect other metrics
}
```

### Circuit Breaker Pattern

```go
// Circuit breaker for SPIRE operations
type SpireCircuitBreaker struct {
    verifier *verification.SpireIdentityVerifier
    cb       *gobreaker.CircuitBreaker
}

func (s *SpireCircuitBreaker) VerifyIdentity(ctx context.Context, id spiffeid.ID) (*ports.IdentityVerificationResult, error) {
    result, err := s.cb.Execute(func() (interface{}, error) {
        return s.verifier.VerifyIdentity(ctx, id)
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*ports.IdentityVerificationResult), nil
}
```

## Best Practices

### 1. Error Handling

```go
// Graceful degradation when SPIRE is unavailable
result, err := verifier.VerifyIdentity(ctx, expectedID)
if err != nil {
    // Log error but don't fail the request
    log.Printf("SPIRE verification failed, continuing with basic auth: %v", err)
    return validateWithBasicAuth(ctx, req)
}

if !result.Valid {
    return fmt.Errorf("SPIFFE identity verification failed: %s", result.Message)
}
```

### 2. Caching

```go
// Cache identity verification results
type CachedVerifier struct {
    verifier *verification.SpireIdentityVerifier
    cache    *ttlcache.Cache
}

func (c *CachedVerifier) VerifyIdentity(ctx context.Context, id spiffeid.ID) (*ports.IdentityVerificationResult, error) {
    cacheKey := id.String()
    
    if cached := c.cache.Get(cacheKey); cached != nil {
        return cached.(*ports.IdentityVerificationResult), nil
    }
    
    result, err := c.verifier.VerifyIdentity(ctx, id)
    if err == nil && result.Valid {
        // Cache successful verifications for a short time
        c.cache.Set(cacheKey, result, 5*time.Minute)
    }
    
    return result, err
}
```

### 3. Timeout Configuration

```go
// Production timeout configuration
config := &ports.VerificationConfig{
    WorkloadAPISocket: "unix:///tmp/spire-agent/public/api.sock",
    Timeout:           10 * time.Second, // Conservative timeout for production
}

// Use context with timeout for operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := verifier.VerifyIdentity(ctx, expectedID)
```

### 4. Security Considerations

```go
// Validate trust domain in production
config := &ports.VerificationConfig{
    TrustDomain: spiffeid.RequireTrustDomainFromString("prod.company.com"),
    AllowedSPIFFEIDs: []spiffeid.ID{
        // Explicitly list allowed SPIFFE IDs for security
        spiffeid.RequireFromString("spiffe://prod.company.com/api-server"),
        spiffeid.RequireFromString("spiffe://prod.company.com/auth-service"),
    },
}

// Always validate the trust domain
if result.TrustDomain.String() != "prod.company.com" {
    return fmt.Errorf("untrusted domain: %s", result.TrustDomain)
}
```

## Troubleshooting

### Common Issues

1. **Socket Permission Errors**
   ```bash
   # Fix socket permissions
   sudo chmod 666 /tmp/spire-agent/public/api.sock
   ```

2. **Connection Timeouts**
   ```bash
   # Increase timeout for slow environments
   ./ephemos-cli verify current --timeout 60s
   ```

3. **SPIRE CLI Not Found**
   ```bash
   # Ensure SPIRE binaries are in PATH
   export PATH="/opt/spire/bin:$PATH"
   which spire-server spire-agent
   ```

4. **Identity Not Available**
   ```bash
   # Check SPIRE agent logs
   journalctl -u spire-agent -f
   
   # Verify workload registration
   spire-server entry show
   ```

### Debug Mode

Enable debug logging:

```go
// Add debug logging to verifier
verifier, err := verification.NewSpireIdentityVerifier(config)
if err != nil {
    return err
}

// Enable debug mode (if supported)
log.SetLevel(logrus.DebugLevel)
```

## Related Documentation

- [CLI Usage Guide](CLI_USAGE.md) - Detailed CLI command examples
- [SPIRE Documentation](https://spiffe.io/docs/latest/spire/) - Official SPIRE documentation
- [go-spiffe Library](https://github.com/spiffe/go-spiffe) - Go SPIFFE library documentation
- [Ephemos Health Monitoring](../health_monitoring/) - Health monitoring examples