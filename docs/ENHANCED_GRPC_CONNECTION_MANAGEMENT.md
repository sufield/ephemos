# Enhanced gRPC Connection Management

This document describes the enhanced gRPC connection management capabilities in ephemos, which leverage the official gRPC-Go library's robust support for connection management, dial options, backoff strategies, and retry policies.

## Overview

The enhanced transport layer provides:

- **Advanced Connection Configuration** - Customizable timeouts, backoff, and retry policies
- **Connection Pooling** - Efficient connection reuse with health monitoring
- **Automatic Cleanup** - Background cleanup of idle and unhealthy connections
- **Multiple Presets** - Pre-configured settings for different environments and use cases
- **Full gRPC Feature Support** - Leverages all gRPC-Go connection management features

## Key Features

### 1. Dial Options and Connection Backoff

The enhanced transport supports all standard gRPC dial options:

```go
// Connection configuration with advanced backoff
config := &transport.ConnectionConfig{
    ConnectTimeout: 30 * time.Second,
    BackoffConfig: backoff.Config{
        BaseDelay:  1.0 * time.Second,
        Multiplier: 1.6,
        Jitter:     0.2,
        MaxDelay:   120 * time.Second, // Cap backoff at 2 minutes
    },
    // ... other options
}

// Converts to gRPC dial options
options := config.ToDialOptions()
// Includes: grpc.WithConnectParams, grpc.WithKeepaliveParams, etc.
```

### 2. Keepalive Configuration

Configure connection health monitoring:

```go
KeepaliveParams: keepalive.ClientParameters{
    Time:                10 * time.Second, // Send keepalive ping every 10s
    Timeout:             5 * time.Second,  // Wait 5s for ping response  
    PermitWithoutStream: true,             // Send pings even without active streams
},
```

### 3. Service Configuration and Retries

Built-in retry policies with exponential backoff:

```go
ServiceConfig: `{
    "methodConfig": [
        {
            "name": [{"service": ""}],
            "retryPolicy": {
                "maxAttempts": 5,
                "initialBackoff": "1s", 
                "maxBackoff": "30s",
                "backoffMultiplier": 2.0,
                "retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED", "ABORTED"]
            }
        }
    ]
}`
```

### 4. Idle Timeout Management

Control connection lifecycle:

```go
// 30-minute idle timeout (default)
IdleTimeout: 30 * time.Minute,

// Disable idle timeout for persistent connections
IdleTimeout: 0,
```

## Configuration Presets

### Default Production Configuration

```go
config := transport.DefaultConnectionConfig()
```

**Characteristics:**
- **Connect Timeout**: 30 seconds
- **Backoff**: 1s base, 1.6x multiplier, 120s max
- **Keepalive**: 10s intervals, 5s timeout
- **Idle Timeout**: 30 minutes
- **Message Size**: 4MB max
- **Pooling**: Disabled
- **Retries**: 5 attempts with exponential backoff

**Use Cases:** General production services, microservices, APIs

### Development Configuration

```go
config := transport.DevelopmentConnectionConfig()
```

**Characteristics:**
- **Connect Timeout**: 5 seconds (faster for development)
- **Backoff**: 500ms base, 10s max (quicker recovery)
- **Idle Timeout**: 5 minutes (shorter for development)
- **Retries**: 3 attempts (fewer retries for faster feedback)

**Use Cases:** Local development, testing, debugging

### High-Throughput Configuration  

```go
config := transport.HighThroughputConnectionConfig()
```

**Characteristics:**
- **Connect Timeout**: 10 seconds
- **Keepalive**: 30s intervals (less frequent)
- **Idle Timeout**: Disabled (persistent connections)
- **Message Size**: 16MB max
- **Pooling**: Enabled with 10 connections
- **Optimized for**: High-volume, long-lived connections

**Use Cases:** Data streaming, batch processing, high-volume APIs

## Connection Pooling

The enhanced transport includes intelligent connection pooling:

### Features

- **Health Monitoring** - Automatically removes unhealthy connections
- **Usage Tracking** - Tracks active connection usage
- **Background Cleanup** - Periodic cleanup of idle connections
- **Thread-Safe** - Concurrent access with proper synchronization

### Configuration

```go
config := &transport.ConnectionConfig{
    EnablePooling: true,
    PoolSize:      5,
    // ... other options
}
```

### Pool Management

```go
// Connections are automatically:
// 1. Created on demand
// 2. Reused when healthy  
// 3. Removed when unhealthy or idle too long
// 4. Cleaned up periodically (every 30 seconds)
```

## Connection Health Monitoring

Enhanced connections provide health monitoring:

```go
conn, err := client.Connect("my-service", "localhost:50051")
if err != nil {
    return err
}

// Check connection health
if !conn.IsHealthy() {
    log.Warn("Connection is not healthy")
}

// Get current state
state := conn.GetState() // Ready, Connecting, Idle, etc.

// Wait for state change
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
conn.WaitForStateChange(ctx, connectivity.Connecting)
```

## Usage Examples

### Basic Usage

```go
// Create provider with default configuration
spiffeProvider := spiffe.NewProvider()
transport := transport.NewGRPCProvider(spiffeProvider)

// Or with custom configuration
config := transport.HighThroughputConnectionConfig()
transport := transport.NewGRPCProviderWithConfig(spiffeProvider, config)
```

### Custom Configuration

