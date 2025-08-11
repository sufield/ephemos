# Graceful Shutdown Implementation

## Overview

Ephemos provides comprehensive graceful shutdown capabilities that ensure all resources are properly cleaned up when a server stops. This includes:

- **SVID Watcher Cleanup**: Properly closes SPIFFE X.509 source and stops certificate watchers
- **Connection Pool Draining**: Gracefully closes all pooled connections
- **Request Completion**: Allows in-flight requests to complete before shutdown
- **Context Deadline Support**: Respects context deadlines for time-bounded shutdowns
- **Custom Cleanup Hooks**: Supports application-specific cleanup logic

## Architecture

### Components

1. **GracefulShutdownManager**: Coordinates shutdown of all registered resources
2. **EnhancedServer**: Production-ready server with built-in graceful shutdown
3. **ShutdownConfig**: Configures timeouts and behavior during shutdown
4. **Resource Registry**: Tracks servers, clients, listeners, and SPIFFE providers

### Shutdown Phases

The shutdown process occurs in carefully orchestrated phases:

```
Phase 1: Stop Servers (Grace Period)
  ├─ Stop accepting new connections
  └─ Signal servers to begin graceful stop

Phase 2: Close Listeners (Immediate)
  └─ Prevent any new incoming connections

Phase 3: Close Clients (Drain Timeout)
  ├─ Close connection pools
  └─ Clean up client resources

Phase 4: Close SPIFFE Providers (Grace Period)
  ├─ Stop SVID watchers
  ├─ Close X509Source
  └─ Clean up certificate resources

Phase 5: Run Cleanup Functions (Sequential)
  └─ Execute custom cleanup hooks
```

## Usage

### Basic Server with Graceful Shutdown

```go
import (
    "context"
    "github.com/sufield/ephemos/pkg/ephemos"
)

// Create shutdown configuration
shutdownConfig := &ephemos.ShutdownConfig{
    GracePeriod:  30 * time.Second,  // Time for servers to stop
    DrainTimeout: 20 * time.Second,  // Time to drain connections
    ForceTimeout: 45 * time.Second,  // Maximum total shutdown time
}

// Create server with graceful shutdown
serverOpts := &ephemos.ServerOptions{
    ConfigPath:           "config/server.yaml",
    ShutdownConfig:       shutdownConfig,
    EnableSignalHandling: true,  // Handle SIGINT/SIGTERM
}

server, err := ephemos.NewEnhancedIdentityServer(ctx, serverOpts)
if err != nil {
    log.Fatal(err)
}

// Server will shutdown gracefully on:
// - SIGINT (Ctrl+C)
// - SIGTERM
// - Context cancellation
// - Explicit Shutdown() call
```

### Advanced Configuration with Hooks

```go
serverOpts := &ephemos.ServerOptions{
    ShutdownConfig: &ephemos.ShutdownConfig{
        GracePeriod:  30 * time.Second,
        DrainTimeout: 20 * time.Second,
        ForceTimeout: 45 * time.Second,
        OnShutdownStart: func() {
            log.Info("Shutdown initiated")
            // Notify monitoring systems
            // Stop accepting new work
        },
        OnShutdownComplete: func(err error) {
            if err != nil {
                log.Error("Shutdown failed", err)
                // Alert on-call
            } else {
                log.Info("Clean shutdown completed")
            }
        },
    },
    PreShutdownHook: func(ctx context.Context) error {
        // Save application state
        // Flush metrics
        // Close database transactions
        return nil
    },
    PostShutdownHook: func(err error) {
        // Final cleanup
        // Log shutdown metrics
    },
}
```

### Custom Cleanup Functions

```go
// Register cleanup for application resources
server.RegisterCleanupFunc(func() error {
    log.Info("Closing database connections")
    return db.Close()
})

server.RegisterCleanupFunc(func() error {
    log.Info("Flushing cache")
    return cache.Flush()
})

server.RegisterCleanupFunc(func() error {
    log.Info("Saving checkpoint")
    return saveCheckpoint()
})
```

### Context Deadline Support

