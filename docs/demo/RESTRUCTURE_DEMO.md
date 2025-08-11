# Production-Friendly Go Layout: Ephemos Restructure

## 🎯 New Standard Structure

```
/cmd/api/                  # Main service composition root
/cmd/ephemos-cli/          # CLI tool composition root  
/internal/
  /domain/                 # Entities, value objects, domain services
    identity.go            # ServiceIdentity, Certificate, AuthenticationPolicy
    identity_test.go
  /app/                    # Use cases + ports (application layer)
    identity_service.go    # IdentityService use case
    ports.go              # IdentityProvider, TransportProvider interfaces
    errors.go             # Application errors
  /infra/                  # All adapters
    /grpc/                # gRPC transport adapters
      server.go           # IdentityServer
      client.go           # IdentityClient  
    /cli/                 # CLI adapters
      registrar.go        # SPIRE registration
    /spiffe/              # SPIFFE identity provider
      provider.go
    /config/              # Configuration provider
      provider.go
/pkg/ephemos/              # Public library API
```

## ✅ What Belongs Where

### `/internal/domain/` - Pure Business Logic
```go
// ServiceIdentity, Certificate, AuthenticationPolicy
// NO imports of frameworks, IO, HTTP, DB, etc.
// Only standard library + other domain types
```

### `/internal/app/` - Use Cases & Ports  
```go
// IdentityService - orchestrates domain + ports
// Ports: IdentityProvider, TransportProvider interfaces
// Application DTOs, application errors
// NO framework imports, only domain types + interfaces
```

### `/internal/infra/` - All Adapters
```go
// gRPC servers/clients, CLI handlers, SPIFFE providers
// Database repos, HTTP handlers, config readers
// Framework imports OK here - implements app ports
```

## 🔗 Import Rules (Enforced)

```go
cmd/*           → internal/app      → internal/domain
internal/infra  → internal/app      (implements ports)
                  internal/domain   (uses entities)

❌ NEVER: domain importing infra
❌ NEVER: app importing infra directly  
```

## 📝 Files Moved

### Use Cases: `core/services/` → `app/`
- ✅ `internal/app/identity_service.go` (was `core/services/`)

### Ports: `core/ports/` → `app/` 
- ✅ `internal/app/identity_provider.go`
- ✅ `internal/app/transport.go`
- ✅ `internal/app/configuration.go`

### Domain: `core/domain/` → `domain/`
- ✅ `internal/domain/identity.go`

### Infrastructure: `adapters/*` → `infra/`
- ✅ `internal/infra/grpc/` (was `adapters/primary/api/`)
- ✅ `internal/infra/cli/` (was `adapters/primary/cli/` + `cli/`)
- ✅ `internal/infra/spiffe/` (was `adapters/secondary/spiffe/`)
- ✅ `internal/infra/config/` (was `adapters/secondary/config/`)

## 🧪 Test Structure

```go
internal/domain/*_test.go    # Pure unit tests, no mocks
internal/app/*_test.go       # Unit tests with mocked ports  
internal/infra/*_test.go     # Integration tests (DB, gRPC)
```

## ✅ Benefits of Standard Layout

1. **Familiar**: Most Go teams recognize this structure
2. **Clear separation**: Domain/App/Infra boundaries obvious
3. **Testable**: Easy to test each layer in isolation
4. **Scalable**: Works for single service and multi-service repos
5. **Import rules**: Clear dependency flow, easy to enforce

## 📋 Quick Checklist for PRs

- [ ] All use cases in `internal/app/`?
- [ ] Ports defined in `internal/app/`, not near adapters?
- [ ] Domain pure (no framework imports)?
- [ ] Infrastructure implements app ports?
- [ ] Wiring only in `cmd/*/main.go`?
- [ ] Tests in correct layers?

## 🚀 Migration Status

- ✅ Created new structure
- ✅ Moved key files to demonstrate  
- ⏳ Update all import statements
- ⏳ Update tests and mocks
- ⏳ Verify build works