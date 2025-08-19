# Ephemos Gin Middleware

SPIFFE identity middleware for the [Gin web framework](https://gin-gonic.com/).

This middleware enables automatic SPIFFE certificate validation and identity context propagation for HTTP services using Gin, providing seamless integration with Ephemos core identity primitives.

## Features

- 🔒 **SPIFFE Certificate Validation**: Automatic validation of client SPIFFE certificates
- 🏷️ **Identity Context**: Service identity available in Gin context and standard Go context
- 🛡️ **Trust Domain Filtering**: Restrict access to specific trust domains
- 🎯 **Service Authorization**: Require specific service identities for endpoints
- 🔧 **Flexible Configuration**: Optional or required client certificates
- 📝 **Structured Logging**: Built-in logging with configurable logger

## Quick Start

### Installation

```bash
go get github.com/sufield/ephemos/contrib/middleware/gin
```

### Basic Usage

```go
package main

import (
    "github.com/gin-gonic/gin"
    ginmiddleware "github.com/sufield/ephemos/contrib/middleware/gin"
)

func main() {
    r := gin.Default()

    // Configure SPIFFE identity middleware
    config := &ginmiddleware.IdentityConfig{
        ConfigPath:        "/etc/ephemos/config.yaml",
        RequireClientCert: true,
        TrustDomains:      []string{"prod.company.com"},
    }

    // Apply to all routes
    r.Use(ginmiddleware.IdentityMiddleware(config))

    r.GET("/api/data", func(c *gin.Context) {
        identity := ginmiddleware.IdentityFromGinContext(c)
        c.JSON(200, gin.H{
            "message": "Hello, " + identity.Name,
            "caller":  identity.ID,
        })
    })

    r.Run(":8080")
}
```

## Configuration

### IdentityConfig

```go
type IdentityConfig struct {
    // ConfigPath is the path to the ephemos configuration file
    ConfigPath string

    // RequireClientCert determines if client certificates are required
    // Default: true (mutual TLS required)
    RequireClientCert bool

    // TrustDomains specifies allowed trust domains. Empty means allow all.
    TrustDomains []string

    // Logger for middleware events
    Logger *slog.Logger
}
```

### Examples

#### Optional Authentication
```go
config := &ginmiddleware.IdentityConfig{
    ConfigPath:        "/etc/ephemos/config.yaml",
    RequireClientCert: false, // Allow requests without client certs
}
r.Use(ginmiddleware.IdentityMiddleware(config))
```

#### Trust Domain Restrictions
```go
config := &ginmiddleware.IdentityConfig{
    ConfigPath:   "/etc/ephemos/config.yaml",
    TrustDomains: []string{"prod.company.com", "staging.company.com"},
}
r.Use(ginmiddleware.IdentityMiddleware(config))
```

## Middleware Functions

### Core Middleware

#### `IdentityMiddleware(config *IdentityConfig) gin.HandlerFunc`
Main middleware that validates SPIFFE certificates and injects identity into context.

### Authorization Middleware

#### `RequireIdentity(c *gin.Context)`
Ensures a valid SPIFFE identity is present. Returns 401 if not authenticated.

```go
authenticated := r.Group("/api")
authenticated.Use(ginmiddleware.RequireIdentity)
```

#### `RequireService(allowedServices ...string) gin.HandlerFunc`
Restricts access to specific service identities.

```go
r.Use(ginmiddleware.RequireService("payment-service", "order-service"))
```

#### `RequireTrustDomain(allowedDomains ...string) gin.HandlerFunc`
Restricts access to specific trust domains.

```go
r.Use(ginmiddleware.RequireTrustDomain("prod.company.com"))
```

## Identity Access

### From Gin Context (Recommended)

```go
func handler(c *gin.Context) {
    identity := ginmiddleware.IdentityFromGinContext(c)
    if identity != nil {
        c.JSON(200, gin.H{
            "service": identity.Name,
            "domain":  identity.Domain,
            "id":      identity.ID,
        })
    }
}
```

### From Standard Context

```go
func handler(c *gin.Context) {
    identity := ginmiddleware.IdentityFromContext(c.Request.Context())
    if identity != nil {
        // Use identity...
    }
}
```

## Advanced Usage

### Route Groups with Different Requirements

```go
r := gin.Default()

// Apply identity middleware globally
r.Use(ginmiddleware.IdentityMiddleware(config))

// Public routes (identity optional)
r.GET("/health", healthHandler)

// Authenticated routes
authenticated := r.Group("/api")
authenticated.Use(ginmiddleware.RequireIdentity)
{
    authenticated.GET("/whoami", whoamiHandler)
    authenticated.GET("/data", dataHandler)
}

// Service-specific routes
payment := r.Group("/payment")
payment.Use(ginmiddleware.RequireIdentity)
payment.Use(ginmiddleware.RequireService("payment-service"))
{
    payment.POST("/charge", chargeHandler)
}

// Internal routes (specific trust domain)
internal := r.Group("/internal")
internal.Use(ginmiddleware.RequireIdentity)
internal.Use(ginmiddleware.RequireTrustDomain("prod.company.com"))
{
    internal.GET("/metrics", metricsHandler)
}
```

### Custom Authorization Logic

```go
func requireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        identity := ginmiddleware.IdentityFromGinContext(c)
        if identity == nil {
            c.JSON(401, gin.H{"error": "Authentication required"})
            c.Abort()
            return
        }

        // Extract role from service name or identity claims
        if !hasRole(identity, role) {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// Usage
r.Use(requireRole("admin"))
```

## Error Handling

The middleware returns appropriate HTTP status codes:

- **400 Bad Request**: No TLS connection when required
- **401 Unauthorized**: Invalid or missing client certificate
- **403 Forbidden**: Valid certificate but access denied (wrong service/domain)

Error responses use Gin's JSON format:

```json
{
    "error": "Client certificate required"
}
```

## Testing

### Running Tests

```bash
cd contrib/middleware/gin
go test ./...
```

### Mock Testing

For unit tests, you can test the validation logic without requiring actual SPIFFE certificates:

```go
func TestMyHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    identity := &ginmiddleware.ServiceIdentity{
        ID:     "spiffe://test.org/my-service",
        Name:   "my-service", 
        Domain: "test.org",
    }
    
    // Create test context with identity
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("spiffe_identity", identity)
    
    // Test your handler
    myHandler(c)
}
```

## Example Application

See [`examples/main.go`](examples/main.go) for a complete example application demonstrating:

- Public and authenticated endpoints
- Service-specific authorization
- Trust domain restrictions
- Proper error handling
- Structured logging

Run the example:

```bash
cd examples
go run main.go
```

## Integration with Core

This middleware consumes the following core Ephemos primitives:

- `ephemos.IdentityClient()` - For configuration validation
- SPIFFE certificate validation using `go-spiffe` library
- Trust domain and service name parsing

The middleware follows Ephemos architecture principles:
- **Zero Core Bloat**: No framework dependencies in core
- **Plugin Architecture**: Consumes only public core APIs
- **Standard Patterns**: Follows Gin middleware conventions

## Comparison with Chi Middleware

| Feature | Gin Middleware | Chi Middleware |
|---------|----------------|----------------|
| Framework | Gin | Chi |
| Identity Access | `IdentityFromGinContext(c)` | `IdentityFromContext(ctx)` |
| Error Format | JSON responses | Plain text |
| Context Storage | Gin context + Go context | Go context only |
| Authorization | Same patterns | Same patterns |

## Requirements

- Go 1.24+
- Gin v1.10.0+
- SPIFFE Workload API socket available
- Valid Ephemos configuration

## Contributing

1. Follow existing patterns from Chi middleware
2. Maintain compatibility with core primitives
3. Add tests for new functionality
4. Update documentation

## See Also

- [Chi Middleware](../chi/) - Chi router middleware
- [Core HTTP Client Examples](../../examples/http_client.go) - Direct HTTP integration
- [Ephemos Core Documentation](../../../README.md)