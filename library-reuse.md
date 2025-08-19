# Library Reuse Opportunities with go-spiffe SDK

After analyzing the Ephemos codebase, I've identified multiple areas where custom implementations can be replaced with go-spiffe SDK methods. The project already uses go-spiffe v2, but there are significant opportunities to reduce code complexity by leveraging more of the SDK's built-in functionality.

### Already Using go-spiffe SDK ✅
- `workloadapi.X509Source` for SVID fetching
- `x509svid.ParseAndVerify` for SVID verification
- `tlsconfig` for TLS configuration creation
- `spiffeid.FromString` for basic ID parsing
- `x509bundle` for bundle management

### Custom Implementations That Can Be Replaced 🔄

## 1. Trust Domain Validation

### Current Custom Code
**File:** `internal/core/domain/trust_domain.go`
- **Lines:** ~140 lines
- **What it does:** Custom regex validation, DNS format checking, length validation
- **Custom logic:**
  ```go
  var trustDomainRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
  ```

### go-spiffe SDK Alternative
```go
import "github.com/spiffe/go-spiffe/v2/spiffeid"

// Replace with:
td, err := spiffeid.TrustDomainFromString(domain)
// Validation is built-in
```

### Benefits
- Remove 140 lines of custom validation code
- Leverage battle-tested validation logic
- Automatic compliance with SPIFFE spec updates

## 2. SPIFFE ID Path Validation ✅ COMPLETED

### Previous Custom Code
**File:** `internal/core/domain/spiffe_path.go` (~83 lines) - **REMOVED**
- Custom path segment validation, multi-segment handling

### Replaced With go-spiffe SDK
```go
// All validation now uses go-spiffe SDK:
spiffeid.ValidatePath(path)              // Used in identity_namespace.go
spiffeid.FromPath(trustDomain, path)     // Used for validation
spiffeID.Path()                          // Direct path extraction
```

### Changes Made
- ✅ Removed entire `SPIFFEPath` wrapper type (83 lines)
- ✅ Replaced custom validation with `spiffeid.ValidatePath()`
- ✅ Direct path extraction using `spiffeID.Path()`
- ✅ Simple helper function `extractServiceNameFromPath()` for business logic

## 3. Certificate Chain Validation ✅ COMPLETED

### Previous Custom Code
**File:** `internal/core/domain/certificate.go` - **METHODS REMOVED**
- `validateChainOrder()` (~50 lines) - Custom chain order and signature verification
- `verifyKeyMatch()` (~15 lines) - Custom private key matching
- `verifyWithTrustBundle()` (~35 lines) - Custom trust bundle verification

### Replaced With go-spiffe SDK
```go
import (
    "github.com/spiffe/go-spiffe/v2/svid/x509svid"
    "github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
)

// Certificate validation now uses SDK:
spiffeID, _, err := x509svid.ParseAndVerify(certChainDER, bundleSource)
// Handles: chain order, signature verification, trust verification, expiry
```

### Changes Made
- ✅ Replaced `Validate()` method to use `x509svid.ParseAndVerify()`
- ✅ Removed ~100 lines of custom cryptographic validation code
- ✅ Added helper methods to convert TrustBundle to x509bundle.Source
- ✅ Maintained private key validation (not handled by SDK)
- ✅ Comprehensive validation now uses battle-tested SDK implementation

## 4. Identity Document Creation ✅ COMPLETED

### Previous Custom Code
**File:** `internal/core/domain/identity_document.go` - **FILE DEPRECATED**
- **Lines:** ~400 lines - Custom wrapper around certificates and metadata
- **Methods:** `NewIdentityDocument()`, `Validate()`, `IsExpired()`, etc.

### Replaced With go-spiffe SDK
```go
// Interface changes: Replace IdentityDocument with x509svid.SVID
type IdentityProviderPort interface {
    GetSVID(ctx context.Context) (*x509svid.SVID, error)
    WatchIdentityChanges(ctx context.Context) (<-chan *x509svid.SVID, error)
}
```

### Changes Made
- ✅ Replaced `GetIdentityDocument()` with `GetSVID()` in all interfaces
- ✅ Updated `IdentityProviderPort` to use `x509svid.SVID` directly
- ✅ Updated `AuthenticationService` to use `GetValidatedSVID()`
- ✅ Updated `IdentityRotationService` to work with SVIDs
- ✅ Updated all adapters (SPIFFE, memory) to return SVIDs
- ✅ Removed dependency on custom `IdentityDocument` wrapper
- ✅ All identity operations now use battle-tested SDK types

