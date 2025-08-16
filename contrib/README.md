# Ephemos Contrib

**Extensions for frameworks; consumes core primitives like certs, bundles, and authorizers.**

This directory contains framework-specific integrations for Ephemos that extend the core SPIFFE identity and gRPC functionality to HTTP frameworks like Chi and Gin.

## Architecture

The Ephemos core provides:
- ✅ **SPIFFE certificates** via `IdentityService.GetCertificate()`
- ✅ **Trust bundles** via `IdentityService.GetTrustBundle()`
- ✅ **Authorizers** for validating peer identities
- ✅ **gRPC connectivity** with automatic mTLS

Contrib extensions consume these primitives to provide:
- 🌐 **HTTP middleware** for popular Go web frameworks
- 📚 **Examples** showing HTTP client integration
- 📖 **Guides** for framework-specific setup

## Plugin Points

The core exposes these interfaces for contrib extensions:

### Identity Access
```go
// Access service certificates and trust bundles
type IdentityService interface {
    GetCertificate() (*x509.Certificate, error)
    GetTrustBundle() (*x509.CertPool, error)
}
```

### Authorization Policies  
```go
// Build authorizers for peer validation
func NewAuthorizerFromConfig(cfg AuthConfig) tlsconfig.Authorizer
func AuthorizeMemberOf(domain string) tlsconfig.Authorizer
```

### TLS Configuration
```go
// Build HTTP transport with mTLS
func MTLSClientConfig(source, bundle, authorizer) *tls.Config
```

## Available Extensions

### HTTP Middleware

#### Chi Router
```go
import "github.com/sufield/ephemos/contrib/middleware/chi"

r := chi.NewRouter()
r.Use(chimiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"spiffe://prod.company.com/service-b"},
}))
r.Get("/api/data", dataHandler) // Handler receives verified identity document
```

- **Location**: [`middleware/chi/`](middleware/chi/)
- **Purpose**: SPIFFE authentication middleware for [Chi router](https://github.com/go-chi/chi)
- **Features**: Identity extraction, authorization policies, error handling

#### Gin Framework  
```go
import "github.com/sufield/ephemos/contrib/middleware/gin"

r := gin.Default()
r.Use(ginmiddleware.IdentityAuthentication(ephemos.IdentitySetting{
    AllowedServices: []string{"spiffe://prod.company.com/service-b"},
}))
r.GET("/api/data", dataHandler) // Handler receives verified identity document
```

- **Location**: [`middleware/gin/`](middleware/gin/)  
- **Purpose**: SPIFFE authentication middleware for [Gin framework](https://github.com/gin-gonic/gin)
- **Features**: Identity extraction, authorization policies, error handling

### HTTP Client Examples

#### Using Core Primitives with net/http
```go
// Get certificates and trust bundle from core
cert, _ := identityService.GetCertificate()
bundle, _ := identityService.GetTrustBundle()
authorizer := ephemos.AuthorizeMemberOf("prod.company.com")

// Build HTTP client with mTLS
tlsConfig := tlsconfig.MTLSClientConfig(cert, bundle, authorizer)
httpClient := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
    },
}

// Make authenticated requests
resp, _ := httpClient.Get("https://api-server.prod.company.com/data")
```

- **Location**: [`examples/`](examples/)
- **Purpose**: Show HTTP client patterns using core certificates/bundles  
- **Patterns**: Client creation, connection pooling, error handling

### Documentation

#### HTTP Integration Guide
Detailed instructions for integrating Ephemos with HTTP services.

- **Location**: [`docs/HTTP_CLIENT.md`](docs/HTTP_CLIENT.md)
- **Content**: Step-by-step HTTP setup, best practices, troubleshooting
- **Focus**: Core provides certs/bundles; contrib glues to http.Transport

## Design Principles

### ✅ **Zero Core Bloat**
- Contrib depends on core, never the reverse
- Core remains lightweight and gRPC-focused
- HTTP extensions live separately

### ✅ **Standard Go Patterns**  
- Follow middleware patterns established by each framework
- Use stdlib `net/http` where possible
- Composable middleware design (like Chi/Gin ecosystem)

### ✅ **Plugin Architecture**
- Contrib consumes only public core APIs
- No internal imports from core
- Clear interface boundaries for extension

### ✅ **Framework Agnostic Core**
- Core provides identity primitives, not framework-specific code
- Similar to OpenTelemetry-go: core APIs + contrib integrations
- Easy to add new frameworks without changing core

## Getting Started

### 1. Set Up Core
First, set up Ephemos core for your service:
```go
// Core setup - same for all frameworks
config := &ports.Configuration{
    Service: ports.ServiceConfig{
        Name: "my-service",
        Domain: "prod.company.com",
    },
}
client, _ := ephemos.IdentityClient(ctx, ephemos.WithConfig(config))
```

### 2. Choose Your Framework
Then, add the appropriate contrib middleware:

**For Chi:**
```bash
go get github.com/sufield/ephemos/contrib/middleware/chi
```

**For Gin:**  
```bash
go get github.com/sufield/ephemos/contrib/middleware/gin
```

**For Custom HTTP:**
See [`examples/http_client.go`](examples/http_client.go)

### 3. Add Middleware
Integrate with your HTTP framework using contrib middleware.

## Contributing

### Adding New Framework Support

To add support for a new framework (e.g., Echo, Fiber):

1. **Create framework directory**: `mkdir middleware/echo`
2. **Implement middleware interface**: Follow patterns from `chi/` or `gin/`
3. **Use only public core APIs**: Import only `pkg/ephemos`, never `internal/`
4. **Add examples**: Include basic usage example
5. **Update this README**: Add section for new framework

### Guidelines

- **No internal imports**: Only use `github.com/sufield/ephemos/pkg/ephemos`
- **Follow framework patterns**: Use each framework's middleware conventions
- **Include tests**: Test middleware with framework's testing patterns  
- **Add documentation**: Update README and add example code

## FAQ

### Q: Why separate core and contrib?

**A:** This follows Go community best practices seen in projects like OpenTelemetry-go:
- **Core stays lightweight**: Only SPIFFE identity + gRPC, no HTTP bloat
- **Framework choice**: Users pick Chi, Gin, or custom without forcing dependencies
- **Independent versioning**: Contrib can evolve faster than core
- **Zero-trust focus**: Core provides identity; frameworks add transport

### Q: Can I use core without contrib?

**A:** Yes! Core provides complete gRPC functionality. Use contrib only if you need HTTP/REST APIs.

### Q: How do I build custom middleware?

**A:** Use the same patterns as Chi/Gin middleware - consume core's `GetCertificate()`, `GetTrustBundle()`, and authorizer functions. See [`examples/http_client.go`](examples/http_client.go) for patterns.

### Q: What about WebSockets, GraphQL, etc.?

**A:** Follow the same pattern - create contrib extensions that consume core primitives. The core identity and certificate management works with any transport.