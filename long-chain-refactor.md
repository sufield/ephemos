# Long Chain Refactoring Guide

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

**Risk:** Low - Additive changes only

### Phase 2: Configuration Refactoring (Medium Risk)
4. 🔄 **Extract TrustDomainProvider interface** - Create capability abstraction
5. 🔄 **Update transport adapters** - Remove config deep access
6. 🔄 **Configuration builder pattern** - Improve construction ergonomics

**Risk:** Medium - Requires constructor changes

### Phase 3: Interceptor Decoupling (Medium Risk)
7. 🔄 **Direct capability injection** - Remove config dependencies from interceptors
8. 🔄 **Functional options pattern** - Improve construction flexibility
9. 🔄 **Update factory construction** - Wire new dependencies

**Risk:** Medium - Interface changes

### Phase 4: Architecture Testing (Low Risk)
10. 📋 **Add selector chain detection** - Prevent future violations
11. 📋 **Cross-package dependency analysis** - Monitor architecture health
12. 📋 **Documentation updates** - Update architectural decision records

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

## Takeaways

- **Package-level encapsulation**: Go encapsulates at package boundaries, not class boundaries
- **"Tell, don't ask"**: Move behavior next to data instead of exposing internals  
- **Capability injection**: Inject capabilities directly rather than configuration structures
- **Domain methods**: Add methods to domain types instead of exposing fields
- **Architectural testing**: Use static analysis to prevent future violations

This refactoring improves maintainability, testability, and follows Go idioms while reducing coupling between packages.

### General Refactoring Principles
Before diving into file-specific guidelines, here are overarching principles for refactoring chained field/method calls (e.g., `x.y.z`) in Go code:
1. **Identify chains**: Scan for dots (`.`) connecting fields or methods, especially those spanning 2+ levels. Chains can obscure errors (e.g., nil panics) and reduce readability.
2. **Break into intermediates**: Assign intermediate values to local variables for clarity and easier debugging. Add nil checks or error handling where chains might fail.
3. **Handle errors and safety**: In Go, methods often return errors; propagate them explicitly. Use guards for nil values.
4. **Preserve semantics**: Ensure refactoring doesn't change behavior. Use meaningful variable names.
5. **Test thoroughly**: After changes, run unit tests, integration tests, and check for race conditions (especially with mutexes).
6. **Batch changes**: Refactor in small commits per file to avoid overwhelming diffs.
7. **Tools aid**: Use `go vet`, `staticcheck`, or IDE features to spot remaining chains.

Now, file-specific step-by-step guidelines, tailored to the provided samples and occurrence counts. Focus on high-impact areas first (e.g., mutex locks, config accesses).

### internal/core/services/identity_service.go (58 occurrences)
Sample: `identity := domain.NewServiceIdentity(config.Service.Name, config.Service.Domain)`
1. Locate all chains like `config.Service.Name` (field accesses on nested structs).
2. Introduce intermediates: e.g., `service := config.Service; if service == nil { return errors.New("service config missing") }; identity := domain.NewServiceIdentity(service.Name, service.Domain)`.
3. Add nil checks for `config` or `config.Service` if not guaranteed non-nil.
4. For method chains, extract results: e.g., if a chain like `obj.Method().Field`, do `result := obj.Method(); if result == nil { ... }; value := result.Field`.
5. Refactor in batches (e.g., 10 occurrences per pass) due to high count.
6. Verify: Recompile and test identity creation flows.

### internal/adapters/secondary/transport/grpc_provider_rotatable.go (39 occurrences)
Sample: `p.mu.Lock()`
1. Find mutex-related chains like `p.mu.Lock()` or similar (e.g., `p.mu.Unlock()`).
2. Extract mutex: `mu := &p.mu; mu.Lock()` (use pointer if embedded).
3. Add context-aware locking if applicable (e.g., integrate with `sync.Mutex` best practices).
4. For deeper chains (if any, like `p.conn.mu.Lock()`), break further: `conn := p.conn; mu := &conn.mu; mu.Lock()`.
5. Check for data races post-refactor using `go test -race`.
6. Verify: Test concurrent gRPC provider rotations.

### internal/core/services/mtls_connection_registry.go (37 occurrences)
Sample: `c.mu.RLock()`
1. Identify read-lock chains like `c.mu.RLock()` (and `RUnlock()` pairs).
2. Assign intermediate: `mu := &c.mu; mu.RLock(); defer mu.RUnlock()`.
3. Ensure defer is used for unlocks to prevent leaks.
4. If chains involve nested structs (e.g., `c.registry.mu.RLock()`), split: `registry := c.registry; mu := &registry.mu; mu.RLock()`.
5. Refactor symmetrically for all lock/unlock pairs.
6. Verify: Run concurrency tests for mTLS connections.

