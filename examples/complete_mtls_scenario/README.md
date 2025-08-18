# Complete mTLS Scenario Example

This example demonstrates Ephemos' complete mTLS capabilities including invariant enforcement, connection management, and rotation continuity in a realistic multi-service scenario.

## What This Example Demonstrates

### 🔒 Complete Security Stack
- **Service Identity Management**: Secure SPIFFE-based service identities
- **mTLS Connection Management**: Managed connections with full lifecycle tracking
- **Invariant Enforcement**: Real-time security invariant monitoring and enforcement
- **Zero-Downtime Rotation**: Certificate rotation with service continuity
- **Authorization Policies**: Client and server authorization rules

### 🏗️ Architecture Components
- **API Server**: Main application service with client authorization
- **Auth Service**: Authentication service with server authorization
- **Connection Manager**: Handles all mTLS connections with state tracking
- **Invariant Enforcer**: Monitors 5 default security invariants continuously
- **Rotation Manager**: Coordinates certificate rotation with overlap periods

## Running the Example

```bash
cd examples/complete_mtls_scenario
go run main.go
```

## Example Output

The example will demonstrate:

1. **Service Setup**: Creating API server and auth service with mTLS enforcement
2. **Connection Establishment**: Secure inter-service mTLS connections
3. **Invariant Monitoring**: Real-time security invariant checking
4. **Rotation Continuity**: Zero-downtime certificate rotation
5. **Security Validation**: Comprehensive end-to-end security verification

## Security Invariants Enforced

### 1. Certificate Validity
- Ensures certificates are not expired
- Validates certificate structure and chains
- Checks certificate timing (not before/not after)

### 2. Mutual Authentication
- Verifies TLS handshake completion
- Ensures peer certificates are present
- Validates bidirectional authentication

### 3. Trust Domain Validation
- Confirms certificates belong to expected trust domains
- Validates trust domain consistency across connections
- Ensures proper trust boundaries

### 4. Certificate Rotation
- Monitors certificate expiry approaching
- Triggers rotation before expiration
- Validates rotation timing and frequency

### 5. Identity Matching
- Verifies certificate SPIFFE IDs match expected identities
- Ensures service identity consistency
- Validates identity-certificate binding

## Rotation Continuity Process

The example demonstrates the 4-phase rotation process:

### Phase 1: Preparation
- New certificate generation
- Validation of new credentials
- Observer notification of rotation start

### Phase 2: Overlap
- Both old and new certificates active simultaneously
- Gradual traffic migration
- Continuous health monitoring

### Phase 3: Validation
- New certificate stability verification
- Connection health checks
- Performance validation

### Phase 4: Completion
- Graceful shutdown of old certificate
- Cleanup and observer notification
- Statistics update

## Configuration Options

### Enforcement Policy
```go
policy := &services.EnforcementPolicy{
    FailOnViolation: true,              // Fail fast on violations
    CheckInterval:   30 * time.Second,  // Check frequency
    MaxViolations:   3,                 // Max violations before action
    ViolationAction: ActionCloseConnection, // Action on max violations
}
```

### Continuity Policy
```go
policy := &services.ContinuityPolicy{
    OverlapDuration:            3 * time.Second,  // Overlap period
    GracefulShutdownTimeout:    1 * time.Second,  // Shutdown timeout
    PreRotationPrepTime:        500 * time.Millisecond, // Prep time
    PostRotationValidationTime: 500 * time.Millisecond, // Validation time
    MaxConcurrentRotations:     2, // Concurrent rotation limit
}
```

## Service Authorization

### Server-side Authorization (API Server)
```go
AuthorizedClients: []string{
    "spiffe://production.company.com/web-client",
    "spiffe://production.company.com/mobile-client",
}
```

### Client-side Authorization (API Server)
```go
TrustedServers: []string{
    "spiffe://production.company.com/auth-service",
    "spiffe://production.company.com/db-proxy",
}
```

## Monitoring and Observability

The example shows how to:

### Monitor Connection Statistics
```go
stats := identityService.GetConnectionStats()
fmt.Printf("Total connections: %d\n", stats.TotalConnections)
```

### Check Invariant Status
```go
status := identityService.GetInvariantStatus(ctx)
for name, result := range status.InvariantResults {
    fmt.Printf("%s: %d pass, %d fail\n", name, result.PassCount, result.FailCount)
}
```

### Observe Rotation Events
```go
observer := &rotationObserver{}
identityService.AddRotationObserver(observer)
// Events: OnRotationStarted, OnRotationCompleted, OnRotationFailed
```

## Production Considerations

### Security Best Practices
- ✅ All connections use mTLS with certificate validation
- ✅ Invariants enforced continuously in real-time  
- ✅ Certificate rotation automated with zero downtime
- ✅ Authorization policies prevent unauthorized access
- ✅ Trust domain boundaries strictly enforced

### Performance Optimizations
- ✅ Connection pooling and reuse
- ✅ Efficient invariant checking with configurable intervals
- ✅ Concurrent rotation support with limits
- ✅ Observer pattern for decoupled event handling

### Operational Features
- ✅ Comprehensive metrics and statistics
- ✅ Structured logging for audit trails
- ✅ Configurable policies for different environments
- ✅ Graceful error handling and recovery

## Integration Points

This example demonstrates integration with:
- **SPIFFE/SPIRE**: For identity and certificate management
- **Transport Layer**: gRPC, HTTP, or custom protocols
- **Monitoring Systems**: Through metrics and observers
- **Service Mesh**: Compatible with Istio, Linkerd, Consul Connect
- **Container Orchestration**: Kubernetes, Docker Swarm support

## Next Steps

After running this example, explore:
1. The comprehensive integration tests in `internal/integration/`
2. The individual invariant implementations in `internal/core/services/`
3. The rotation continuity service details
4. Custom invariant and observer implementations
5. Production deployment configurations

This example provides a complete foundation for implementing zero-trust mTLS communication in production microservices environments.