# HTTP Client with SPIFFE Authentication

The ephemos library now provides an HTTP client with built-in SPIFFE certificate authentication, enabling secure HTTP communication between services using the same identity verification as gRPC connections.

## Features

- 🔒 **SPIFFE Certificate Authentication**: HTTP requests use the same SPIFFE certificates as gRPC connections
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

### Using with Existing Service Discovery

Ephemos focuses on authentication - use your existing service discovery solution:

```go
// Use your existing service discovery (Kubernetes, Consul, etc.)
address, err := myServiceRegistry.Lookup("payment-service")
if err != nil {
    log.Fatalf("Service lookup failed: %v", err)
}

// Ephemos handles authentication to the discovered service
conn, err := client.Connect(ctx, "payment-service", address)
if err != nil {
    log.Fatalf("Connection failed: %v", err)
}
defer conn.Close()

// Use authenticated HTTP client
httpClient := conn.HTTPClient()
resp, err := httpClient.Get("https://" + address + "/api/balance")
```
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
// Look up service addresses using your service registry
paymentAddr, _ := serviceRegistry.Lookup("payment-service") 
orderAddr, _ := serviceRegistry.Lookup("order-service")

// Connect to multiple services with authentication
paymentConn, err := client.Connect(ctx, "payment-service", paymentAddr)
if err != nil {
    log.Fatalf("Failed to connect to payment service: %v", err)
}
defer paymentConn.Close()

orderConn, err := client.Connect(ctx, "order-service", orderAddr)
if err != nil {
    log.Fatalf("Failed to connect to order service: %v", err)
}
defer orderConn.Close()

// Each connection has its own authenticated HTTP client
paymentClient := paymentConn.HTTPClient()
orderClient := orderConn.HTTPClient()

// Make authenticated requests to different services
paymentResp, _ := paymentClient.Get("https://" + paymentAddr + "/api/balance")
orderResp, _ := orderClient.Get("https://" + orderAddr + "/api/orders")
```

### Error Handling

```go
// Service discovery is handled by your infrastructure
address, err := serviceRegistry.Lookup("payment-service")
if err != nil {
    log.Printf("Service discovery failed: %v", err)
    return
}

conn, err := client.Connect(ctx, "payment-service", address)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "certificate"):
        log.Printf("Certificate error: %v", err)
        // Handle certificate issues
    case strings.Contains(err.Error(), "connection"):
        log.Printf("Connection error: %v", err)
        // Handle connection errors
    default:
        log.Printf("Authentication error: %v", err)
        // Handle other authentication errors
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
// Use existing service discovery
address, _ := serviceRegistry.Lookup("payment-service")

conn, err := ephemosClient.Connect(ctx, "payment-service", address)
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

httpClient := conn.HTTPClient() // Now with SPIFFE authentication
resp, err := httpClient.Get("https://" + address + "/api/balance")
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
4. **Connection Reuse**: Reuse connections when making multiple requests

## Troubleshooting

### Common Issues

1. **Service Connection Failures**
   ```
   Error: failed to connect to service payment-service at address
   ```
   - Verify service is running and accessible at the address
   - Check network connectivity
   - Ensure address format is correct (host:port)

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
| Certificate Rotation | ✅ | ❌ | ✅ |
| REST API Support | ✅ | ✅ | ❌ |
| Binary Protocol | ❌ | ❌ | ✅ |
| Browser Compatible | ✅ | ✅ | ❌ |
| Identity-based Auth | ✅ | ❌ | ✅ |

## Future Enhancements

- [ ] Load balancing support
- [ ] Circuit breaker integration
- [ ] Metrics and tracing
- [ ] Advanced connection pooling strategies
- [ ] WebSocket support with SPIFFE authentication

## See Also

- [SPIFFE Certificate Validation](SPIFFE_CERTIFICATES.md)
- [Service Discovery Guide](SERVICE_DISCOVERY.md)
- [Security Best Practices](SECURITY.md)
- [Performance Tuning](PERFORMANCE.md)