```go
// Create custom configuration for specific requirements
config := &transport.ConnectionConfig{
    ConnectTimeout: 15 * time.Second,
    BackoffConfig: backoff.Config{
        BaseDelay:  500 * time.Millisecond,
        Multiplier: 2.0,
        MaxDelay:   30 * time.Second,
    },
    KeepaliveParams: keepalive.ClientParameters{
        Time:    20 * time.Second,
        Timeout: 5 * time.Second,
    },
    EnablePooling: true,
    PoolSize:      3,
    ServiceConfig: `{
        "methodConfig": [{
            "name": [{"service": ""}],
            "retryPolicy": {
                "maxAttempts": 4,
                "initialBackoff": "1s",
                "maxBackoff": "15s",
                "backoffMultiplier": 1.8,
                "retryableStatusCodes": ["UNAVAILABLE"]
            }
        }]
    }`,
}
```

### Environment-Specific Usage

```go
var config *transport.ConnectionConfig

switch os.Getenv("ENVIRONMENT") {
case "development":
    config = transport.DevelopmentConnectionConfig()
case "production":
    config = transport.DefaultConnectionConfig()
case "high-throughput":
    config = transport.HighThroughputConnectionConfig()
default:
    config = transport.DefaultConnectionConfig()
}

provider := transport.NewGRPCProviderWithConfig(spiffeProvider, config)
```

## Best Practices

### 1. Choose Appropriate Configuration

- **Development**: Use `DevelopmentConnectionConfig()` for faster feedback
- **Production APIs**: Use `DefaultConnectionConfig()` for reliability
- **High-Volume Systems**: Use `HighThroughputConnectionConfig()` with pooling
- **Custom Requirements**: Create custom configuration as needed

### 2. Connection Pooling Guidelines

- **Enable pooling** for high-volume, long-lived connections
- **Disable pooling** for low-volume or short-lived connections
- **Monitor pool utilization** and adjust pool size accordingly
- **Consider memory usage** when setting pool sizes

### 3. Timeout and Backoff Tuning

- **Connect timeouts**: Balance responsiveness vs. network conditions
- **Backoff parameters**: Adjust based on service recovery characteristics
- **Retry attempts**: Consider downstream service capacity
- **Keepalive settings**: Balance health monitoring vs. network overhead

### 4. Service Configuration

- **Configure retries** for appropriate error codes
- **Set method-specific timeouts** when needed
- **Use load balancing** policies when multiple endpoints are available
- **Monitor retry rates** to avoid overwhelming downstream services

## Performance Characteristics

### Memory Usage

- **Default configuration**: ~500 bytes per connection
- **With pooling**: Additional ~200 bytes per pooled connection
- **Service configuration**: ~1KB per configuration

### Network Overhead

- **Keepalive pings**: Minimal overhead (few bytes every 10-30 seconds)
- **Connection establishment**: Reduced with pooling
- **Retry attempts**: Configurable to balance reliability vs. load

### Connection Limits

- **Pool size**: Recommended 3-10 connections per service
- **Concurrent connections**: Limited by system resources
- **Idle timeout**: Balance connection reuse vs. resource usage

## Monitoring and Observability

### Connection Health

```go
// Log connection state changes
conn.WaitForStateChange(ctx, currentState)
log.Info("Connection state changed", "new_state", conn.GetState())
```

### Pool Utilization

```go
// Monitor pool statistics (in custom implementation)
// - Active connections
// - Pool hit rate  
// - Connection creation rate
// - Cleanup frequency
```

### Metrics Integration

The enhanced transport integrates with the existing metrics interceptors to provide:

- Connection establishment time
- Pool hit/miss rates
- Connection health statistics
- Retry attempt counts
- Backoff delay distributions

## Integration with Existing Interceptors

The enhanced connection management works seamlessly with existing interceptors:

```go
// Create provider with enhanced connection management
provider := transport.NewGRPCProviderWithConfig(spiffeProvider, config)

// Create client with interceptors
client, err := provider.CreateClient(cert, trustBundle, authPolicy)

// Connections will use:
// - Authentication interceptors
// - Identity propagation  
// - Logging interceptors
// - Metrics interceptors
// - Enhanced connection management
```

## Migration Guide

### From Basic Transport

```go
// Before: Basic transport
provider := transport.NewGRPCProvider(spiffeProvider)

// After: Enhanced transport with explicit configuration  
config := transport.DefaultConnectionConfig() // Same behavior
provider := transport.NewGRPCProviderWithConfig(spiffeProvider, config)
```

### Gradual Enhancement

1. **Start with default configuration** (no behavior change)
2. **Add environment-specific configurations** 
3. **Enable pooling for high-volume services**
4. **Fine-tune based on monitoring data**

## Testing

Comprehensive tests cover:

- Configuration validation
- Connection pooling logic
- Health monitoring
- Cleanup routines
- Concurrent access safety
- Integration with existing components

Run tests:
```bash
go test ./internal/adapters/secondary/transport/ -v
```

## Future Enhancements

Potential future improvements:

- **Circuit breaker integration** for fault tolerance
- **Connection warming** for improved cold start performance  
- **Advanced load balancing** strategies
- **Connection metrics dashboards** for monitoring
- **Automatic configuration tuning** based on observed patterns

---

This enhanced gRPC connection management provides production-ready, scalable, and observable connection handling while maintaining compatibility with existing ephemos components.