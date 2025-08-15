# HTTP Client with SPIFFE Authentication

The ephemos library now provides an HTTP client with built-in SPIFFE certificate authentication, enabling secure HTTP communication between services using the same identity verification as gRPC connections.

## Features

- 🔒 **SPIFFE Certificate Authentication**: HTTP requests use the same SPIFFE certificates as gRPC connections
- 🌐 **Service Discovery**: Automatic discovery of service endpoints
- 🔄 **Certificate Rotation**: Automatic handling of certificate lifecycle
- ⚡ **Connection Pooling**: Optimized HTTP transport configuration
- 🛡️ **Security by Default**: Secure TLS configuration with proper validation

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"

    "github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
    ctx := context.Background()

    // Create ephemos client
    client, err := ephemos.IdentityClient(ctx, "/etc/ephemos/config.yaml")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Connect to a service
    conn, err := client.Connect(ctx, "payment-service", "https://payment.example.com")
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // Get HTTP client with SPIFFE authentication
    httpClient := conn.HTTPClient()

    // Make authenticated HTTP requests
    resp, err := httpClient.Get("https://payment.example.com/api/balance")
    if err != nil {
        log.Fatalf("HTTP request failed: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Failed to read response: %v", err)
    }

    fmt.Printf("Response: %s\n", body)
}
```

### Service Discovery

The HTTP client supports automatic service discovery, eliminating the need to hardcode service addresses:

```go
// Connect using service discovery (no address needed)
conn, err := client.ConnectByName(ctx, "payment-service")
if err != nil {
    log.Fatalf("Service discovery failed: %v", err)
}
defer conn.Close()

// Use HTTP client as normal
httpClient := conn.HTTPClient()
resp, err := httpClient.Get("/api/balance") // Relative URL - uses discovered endpoint
```

### Alternative: Connect with Empty Address

```go
// Connect with empty address triggers service discovery
conn, err := client.Connect(ctx, "payment-service", "")
if err != nil {
    log.Fatalf("Service discovery failed: %v", err)
}
defer conn.Close()
```

## Service Discovery

The HTTP client includes built-in service discovery that attempts to locate services using common patterns:

### Discovery Patterns

1. **Kubernetes**: `{service}.default.svc.cluster.local:443`
2. **Consul**: `{service}.service.consul:443`
3. **AWS/Cloud**: `{service}.internal:443`
4. **Direct DNS**: `{service}:443`

### Custom Discovery

For custom service discovery, extend the `discoverService` method or configure external service registries.

## Security Features

### SPIFFE Certificate Authentication

The HTTP client automatically:
- Uses the same SPIFFE certificates as gRPC connections
- Validates server certificates against the trust bundle
- Performs mutual TLS authentication
- Handles certificate rotation seamlessly

### TLS Configuration

```go
// The HTTP client includes secure TLS defaults:
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
    },
    InsecureSkipVerify: false, // Always validate certificates
}
```

### Connection Security

- **Redirect Limiting**: Maximum 10 redirects to prevent attacks
- **Timeout Protection**: Configurable timeouts for all operations
- **Certificate Validation**: Full certificate chain verification
- **Connection Pooling**: Secure connection reuse

## Configuration

### Transport Settings

The HTTP client includes optimized transport configuration:

```go
transport := &http.Transport{
    MaxIdleConns:          100,           // Connection pool size
    IdleConnTimeout:       90 * time.Second,  // Connection reuse timeout
    TLSHandshakeTimeout:   10 * time.Second,  // TLS handshake timeout
    ExpectContinueTimeout: 1 * time.Second,   // 100-continue timeout
}
```

### Client Settings

```go
client := &http.Client{
    Timeout:   30 * time.Second, // Request timeout
    Transport: transport,        // Configured transport
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 10 {
            return fmt.Errorf("stopped after 10 redirects")
        }
        return nil
    },
}
```

## Advanced Usage

### Multiple Service Connections

```go
// Connect to multiple services
paymentConn, err := client.ConnectByName(ctx, "payment-service")
if err != nil {
    log.Fatalf("Failed to connect to payment service: %v", err)
}
defer paymentConn.Close()

orderConn, err := client.ConnectByName(ctx, "order-service")
if err != nil {
    log.Fatalf("Failed to connect to order service: %v", err)
}
defer orderConn.Close()

// Each connection has its own HTTP client
paymentClient := paymentConn.HTTPClient()
orderClient := orderConn.HTTPClient()