### internal/core/services/rotation_continuity_service.go (31 occurrences)
Sample: `p.mu.RLock()`
1. Spot read-lock chains similar to above.
2. Break: `mu := &p.mu; mu.RLock(); defer mu.RUnlock()`.
3. Add comments for lock purpose (e.g., `// Protect continuity state`).
4. For any method chains in context (e.g., `p.service.mu.RLock()`), extract: `service := p.service; mu := &service.mu; ...`.
5. Handle high count by focusing on functions with multiple locks first.
6. Verify: Test rotation continuity under load.

### internal/adapters/secondary/spiffe/identity_adapter.go (28 occurrences)
Sample: `svid, err := a.x509Source.GetX509SVID()`
1. Find method chains like `a.x509Source.GetX509SVID()`.
2. Split: `source := a.x509Source; if source == nil { return nil, errors.New("x509 source missing") }; svid, err := source.GetX509SVID()`.
3. Propagate errors explicitly if chained further.
4. For repeated calls, cache intermediates if safe (e.g., in a func-local var).
5. Refactor error-handling blocks together.
6. Verify: Test SPIFFE identity fetching.

### internal/core/services/health_monitor.go (27 occurrences)
Sample: `h.mu.Lock()`
1. Locate mutex chains like `h.mu.Lock()`.
2. Extract: `mu := &h.mu; mu.Lock(); defer mu.Unlock()`.
3. Ensure no deadlocks by reviewing lock order.
4. If nested (e.g., `h.monitor.mu.Lock()`), break: `monitor := h.monitor; mu := &monitor.mu; ...`.
5. Group refactors by function.
6. Verify: Simulate health monitoring scenarios.

### internal/core/application/identity_rotation_service.go (25 occurrences)
Sample: `s.mu.Lock()`
1. Identify lock chains.
2. Assign: `mu := &s.mu; mu.Lock(); defer mu.Unlock()`.
3. Add nil checks if `s` could be nil (unlikely, but safe).
4. For rotation-specific chains, extract service refs.
5. Prioritize critical sections.
6. Verify: Test identity rotations.

### internal/core/services/mtls_enforcement_service.go (24 occurrences)
Sample: `s.mu.Lock()`
1. Similar to above: `mu := &s.mu; mu.Lock(); defer mu.Unlock()`.
2. Review for write vs. read locks if mixed.
3. Break any nested enforcement chains.
4. Refactor in enforcement logic blocks.
5. Verify: Enforce mTLS in tests.

### internal/adapters/secondary/health/spire_client.go (24 occurrences)
Sample: `if c.config.Server == nil {`
1. Find config chains like `c.config.Server`.
2. Split: `config := c.config; if config == nil { ... }; if config.Server == nil { ... }`.
3. Use early returns for nil cases.
4. For deeper configs, layer variables (e.g., `server := config.Server; ...`).
5. Handle all conditional chains.
6. Verify: Test Spire client health.

### pkg/ephemos/public_api.go (21 occurrences)
Sample: `c.mu.RLock()`
1. Extract lock: `mu := &c.mu; mu.RLock(); defer mu.RUnlock()`.
2. Focus on API-critical sections.
3. Break any API param chains.
4. Verify: Public API calls.

### internal/core/domain/workload.go (20 occurrences)
Sample: `if config.Identity.IsZero() {`
1. Chain: `config.Identity.IsZero()`.
2. Split: `identity := config.Identity; if identity.IsZero() { ... }`.
3. Add nil check: `if identity == (domain.Identity{}) { ... }` if needed.
4. Refactor workload validations.
5. Verify: Workload configs.

### internal/core/domain/identity_document.go (18 occurrences)
Sample: `subject := leafCert.Subject.String()`
1. Break: `subject := leafCert.Subject; subjectStr := subject.String()`.
2. Handle cert nil: `if leafCert == nil { ... }`.
3. For parsing chains, add error checks.
4. Verify: Identity doc parsing.

### internal/shutdown/coordinator.go (17 occurrences)
Sample: `c.mu.Lock()`
1. Extract: `mu := &c.mu; mu.Lock(); defer mu.Unlock()`.
2. Ensure shutdown safety.
3. Verify: Shutdown sequences.

### internal/core/domain/certificate.go (17 occurrences)
Sample: `if now.Before(c.Cert.NotBefore) {`
1. Split: `cert := c.Cert; if now.Before(cert.NotBefore) { ... }`.
2. Nil check: `if cert == nil { ... }`.
3. Verify: Cert validations.

### internal/adapters/secondary/verification/spire_verifier.go (17 occurrences)
Sample: `workloadapi.WithAddr(v.config.WorkloadAPISocket)`
1. Break builders: `config := v.config; addr := config.WorkloadAPISocket; workloadapi.WithAddr(addr)`.
2. Verify: Verifier setups.

### internal/adapters/secondary/memidentity/provider.go (17 occurrences)
Sample: `p.mu.Lock()`
1. Similar lock extraction.
2. Verify: Memory provider.

### internal/adapters/interceptors/identity_propagation.go (17 occurrences)
Sample: `if i.config.MetricsCollector != nil {`
1. Split: `config := i.config; if config.MetricsCollector != nil { ... }`.
2. Verify: Interceptors.

