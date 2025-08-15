# Ephemos Chi Middleware

SPIFFE identity middleware for the [Chi router](https://github.com/go-chi/chi) framework. This middleware provides automatic SPIFFE certificate validation and identity context propagation for HTTP services.

## Features

- 🔒 **SPIFFE Certificate Validation**: Automatic validation of client SPIFFE certificates
- 🌐 **Identity Context Propagation**: Service identity available in request context
- 🛡️ **Flexible Authentication**: Optional or required client certificates per route
- 🎯 **Service-Based Authorization**: Route access control based on service identity
- 📝 **Structured Logging**: Integration with `slog` for security events
- ⚡ **Zero-Configuration Defaults**: Works out of the box with sensible defaults

## Installation

```bash
go get github.com/sufield/ephemos/contrib/middleware/chi
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/sufield/ephemos/contrib/middleware/chi"
)

func main() {
    r := chi.NewRouter()

    // Configure identity middleware
    config := &chimiddleware.IdentityConfig{
        ConfigPath: "/etc/ephemos/config.yaml",
        RequireClientCert: true,
    }

    // Apply middleware to protected routes
    r.Route("/api", func(r chi.Router) {
        r.Use(chimiddleware.IdentityMiddleware(config))
        r.Get("/protected", protectedHandler)
    })

    http.ListenAndServe(":8080", r)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    identity := chimiddleware.IdentityFromContext(r.Context())
    // identity contains SPIFFE ID, service name, and trust domain
}
```

## Configuration

### IdentityConfig

```go
type IdentityConfig struct {
    // Path to ephemos configuration file
    ConfigPath string

    // Whether client certificates are required (default: true)
    RequireClientCert bool

    // Allowed trust domains (empty = allow all)
    TrustDomains []string

    // Logger for middleware events
    Logger *slog.Logger
}
```

### Example Configuration

```go
config := &chimiddleware.IdentityConfig{
    ConfigPath:        "/etc/ephemos/config.yaml",
    RequireClientCert: false,  // Allow optional client certs
    TrustDomains:      []string{"example.org", "trusted.com"},
    Logger:            slog.Default(),
}
```

## Middleware Functions

### IdentityMiddleware

Primary middleware that validates SPIFFE certificates and injects identity into request context.

```go
r.Use(chimiddleware.IdentityMiddleware(config))
```

### RequireIdentity

Helper middleware that ensures client identity is present. Use after `IdentityMiddleware` when you need to enforce authentication on specific routes.

```go
r.Use(chimiddleware.IdentityMiddleware(config))
r.Use(chimiddleware.RequireIdentity)  // Enforce authentication
```

### RequireService

Middleware that restricts access to specific service names.

```go
// Only allow admin-service and operator-service
r.Use(chimiddleware.RequireService("admin-service", "operator-service"))
```

## Identity Context

### ServiceIdentity

The middleware injects a `ServiceIdentity` into the request context:

```go
type ServiceIdentity struct {
    ID     string // Full SPIFFE ID (e.g., "spiffe://example.org/workload")
    Name   string // Service name extracted from SPIFFE ID
    Domain string // Trust domain (e.g., "example.org")
}
```

### Accessing Identity

```go
func handler(w http.ResponseWriter, r *http.Request) {
    identity := chimiddleware.IdentityFromContext(r.Context())
    if identity != nil {
        fmt.Printf("Service: %s, SPIFFE ID: %s", identity.Name, identity.ID)
    }
}
```

## Route Patterns

### Mixed Authentication (Public + Protected)

```go
r := chi.NewRouter()

// Public routes (no authentication)
r.Get("/health", healthHandler)
r.Get("/public", publicHandler)

// Protected routes
r.Route("/api", func(r chi.Router) {
    r.Use(chimiddleware.IdentityMiddleware(config))
    r.Get("/protected", protectedHandler)
})
```

### Service-Specific Routes

```go
r.Route("/api", func(r chi.Router) {
    r.Use(chimiddleware.IdentityMiddleware(config))
    
    // Payment service only
    r.Route("/payment", func(r chi.Router) {
        r.Use(chimiddleware.RequireService("payment-service"))
        r.Post("/charge", chargeHandler)
    })
    
    // Admin services only
    r.Route("/admin", func(r chi.Router) {
        r.Use(chimiddleware.RequireService("admin-service", "operator-service"))
        r.Get("/users", usersHandler)
    })
})
```

### Strict Authentication

```go
// Always require client certificates
strictConfig := &chimiddleware.IdentityConfig{
    ConfigPath:        "/etc/ephemos/config.yaml",
    RequireClientCert: true,  // Always require client certs
    TrustDomains:      []string{"example.org"},
}

r.Route("/secure", func(r chi.Router) {
    r.Use(chimiddleware.IdentityMiddleware(strictConfig))
    r.Use(chimiddleware.RequireIdentity)
    r.Get("/sensitive", sensitiveHandler)
})
```

## TLS Configuration

The middleware requires HTTPS with proper TLS configuration to validate client certificates:

```go
server := &http.Server{
    Addr:    ":8080",
    Handler: r,
    TLSConfig: &tls.Config{
        ClientAuth: tls.RequestClientCert,  // Request client certificates
        MinVersion: tls.VersionTLS12,
    },
}

// Start with TLS
server.ListenAndServeTLS("server.crt", "server.key")
```

## Security Considerations

### Certificate Validation

- Client certificates are validated using SPIFFE standards
- SPIFFE ID format validation ensures proper certificate structure  
- Trust domain validation restricts access to approved domains
- Certificate parsing errors result in authentication failure

### Logging

The middleware logs important security events:

```go
// Authentication success
slog.Debug("Client identity authenticated", 
    slog.String("spiffe_id", identity.ID),
    slog.String("service_name", identity.Name))

// Authentication failure  
slog.Error("Client certificate validation failed",
    slog.String("error", err.Error()),
    slog.String("remote_addr", r.RemoteAddr))

// Access denied
slog.Warn("Service access denied",
    slog.String("service", identity.Name),
    slog.Any("allowed_services", allowedServices))
```

### Best Practices

1. **Use HTTPS**: Always run your server with TLS enabled
2. **Trust Domain Validation**: Specify allowed trust domains in production
3. **Service-Specific Routes**: Use `RequireService` for sensitive operations
4. **Structured Logging**: Configure proper logging for security monitoring
5. **Graceful Degradation**: Consider optional authentication for public endpoints

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic Usage](examples/main.go): Complete server with multiple authentication patterns
- [Service Authorization](examples/service-auth.go): Advanced service-based access control
- [Development Setup](examples/dev-server.go): Development-friendly configuration

## Error Handling

The middleware handles various error conditions:

| Condition | Response | Status Code |
|-----------|----------|-------------|
| No TLS connection | Configurable (continue or 400) | 400 Bad Request |
| Invalid client certificate | HTTP error | 401 Unauthorized |
| Missing required certificate | HTTP error | 401 Unauthorized |
| Invalid SPIFFE ID format | HTTP error | 401 Unauthorized |
| Untrusted domain | HTTP error | 401 Unauthorized |
| Service not allowed | HTTP error | 403 Forbidden |

## Integration with Ephemos

This middleware integrates with the core [ephemos](https://github.com/sufield/ephemos) library:

- Uses ephemos configuration files
- Leverages ephemos certificate management
- Compatible with ephemos service identity patterns
- Follows ephemos security best practices

## Contributing

Contributions are welcome! Please see the main [ephemos repository](https://github.com/sufield/ephemos) for contribution guidelines.

## License

This middleware is part of the ephemos project and follows the same licensing terms.