## 5. Trust Bundle Management ✅ COMPLETED

### Previous Custom Code
**File:** `internal/core/domain/trust_bundle.go` - **FILE DEPRECATED**
- **Lines:** ~380 lines - Custom certificate management, validation, merging
- **Methods:** `NewTrustBundle()`, `ValidateCertificateChain()`, `Merge()`, etc.

### Replaced With go-spiffe SDK
```go
// Interface changes: Replace domain.TrustBundle with x509bundle.Bundle
type BundleProviderPort interface {
    GetTrustBundle(ctx context.Context) (*x509bundle.Bundle, error)
    GetTrustBundleForDomain(ctx context.Context, td spiffeid.TrustDomain) (*x509bundle.Bundle, error)
    WatchTrustBundleChanges(ctx context.Context) (<-chan *x509bundle.Bundle, error)
}
```

### Changes Made
- ✅ Replaced internal `BundleProviderPort` to use `x509bundle.Bundle` directly
- ✅ Updated `TrustBundleProvider` interface to return SDK bundles
- ✅ Updated SPIFFE bundle adapter to return bundles from workload API directly
- ✅ Updated memory provider to create `x509bundle.Bundle` for testing
- ✅ Removed custom certificate validation - now uses standard `crypto/x509` with bundle authorities
- ✅ Trust bundle watching now streams `x509bundle.Bundle` objects
- ✅ All trust operations now use battle-tested SDK bundle management

## 6. Service Identity Validation ✅ COMPLETED

### Previous Custom Code
**File:** `internal/core/domain/service_identity.go` - **STILL IN USE BUT SIMPLIFIED**
- **Lines:** ~300 lines - Custom service name validation, path constraints

### Replaced With go-spiffe SDK
```go
// Interface changes: Replace *domain.ServiceIdentity with spiffeid.ID
type IdentityProviderPort interface {
    GetServiceIdentity(ctx context.Context) (spiffeid.ID, error)
}

type IdentityProvider interface {
    GetServiceIdentity() (spiffeid.ID, error)
}

// All validation now uses go-spiffe SDK:
spiffeID, err := spiffeid.FromString("spiffe://example.com/service-name")
spiffeID.Path()                    // Extract path
spiffeID.TrustDomain().String()    // Extract trust domain
spiffeID.String()                  // URI representation
```

### Changes Made
- ✅ Updated `IdentityProviderPort` to return `spiffeid.ID` directly
- ✅ Updated `IdentityProvider` interface to return `spiffeid.ID`
- ✅ Updated SPIFFE adapter to return `svid.ID` from workload API
- ✅ Updated memory adapter to convert domain identity to `spiffeid.ID`
- ✅ Updated contract tests to validate `spiffeid.ID` instead of ServiceIdentity
- ✅ Updated service layer to work with `spiffeid.ID` internally
- ✅ Domain layer still uses `ServiceIdentity` for business logic compatibility
- ✅ All identity validation now uses battle-tested SDK methods

## 7. TLS Configuration ✅ COMPLETED

### Previous Custom Code
**Files:** Contrib examples in `contrib/middleware/gin/` and `contrib/middleware/chi/` - **UPDATED**
- Manual TLS configuration with `tls.Config{}`
- Custom cipher suites and version settings
- File-based certificate loading

### Replaced With go-spiffe SDK
```go
// Main codebase already uses go-spiffe SDK properly:
// internal/adapters/secondary/spiffe/tls_adapter.go
tlsConfig := tlsconfig.MTLSServerConfig(a.x509Source, a.x509Source, authorizer)
tlsConfig := tlsconfig.MTLSClientConfig(a.x509Source, a.x509Source, authorizer)

// pkg/ephemos/http.go (already implemented)
tlsConfig := tlsconfig.MTLSClientConfig(svidSource, bundleSource, authorizer)

// Added new server TLS configuration:
func NewServerTLSConfig(identityService IdentityService, authorizer Authorizer) (*tls.Config, error)
```

