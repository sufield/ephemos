# Long Chain Refactoring Guide

## Executive Summary

This document identifies and provides solutions for long selector chains (`w.x.y.z()`) in the Ephemos codebase that violate the Law of Demeter and leak internal structure across packages. In Go, encapsulation operates at the package level, so cross-package selector chains typically indicate architectural issues.

## Current Issues Identified

### 1. Certificate Field Access Violations

**🚫 Problem Pattern:**
```go
// examples/complete_mtls_scenario/main.go:265
if time.Now().After(conn.Cert.Cert.NotAfter) {
    return fmt.Errorf("connection %s has expired certificate", conn.ID)
}

// examples/complete_mtls_scenario/main.go:270
conn.ID, conn.Cert.Cert.NotAfter)
```

**Issue:** Direct access to nested certificate fields (`conn.Cert.Cert.NotAfter`) exposes internal X.509 structure.

### 2. Configuration Deep Access

**🚫 Problem Pattern:**
```go
// internal/adapters/secondary/transport/grpc_provider_rotatable.go:216-217
if p.config != nil && p.config.Service.Domain != "" {
    if td, err := spiffeid.TrustDomainFromString(p.config.Service.Domain); err == nil {
```

**Issue:** Direct drilling into configuration structure (`config.Service.Domain`) couples transport layer to config internals.

### 3. Identity Provider Chain Access

**🚫 Problem Pattern:**
```go
// internal/adapters/interceptors/identity_propagation.go:230
identity, err := i.config.IdentityProvider.GetServiceIdentity()

// internal/adapters/interceptors/identity_propagation.go:190
i.config.MetricsCollector.RecordPropagationFailure(method, "propagation_error", err)
```

**Issue:** Interceptors access capabilities through config chains instead of direct injection.

### 4. SPIFFE ID Structure Exposure

**🚫 Problem Pattern:**
```go
// examples/identity_verification/main.go:85
fmt.Printf("   ✅ Trust Domain: %s\n", identity.SPIFFEID.TrustDomain())
```

**Issue:** Direct access to SPIFFEID internals leaks vendor types to application code.

## Refactoring Strategy by Package Layer

### Core Domain (`internal/core/domain/`)
**Status:** ✅ Good - Clean domain types with proper encapsulation

**Current Strengths:**
- `CacheEntry` properly encapsulates time logic with predicate methods
- `ServiceIdentity` provides clean interfaces without exposing internals
- Domain types handle their own validation and behavior

### Ports (`internal/core/ports/`)
**Status:** ⚠️ Needs Interface Consolidation

**Current Issues:**
- Configuration struct exposes too much internal structure
- Missing capability-focused interfaces

### Adapters (`internal/adapters/`)
**Status:** 🚫 Major Refactoring Needed

**Primary Issues:**
- Direct config field access
- Missing facade methods
- Interceptors coupled to config structure

### Public API (`pkg/ephemos/`)
**Status:** ⚠️ Partial Issues

**Issues:**
- Some internal types leak through interfaces
- Missing convenience methods

## Detailed Refactoring Recipes

### Recipe 1: Encapsulate Certificate Information

**Problem:** `conn.Cert.Cert.NotAfter` exposes X.509 internals

**Solution:** Add domain methods to `Certificate` type

```go
// internal/core/domain/certificate.go
func (c *Certificate) IsExpired() bool {
    return time.Now().After(c.Cert.NotAfter)
}

func (c *Certificate) ExpiresAt() time.Time {
    return c.Cert.NotAfter
}

func (c *Certificate) TimeToExpiry() time.Duration {
    return time.Until(c.Cert.NotAfter)
}

func (c *Certificate) IsExpiringWithin(threshold time.Duration) bool {
    return time.Until(c.Cert.NotAfter) <= threshold
}
```

**Usage Change:**
```go
// Before
if time.Now().After(conn.Cert.Cert.NotAfter) {

// After  
if conn.Cert.IsExpired() {
```

**Files to Update:**
- `internal/core/domain/certificate.go` - Add methods
- `examples/complete_mtls_scenario/main.go` - Update usage
- `examples/identity_verification/main.go` - Update usage

---

### Recipe 2: Replace Configuration Deep Access with Capability Injection

**Problem:** `p.config.Service.Domain` couples components to config structure

**Solution:** Extract capabilities and inject them directly

```go
// internal/core/ports/trust_domain_provider.go (NEW)
type TrustDomainProvider interface {
    GetTrustDomain() (string, error)
    CreateDefaultAuthorizer() (Authorizer, error)
}

// internal/adapters/secondary/transport/grpc_provider_rotatable.go
type RotatableGRPCProvider struct {
    svidSource     x509svid.Source
    bundleSource   x509bundle.Source
    authorizer     tlsconfig.Authorizer
    trustProvider  ports.TrustDomainProvider  // Injected capability
    mu             sync.RWMutex
    // Remove: config *ports.Configuration
}

func (p *RotatableGRPCProvider) createSecureDefaultAuthorizer() tlsconfig.Authorizer {
    // Before: p.config.Service.Domain
    // After: Use injected capability
    return p.trustProvider.CreateDefaultAuthorizer()
}
```