// Make requests to different services
paymentResp, _ := paymentClient.Get("/api/balance")
orderResp, _ := orderClient.Get("/api/orders")
```

### Error Handling

```go
conn, err := client.ConnectByName(ctx, "payment-service")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "service discovery failed"):
        log.Printf("Service not found: %v", err)
        // Handle service discovery failure
    case strings.Contains(err.Error(), "certificate"):
        log.Printf("Certificate error: %v", err)
        // Handle certificate issues
    default:
        log.Printf("Connection error: %v", err)
        // Handle other connection errors
    }
    return
}
defer conn.Close()
```

### Request Customization

```go
httpClient := conn.HTTPClient()

// Create custom request
req, err := http.NewRequestWithContext(ctx, "POST", "/api/transfer", bytes.NewReader(data))
if err != nil {
    log.Fatalf("Failed to create request: %v", err)
}

// Add custom headers
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-Request-ID", "12345")

// Execute request
resp, err := httpClient.Do(req)
if err != nil {
    log.Fatalf("Request failed: %v", err)
}
defer resp.Body.Close()
```

## Integration with Existing Code

### Migration from Standard HTTP Clients

**Before:**
```go
httpClient := &http.Client{Timeout: 30 * time.Second}
resp, err := httpClient.Get("https://payment.example.com/api/balance")
```

**After:**
```go
conn, err := ephemosClient.ConnectByName(ctx, "payment-service")
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

httpClient := conn.HTTPClient() // Now with SPIFFE authentication
resp, err := httpClient.Get("/api/balance")
```

### Framework Integration

The HTTP client works seamlessly with existing HTTP frameworks and libraries:

```go
// Works with any HTTP client library
import "github.com/go-resty/resty/v2"

httpClient := conn.HTTPClient()
restyClient := resty.NewWithClient(httpClient)

resp, err := restyClient.R().
    SetHeader("Accept", "application/json").
    Get("/api/data")
```

## Best Practices

### Connection Management

1. **Reuse Connections**: Create connections once and reuse them
2. **Proper Cleanup**: Always defer `conn.Close()`
3. **Context Usage**: Use proper context for cancellation
4. **Error Handling**: Handle both connection and request errors

### Security

1. **Certificate Validation**: Never disable certificate validation
2. **Timeouts**: Set appropriate timeouts for your use case
3. **Redirect Limits**: Keep redirect limits reasonable
4. **Service Identity**: Verify service identity in responses

### Performance

1. **Connection Pooling**: Leverage built-in connection pooling
2. **Keep-Alive**: Use persistent connections when possible
3. **Request Batching**: Batch related requests when appropriate
4. **Service Discovery Caching**: Cache discovered service addresses

## Troubleshooting

### Common Issues

1. **Service Discovery Failures**
   ```
   Error: service payment-service not found via discovery
   ```
   - Verify service is running and accessible
   - Check DNS resolution for discovery patterns
   - Ensure network connectivity

2. **Certificate Errors**
   ```
   Error: failed to extract TLS config for HTTP client
   ```
   - Verify SPIFFE certificates are valid
   - Check certificate expiration
   - Ensure trust bundle is current

3. **Connection Timeouts**
   ```
   Error: context deadline exceeded
   ```
   - Increase timeout values
   - Check network latency
   - Verify service responsiveness

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
// Add debug logging
log.Printf("Connecting to service: %s", serviceName)
conn, err := client.ConnectByName(ctx, serviceName)
if err != nil {
    log.Printf("Connection failed: %v", err)
    return
}
defer conn.Close()

log.Printf("Connection established, creating HTTP client")
httpClient := conn.HTTPClient()
```

## Comparison with Other Solutions

| Feature | Ephemos HTTP Client | Standard HTTP | gRPC |
|---------|-------------------|---------------|------|
| SPIFFE Authentication | ✅ | ❌ | ✅ |
| Service Discovery | ✅ | ❌ | ❌ |
| Certificate Rotation | ✅ | ❌ | ✅ |
| REST API Support | ✅ | ✅ | ❌ |
| Binary Protocol | ❌ | ❌ | ✅ |
| Browser Compatible | ✅ | ✅ | ❌ |

## Future Enhancements

- [ ] Load balancing support
- [ ] Circuit breaker integration
- [ ] Metrics and tracing
- [ ] Custom service discovery providers
- [ ] WebSocket support with SPIFFE authentication

## See Also

- [SPIFFE Certificate Validation](SPIFFE_CERTIFICATES.md)
- [Service Discovery Guide](SERVICE_DISCOVERY.md)
- [Security Best Practices](SECURITY.md)
- [Performance Tuning](PERFORMANCE.md)