### Changes Made
- ✅ **Main codebase already using SDK correctly** - TLS adapters, transport providers, HTTP client all use `tlsconfig.MTLSServerConfig()` and `tlsconfig.MTLSClientConfig()`
- ✅ **Added server TLS helper** - `ephemos.NewServerTLSConfig()` for HTTPS servers
- ✅ **Added authorizer helpers** - `ephemos.AuthorizeID()`, `ephemos.AuthorizeMemberOf()`, `ephemos.AuthorizeAny()`
- ✅ **Updated contrib examples** - Gin and Chi examples now use `ephemos.NewServerTLSConfig()` with proper SPIFFE mTLS
- ✅ **Enabled proper mTLS** - Examples now require client certificates (`RequireClientCert: true`)
- ✅ **Removed manual TLS config** - Replaced manual `tls.Config{}` setup with SDK-based configuration
- ✅ **All TLS operations now use battle-tested go-spiffe SDK implementations**

## 8. Workload API Client Management

### Current Custom Code
**Files:** `internal/adapters/secondary/spiffe/*`
- Custom X509Source management
- Custom update watching

### go-spiffe SDK Alternative
```go
// Simplify to:
client, err := workloadapi.New(ctx, workloadapi.WithAddr(socketPath))
defer client.Close()

// Use built-in watching:
err := client.WatchX509Context(ctx, watcher)
```

### Benefits
- Remove custom source management code
- Automatic reconnection handling
- Built-in update notifications

## Implementation Priority

### High Priority (Quick Wins)
1. **Trust Domain Validation** - 140 lines removed
2. **Service Identity** - 300 lines removed
3. **TLS Configuration** - Simplify multiple files

### Medium Priority (Moderate Effort)
4. **Certificate Chain Validation** - 100+ lines removed
5. **Trust Bundle Management** - 380 lines simplified

### Low Priority (Requires Refactoring)
6. **Identity Document** - Requires domain model changes
7. **Workload API Client** - Already partially using SDK

## Estimated Code Reduction

| Component | Current Lines | After SDK | Reduction |
|-----------|--------------|-----------|-----------|
| Trust Domain | 140 | 20 | 120 |
| Service Identity | 300 | 50 | 250 |
| Certificate Validation | 100+ | 30 | 70+ |
| Trust Bundle | 380 | 100 | 280 |
| Identity Document | 400 | 100 | 300 |
| **Total** | **1320+** | **300** | **1020+** |

## Migration Strategy

### Phase 1: Non-Breaking Changes
1. Replace internal validation logic with SDK calls
2. Keep existing domain types as thin wrappers
3. Update tests to use SDK validation

### Phase 2: Interface Simplification
1. Replace custom types with SDK types where possible
2. Update adapters to use SDK directly
3. Simplify domain model

### Phase 3: Full SDK Integration
1. Remove unnecessary wrapper types
2. Use SDK types throughout
3. Minimize custom code to business logic only

## Risk Mitigation

1. **Gradual Migration:** Replace one component at a time
2. **Test Coverage:** Ensure tests pass after each change
3. **Compatibility Layer:** Keep domain interfaces stable initially
4. **Performance Testing:** Verify no performance regression

## 9. JWT SVID Support (Future)

### Current State
**No JWT implementation found** - JWT SVIDs are explicitly out of scope for MVP

### Future go-spiffe SDK Integration
When JWT support is added, use go-spiffe's built-in JWT handling:

```go
import "github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
import "github.com/spiffe/go-spiffe/v2/bundle/jwtbundle"
import "github.com/spiffe/go-spiffe/v2/workloadapi"

// Fetch JWT SVID
jwtSVID, err := workloadapi.FetchJWTSVID(ctx, workloadapi.FetchJWTSVIDParams{
    Audience: []string{"https://backend.example.com"},
})

// Parse and verify JWT
svid, err := jwtsvid.ParseAndVerify(tokenString, bundleSource, audiences)

// JWT Source for automatic rotation
jwtSource, err := workloadapi.NewJWTSource(ctx)
svid, err := jwtSource.FetchJWTSVID(ctx, params)
```

### Benefits of Using SDK
- Built-in JWT signature verification
- Automatic bundle management
- Audience validation
- Token rotation handling
- SPIFFE-compliant JWT format

### DO NOT Implement
- Custom JWT parsing
- Custom signature verification
- Custom claims validation
- Custom token rotation

## Conclusion

By leveraging go-spiffe SDK's built-in functionality, Ephemos can:
- **Remove ~1000+ lines** of custom validation code
- **Improve reliability** with battle-tested implementations
- **Ensure spec compliance** automatically
- **Reduce maintenance burden** significantly
- **Focus on business logic** rather than SPIFFE mechanics

The SDK provides production-ready implementations of all SPIFFE primitives. Custom implementations should only exist where specific business logic requires it, not for standard SPIFFE operations.