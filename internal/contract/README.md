# Port Contract Tests

This directory contains **behavioral contract tests** for each port interface. Contract tests ensure that all implementations of a port behave consistently for the same inputs/outputs.

## Philosophy

- **No mocks**: Test real implementations against behavioral contracts
- **Reusable suites**: One test suite per port, parameterized by factory
- **Multiple implementations**: Same suite runs against fakes, real adapters, testcontainer-backed implementations

## Structure

```
internal/contract/
├── identityprovider/
│   └── suite.go           # IdentityProvider contract suite
├── transportprovider/ 
│   └── suite.go           # TransportProvider contract suite
└── configurationprovider/
    └── suite.go           # ConfigurationProvider contract suite
```

## Usage Pattern

### 1. Contract Suite (Reusable)

```go
// internal/contract/identityprovider/suite.go
package identityprovider

func Run(t *testing.T, newImpl Factory) {
    t.Run("get service identity", func(t *testing.T) {
        provider := newImpl(t)
        defer provider.Close()
        
        identity, err := provider.GetServiceIdentity()
        // Test contract behavior...
    })
}
```

### 2. Adapter Test (Real Implementation)

```go  
// internal/adapters/secondary/spiffe/provider_contract_test.go
func TestSPIFFEProvider_Conformance(t *testing.T) {
    identityprovider.Run(t, func(t *testing.T) ports.IdentityProvider {
        return spiffe.NewProvider(nil)
    })
}
```

### 3. Fake Test (Fast, Hermetic)

```go
// internal/adapters/secondary/memidentity/provider_contract_test.go  
func TestMemIdentityProvider_Conformance(t *testing.T) {
    identityprovider.Run(t, func(t *testing.T) ports.IdentityProvider {
        return memidentity.New()
    })
}
```

## Available Contracts

### IdentityProvider

Tests that any `IdentityProvider` implementation:
- Returns valid service identities with proper fields
- Handles certificates and trust bundles correctly
- Implements idempotent `Close()` 
- Provides consistent results across multiple calls
- Handles error conditions gracefully

### TransportProvider  

Tests that any `TransportProvider` implementation:
- Creates functional servers and clients
- Rejects nil parameters appropriately
- Validates required dependencies
- Manages resource lifecycles properly

### ConfigurationProvider

Tests that any `ConfigurationProvider` implementation:
- Loads valid configurations successfully
- Rejects invalid configurations with errors
- Provides sensible defaults (if supported)
- Validates configuration structure
- Handles edge cases (empty paths, etc.)

## Running Contract Tests

### Fast Tests (Fakes)
```bash
# Run against in-memory fakes
go test ./internal/adapters/secondary/memidentity -v
```

### Integration Tests (Real Infrastructure)
```bash  
# Run against real SPIFFE (requires SPIRE running)
go test ./internal/adapters/secondary/spiffe -run Conformance -v
```

### All Contract Tests
```bash
# Run all contract compliance tests  
go test ./internal/adapters/... -run Conformance -v
```

## Contract Test Behavior

Contract tests verify **observable behavior**, not internal implementation:

✅ **Good**: "Returns valid identity with non-empty fields"  
❌ **Avoid**: "Calls X method exactly once"

✅ **Good**: "Idempotent Close() operations"  
❌ **Avoid**: "Close() sets internal closed flag"

## When to Use What

### Fakes (Fast, Hermetic)
- **Unit testing**: Core business logic tests
- **CI/CD pipelines**: Fast feedback loops
- **Contract validation**: Prove your fake behaves like real thing

### Real Adapters (High Confidence)  
- **Integration testing**: End-to-end scenarios with real infrastructure
- **Production validation**: Testcontainers with Redis/Postgres/SPIRE
- **Regression testing**: Catch infrastructure-specific issues

### Mocks (Rarely Needed)
- **Interaction testing**: When you must verify call sequences
- **Failure simulation**: Hard-to-reproduce error conditions
- **Legacy integration**: When dealing with unmockable third-party code

## Adding New Contracts

To add a contract for a new port:

1. **Create contract suite**:
   ```bash
   mkdir internal/contract/mynewport
   ```

2. **Define factory and suite**:
   ```go
   // internal/contract/mynewport/suite.go
   type Factory func(t *testing.T) ports.MyNewPort
   
   func Run(t *testing.T, newImpl Factory) {
       // Test contract behavior
   }
   ```

3. **Add conformance tests to adapters**:
   ```go
   // internal/adapters/*/mynewport/*_contract_test.go
   func TestMyAdapter_Conformance(t *testing.T) {
       mynewport.Run(t, func(t *testing.T) ports.MyNewPort {
           return myadapter.New()
       })
   }
   ```

## Benefits

1. **Consistent Behavior**: All implementations satisfy same contract
2. **Easy Extension**: New adapters get comprehensive test coverage automatically  
3. **Fast Feedback**: Fakes enable rapid TDD cycles
4. **High Confidence**: Real adapters catch infrastructure issues
5. **Regression Prevention**: Changes that break contracts fail immediately
6. **Living Documentation**: Contract tests document expected port behavior

## Examples in Codebase

- **IdentityProvider**: SPIFFE (real) vs MemIdentity (fake)
- **ConfigurationProvider**: FileProvider (real) vs InMemoryProvider (fake)  
- **TransportProvider**: gRPC (real) vs Mock (test double)

Each pair runs the same contract suite, proving behavioral equivalence.