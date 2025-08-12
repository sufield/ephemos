# Integration Tests

This package contains vertical integration tests that prove the identity story works end-to-end.

## Test Coverage

### Identity Story End-to-End (`identity_vertical_test.go`)
- Service identity creation and validation
- Server startup with identity-based authentication  
- Client connection with identity verification
- Authenticated communication between services
- Rejection of unauthorized clients
- Identity metadata propagation
- Concurrent multi-client scenarios
- Identity lifecycle management

### Test Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Test Client   │────│  gRPC Channel   │────│   Test Server   │
│                 │    │   (mTLS/SPIFFE) │    │                 │
│ • Identity      │    │                 │    │ • Identity      │
│ • Certificates  │    │ • Auth Metadata │    │ • Auth Policy   │
│ • Trust Policy  │    │ • Encryption    │    │ • Service Logic │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         v                        v                        v
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ SPIFFE Provider │    │  Interceptors   │    │ Identity Service│
│ (In-Memory)     │    │                 │    │                 │
│ • Mock SPIRE    │    │ • Auth Check    │    │ • Cert Mgmt     │
│ • Test Certs    │    │ • Identity Prop │    │ • Trust Bundle  │
│ • Trust Bundle  │    │ • Metrics/Logs  │    │ • Policy Eval   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Running Tests

```bash
# Run all integration tests
go test -v ./internal/integration/...

# Run specific test
go test -v ./internal/integration/ -run TestIdentityStoryEndToEnd

# Run with race detection
go test -race -v ./internal/integration/...

# Run with coverage
go test -cover -v ./internal/integration/...
```

## Test Scenarios

### 1. Happy Path Authentication
- Server starts with valid identity
- Authorized client connects successfully  
- gRPC calls succeed with proper identity metadata

### 2. Authorization Enforcement
- Unauthorized clients are rejected at connection time
- Authentication failures return appropriate error codes
- No service logic is executed for failed auth

### 3. Concurrent Operations
- Multiple clients with different identities
- Concurrent connections and calls
- No interference between client sessions

### 4. Lifecycle Management
- Identity validation edge cases
- Connection resilience under load
- Graceful error handling and recovery

## Test Dependencies

The integration tests use:
- In-memory identity providers (no external SPIRE required)
- Mock gRPC services for end-to-end validation
- Configurable authentication policies
- Dynamic port allocation for parallel test execution

## Security Validation

These tests prove:
- ✅ Identity-based authentication works end-to-end
- ✅ Unauthorized access is properly blocked
- ✅ Certificate-based mTLS is enforced  
- ✅ Identity metadata flows correctly
- ✅ Concurrent access is properly isolated
- ✅ Error handling maintains security invariants