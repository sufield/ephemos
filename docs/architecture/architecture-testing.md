# Architecture Testing

This document describes the architectural testing approach used to maintain code quality and prevent design violations in the ephemos project.

## Overview
Architecture tests use static analysis to enforce design constraints and prevent regressions. These tests run as part of the normal test suite and will fail the build if violations are detected.

## Test Categories

### 1. Selector Chain Depth (`TestNoLongSelectorChains`)
Prevents excessive chaining of field/method accesses that violate the Law of Demeter.

**Limits:**
- Public API: 2 levels (e.g., `client.connection()`)
- Core packages: 3 levels (e.g., `service.config.value()`)  
- Adapters: 4 levels (e.g., `adapter.config.section.field()`)

**Example Violation:**
```go
// Depth 4 in core package (limit: 3)
s.config.Service.Cache.TTLMinutes = minutes
```

**Fix:**
```go
// Extract intermediate variable
service := s.config.Service
if service.Cache != nil {
    service.Cache.TTLMinutes = minutes
}
```

### 2. Cross-Package Deep Access (`TestNoCrossPackageDeepAccess`)
Detects configuration coupling violations where adapters directly access nested config fields.

**Detected Patterns:**
- `.config.Service.`
- `.config.Agent.`
- `.config.Health.`

**Example Violation:**
```go
// Direct config access in adapter
if adapter.config.Service.Domain == "prod" {
    // ...
}
```

**Fix:**
```go
// Use injected capability
if adapter.trustProvider.GetTrustDomain() == "prod" {
    // ...
}
```

### 3. Vendor Type Isolation (`TestVendorTypeIsolation`)
Ensures vendor-specific types don't leak into public APIs.

**Monitored Types:**
- `spiffeid.ID`
- `spiffeid.TrustDomain`
- `x509svid.SVID`
- `tlsconfig.Authorizer`
- `x509bundle.Bundle`

**Example Violation:**
```go
// go-spiffe type in public API
func GetIdentity() spiffeid.ID {
    // ...
}
```

**Fix:**
```go
// Use domain abstraction
func GetIdentity() *domain.ServiceIdentity {
    // ...
}
```

### 4. HTTP Framework Isolation (`TestHTTPFrameworkIsolation`)
Prevents HTTP framework dependencies from leaking into core business logic.

### 5. Core Architecture Boundaries (`TestCoreArchitectureBoundaries`)
Ensures core packages only import other core packages, maintaining architectural layering.

## Running Architecture Tests

### All Tests
```bash
go test ./internal/arch/ -v
```

### Specific Test Categories
```bash
# Selector chain violations only
go test ./internal/arch/ -run TestNoLongSelectorChains -v

# Config coupling violations
go test ./internal/arch/ -run TestNoCrossPackageDeepAccess -v

# Vendor type leakage
go test ./internal/arch/ -run TestVendorTypeIsolation -v
```

## Integration with CI/CD
Architecture tests are part of the standard test suite and run automatically:
- On pull requests
- During CI/CD builds
- In pre-commit hooks (if configured)

A failing architecture test will prevent code from being merged until violations are fixed.

## Interpreting Results

### Selector Chain Violations
```
selector_depth_test.go:88: ../../internal/core/services/identity_service.go:299:40: 
Long selector chain (depth=4, max=3) - Core packages should minimize deep access
    if s.config.Service.Cache != nil && s.config.Service.Cache.ProactiveRefreshMinutes > 0 {
```

**Solution:** Extract intermediate variables or create helper methods.

### Config Deep Access Violations
```
selector_depth_test.go:35: Found 12 config deep access violations that should use capability injection:
    ../../internal/adapters/secondary/health/spire_client.go:91: address = c.config.Agent.Address
```

**Solution:** Use dependency injection to provide specific capabilities instead of passing entire config objects.

### Vendor Type Violations
```
selector_depth_test.go:229: ../../pkg/ephemos/authorizer.go:13: 
Vendor type tlsconfig.Authorizer leaked into public interface
    type Authorizer = tlsconfig.Authorizer
```

**Solution:** Create domain abstractions that wrap vendor types.

## Exemptions and Special Cases

### Configuration Adapters
Files like `trust_domain_adapter.go` are specifically designed to encapsulate config access, so some deep access patterns are acceptable within these implementations.

### Test Files
Architecture tests exclude `*_test.go` files since test code may legitimately need deep access for verification purposes.

### Builder Patterns
Configuration builders may have controlled deep access during object construction phases.

## Maintenance

### Adding New Constraints
1. Identify problematic patterns through code review
2. Add detection logic to appropriate test function
3. Update documentation with examples
4. Consider exemptions for legitimate use cases

### Updating Limits
Selector chain depth limits can be adjusted based on empirical analysis:
```go
// Update limits in TestNoLongSelectorChains
checkSelectorDepth(t, "../../pkg", 2, "Public API description")
checkSelectorDepth(t, "../../internal/core", 3, "Core package description")
```

### Performance Considerations
Architecture tests use static analysis and may take several seconds to complete. They scan all Go files in specified directories, so performance scales with codebase size.

## Related Documentation
- [ADR-004: Selector Chain Constraints](./adr-004-selector-chain-constraints.md)
- [Long Chain Refactoring Guide](../../long-chain-refactor.md)
- [Dependency Injection Patterns](./dependency-injection.md)