# Out-of-Scope Functionality Analysis

This document identifies functionality in the ephemos codebase that is out of scope for an identity-based authentication library focused on SPIFFE certificates.

## Core Scope Definition

**Ephemos should focus ONLY on:**
- SPIFFE identity validation and authentication
- Certificate-based authentication (mTLS)
- Identity propagation between services
- Authentication interceptors and middleware
- Identity context management
- Service registration with SPIRE

## Out-of-Scope Functionality Found

### 1. File/Data Management Services ❌

**Files:**
- `internal/adapters/http/adapter.go` (lines 124-212)
- `internal/core/ports/service.go` (FileService interface)

**Description:**
- Complete file upload/download/listing service
- HTTP REST endpoints: POST `/upload/{filename}`, GET `/download/{filename}`, GET `/list`
- File system operations and data storage

**Why out of scope:** File management is business logic unrelated to identity authentication. An authentication library should handle certificates and identity validation, not file operations.

**Action:** **REMOVE** - This belongs in application-specific services, not an authentication library.

### 2. Business Logic Services (Echo Service) ❌

**Files:**
- `internal/adapters/http/adapter.go` (lines 68-122)
- `internal/core/ports/service.go` (lines 68-75)

**Description:**
- Echo service with `Echo(message)` and `Ping()` methods
- HTTP endpoints: POST `/echo`, POST `/ping`
- Example business logic for demonstration

**Why out of scope:** Echo services are demo/example functionality. An identity authentication library should not include sample business services.

**Action:** **REMOVE** or move to separate examples repository.

### 3. Generic HTTP REST Framework ❌

**Files:**
- `internal/adapters/http/adapter.go` (entire file)
- `internal/transport/http_adapter.go` (entire file)
- `internal/transport/server.go` (entire file)

**Description:**
- Generic HTTP service mounting and routing infrastructure
- Automatic REST endpoint generation from service interfaces
- Transport-agnostic server framework

**Why out of scope:** While HTTP support is needed for SPIFFE certificate validation, generic service hosting and REST framework capabilities go beyond authentication scope.

**Action:** **SIMPLIFY** - Keep minimal HTTP client support for authentication, remove generic service hosting.

### 4. Generic Service Registry/Discovery ❌

**Files:**
- `internal/core/ports/service.go` (lines 9-63)
- `internal/adapters/http/adapter.go` (service mounting logic)

**Description:**
- `ServiceRegistry` for mapping service types to implementations
- Generic service descriptor and method introspection
- Dynamic service registration system

**Why out of scope:** Service discovery is an infrastructure concern separate from identity authentication. An authentication library should validate identities, not manage arbitrary service registration.

**Action:** **REMOVE** - Service discovery should be handled by external systems (Kubernetes, Consul, etc.).

### 5. Generic Health Checking Services ❌

**Files:**
- `internal/adapters/http/adapter.go` (lines 214-251)
- `internal/core/ports/service.go` (lines 77-96)

**Description:**
- `HealthService` interface with `Check(serviceName)` method
- Health status types (Unknown, Serving, NotServing, ServiceUnknown)
- HTTP endpoints for arbitrary service health checking

**Why out of scope:** While identity services need health checks, generic health checking for arbitrary services is beyond authentication scope.

**Action:** **REMOVE** generic health service. Keep internal health checks for identity/certificate validation only.

### 6. Demo/Example Business Logic ❌

**Files:**
- `scripts/demo/` (entire directory)
- `contrib/middleware/chi/examples/main.go` (payment handlers)

**Description:**
- Mock payment service handlers and business logic
- Demo infrastructure with business-specific examples
- SPIRE installation scripts (these are OK)

**Why out of scope:** Demo infrastructure with mock business services like payment processing is not core authentication functionality.

**Action:** **SIMPLIFY** - Keep SPIRE setup scripts and basic identity examples. Remove business logic examples.

### 7. Generic Transport Framework ❌

**Files:**
- `internal/transport/server.go` (entire file)
- `internal/transport/http_adapter.go` (generic mounting)

**Description:**
- Transport-agnostic server framework
- Generic service mounting with automatic protocol adaptation
- Configurable transport types beyond identity needs

**Why out of scope:** Generic transport abstractions go beyond identity authentication needs. The library should provide SPIFFE-authenticated connections, not generic transport frameworks.

**Action:** **SIMPLIFY** - Focus on SPIFFE-specific HTTP client and gRPC connection management only.

### 8. CLI Tools for Non-Identity Purposes ⚠️

**Files:**
- `cmd/config-validator/main.go` (generic config validation)

**Description:**
- Generic configuration validation tool
- Production readiness checking beyond identity scope

**Why partially out of scope:** While SPIFFE service registration CLI is appropriate, generic configuration validation extends beyond authentication.

**Action:** **SIMPLIFY** - Keep identity-specific validation (service names, trust domains, socket paths). Remove generic application configuration support.

## Recommendations

### Remove Entirely:
1. **FileService** and all file management functionality
2. **EchoService** and demo business logic  
3. **Generic service registry** and discovery
4. **Generic health checking** services
5. **Generic transport framework** beyond SPIFFE needs
6. **Business logic examples** (payment handlers, etc.)

### Keep But Simplify:
1. **HTTP client support** (for SPIFFE certificate authentication)
2. **Configuration management** (limited to identity settings: service name, trust domain, socket path)
3. **CLI tools** (limited to SPIFFE service registration and identity validation)
4. **Contrib middleware examples** (remove business logic, keep identity examples)

### Core Identity Scope to Maintain:
- ✅ SPIFFE certificate validation and management
- ✅ mTLS connection establishment with SPIFFE certificates  
- ✅ Identity context propagation and call chain tracking
- ✅ Authentication interceptors and middleware (auth, logging, metrics)
- ✅ Service registration with SPIRE
- ✅ Trust domain and service name configuration
- ✅ Identity-based authorization policies

## Impact Assessment

**High Priority Removals (No Dependencies):**
- FileService (pure business logic)
- EchoService (demo functionality)
- Generic health checking

**Medium Priority Simplifications:**
- HTTP adapter (keep client features, remove server hosting)
- Service registry (keep identity services only)
- Transport framework (focus on SPIFFE connections)

**Low Priority (Documentation/Examples):**
- CLI tools (simplify to identity scope)
- Demo scripts (keep SPIRE setup, remove business examples)

## Post-Cleanup Benefits

After removing out-of-scope functionality:
1. **Clearer purpose** - Focus on identity authentication only
2. **Smaller footprint** - Reduced dependencies and complexity
3. **Better maintainability** - Single responsibility principle
4. **Easier adoption** - Clear value proposition for identity authentication
5. **Security focus** - Concentrate on authentication security rather than generic application features

---

**Next Steps:** Prioritize removal of FileService and EchoService as they are purely business logic with no relation to identity authentication.