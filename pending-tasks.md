# Ephemos Pending Tasks

This document outlines the implementation gaps, TODO items, and missing functionality identified in the ephemos codebase. Tasks are organized by priority and complexity.

## 🎯 **Architectural Focus**

Ephemos is **identity-based authentication library** that focuses solely on:
- **SPIFFE identity verification** using go-spiffe library  
- **Service-to-service authentication** with SPIFFE certificates
- **Middleware integration** for existing HTTP frameworks (chi, gin, net/http)
- **CLI tools for administrators** (service registration, management)

**What Ephemos does NOT do:**
- Custom HTTP server/client implementation
- Transport layer reinvention  
- Framework replacement

**Integration approach (following industry patterns):**
- **Separate packages for each framework** (like OpenTelemetry contrib model)
- Use go-spiffe as the standard SPIFFE implementation
- Focus on developer experience with simple APIs (`IdentityServer()`, `IdentityClient()`)

**Architectural Decision: Separate Packages**
Following patterns from OpenTelemetry, Auth0, and Casbin:
- `github.com/sufield/ephemos-contrib/middleware/chi` 
- `github.com/sufield/ephemos-contrib/middleware/gin` (future)
- Core ephemos library stays framework-agnostic
- Each framework gets its own optimized integration package

**Consistent API Pattern:**
```go
// Chi Framework (separate package)
import "github.com/sufield/ephemos-contrib/middleware/chi"
r := chi.NewRouter()
r.Use(chi.IdentityMiddleware(config))

// Future: Gin Framework (separate package) 
import "github.com/sufield/ephemos-contrib/middleware/gin"
r := gin.Default()
r.Use(gin.IdentityMiddleware(config))
```

---

## 🚨 High Priority (Core Functionality)

### 1. Remove Public Service Registration API
**File**: `pkg/ephemos/public_api.go:163-167`
**Status**: Should be removed from public API
**Description**: Service registration will be CLI-only for admin use, not exposed to developers as public API. The `RegisterService` method should be removed from the public `Server` interface.
**Implementation needed**:
- Remove `RegisterService` method from public `Server` interface
- Move service registration to CLI commands in `cmd/ephemos-cli`
- Update documentation to clarify admin vs developer APIs
- Simplify public API for developers

### 2. SPIFFE Certificate Validation  
**File**: `internal/adapters/secondary/transport/grpc_provider.go:71,80`
**Status**: Using insecure TLS configuration
**Description**: Both client and server TLS configs use `InsecureSkipVerify: true`
**Implementation needed**:
- Proper SPIFFE X.509 certificate validation
- Integration with SPIRE-issued certificates
- Secure mTLS handshake implementation using go-spiffe

## 🔧 Medium Priority (Public API Features)

### 3. Chi Framework Middleware (Separate Repository)
**Files**: New `github.com/sufield/ephemos-contrib` repository needed
**Status**: Create separate contrib repository following industry patterns
**Description**: Chi middleware as separate package (following OpenTelemetry contrib model)
**Implementation needed**:
- Create `ephemos-contrib` repository
- Package: `github.com/sufield/ephemos-contrib/middleware/chi`
- `chi.IdentityMiddleware(config)` function for SPIFFE authentication
- Identity context propagation in Chi request context
- Certificate validation using go-spiffe library
- Documentation and examples

### 4. Connection Interface Implementation
**File**: `pkg/ephemos/public_api.go:134-143` 
**Status**: Client.Connect() method needs proper implementation for HTTP clients
**Description**: Client should provide authenticated HTTP client with SPIFFE certificates
**Implementation needed**:
- HTTP client with SPIFFE certificate-based authentication
- Integration with go-spiffe for certificate management
- Service discovery for target service endpoints
- Proper certificate validation and rotation

Developers should only be exposed to Certificate (not Spiffe Certificate)

## 🔧 Administrative (CLI Implementation)

### 5. Service Registration CLI Commands
**File**: `cmd/ephemos-cli/` (needs new commands)
**Status**: CLI commands for service registration need implementation
**Description**: Since service registration is admin-only, implement CLI commands for service management
**Implementation needed**:
- Add `ephemos-cli register-service` command
- Add `ephemos-cli list-services` command  
- Add `ephemos-cli remove-service` command
- Add service discovery and status commands
- SPIRE integration for service identity management
- Admin authentication/authorization for CLI operations

## 🏗️ Architecture & Dependencies

### 6. Remove Custom HTTP Transport Implementation
**Files**: `internal/adapters/http/`, `internal/adapters/secondary/transport/`
**Status**: Should be simplified/removed
**Description**: Focus on go-spiffe authentication only, not custom HTTP transport. Integrate with existing frameworks instead.
**Implementation needed**:
- Remove custom HTTP server/client implementations
- Keep only SPIFFE certificate management and validation
- Simplify transport layer to focus on authentication
- Provide middleware hooks for existing HTTP frameworks