```go
// Shutdown with deadline
ctx, cancel := context.WithDeadline(context.Background(), 
    time.Now().Add(30*time.Second))
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Error("Shutdown deadline exceeded")
    }
}

// Or use ServeWithDeadline for time-bounded operation
deadline := time.Now().Add(1 * time.Hour)
err := server.ServeWithDeadline(ctx, listener, deadline)
```

## Configuration

### Timeout Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `GracePeriod` | 30s | Maximum time for servers to stop gracefully |
| `DrainTimeout` | 20s | Time to wait for existing requests to complete |
| `ForceTimeout` | 45s | Maximum total shutdown time before forcing |

### Recommended Settings

**For microservices (fast shutdown):**
```go
&ShutdownConfig{
    GracePeriod:  10 * time.Second,
    DrainTimeout: 5 * time.Second,
    ForceTimeout: 15 * time.Second,
}
```

**For long-running services (careful shutdown):**
```go
&ShutdownConfig{
    GracePeriod:  60 * time.Second,
    DrainTimeout: 30 * time.Second,
    ForceTimeout: 90 * time.Second,
}
```

**For batch processors (complete work):**
```go
&ShutdownConfig{
    GracePeriod:  5 * time.Minute,
    DrainTimeout: 2 * time.Minute,
    ForceTimeout: 10 * time.Minute,
}
```

## SVID Watcher Cleanup

The graceful shutdown properly handles SPIFFE/SPIRE resources:

1. **X509Source Closure**: Stops the background SVID renewal process
2. **Watcher Termination**: Cancels all certificate update watchers
3. **Connection Cleanup**: Closes the workload API connection
4. **Resource Release**: Frees memory used by cached certificates

```go
// SPIFFE provider is automatically registered and cleaned up
spiffeProvider := server.GetSPIFFEProvider()
// No manual cleanup needed - handled by shutdown manager
```

## Signal Handling

When `EnableSignalHandling` is true, the server automatically handles:

- **SIGINT** (Ctrl+C): Initiates graceful shutdown
- **SIGTERM**: Initiates graceful shutdown (Kubernetes/Docker)

```go
// Automatic signal handling
serverOpts := &ephemos.ServerOptions{
    EnableSignalHandling: true,  // Default behavior
}

// Or handle signals manually
serverOpts := &ephemos.ServerOptions{
    EnableSignalHandling: false,
}

// Manual signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigChan
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}()
```

## Testing Graceful Shutdown

### Unit Testing

```go
func TestGracefulShutdown(t *testing.T) {
    config := &ephemos.ShutdownConfig{
        GracePeriod:  100 * time.Millisecond,
        DrainTimeout: 50 * time.Millisecond,
        ForceTimeout: 200 * time.Millisecond,
    }
    
    manager := ephemos.NewGracefulShutdownManager(config)
    
    // Register mock resources
    server := &mockServer{}
    manager.RegisterServer(server)
    
    // Perform shutdown
    ctx := context.Background()
    err := manager.Shutdown(ctx)
    
    assert.NoError(t, err)
    assert.True(t, server.stopCalled)
}
```

### Integration Testing

```bash
# Start server
go run examples/graceful-shutdown/main.go &
SERVER_PID=$!

# Send requests
for i in {1..10}; do
    grpcurl -plaintext -d '{"message": "Test"}' \
        localhost:50051 ephemos.EchoService/Echo &
done

# Trigger graceful shutdown
kill -TERM $SERVER_PID

# Verify clean shutdown
wait $SERVER_PID
echo "Server shutdown cleanly with exit code: $?"
```

### Load Testing Shutdown

```bash
# Start server with deadline
SERVER_DEADLINE=1m go run examples/graceful-shutdown/main.go &

# Generate load
hey -n 10000 -c 100 -q 100 \
    -m POST -T "application/grpc" \
    http://localhost:50051/ephemos.EchoService/Echo

# Server will shutdown gracefully after 1 minute
```

## Monitoring and Observability

### Metrics to Track

1. **Shutdown Duration**: Time from shutdown signal to completion
2. **Requests Completed**: Number of requests finished during grace period
3. **Requests Dropped**: Requests that couldn't complete in time
4. **Resource Cleanup Time**: Time to close each resource type
5. **Cleanup Errors**: Errors encountered during shutdown

