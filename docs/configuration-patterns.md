# Ephemos Configuration Patterns

Ephemos supports multiple configuration patterns to fit different deployment scenarios and development workflows. The YAML adapter remains the primary configuration method, with added support for environment variable overrides and pure-code configuration.

## Configuration Sources

### 1. YAML with Environment Overrides (Default)

The traditional approach that keeps YAML files as the source of truth while allowing environment variables to override specific values.

```go
config, err := ephemos.NewConfigBuilder().
    WithSource(ephemos.ConfigSourceYAML).
    WithYAMLFile("config/production.yaml").
    Build(ctx)
```

**Environment Variables for Overrides:**
- `EPHEMOS_SERVICE_NAME` - Override service name
- `EPHEMOS_SERVICE_DOMAIN` - Override service domain  
- `EPHEMOS_SPIFFE_SOCKET` - Override SPIFFE socket path
- `EPHEMOS_TRANSPORT_TYPE` - Override transport type (grpc/http)
- `EPHEMOS_TRANSPORT_ADDRESS` - Override transport address
- `EPHEMOS_AUTHORIZED_CLIENTS` - Override authorized clients (comma-separated)
- `EPHEMOS_TRUSTED_SERVERS` - Override trusted servers (comma-separated)

### 2. Environment Variables Only

Pure environment variable configuration, ideal for containerized deployments and 12-factor apps.

```go
config, err := ephemos.NewConfigBuilder().
    WithSource(ephemos.ConfigSourceEnvOnly).
    WithEnvPrefix("EPHEMOS").
    Build(ctx)
```

### 3. Pure Code Configuration

Programmatic configuration defined entirely in code, perfect for testing and embedded applications.

```go
config, err := ephemos.NewConfigBuilder().
    WithSource(ephemos.ConfigSourcePureCode).
    WithServiceName("my-service").
    WithServiceDomain("production.company.com").
    WithSPIFFESocket("/tmp/spire-agent/public/api.sock").
    WithTransport("grpc", ":443").
    WithAuthorizedClients([]string{"client-a", "client-b"}).
    WithTrustedServers([]string{"server-1", "server-2"}).
    Build(ctx)
```

## Flexible Configuration API

For cleaner, more composable configuration:

```go
// Production environment
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithYAMLSource("config/production.yaml"),
)

// Development environment  
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithEnvSource("DEV"),
    ephemos.WithService("dev-service", "dev.company.com"),
)

// Testing environment
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithPureCodeSource(),
    ephemos.WithService("test-service", "test.local"),
    ephemos.WithTransportOption("grpc", ":0"), // Random port
)
```

## Use Cases by Environment

### Production
- **Pattern**: YAML + Environment Overrides
- **Benefits**: Version-controlled configuration with runtime flexibility
- **Example**: YAML defines structure, env vars provide secrets and environment-specific values

```yaml
# config/production.yaml
service:
  name: "api-service"
  domain: "production.company.com"
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"  
transport:
  type: "grpc"
  address: ":443"
```

```bash
# Environment overrides for sensitive data
export EPHEMOS_AUTHORIZED_CLIENTS="service-a,service-b,service-c"
export EPHEMOS_TRUSTED_SERVERS="auth.company.com,data.company.com"
```

### Containerized Deployments
- **Pattern**: Environment Variables Only
- **Benefits**: No config files to manage, perfect for Docker/Kubernetes
- **Example**: All configuration through environment variables

```bash
export EPHEMOS_SERVICE_NAME="containerized-service"
export EPHEMOS_SERVICE_DOMAIN="k8s.company.com"
export EPHEMOS_TRANSPORT_TYPE="grpc"
export EPHEMOS_TRANSPORT_ADDRESS=":8080"
export EPHEMOS_SPIFFE_SOCKET="/run/spire/sockets/agent.sock"
```

### Testing
- **Pattern**: Pure Code Configuration
- **Benefits**: Predictable, isolated, no external dependencies
- **Example**: Each test defines its own configuration

```go
func TestServiceIntegration(t *testing.T) {
    config, err := ephemos.LoadConfigFlexible(ctx,
        ephemos.WithPureCodeSource(),
        ephemos.WithService("test-service", "test.local"),
        ephemos.WithTransportOption("grpc", ":0"), // Random port
    )
    // ... test logic
}
```

### Development  
- **Pattern**: YAML + Environment Overrides or Environment Only
- **Benefits**: Fast iteration, easy debugging, local overrides
- **Example**: Base config in YAML, local overrides via env vars

```bash
# Override just what's needed for local development
export EPHEMOS_SERVICE_DOMAIN="localhost" 
export EPHEMOS_SPIFFE_SOCKET="/tmp/local-spire/api.sock"
export EPHEMOS_TRANSPORT_ADDRESS=":8080"
```

## Custom Environment Prefixes

Different applications can use different environment prefixes to avoid conflicts:

```go
// Application A
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithEnvSource("APPA"),
)

// Application B  
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithEnvSource("APPB"),
)
```

This allows:
```bash
export APPA_SERVICE_NAME="app-a"
export APPB_SERVICE_NAME="app-b"  
```

## Migration from Existing Code

The enhanced configuration is fully backward compatible:

```go
// Old way (still works)
config, err := loadAndValidateConfig(ctx, "config/service.yaml")

// New way with same behavior
config, err := ephemos.NewConfigBuilder().
    WithSource(ephemos.ConfigSourceYAML).
    WithYAMLFile("config/service.yaml").
    Build(ctx)

// New way with added flexibility
config, err := ephemos.LoadConfigFlexible(ctx,
    ephemos.WithYAMLSource("config/service.yaml"),
)
```

## Error Handling

All configuration patterns return the same rich error types:

```go
config, err := ephemos.LoadConfigFlexible(ctx, ...)
if err != nil {
    if ephemos.IsConfigurationError(err) {
        // Handle configuration-specific errors
        if configErr := ephemos.GetConfigValidationError(err); configErr != nil {
            log.Printf("Config validation failed in %s: %s", 
                configErr.File, configErr.Message)
        }
    }
    return err
}
```

## Best Practices

1. **Use YAML + Env Overrides for production** - Version control structure, environment variables for secrets
2. **Use Environment Only for containers** - Simplifies deployment and follows 12-factor principles  
3. **Use Pure Code for testing** - Predictable, isolated test environments
4. **Validate early** - All patterns validate configuration before use
5. **Use custom prefixes** - Avoid environment variable conflicts in multi-app deployments
6. **Leverage flexible API** - Cleaner, more composable configuration code

## Environment Variable Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `{PREFIX}_SERVICE_NAME` | Service identifier | `api-service` |
| `{PREFIX}_SERVICE_DOMAIN` | Service domain | `production.company.com` |
| `{PREFIX}_SPIFFE_SOCKET` | SPIFFE agent socket path | `/tmp/spire-agent/public/api.sock` |
| `{PREFIX}_TRANSPORT_TYPE` | Transport protocol | `grpc` or `http` |
| `{PREFIX}_TRANSPORT_ADDRESS` | Listen address | `:443`, `:8080`, `0.0.0.0:9000` |
| `{PREFIX}_AUTHORIZED_CLIENTS` | Comma-separated client list | `client-a,client-b,client-c` |
| `{PREFIX}_TRUSTED_SERVERS` | Comma-separated server list | `server1.com,server2.com` |

Default prefix is `EPHEMOS`, but can be customized per application.