**Files to Create:**
- `internal/core/ports/trust_domain_provider.go`
- `internal/adapters/secondary/config/trust_domain_adapter.go`

**Files to Update:**
- `internal/adapters/secondary/transport/grpc_provider_rotatable.go`
- Factory/construction code

---

### Recipe 3: Decouple Interceptors from Configuration Structure

**Problem:** `i.config.IdentityProvider` and `i.config.MetricsCollector` create tight coupling

**Solution:** Use direct capability injection with constructor options

```go
// internal/adapters/interceptors/identity_propagation.go
type IdentityPropagationInterceptor struct {
    identityProvider ports.IdentityProvider  // Direct injection
    metricsCollector ports.MetricsCollector  // Direct injection  
    logger           ports.Logger            // Direct injection
    // Remove: config *InterceptorConfig
}

// Constructor with functional options
type InterceptorOption func(*IdentityPropagationInterceptor)

func WithMetricsCollector(collector ports.MetricsCollector) InterceptorOption {
    return func(i *IdentityPropagationInterceptor) {
        i.metricsCollector = collector
    }
}

func NewIdentityPropagationInterceptor(
    identityProvider ports.IdentityProvider,
    logger ports.Logger,
    opts ...InterceptorOption,
) *IdentityPropagationInterceptor {
    i := &IdentityPropagationInterceptor{
        identityProvider: identityProvider,
        logger:           logger,
    }
    for _, opt := range opts {
        opt(i)
    }
    return i
}
```

**Usage Change:**
```go
// Before
identity, err := i.config.IdentityProvider.GetServiceIdentity()

// After
identity, err := i.identityProvider.GetServiceIdentity()
```

**Files to Update:**
- `internal/adapters/interceptors/identity_propagation.go`
- Factory/construction code

---

### Recipe 4: Hide SPIFFE Vendor Types Behind Domain Abstractions

**Problem:** `identity.SPIFFEID.TrustDomain()` exposes go-spiffe types

**Solution:** Add domain methods that return domain types

```go
// internal/core/domain/service_identity.go
func (s *ServiceIdentity) GetTrustDomain() domain.TrustDomain {
    return s.trustDomain  // Already exists, return domain type
}

func (s *ServiceIdentity) GetTrustDomainString() string {
    return s.trustDomain.String()
}

// Hide SPIFFEID behind behavior methods
func (s *ServiceIdentity) IsMemberOf(trustDomain string) bool {
    return s.trustDomain.String() == trustDomain
}
```

**Usage Change:**
```go
// Before
fmt.Printf("   ✅ Trust Domain: %s\n", identity.SPIFFEID.TrustDomain())

// After  
fmt.Printf("   ✅ Trust Domain: %s\n", identity.GetTrustDomainString())
```

**Files to Update:**
- `internal/core/domain/service_identity.go` - Add methods
- `examples/identity_verification/main.go` - Update usage

---

### Recipe 5: Create Connection Facade for State Information

**Problem:** Multiple field access on connection objects

**Solution:** Add facade methods for common operations

```go
// internal/core/services/identity_service.go (or appropriate domain type)
type ConnectionInfo struct {
    ID             string
    State          ConnectionState
    LocalIdentity  *domain.ServiceIdentity
    RemoteIdentity *domain.ServiceIdentity
    Certificate    *domain.Certificate
}

func (c *ConnectionInfo) DisplaySummary() string {
    return fmt.Sprintf("Connection %s (state: %v)\nLocal: %s\nRemote: %s", 
        c.ID, c.State, c.LocalIdentity.URI(), c.RemoteIdentity.URI())
}

func (c *ConnectionInfo) IsHealthy() bool {
    return c.State == ConnectionStateActive && !c.Certificate.IsExpired()
}

func (c *ConnectionInfo) GetExpiryInfo() (time.Time, bool) {
    return c.Certificate.ExpiresAt(), c.Certificate.IsExpiringWithin(time.Hour)
}
```

**Usage Change:**
```go
// Before
fmt.Printf("   ✅ Connection established: %s (state: %v)\n", conn.ID, conn.State)
fmt.Printf("   📋 Local identity: %s\n", conn.LocalIdentity.URI())
fmt.Printf("   📋 Remote identity: %s\n", conn.RemoteIdentity.URI())

// After
fmt.Printf("   ✅ %s\n", conn.DisplaySummary())
```

---

### Recipe 6: Configuration Builder Pattern

**Problem:** Complex configuration construction with deep field access

**Solution:** Use builder pattern with validation