### 7. Core Domain Dependency Cleanup  
**File**: Architecture boundary test failures
**Status**: Core domain imports external SPIFFE libraries, should use go-spiffe directly
**Description**: Embrace go-spiffe as the standard library for SPIFFE operations
**Implementation needed**:
- Use go-spiffe types directly where appropriate
- Focus domain on authentication policies and identity validation
- Remove unnecessary abstractions over go-spiffe functionality

### 8. Public API Boundary Violations
**Status**: pkg/ephemos imports internal packages directly
**Description**: Public API directly imports internal adapters, violating API boundaries
**Implementation needed**:
- Create proper abstraction layer
- Remove direct internal imports from public API
- Implement facade pattern for public API

## 🧪 Testing & Quality

### 9. Integration Test Coverage
**Files**: Multiple test files with `t.Skip()` calls
**Status**: Several integration tests are skipped
**Description**: Tests are skipped due to missing dependencies or implementation gaps
**Implementation needed**:
- Enable SPIFFE provider contract tests when SPIRE is available
- Complete server registration tests  
- Implement full client connection tests

### 10. Mock Implementations
**Files**: 
- `internal/adapters/interceptors/identity_propagation_test.go:32,36`
**Status**: Mock providers return "not implemented" errors
**Description**: Test mocks are incomplete, limiting test coverage
**Implementation needed**:
- Complete mock identity provider implementations
- Add proper certificate and trust bundle mocks
- Enable comprehensive testing scenarios

### 11. gRPC Mock Response Serialization
**File**: `internal/adapters/interceptors/integration_test.go:368`
**Status**: gRPC response testing is limited
**Description**: Integration tests skip full response validation due to serialization issues
**Implementation needed**:
- Fix gRPC mock response serialization
- Enable complete end-to-end response testing
- Add proper message validation

## 📋 Documentation & Maintenance

### 12. Bazel Build System Tests
**File**: `scripts/BUILD.bazel:105-107`
**Status**: Go-based Bazel tests are commented out
**Description**: Bazel build system needs Go test integration
**Implementation needed**:
- Add Go tests to Bazel build
- Ensure build system completeness
- Integrate with CI pipeline

### 13. Security Placeholder Updates
**File**: `docs/security/CONFIGURATION_SECURITY.md:671`
**Status**: Contains placeholder contact information
**Description**: Security documentation has placeholder phone number (XXX-XXX-XXXX)
**Implementation needed**:
- Update with real security contact information
- Ensure incident response procedures are accurate

## 🚀 Future Enhancements

### 14. Gin Framework Middleware (Contrib Repository)
**Files**: `github.com/sufield/ephemos-contrib/middleware/gin/`
**Status**: Follow-up after Chi implementation in same contrib repo
**Description**: Extend consistent API pattern to Gin framework in contrib repository
**Implementation needed**:
- Package: `github.com/sufield/ephemos-contrib/middleware/gin`
- `gin.IdentityMiddleware(config)` function with same behavior as Chi version
- Gin-specific context integration
- Same API pattern as Chi but optimized for Gin
- Ensure consistent developer experience across frameworks

### 15. Tracing Integration
**Status**: Mentioned in documentation but not implemented
**Description**: Distributed tracing integration for service communications  
**Implementation needed**:
- OpenTelemetry integration with go-spiffe
- Trace context propagation through SPIFFE identity
- Performance monitoring for certificate operations

### 15. Metrics Collection
**Status**: Basic structure exists in middleware
**Description**: Comprehensive metrics collection for identity operations
**Implementation needed**:
- Prometheus metrics integration
- Authentication success/failure rates  
- Certificate rotation and validation metrics
- go-spiffe operation performance metrics

### 16. Advanced Authorization Policies  
**Status**: Basic authentication implemented
**Description**: Fine-grained authorization beyond service identity
**Implementation needed**:
- Role-based access control
- Attribute-based policies
- Dynamic policy evaluation

## 💡 Implementation Notes

### Priority Guidelines:
1. **High Priority**: Blocks core functionality, affects smoke test completeness
2. **Medium Priority**: Public API completeness, developer experience 
3. **Administrative**: CLI tools for admins (service registration, management)
4. **Architecture**: Code quality, maintainability, proper boundaries
5. **Testing**: Coverage, reliability, debugging capability
6. **Future**: Performance, monitoring, advanced features

### Estimated Effort:
- **High Priority**: 1-2 weeks total (mainly certificate validation)
- **Medium Priority**: 1-2 weeks total  
- **Administrative**: 1-2 weeks (CLI command implementation)
- **Architecture**: 2-3 weeks (requires careful refactoring)
- **Testing**: 1 week
- **Future**: 3-4 weeks (can be done incrementally)

### Dependencies:
- SPIFFE certificate validation depends on proper domain abstraction
- CLI service registration requires SPIRE integration and admin auth
- Full integration testing depends on core functionality completion
- Public API cleanup can proceed independently of service registration

---
*Last Updated: 2025-08-15*
*Generated from codebase analysis of TODO comments, test failures, and missing implementations*