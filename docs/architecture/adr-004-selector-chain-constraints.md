# ADR-004: Selector Chain Constraints

## Status
Accepted

## Context
During the long-chain refactoring effort, we identified numerous instances of deep selector chains (e.g., `obj.field.method.value`) that violate the Law of Demeter and create tight coupling between components. These patterns make code harder to test, debug, and maintain.

## Decision
We will enforce architectural constraints through automated testing to prevent selector chain violations and maintain code quality.

### Selector Chain Depth Limits
- **Public API packages** (`pkg/`): Maximum depth of 2 selectors
- **Core packages** (`internal/core/`): Maximum depth of 3 selectors  
- **Adapter packages** (`internal/adapters/`): Maximum depth of 4 selectors

### Rationale for Limits
- **Public API**: Should provide clean, facade-style interfaces with minimal chaining
- **Core packages**: Business logic should avoid deep object navigation
- **Adapters**: May need slightly deeper access for integration, but still constrained

### Acceptable Patterns
```go
// ✅ Good: Single-level access
user.Name()

// ✅ Good: Two-level access with clear intent
client.Connection().Close()

// ✅ Good: Intermediate variables break up chains
config := service.GetConfig()
cache := config.Cache
if cache != nil && cache.TTLMinutes > 0 {
    // use cache settings
}

// ❌ Bad: Deep selector chain
service.config.cache.settings.ttl.minutes
```

### Exemptions
Certain patterns are acceptable within their appropriate contexts:

1. **Configuration Adapters**: The `TrustDomainAdapter` and similar adapters are specifically designed to encapsulate config access, so patterns like `t.config.Service.Domain` are acceptable within these adapter implementations.

2. **Builder Patterns**: Configuration builders may have controlled deep access during construction phases.

3. **Test Files**: Test files are excluded from these constraints as they may legitimately need deep access for verification.

## Implementation
Architectural constraints are enforced through the `internal/arch/selector_depth_test.go` test suite:

- `TestNoLongSelectorChains`: Enforces depth limits across packages
- `TestNoCrossPackageDeepAccess`: Detects config coupling violations
- `TestVendorTypeIsolation`: Prevents vendor type leakage

### Benefits
1. **Improved Maintainability**: Shorter chains are easier to understand and modify
2. **Better Testability**: Components with fewer dependencies are easier to unit test
3. **Reduced Coupling**: Limits knowledge of internal object structure
4. **Error Prevention**: Reduces null pointer exceptions from deep navigation
5. **Code Review Guidance**: Provides objective criteria for architectural review

## Consequences
- Developers must use intermediate variables or helper methods instead of long chains
- Some refactoring may be required to meet depth constraints
- Test suite will catch violations during CI/CD pipeline
- Architecture tests provide ongoing regression prevention

## Related Patterns
- **Facade Pattern**: Use wrapper methods instead of exposing deep structure
- **Dependency Injection**: Inject capabilities directly rather than config objects
- **Builder Pattern**: Encapsulate complex construction logic
- **Domain Methods**: Add behavior to domain objects rather than exposing fields

## Examples
### Before (Violates Constraints)
```go
// Depth 4 - violates core package limit of 3
if s.config.Service.Cache.ProactiveRefreshMinutes > 0 {
    threshold = time.Duration(s.config.Service.Cache.ProactiveRefreshMinutes) * time.Minute
}
```

### After (Compliant)
```go
// Extract intermediate to reduce depth
service := s.config.Service
if service.Cache != nil && service.Cache.ProactiveRefreshMinutes > 0 {
    threshold = time.Duration(service.Cache.ProactiveRefreshMinutes) * time.Minute
}
```

## Monitoring
The architecture tests run automatically in CI/CD and will fail the build if violations are introduced. Regular reviews should examine test output to identify patterns and guide further refactoring efforts.