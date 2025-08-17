# SPIRE Health Monitoring Example

This example demonstrates how to use Ephemos health monitoring capabilities to monitor SPIRE infrastructure components using their built-in HTTP health endpoints.

## Overview

SPIRE provides built-in health check endpoints that can be enabled via configuration:
- `/live` - Liveness check (process running)
- `/ready` - Readiness check (ready to serve requests)

Ephemos leverages these endpoints rather than implementing custom health checks from scratch, following SPIRE best practices.

## Prerequisites

1. **SPIRE Server** running with health checks enabled:
   ```hcl
   health_checks {
       listener_enabled = true
       bind_address = "localhost"
       bind_port = "8080"
       live_path = "/live"
       ready_path = "/ready"
   }
   ```

2. **SPIRE Agent** running with health checks enabled:
   ```hcl
   health_checks {
       listener_enabled = true
       bind_address = "localhost"
       bind_port = "8081"
       live_path = "/live"
       ready_path = "/ready"
   }
   ```

## Running the Example

### Option 1: Using the Example Program

```bash
# Build and run the example
go run main.go
```

This will:
1. Perform a one-time health check of both SPIRE server and agent
2. Start continuous monitoring with 30-second intervals
3. Log health status changes

### Option 2: Using the Ephemos CLI

```bash
# One-time health check
ephemos health --server-address localhost:8080 --agent-address localhost:8081

# Continuous monitoring
ephemos health monitor --interval 30s --server-address localhost:8080 --agent-address localhost:8081

# Using configuration file
ephemos health --config config.yaml

# JSON output for automation
ephemos health --config config.yaml --format json

# Verbose output with details
ephemos health --config config.yaml --verbose
```

## Configuration

The health monitoring can be configured via:

### 1. Configuration File (config.yaml)
```yaml
health:
  enabled: true
  timeout: "10s"
  interval: "30s"
  
  server:
    address: "localhost:8080"
    live_path: "/live"
    ready_path: "/ready"
    use_https: false
    
  agent:
    address: "localhost:8081"
    live_path: "/live"
    ready_path: "/ready"
    use_https: false
```

### 2. Environment Variables
```bash
export EPHEMOS_HEALTH_ENABLED=true
export EPHEMOS_HEALTH_TIMEOUT=10s
export EPHEMOS_HEALTH_INTERVAL=30s
export EPHEMOS_SPIRE_SERVER_ADDRESS=localhost:8080
export EPHEMOS_SPIRE_AGENT_ADDRESS=localhost:8081
```

### 3. Command Line Flags
```bash
ephemos health \
  --server-address localhost:8080 \
  --agent-address localhost:8081 \
  --check-timeout 10s \
  --interval 30s
```

## Integration Examples

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: health-monitor
    image: my-app:latest
    command: ["/usr/local/bin/ephemos"]
    args: ["health", "monitor", "--config", "/etc/ephemos/config.yaml"]
    volumeMounts:
    - name: config
      mountPath: /etc/ephemos
    livenessProbe:
      exec:
        command: ["/usr/local/bin/ephemos", "health", "check", "--quiet"]
      initialDelaySeconds: 30
      periodSeconds: 30
```

### Docker Compose

```yaml
version: '3.8'
services:
  health-monitor:
    image: ephemos:latest
    command: ["ephemos", "health", "monitor", "--server-address", "spire-server:8080", "--agent-address", "spire-agent:8081"]
    depends_on:
      - spire-server
      - spire-agent
    restart: unless-stopped
```

### Prometheus Integration

The health monitoring service can be extended with Prometheus metrics:

```go
// Custom Prometheus reporter
type PrometheusHealthReporter struct {
    healthGauge *prometheus.GaugeVec
}

func (r *PrometheusHealthReporter) ReportHealth(result *ports.HealthResult) error {
    status := 0.0
    if result.Status == ports.HealthStatusHealthy {
        status = 1.0
    }
    
    r.healthGauge.WithLabelValues(result.Component).Set(status)
    return nil
}
```

## Expected Output

### Successful Health Check
```
✅ Overall Health: HEALTHY

✅ spire-server: HEALTHY (15ms) - liveness and readiness checks passed
✅ spire-agent: HEALTHY (8ms) - liveness and readiness checks passed
```

### Failed Health Check
```
❌ Overall Health: UNHEALTHY

❌ spire-server: UNHEALTHY (503ms) - readiness check failed: service unavailable
✅ spire-agent: HEALTHY (12ms) - liveness and readiness checks passed
```

### JSON Output
```json
{
  "overall_health": "healthy",
  "components": {
    "spire-server": {
      "status": "healthy",
      "component": "spire-server",
      "message": "Component is healthy and ready",
      "checked_at": "2024-01-15T10:30:00Z",
      "response_time": "15ms",
      "details": {
        "liveness_status": "healthy",
        "readiness_status": "healthy",
        "url": "http://localhost:8080/live"
      }
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   ❌ spire-server: UNHEALTHY - Health check failed: dial tcp localhost:8080: connection refused
   ```
   - Ensure SPIRE server is running
   - Check that health checks are enabled in SPIRE configuration
   - Verify the correct address and port

2. **503 Service Unavailable**
   ```
   ❌ spire-server: UNHEALTHY - readiness check failed: service unavailable
   ```
   - SPIRE server is running but not ready to serve requests
   - Check SPIRE server logs for initialization issues
   - Verify trust domain and datastore configuration

3. **Timeout Errors**
   ```
   ❌ spire-agent: UNHEALTHY - Health check failed: context deadline exceeded
   ```
   - Increase timeout with `--check-timeout` flag
   - Check network connectivity
   - Verify SPIRE agent is responsive

### Debug Mode

Enable verbose logging for detailed health check information:

```bash
ephemos health --verbose --config config.yaml
```

This will show:
- HTTP request/response details
- Response times for each endpoint
- Detailed error messages
- Component-specific health information

## Security Considerations

1. **Network Security**: Health endpoints should only be accessible from trusted networks
2. **Authentication**: Consider adding custom headers for authentication if needed
3. **TLS**: Use HTTPS in production environments by setting `use_https: true`
4. **Firewall Rules**: Restrict access to health check ports to monitoring systems only

## Best Practices

1. **Monitoring Intervals**: Use reasonable intervals (30s-60s) to avoid overwhelming SPIRE components
2. **Timeout Configuration**: Set timeouts shorter than monitoring intervals
3. **Alert Thresholds**: Configure alerts for consecutive failed checks, not single failures
4. **Log Aggregation**: Centralize health check logs for monitoring and alerting
5. **Health Check Validation**: Test health check configuration in development environments first