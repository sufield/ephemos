# Built-in Interceptors Guide

Ephemos provides a comprehensive set of built-in gRPC interceptors that handle authentication, identity propagation, logging, and metrics collection automatically. These interceptors eliminate boilerplate code while providing enterprise-grade observability and security for your microservices.

## Table of Contents

- [Overview](#overview)
- [Who Uses Interceptors](#who-uses-interceptors)
- [Available Interceptors](#available-interceptors)
- [Quick Start](#quick-start)
- [Configuration Options](#configuration-options)
- [Real-World Examples](#real-world-examples)
- [Service Handler Integration](#service-handler-integration)
- [Custom Implementations](#custom-implementations)
- [Performance Considerations](#performance-considerations)
- [Testing](#testing)

## Overview

Built-in interceptors provide cross-cutting concerns for gRPC services:

- **🔐 Authentication**: Automatic mTLS with SPIFFE identity validation
- **🔗 Identity Propagation**: Distributed tracing and call chain tracking  
- **📝 Logging**: Structured audit logs with security redaction
- **📊 Metrics**: Request/response observability and performance monitoring

All interceptors are designed to work together seamlessly and can be enabled/disabled independently based on your requirements.

## Who Uses Interceptors

### Primary Users

- **Backend Service Developers**: Building microservices with ephemos
- **Platform Engineers**: Setting up service mesh infrastructure
- **DevOps/SRE Teams**: Implementing observability and security
- **Security Engineers**: Enforcing zero-trust policies

### Use Cases

- **Enterprise Microservices**: Multi-service applications with strict security
- **Financial Services**: Regulatory compliance and audit trails
- **E-commerce Platforms**: High-throughput with observability requirements
- **Startup MVPs**: Quick setup with production-ready security

## Available Interceptors

### Authentication Interceptor

Provides automatic mTLS authentication using SPIFFE identities.

**Features:**
- X.509-SVID certificate validation from SPIRE
- Service-to-service authorization policies (allow/deny lists)
- Method-level authentication skipping
- Identity context injection

**Use When:**
- Implementing zero-trust security
- Requiring service-to-service authentication
- Enforcing authorization policies

### Identity Propagation Interceptor

Enables distributed tracing and maintains call context across services.

**Features:**
- Request ID generation and propagation
- Call chain tracking with circular detection
- Original caller preservation
- Custom header forwarding

**Use When:**
- Building distributed systems
- Requiring audit trails
- Implementing distributed tracing
- Debugging service interactions

### Logging Interceptor

Provides structured audit logging with security best practices.

**Features:**
- Request/response lifecycle logging
- Automatic PII/secret redaction
- Slow request detection
- Method exclusion filtering
- Identity and tracing information inclusion

**Use When:**
- Meeting compliance requirements
- Debugging production issues
- Performance monitoring
- Security auditing

### Metrics Interceptor

Collects observability metrics for monitoring and alerting.

**Features:**
- Request rate, latency, and error metrics
- Active request tracking
- Stream message counting
- Payload size observation
- Custom metrics backend integration

**Use When:**
- Setting up monitoring and alerting
- Tracking SLA compliance
- Performance optimization
- Capacity planning

## Quick Start

### Option 1: Preset Configurations (Recommended)

Use environment-specific presets for common scenarios:

```go
package main

import (
    "context"
    "net"
    
    "github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
    ctx := context.Background()
    
    // Create server with automatic interceptor configuration
    server, err := ephemos.NewIdentityServer(ctx, "")
    if err != nil {
        panic(err)
    }
    defer server.Close()
    
    // Choose configuration based on environment
    var config *ephemos.InterceptorConfig
    switch os.Getenv("ENVIRONMENT") {
    case "development":
        config = ephemos.NewDevelopmentInterceptorConfig("my-service")
    case "production":
        config = ephemos.NewProductionInterceptorConfig("my-service")
    default:
        config = ephemos.NewDefaultInterceptorConfig()
    }
    
    // Register service - interceptors are automatically applied
    serviceRegistrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
        myservice.RegisterMyServiceServer(s, &MyServiceImpl{})
    })
    
    if err := server.RegisterService(ctx, serviceRegistrar); err != nil {
        panic(err)
    }
    
    // Start server
    lis, _ := net.Listen("tcp", ":50051")
    defer lis.Close()
    
    server.Serve(ctx, lis)
}
```

### Option 2: Manual Configuration

For fine-grained control over interceptor behavior:

```go
// Create custom configuration using public API
config := &ephemos.InterceptorConfig{
    EnableAuth:                true,
    EnableIdentityPropagation: true,
    EnableLogging:             true,
    EnableMetrics:             true,
    ServiceName:              "my-service",
}

// Apply to server (done automatically with RegisterService)
// Note: Advanced configuration is handled through ephemos public API methods
serverInterceptors, streamInterceptors := ephemos.CreateServerInterceptors(
    config, identityProvider)
```

## Configuration Options

### Environment Presets

#### Development Configuration
```go
config := ephemos.NewDevelopmentInterceptorConfig("my-service")
// - Authentication: Disabled for easier testing
// - Identity Propagation: Enabled for debugging
// - Logging: Verbose with payload logging
// - Metrics: Basic collection enabled
```

#### Production Configuration
```go
config := ephemos.NewProductionInterceptorConfig("my-service")
// - Authentication: Strict with allow lists
// - Identity Propagation: Full tracing enabled
// - Logging: Security-focused, no payloads
// - Metrics: Complete observability
```

#### Default Configuration
```go
config := ephemos.NewDefaultInterceptorConfig()
// - Authentication: Enabled with basic policies
// - Identity Propagation: Disabled by default
// - Logging: Secure configuration
// - Metrics: Standard collection
```

### Authentication Configuration

```go
// Allow-list configuration (whitelist mode)
authConfig := interceptors.NewAllowListAuthConfig([]string{
    "spiffe://company.com/user-service",
    "spiffe://company.com/payment-service",
})

// Deny-list configuration (blacklist mode)  
authConfig := interceptors.NewDenyListAuthConfig([]string{
    "spiffe://company.com/deprecated-service",
})

// Advanced configuration
authConfig := &interceptors.AuthConfig{
    RequireAuthentication: true,
    AllowedServices: []string{
        "spiffe://company.com/trusted-*", // Wildcard support
    },
    RequiredClaims: map[string]string{
        "environment": "production",
        "compliance":  "pci-dss",
    },
    SkipMethods: []string{
        "/grpc.health.v1.Health/Check",  // Health checks
        "/my.Service/PublicMethod",       // Public endpoints
    },
}
```

### Identity Propagation Configuration

```go
identityConfig := &interceptors.IdentityPropagationConfig{
    IdentityProvider:        myIdentityProvider,
    PropagateOriginalCaller: true,   // Track original request source
    PropagateCallChain:      true,   // Build service call chain
    MaxCallChainDepth:      10,      // Prevent infinite loops
    CustomHeaders: []string{         // Additional headers to propagate
        "x-trace-id",
        "x-user-context",
        "x-correlation-id",
    },
}
```

### Logging Configuration

```go
// Secure configuration (production)
loggingConfig := interceptors.NewSecureLoggingConfig()
// - LogPayloads: false (security)
// - SlowRequestThreshold: 500ms
// - Excludes health checks

// Debug configuration (development)
loggingConfig := interceptors.NewDebugLoggingConfig()  
// - LogPayloads: true (debugging)
// - SlowRequestThreshold: 100ms  
// - Logs all methods

// Custom configuration
loggingConfig := &interceptors.LoggingConfig{
    LogRequests:           true,
    LogResponses:          true,
    LogPayloads:           false,  // Be careful with sensitive data
    SlowRequestThreshold:  1 * time.Second,
    ExcludeMethods: []string{
        "/grpc.health.v1.Health/Check",
        "/my.Service/HighVolumeMethod",
    },
    IncludeHeaders: []string{
        "authorization",
        "x-request-id", 
        "x-forwarded-for",
    },
}
```

### Metrics Configuration

```go
// Default configuration with no-op collector
metricsConfig := interceptors.DefaultMetricsConfig("my-service")

// Custom configuration with Prometheus
metricsConfig := &interceptors.MetricsConfig{
    MetricsCollector:     &myPrometheusCollector{},
    ServiceName:          "payment-service", 
    EnablePayloadSize:    true,   // Monitor message sizes
    EnableActiveRequests: true,   // Track concurrent requests
}
```

## Real-World Examples

### E-commerce Platform

```go
// High-security e-commerce service
authConfig := interceptors.NewAllowListAuthConfig([]string{
    "spiffe://ecommerce.com/user-service",
    "spiffe://ecommerce.com/payment-service",
    "spiffe://ecommerce.com/inventory-service",
})
authConfig.RequiredClaims = map[string]string{
    "environment":     "production",
    "pci-compliance":  "required",
}

// Comprehensive audit logging
loggingConfig := interceptors.NewSecureLoggingConfig()
loggingConfig.IncludeHeaders = []string{
    "x-transaction-id",
    "x-customer-id",
    "x-session-id",
}

// Business metrics
metricsConfig := &interceptors.MetricsConfig{
    MetricsCollector:  &ecommerceMetricsCollector{},
    ServiceName:       "order-service",
    EnablePayloadSize: true, // Monitor order sizes
}

config := &ephemos.InterceptorConfig{
    EnableAuth:                true,
    AuthConfig:               authConfig,
    EnableIdentityPropagation: true,  // For order tracking
    EnableLogging:             true,  // For audit compliance
    EnableMetrics:             true,  // For business monitoring
}
```

### Financial Services

```go
// Banking system with regulatory compliance
authConfig := interceptors.DefaultAuthConfig()
authConfig.RequiredClaims = map[string]string{
    "regulatory-zone": "banking",
    "sox-compliance":  "required", 
    "encryption-level": "aes-256",
}

// Regulatory audit logging
loggingConfig := &interceptors.LoggingConfig{
    LogRequests:          true,
    LogResponses:         true, 
    LogPayloads:          false, // Never log financial data
    SlowRequestThreshold: 100 * time.Millisecond, // Strict SLA
    IncludeHeaders: []string{
        "x-transaction-id",
        "x-regulatory-context",
        "x-audit-trail-id",
    },
}

// Regulatory metrics
metricsConfig := &interceptors.MetricsConfig{
    MetricsCollector: &regulatoryMetricsCollector{
        auditLogger:    complianceAuditor,
        alertManager:   regulatoryAlerts,
    },
    ServiceName: "trading-engine",
}
```

### Startup MVP

```go
// Simple setup for rapid development
config := ephemos.NewDevelopmentInterceptorConfig("my-startup-api")

// Override specific settings
config.AuthConfig.SkipMethods = []string{
    "/api.v1.Public/GetStatus",    // Public status endpoint
    "/api.v1.Public/GetVersion",   // Version endpoint
}

// Add basic custom metrics
config.MetricsConfig = &interceptors.MetricsConfig{
    MetricsCollector: &startupMetricsCollector{
        prometheusRegistry: prometheus.DefaultRegisterer,
    },
    ServiceName: "startup-api",
}
```

## Service Handler Integration

### Accessing Identity Information

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
    // Get authenticated client identity (from auth interceptor)
    identity, ok := interceptors.GetIdentityFromContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "authentication required")
    }
    
    log.Info("Processing payment",
        "client_service", identity.ServiceName,
        "client_spiffe_id", identity.SPIFFEID,
        "payment_amount", req.Amount)
    
    // Validate client is authorized for this operation
    if !s.isAuthorizedForPayments(identity.ServiceName) {
        return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
    }
    
    // Use identity for business logic
    return s.processPaymentForClient(ctx, req, identity.ServiceName)
}
```

### Using Distributed Tracing Information

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
    // Get distributed tracing information
    requestID, ok := interceptors.GetRequestID(ctx)
    if !ok {
        requestID = generateOrderID() // Fallback
    }
    
    originalCaller, _ := interceptors.GetOriginalCaller(ctx)
    callChain, _ := interceptors.GetCallChain(ctx)
    
    log.Info("Creating order",
        "request_id", requestID,
        "original_caller", originalCaller,
        "call_chain", callChain)
    
    // Use for correlation across services
    paymentResp, err := s.paymentClient.ChargeCustomer(ctx, &PaymentRequest{
        OrderID:   requestID,  // Correlation ID
        Amount:    req.Total,
        RequestID: requestID,  // Pass through for tracing
    })
    
    return &OrderResponse{OrderID: requestID}, nil
}
```

### Conditional Logic Based on Identity

```go
func (s *UserService) GetUserData(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
    identity, ok := interceptors.GetIdentityFromContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "authentication required")
    }
    
    // Different response based on calling service
    switch identity.ServiceName {
    case "admin-service":
        // Admin gets full user data including sensitive info
        return s.getFullUserData(ctx, req.UserID)
        
    case "recommendation-service":
        // Recommendation service gets limited data
        return s.getPublicUserData(ctx, req.UserID)
        
    case "billing-service": 
        // Billing gets payment-related data only
        return s.getBillingUserData(ctx, req.UserID)
        
    default:
        return nil, status.Error(codes.PermissionDenied, 
            fmt.Sprintf("service %s not authorized", identity.ServiceName))
    }
}
```

## Custom Implementations

### Custom Metrics Collector

```go
// Prometheus metrics collector
type PrometheusMetricsCollector struct {
    requestsTotal    *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    activeRequests   *prometheus.GaugeVec
    payloadSize      *prometheus.HistogramVec
}

func NewPrometheusCollector() *PrometheusMetricsCollector {
    return &PrometheusMetricsCollector{
        requestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "grpc_requests_total",
                Help: "Total number of gRPC requests",
            },
            []string{"method", "service", "code"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "grpc_request_duration_seconds",
                Help:    "Request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "service", "code"},
        ),
        activeRequests: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "grpc_active_requests",
                Help: "Number of active requests",
            },
            []string{"method", "service"},
        ),
        payloadSize: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "grpc_payload_size_bytes", 
                Help:    "Request/response payload size in bytes",
                Buckets: []float64{10, 100, 1000, 10000, 100000, 1000000},
            },
            []string{"method", "service", "direction"},
        ),
    }
}

func (p *PrometheusMetricsCollector) IncRequestsTotal(method, service, code string) {
    p.requestsTotal.WithLabelValues(method, service, code).Inc()
}

func (p *PrometheusMetricsCollector) ObserveRequestDuration(method, service, code string, duration time.Duration) {
    p.requestDuration.WithLabelValues(method, service, code).Observe(duration.Seconds())
}

func (p *PrometheusMetricsCollector) IncActiveRequests(method, service string) {
    p.activeRequests.WithLabelValues(method, service).Inc()
}

func (p *PrometheusMetricsCollector) DecActiveRequests(method, service string) {
    p.activeRequests.WithLabelValues(method, service).Dec()
}

// ... implement other methods
```

### Custom Authentication Logic

```go
// Custom authentication with database lookup
type DatabaseAuthInterceptor struct {
    *interceptors.AuthInterceptor
    db *sql.DB
}

func (d *DatabaseAuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
    baseInterceptor := d.AuthInterceptor.UnaryServerInterceptor()
    
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // First run standard authentication
        newCtx, err := baseInterceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
            return ctx, nil // Just return context for validation
        })
        if err != nil {
            return nil, err
        }
        
        // Additional database-based validation
        identity, _ := interceptors.GetIdentityFromContext(newCtx.(context.Context))
        
        // Check if service is active in database
        var active bool
        err = d.db.QueryRow("SELECT active FROM services WHERE spiffe_id = ?", 
            identity.SPIFFEID).Scan(&active)
        if err != nil || !active {
            return nil, status.Error(codes.PermissionDenied, "service not active")
        }
        
        // Continue with original handler
        return handler(newCtx.(context.Context), req)
    }
}
```

### Integration with OpenTelemetry

```go
// OpenTelemetry integration for distributed tracing
func setupTracingIntegration(config *ephemos.InterceptorConfig) {
    // Add OpenTelemetry headers to identity propagation
    config.IdentityPropagationConfig.CustomHeaders = []string{
        "x-trace-id",
        "x-span-id",
        "x-parent-span-id", 
        "traceparent",      // W3C Trace Context
        "tracestate",       // W3C Trace State
    }
    
    // Custom metrics collector that also creates spans
    config.MetricsConfig.MetricsCollector = &TracingMetricsCollector{
        tracer: otel.Tracer("ephemos-interceptors"),
    }
}

type TracingMetricsCollector struct {
    tracer trace.Tracer
}

func (t *TracingMetricsCollector) IncRequestsTotal(method, service, code string) {
    // Record metric and create span event
    ctx := context.Background()
    _, span := t.tracer.Start(ctx, "grpc.request.completed")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("grpc.method", method),
        attribute.String("grpc.service", service), 
        attribute.String("grpc.status_code", code),
    )
}
```

## Performance Considerations

### Interceptor Ordering

Interceptors are executed in a specific order for optimal performance:

1. **Identity Propagation Server** (extracts metadata)
2. **Authentication** (validates identity)  
3. **Logging** (records request)
4. **Metrics** (measures performance)

### Memory Usage

- **Identity Context**: ~200 bytes per request
- **Metadata Propagation**: ~500 bytes per request
- **Log Entries**: ~1KB per request (varies with payload logging)
- **Metrics**: ~50 bytes per request

### Performance Benchmarks

```go
// Typical overhead per interceptor:
// - Authentication: ~50µs per request
// - Identity Propagation: ~25µs per request  
// - Logging: ~100µs per request (without payloads)
// - Metrics: ~25µs per request
// - Total overhead: ~200µs per request

// High-throughput optimization:
config := ephemos.NewProductionInterceptorConfig("high-perf-service")
config.LoggingConfig.LogPayloads = false           // Reduce logging overhead
config.MetricsConfig.EnablePayloadSize = false     // Skip payload size calculation
config.LoggingConfig.ExcludeMethods = []string{     // Skip high-volume endpoints
    "/api.v1.HighVolume/StreamData",
}
```

### Resource Optimization

```go
// For high-throughput services
func optimizeForThroughput() *ephemos.InterceptorConfig {
    config := ephemos.NewProductionInterceptorConfig("high-throughput-service")
    
    // Minimize logging
    config.LoggingConfig.LogRequests = false
    config.LoggingConfig.LogResponses = false
    
    // Essential metrics only
    config.MetricsConfig.EnableActiveRequests = false
    config.MetricsConfig.EnablePayloadSize = false
    
    // Skip auth for internal services
    config.AuthConfig.SkipMethods = []string{
        "/internal.Service/*",  // Skip all internal methods
    }
    
    return config
}
```

## Testing

The interceptors include comprehensive test coverage (72.7%) with test files for each interceptor:

- **`auth_test.go`**: Authentication and authorization testing
- **`identity_propagation_test.go`**: Distributed tracing testing
- **`logging_test.go`**: Audit logging testing  
- **`metrics_test.go`**: Performance monitoring testing

### Testing Your Integration

```go
func TestServiceWithInterceptors(t *testing.T) {
    // Create test configuration
    config := &ephemos.InterceptorConfig{
        EnableAuth: true,
        AuthConfig: interceptors.DefaultAuthConfig(),
    }
    
    // Create test identity
    identity := &interceptors.AuthenticatedIdentity{
        SPIFFEID:    "spiffe://test.com/test-service",
        ServiceName: "test-service",
    }
    
    // Create context with identity
    ctx := context.WithValue(context.Background(), 
        interceptors.IdentityContextKey{}, identity)
    
    // Test your service handler
    resp, err := myService.MyMethod(ctx, &MyRequest{})
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

### Mock Implementations

```go
// Mock metrics collector for testing
type MockMetricsCollector struct {
    RequestsTotal int
    Durations     []time.Duration
}

func (m *MockMetricsCollector) IncRequestsTotal(method, service, code string) {
    m.RequestsTotal++
}

func (m *MockMetricsCollector) ObserveRequestDuration(method, service, code string, duration time.Duration) {
    m.Durations = append(m.Durations, duration)
}

// Use in tests
collector := &MockMetricsCollector{}
config := &ephemos.InterceptorConfig{
    EnableMetrics: true,
    MetricsConfig: &interceptors.MetricsConfig{
        MetricsCollector: collector,
        ServiceName:      "test-service",
    },
}
```

## Best Practices

### Security
- Always use `NewSecureLoggingConfig()` in production
- Never enable payload logging for sensitive services
- Use allow-lists instead of deny-lists for authentication
- Regularly rotate SPIFFE certificates

### Performance  
- Disable unnecessary interceptors for high-throughput endpoints
- Use method exclusion lists for health checks and metrics endpoints
- Monitor interceptor overhead in production
- Consider payload size limits for logging and metrics

### Observability
- Include correlation IDs in all log entries
- Set appropriate slow request thresholds
- Use structured logging consistently
- Integrate with your existing monitoring stack

### Testing
- Test interceptor configurations in staging environments
- Validate authentication policies before production deployment
- Monitor metrics collection performance
- Test failure scenarios (authentication failures, etc.)

---

For more examples and advanced usage, see the `/examples/interceptors/` directory in the ephemos repository.