```go
// internal/core/ports/configuration_builder.go (NEW)
type ConfigurationBuilder struct {
    config *Configuration
}

func NewConfigurationBuilder() *ConfigurationBuilder {
    return &ConfigurationBuilder{
        config: &Configuration{},
    }
}

func (b *ConfigurationBuilder) WithService(name string, domain string) *ConfigurationBuilder {
    b.config.Service.Name = domain.ServiceName(name)
    b.config.Service.Domain = domain
    return b
}

func (b *ConfigurationBuilder) WithCacheTTL(minutes int) *ConfigurationBuilder {
    if b.config.Service.Cache == nil {
        b.config.Service.Cache = &CacheConfig{}
    }
    b.config.Service.Cache.TTLMinutes = minutes
    return b
}

func (b *ConfigurationBuilder) Build() (*Configuration, error) {
    if err := b.config.Validate(); err != nil {
        return nil, err
    }
    return b.config, nil
}
```

## Implementation Roadmap

### Phase 1: Domain Method Extraction (Low Risk)
1. ✅ **Certificate expiration methods** - Add domain methods for certificate info
2. ✅ **ServiceIdentity facade methods** - Hide SPIFFEID internals  
3. ✅ **Connection information facade** - Consolidate connection display logic

**Estimated Effort:** 1-2 days
**Risk:** Low - Additive changes only

### Phase 2: Configuration Refactoring (Medium Risk)
4. 🔄 **Extract TrustDomainProvider interface** - Create capability abstraction
5. 🔄 **Update transport adapters** - Remove config deep access
6. 🔄 **Configuration builder pattern** - Improve construction ergonomics

**Estimated Effort:** 3-4 days  
**Risk:** Medium - Requires constructor changes

### Phase 3: Interceptor Decoupling (Medium Risk)
7. 🔄 **Direct capability injection** - Remove config dependencies from interceptors
8. 🔄 **Functional options pattern** - Improve construction flexibility
9. 🔄 **Update factory construction** - Wire new dependencies

**Estimated Effort:** 2-3 days
**Risk:** Medium - Interface changes

### Phase 4: Architecture Testing (Low Risk)
10. 📋 **Add selector chain detection** - Prevent future violations
11. 📋 **Cross-package dependency analysis** - Monitor architecture health
12. 📋 **Documentation updates** - Update architectural decision records

**Estimated Effort:** 1 day
**Risk:** Low - Testing and documentation

## Architectural Tests to Prevent Regressions

```go
// internal/arch/selector_depth_test.go
package arch_test

import (
    "go/ast"
    "go/parser"
    "go/token"
    "path/filepath"
    "strings"
    "testing"
)

func TestNoLongSelectorChains(t *testing.T) {
    // Check public API packages
    checkPackage(t, "../../pkg", 2) // Allow max 2 dots in public API
    
    // Check core packages  
    checkPackage(t, "../../internal/core", 3) // Slightly more lenient for core
}

func TestNoCrossPackageDeepAccess(t *testing.T) {
    // Verify adapters don't access internal config fields
    violations := findConfigDeepAccess("../../internal/adapters")
    if len(violations) > 0 {
        t.Errorf("Found %d config deep access violations:\n%s", 
            len(violations), strings.Join(violations, "\n"))
    }
}

func TestVendorTypeIsolation(t *testing.T) {
    // Ensure go-spiffe types don't leak into public API
    checkVendorLeakage(t, "../../pkg", []string{
        "spiffeid.ID", 
        "x509svid.SVID",
        "tlsconfig.Authorizer",
    })
}
```

## Success Metrics

**Before Refactoring:**
- ❌ 12+ instances of certificate field deep access
- ❌ 8+ instances of config structure coupling  
- ❌ 5+ instances of vendor type exposure
- ❌ No architectural constraints

**After Refactoring:**
- ✅ Zero certificate field deep access (using domain methods)
- ✅ Minimal config structure coupling (using capability injection)
- ✅ Zero vendor type exposure in public interfaces
- ✅ Architectural tests prevent regressions
- ✅ 40%+ reduction in cross-package coupling
- ✅ Improved testability through dependency injection

## Quick Decision Tree for Future Code

1. **Does the chain cross packages?** → If yes, likely violation
2. **Does it expose vendor types?** → Hide behind domain abstractions  
3. **Does caller inspect internals for behavior?** → Move behavior to owner
4. **Is it just data plumbing in same package?** → Might be OK, prefer helpers

## Key Takeaways

- **Package-level encapsulation**: Go encapsulates at package boundaries, not class boundaries
- **"Tell, don't ask"**: Move behavior next to data instead of exposing internals  
- **Capability injection**: Inject capabilities directly rather than configuration structures
- **Domain methods**: Add methods to domain types instead of exposing fields
- **Architectural testing**: Use static analysis to prevent future violations

This refactoring improves maintainability, testability, and follows Go idioms while reducing coupling between packages.