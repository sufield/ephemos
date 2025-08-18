# Codebase Issues Report

This document identifies violations of the rules defined in CLAUDE.md after analyzing the Ephemos codebase.

## 1. Authorization Scope Violations 🚫

**Rule Violated:** "Focus only on identity-based authentication. Authorization is out of scope."

### Critical Issues:

#### 1.1 Authorization Policy Implementation
- **File:** `internal/core/domain/authentication_policy.go`
- **Issue:** Implements `NewAuthorizationPolicy` with access control lists
- **Lines:** 38-50
- **Details:** 
  - Contains `AuthorizedClients` and `TrustedServers` lists for access control
  - Implements allowlist-based authorization (SPIFFE IDs)
  - Function explicitly named "AuthorizationPolicy"

#### 1.2 Configuration Contains Authorization Fields
- **File:** `internal/core/ports/configuration.go`
- **Lines:** 52-60
- **Fields:**
  ```go
  AuthorizedClients []string // Line 55
  TrustedServers []string    // Line 60
  ```
- **Issue:** Configuration structure includes authorization-specific fields

#### 1.3 Identity Service Uses Authorization
- **File:** `internal/core/services/identity_service.go`
- **Lines:** 624-630
- **Code:**
  ```go
  if len(serviceConfig.AuthorizedClients) > 0 || len(serviceConfig.TrustedServers) > 0 {
      policy, err := domain.NewAuthorizationPolicy(identity, serviceConfig.AuthorizedClients, serviceConfig.TrustedServers)
  ```
- **Issue:** Service layer actively uses authorization policies

#### 1.4 Public API Exposes Authorizer
- **File:** `pkg/ephemos/authorizer.go`
- **Line:** 13
- **Code:** `type Authorizer = tlsconfig.Authorizer`
- **Issue:** Public API directly exposes authorization concepts

**Recommendation:** Remove all authorization-related code. Focus solely on identity verification and authentication.

## 2. Backward Compatibility Code (Fallback Patterns) ⚠️

**Rule Violated:** "No conditional code paths for legacy behavior"

### Issues Found:

#### 2.1 Fallback Patterns in Transport Layer
- **File:** `internal/adapters/secondary/transport/grpc_provider_rotatable.go`
- **Lines:** 81, 89, 131, 139, 238, 278
- **Pattern:** "Fallback mode" for static certificates when sources unavailable
- **Issue:** Provides backward compatibility path for non-rotation scenarios

#### 2.2 Service Identity Fallback Construction
- **File:** `internal/core/domain/service_identity.go`
- **Lines:** 58-60, 70-73, 81-84, 168-171, 179-182, 201-203
- **Pattern:** "Fallback to simple construction" when SPIFFE ID creation fails
- **Issue:** Maintains backward compatibility for invalid trust domains

#### 2.3 SPIFFE Socket Path Fallbacks
- **Files:** 
  - `internal/adapters/secondary/spiffe/bundle_adapter.go:292`
  - `internal/adapters/secondary/spiffe/identity_adapter.go:337`
  - `internal/adapters/secondary/spiffe/tls_adapter.go:249`
- **Pattern:** Hardcoded fallback to `/tmp/spire-agent/public/api.sock`
- **Issue:** Provides compatibility when SPIFFE_ENDPOINT_SOCKET not set

**Recommendation:** Remove all fallback patterns. Fail fast with clear errors instead.

## 3. Vendor Type Leakage 🔴

**Rule Violated:** Architecture constraint - vendor types should not leak into public interfaces

### Critical Issues:

#### 3.1 Public API Exposes go-spiffe Types
- **File:** `pkg/ephemos/authorizer.go`
  - Line 13: `type Authorizer = tlsconfig.Authorizer`
  - Line 23: Uses `spiffeid.TrustDomain`
  - Line 67: Uses `spiffeid.ID`

- **File:** `pkg/ephemos/http.go`
  - Line 163: Returns `*x509svid.SVID`
  - Line 200: Uses `spiffeid.TrustDomain`
  - Line 204: Returns `*x509bundle.Bundle`
  - Line 225: Returns `spiffeid.ID`

**Recommendation:** Wrap all vendor types in domain abstractions.

## 4. Configuration Deep Access Violations 🔧

**Rule Violated:** Law of Demeter - avoid deep configuration access

### Issues Found:

#### 4.1 Trust Domain Adapter
- **File:** `internal/adapters/secondary/config/trust_domain_adapter.go`
- **Lines:** 32, 36, 64, 69
- **Pattern:** `t.config.Service.Domain`
- **Issue:** Direct access to nested configuration

#### 4.2 Health Check Client
- **File:** `internal/adapters/secondary/health/spire_client.go`
- **Lines:** 91-94, 141-144
- **Pattern:** `c.config.Agent.Address`, `c.config.Agent.LivePath`, etc.
- **Issue:** Multiple deep accesses to agent configuration

**Recommendation:** Use dependency injection of specific capabilities instead of passing entire config.

## 5. Code Quality Issues 📊

### 5.1 Naming Convention Violations
- **Test Failure:** `Test_Port_Naming_Conventions`
- **Issue:** Some interfaces don't follow port naming conventions (should end with Port, Provider, Service, or Repository)

### 5.2 Adapter Structure Violations
- **Test Failure:** `Test_Adapter_Interface_Compliance`
- **Issue:** Some adapter files lack proper struct definitions

### 5.3 Domain Purity Violations
- **Test Failure:** `Test_Domain_Types_Are_Pure`
- **File:** `internal/core/domain/trust_domain.go`
- **Issue:** Contains `MarshalJSON`/`UnmarshalJSON` methods (infrastructure concerns in domain)

## 6. Dependency Analysis ✅

### All Dependencies Are Stable
- Go version: 1.24 (stable)
- All direct dependencies use stable versions:
  - `github.com/spiffe/go-spiffe/v2 v2.5.0` - Active, well-maintained
  - `github.com/stretchr/testify v1.10.0` - Active, widely used
  - `github.com/prometheus/client_golang v1.23.0` - Active, official
  - `google.golang.org/grpc v1.74.2` - Active, official

**No issues found with dependencies.**

## Summary Statistics

- **🚫 Authorization Violations:** 4 major areas
- **⚠️ Backward Compatibility Issues:** 15+ fallback patterns
- **🔴 Vendor Type Leakage:** 11+ instances in public API
- **🔧 Configuration Deep Access:** 12 violations
- **📊 Architecture Test Failures:** 5 categories
- **✅ Dependencies:** All stable and maintained

## Priority Recommendations

1. **CRITICAL:** Remove all authorization code (AuthorizedClients, TrustedServers, AuthorizationPolicy)
2. **HIGH:** Eliminate fallback patterns - fail fast instead
3. **HIGH:** Wrap vendor types in domain abstractions for public API
4. **MEDIUM:** Refactor configuration access to use capability injection
5. **LOW:** Fix naming conventions and domain purity issues

## Next Steps

1. Create separate issues/PRs for each category
2. Start with removing authorization code (biggest scope violation)
3. Refactor public API to hide vendor types
4. Eliminate fallback patterns systematically
5. Apply dependency injection for configuration access

---
*Generated based on CLAUDE.md rules analysis*
*Date: 2024*