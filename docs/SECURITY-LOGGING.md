# Security Guidelines for Logging

## Overview

The Ephemos library implements automatic redaction of sensitive data in all logging output using a custom slog.Handler. This prevents accidental exposure of credentials, certificates, private keys, and other sensitive information in logs.

## Usage

### Basic Setup

All applications using Ephemos should use the secure logger:

```go
import (
    "log/slog"
    "os"
    "github.com/sufield/ephemos/internal/adapters/logging"
)

func main() {
    // Setup secure structured logging
    baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })
    secureLogger := logging.NewSecureLogger(baseHandler)
    slog.SetDefault(secureLogger)
    
    // All subsequent logging will be automatically secured
    slog.Info("Server starting", "port", 8080)
}
```

### Automatic Redaction

The secure logger automatically redacts fields containing sensitive patterns:

```go
// These fields will be automatically redacted
slog.Info("Authentication", "password", "secret123")           // → password=[REDACTED]
slog.Info("Certificate", "private_key", "-----BEGIN PRIVATE KEY...") // → private_key=[REDACTED]
slog.Info("Token", "bearer_token", "eyJhbGci...")            // → bearer_token=[REDACTED]

// These fields will NOT be redacted
slog.Info("Server", "port", 8080)                            // → port=8080
slog.Info("Request", "user_id", "12345")                     // → user_id=12345
```

### Sensitive Field Patterns

The following field name patterns trigger automatic redaction:

- `password`, `secret`, `token`, `key`
- `private_key`, `privatekey`, `private-key`
- `cert`, `certificate`, `credentials`
- `auth`, `bearer`, `authorization`
- Any compound field containing these terms (e.g., `user_password`, `api_token`)


## Content-Based Redaction

The secure logger also performs content-based redaction for string values:

### Certificate Content
```go
certPEM := `-----BEGIN CERTIFICATE-----
MIIDETCCAfmgAwIBAgIRAK+RuN...
-----END CERTIFICATE-----`

slog.Info("Loaded certificate", "data", certPEM)
// Output: data=[REDACTED]
```

### JWT Tokens
```go
jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
slog.Info("Authentication successful", "token", jwtToken)
// Output: token=[REDACTED]
```

## Best Practices

### DO ✅

- **Use secure logger**: Always initialize with `logging.NewSecureLogger()`
- **Log business events**: Service starts, requests processed, connections established
- **Log operational metrics**: Response times, success/failure counts, resource usage
- **Use structured logging**: Key-value pairs instead of formatted strings
- **Log errors safely**: Use `logging.Error()` helper for error messages

```go
slog.Info("Request processed", 
    "method", "GET",
    "path", "/api/users", 
    "duration_ms", 42,
    "status", "success")
```

### DON'T ❌

- **Don't log raw sensitive data**: Even in debug mode
- **Don't bypass the secure logger**: Direct use of standard library loggers
- **Don't log entire request/response bodies**: May contain sensitive data
- **Don't assume field names protect you**: Use the secure logger for all logging

```go
// WRONG - bypasses secure logging
log.Printf("User password: %s", password)

// WRONG - may contain sensitive headers
slog.Info("HTTP request", "headers", allHeaders)

// WRONG - response may contain sensitive data  
slog.Info("API response", "body", responseBody)
```

## Log Levels

### Production Logging Levels

- **ERROR**: System failures, unrecoverable errors
- **WARN**: Recoverable errors, degraded performance, retries
- **INFO**: Business events, successful operations, lifecycle events
- **DEBUG**: Should be disabled in production

### Debug Logging

Debug logging should only be enabled in development or for specific troubleshooting:

```go
baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo, // Production: INFO or higher
})
```

## Testing

### Unit Tests

Test logging behavior without sensitive data exposure:

```go
func TestUserAuthentication(t *testing.T) {
    var buf bytes.Buffer
    baseHandler := slog.NewTextHandler(&buf, nil)
    secureLogger := logging.NewSecureLogger(baseHandler)
    slog.SetDefault(secureLogger)
    
    // Test your code
    authenticateUser("user123", "password123")
    
    output := buf.String()
    
    // Verify sensitive data is redacted
    if strings.Contains(output, "password123") {
        t.Error("Password should not appear in logs")
    }
    if !strings.Contains(output, logging.RedactedValue) {
        t.Error("Expected password field to be redacted")
    }
}
```

## Monitoring & Alerting

### Log Monitoring

Set up monitoring to detect potential sensitive data leaks:

- Alert on log messages containing patterns like `BEGIN PRIVATE KEY`
- Monitor for JWT-like tokens (base64 strings with dots)
- Watch for suspicious field names being logged

### Security Auditing

Regularly audit your logging:

```bash
# Search for potential sensitive data in logs
grep -i "password\|secret\|key\|token" /var/log/myapp.log

# Should only show [REDACTED] values, not actual secrets
```

## Configuration

### Environment-based Configuration

```go
func setupLogging() *slog.Logger {
    level := slog.LevelInfo
    if os.Getenv("DEBUG") == "true" {
        level = slog.LevelDebug
    }
    
    baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    })
    
    return logging.NewSecureLogger(baseHandler)
}
```

### Structured vs Text Handlers

- **Development**: Use `slog.NewTextHandler` for human-readable logs
- **Production**: Use `slog.NewJSONHandler` for machine parsing

## Compliance

This secure logging implementation helps meet compliance requirements:

- **GDPR**: Prevents logging of personal data
- **PCI DSS**: Protects payment card information
- **SOC 2**: Ensures secure handling of sensitive data
- **HIPAA**: Protects healthcare information from logs

## Troubleshooting

### Common Issues

1. **Sensitive data still appearing**: Check that you're using the secure logger, not standard library loggers
2. **Over-redaction**: Review sensitive field patterns if legitimate data is being redacted
3. **Performance concerns**: The redactor has minimal overhead, but excessive debug logging can impact performance

### Debugging Redaction

To understand what's being redacted:

```go
// Temporarily log field names being checked
slog.Debug("Field analysis", 
    "field_name", fieldName,
    "is_sensitive", logging.IsSensitiveField(fieldName))
```

## Examples

See working examples in:
- `examples/echo-server/main.go` - Server-side secure logging setup
- `examples/echo-client/main.go` - Client-side secure logging setup  
- `internal/adapters/logging/redactor_test.go` - Comprehensive test examples