### Example Metrics Implementation

```go
serverOpts := &ephemos.ServerOptions{
    ShutdownConfig: &ephemos.ShutdownConfig{
        OnShutdownStart: func() {
            metrics.ShutdownInitiated.Inc()
            metrics.ShutdownStartTime.Set(time.Now().Unix())
        },
        OnShutdownComplete: func(err error) {
            duration := time.Since(startTime)
            metrics.ShutdownDuration.Observe(duration.Seconds())
            if err != nil {
                metrics.ShutdownErrors.Inc()
            }
        },
    },
}
```

## Best Practices

### 1. Always Set Reasonable Timeouts

```go
// Don't use infinite timeouts
✗ GracePeriod: 0  // Bad: Never terminates

// Use appropriate timeouts for your service
✓ GracePeriod: 30 * time.Second  // Good: Bounded time
```

### 2. Register Cleanup Functions Early

```go
// Register cleanup immediately after resource creation
db := openDatabase()
server.RegisterCleanupFunc(func() error {
    return db.Close()
})
```

### 3. Handle Cleanup Errors

```go
server.RegisterCleanupFunc(func() error {
    if err := saveState(); err != nil {
        // Log but don't fail shutdown
        log.Error("Failed to save state", err)
        return nil  // Continue shutdown
    }
    return nil
})
```

### 4. Test Shutdown Behavior

- Test with active connections
- Test with slow requests
- Test timeout scenarios
- Test cleanup function failures

### 5. Monitor Shutdown Health

- Track shutdown duration trends
- Alert on shutdown failures
- Monitor resource leaks
- Verify SVID watcher cleanup

## Troubleshooting

### Server Hangs During Shutdown

**Cause**: Long-running request or blocking operation
**Solution**: Ensure all operations respect context cancellation

```go
func (s *Service) LongOperation(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-s.doWork():
        return nil
    }
}
```

### SVID Watchers Not Closing

**Cause**: X509Source not properly registered
**Solution**: Ensure SPIFFE provider is registered with shutdown manager

```go
spiffeProvider, _ := spiffe.NewProvider(config)
shutdownManager.RegisterSPIFFEProvider(spiffeProvider)
```

### Timeout Exceeded Errors

**Cause**: Cleanup taking longer than configured timeouts
**Solution**: Increase timeouts or optimize cleanup operations

```go
// Increase timeouts for complex cleanup
shutdownConfig.ForceTimeout = 2 * time.Minute
```

### Resources Not Released

**Cause**: Missing cleanup registration
**Solution**: Register all resources that need cleanup

```go
// Register all resources
manager.RegisterServer(server)
manager.RegisterClient(client)
manager.RegisterListener(listener)
manager.RegisterSPIFFEProvider(provider)
```

## Performance Considerations

### Shutdown Performance Metrics

| Phase | Typical Duration | Optimization Tips |
|-------|-----------------|-------------------|
| Server Stop | 1-5s | Use concurrent stop for multiple servers |
| Listener Close | <100ms | Usually fast, no optimization needed |
| Client Close | 1-10s | Use connection pooling with proper limits |
| SPIFFE Close | <1s | Ensure watchers use context cancellation |
| Custom Cleanup | Variable | Run independent cleanups concurrently |

### Optimization Strategies

1. **Parallel Cleanup**: Cleanup functions run concurrently where possible
2. **Early Connection Rejection**: Stop accepting new connections immediately
3. **Request Deadlines**: Set deadlines on in-flight requests
4. **Progressive Shutdown**: Shutdown less critical resources first

## Conclusion

The graceful shutdown implementation in Ephemos ensures:

- ✅ **No resource leaks**: All resources properly released
- ✅ **Clean SVID cleanup**: Certificate watchers properly terminated
- ✅ **Request completion**: In-flight requests allowed to finish
- ✅ **Configurable behavior**: Flexible timeout and hook configuration
- ✅ **Production ready**: Battle-tested shutdown sequence
- ✅ **Observable**: Built-in hooks for metrics and monitoring

By following these guidelines, your Ephemos services will shutdown cleanly and reliably, maintaining system stability even during deployments and scaling operations.