### internal/adapters/secondary/transport/grpc_provider.go (16 occurrences)
Sample: `s.mu.Lock()`
1. Lock extraction.
2. Verify: gRPC providers.

### internal/arch/runtime_validation.go (14 occurrences)
Sample: `v.enabled.Store(enabled)`
1. Break: `enabledPtr := &v.enabled; enabledPtr.Store(enabled)`.
2. Verify: Runtime validations.

### internal/adapters/primary/api/server.go (14 occurrences)
Sample: `serviceName: cfg.Service.Name`
1. Split: `service := cfg.Service; serviceName: service.Name`.
2. Verify: API servers.

### internal/core/ports/configuration.go (13 occurrences)
Sample: `if c.Service.Cache != nil {`
1. Split: `service := c.Service; if service.Cache != nil { ... }`.
2. Verify: Config ports.

### internal/adapters/secondary/spiffe/bundle_adapter.go (13 occurrences)
Sample: `svid, err := a.x509Source.GetX509SVID()`
1. Similar to identity_adapter.go.
2. Verify: Bundle adapters.

### internal/adapters/primary/api/client.go (13 occurrences)
Sample: `c.mu.Lock()`
1. Lock extraction.
2. Verify: API clients.

### internal/adapters/interceptors/auth.go (12 occurrences)
Sample: `if a.config.RequireAuthentication {`
1. Split: `config := a.config; if config.RequireAuthentication { ... }`.
2. Verify: Auth interceptors.

### internal/adapters/secondary/spiffe/provider.go (10 occurrences)
Sample: `return p.identityAdapter.GetServiceIdentity(ctx)`
1. Break: `adapter := p.identityAdapter; return adapter.GetServiceIdentity(ctx)`.
2. Verify: SPIFFE providers.

### internal/adapters/secondary/config/inmemory_provider.go (9 occurrences)
Sample: `p.mu.Lock()`
1. Lock extraction.
2. Verify: In-memory configs.

### examples/complete_mtls_scenario/main.go (9 occurrences)
Sample: `domain.NewServiceIdentity("api-server", "production.company.com")`
1. Minimal chains; add vars if nested.
2. Verify: Example scenarios.

### internal/core/application/authentication_service.go (8 occurrences)
Sample: `identityDoc, err := s.identityProvider.GetIdentityDocument(ctx)`
1. Split: `provider := s.identityProvider; identityDoc, err := provider.GetIdentityDocument(ctx)`.
2. Verify: Auth services.

### internal/adapters/logging/redactor.go (8 occurrences)
Sample: `return h.handler.Enabled(ctx, level)`
1. Break: `handler := h.handler; return handler.Enabled(ctx, level)`.
2. Verify: Logging.

### internal/core/domain/trust_bundle.go (7 occurrences)
Sample: `if ca.Cert != nil && ca.Cert.Equal(cert) {`
1. Split: `cert := ca.Cert; if cert != nil && cert.Equal(otherCert) { ... }`.
2. Verify: Trust bundles.

### internal/factory/spiffe.go (6 occurrences)
Sample: `internalConn, err := d.client.Connect(ctx, serviceName, address)`
1. Break: `client := d.client; internalConn, err := client.Connect(...)`.
2. Verify: SPIFFE factories.

### internal/core/domain/authentication_policy.go (5 occurrences)
Sample: `if policy.TrustDomain.IsZero() && identity != nil {`
1. Split: `trustDomain := policy.TrustDomain; if trustDomain.IsZero() ...`.
2. Verify: Policies.

### internal/core/application/identity_use_case.go (5 occurrences)
Sample: `return u.identityService.CreateServerIdentity()`
1. Break: `service := u.identityService; return service.CreateServerIdentity()`.
2. Verify: Use cases.

### contrib/middleware/gin/identity.go (5 occurrences)
Sample: `if c.Request.TLS == nil {`
1. Split: `request := c.Request; if request.TLS == nil { ... }`.
2. Verify: Gin middleware.

### contrib/examples/http_client.go (5 occurrences)
Sample: `trustDomain := svid.ID.TrustDomain()`
1. Break: `id := svid.ID; trustDomain := id.TrustDomain()`.
2. Verify: HTTP examples.

### internal/adapters/interceptors/testing.go (4 occurrences)
Sample: `ServiceDomain: "test.example.com",`
1. Minimal; assign if nested.
2. Verify: Testing interceptors.

### contrib/middleware/gin/examples/main.go (4 occurrences)
Sample: `param.TimeStamp.Format(time.RFC1123)`
1. Split: `ts := param.TimeStamp; formatted := ts.Format(time.RFC1123)`.
2. Verify: Gin examples.

### pkg/ephemos/http.go (3 occurrences)
Sample: `cert, err := s.identityService.GetCertificate()`
1. Break: `service := s.identityService; cert, err := service.GetCertificate()`.
2. Verify: HTTP funcs.
