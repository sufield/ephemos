# Ephemos Configuration Guide

## Overview

This directory contains configuration templates and examples for Ephemos services. **Never commit production secrets or sensitive configurations to version control.**

## Directory Structure

```
config/
├── README.md                    # This guide
├── templates/                   # Production-ready templates
│   ├── production.yaml.template
│   ├── staging.yaml.template
│   └── k8s/
│       ├── configmap.yaml
│       └── secret.yaml.template
└── examples/                    # Demo configurations (safe for version control)
    ├── echo-client.yaml
    ├── echo-server.yaml
    └── ephemos.yaml
```

## Security Levels

| Configuration Method | Security | Production Use | Secrets Allowed |
|---------------------|----------|---------------|----------------|
| **Environment Variables** | ✅ High | ✅ Recommended | ✅ Yes |
| **External Secrets** | ✅ High | ✅ Recommended | ✅ Yes |
| **Kubernetes Secrets** | ⚠️ Medium | ✅ Acceptable | ✅ Yes |
| **Configuration Files** | ❌ Low | ❌ Templates Only | ❌ No |

## Production Configuration Methods

### 1. Environment Variables (Recommended)

Environment variables are the most secure way to configure Ephemos in production:

```bash
# Required Configuration
export EPHEMOS_SERVICE_NAME="payment-service"
export EPHEMOS_TRUST_DOMAIN="prod.company.com"

# Optional Configuration
export EPHEMOS_SPIFFE_SOCKET="/run/spire/sockets/api.sock"
export EPHEMOS_AUTHORIZED_CLIENTS="spiffe://prod.company.com/api-gateway,spiffe://prod.company.com/billing-service"
export EPHEMOS_TRUSTED_SERVERS="spiffe://prod.company.com/database-service"

# Advanced Security
export EPHEMOS_REQUIRE_AUTHENTICATION="true"
export EPHEMOS_LOG_LEVEL="warn"
export EPHEMOS_DEBUG_ENABLED="false"
```

### 2. Kubernetes ConfigMap + Secrets

For Kubernetes deployments, use ConfigMaps for non-sensitive data and Secrets for sensitive data:

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ephemos-config
data:
  EPHEMOS_SPIFFE_SOCKET: "/run/spire/sockets/api.sock"
  EPHEMOS_LOG_LEVEL: "warn"
  EPHEMOS_REQUIRE_AUTHENTICATION: "true"

---
# secret.yaml (create manually, never commit)
apiVersion: v1
kind: Secret
metadata:
  name: ephemos-secret
type: Opaque
data:
  EPHEMOS_SERVICE_NAME: <base64-encoded-service-name>
  EPHEMOS_TRUST_DOMAIN: <base64-encoded-trust-domain>
  EPHEMOS_AUTHORIZED_CLIENTS: <base64-encoded-client-list>
```

### 3. External Secret Management

For enterprise deployments, use external secret management:

```go
// Example with HashiCorp Vault
import "github.com/sufield/ephemos/internal/core/ports"

config, err := ports.LoadFromVault(vaultClient, "secret/ephemos/payment-service")
if err != nil {
    log.Fatal("Failed to load configuration from Vault:", err)
}
```

## Configuration Loading Priority

Ephemos loads configuration in this order (highest to lowest priority):

1. **Environment Variables** (overrides everything)
2. **External Secret Management** (Vault, AWS Secrets Manager)
3. **Kubernetes Secrets**
4. **Configuration Files** (templates only)
5. **Default Values**

## Using Templates

### 1. Copy Template for Your Environment

```bash
# Copy production template
cp config/templates/production.yaml.template config/production.yaml

# Edit with your values (use environment variable references)
vim config/production.yaml

# IMPORTANT: Add to .gitignore immediately
echo "config/production.yaml" >> .gitignore
```

### 2. Example Production Configuration

```yaml
# production.yaml - Use environment variable references only
service:
  name: "${EPHEMOS_SERVICE_NAME}"          # Required: Set via environment
  domain: "${EPHEMOS_TRUST_DOMAIN}"       # Required: Set via environment

spiffe:
  socketPath: "${EPHEMOS_SPIFFE_SOCKET:-/run/spire/sockets/api.sock}"

# Use environment variables for security-sensitive lists
authorized_clients: "${EPHEMOS_AUTHORIZED_CLIENTS}"  # Comma-separated
trusted_servers: "${EPHEMOS_TRUSTED_SERVERS}"       # Comma-separated
```

### 3. Loading Configuration with Environment Override

```go
import (
    "github.com/sufield/ephemos/internal/adapters/secondary/config"
    "github.com/sufield/ephemos/internal/core/ports"
)

// Method 1: Environment variables only (most secure)
config, err := ports.LoadFromEnvironment()

// Method 2: File + environment override
provider := config.NewFileProvider()
config, err := provider.LoadConfiguration(ctx, "config/production.yaml")
if err == nil {
    err = config.MergeWithEnvironment()  // Environment overrides file
}

// Method 3: Validate production readiness
if err := config.IsProductionReady(); err != nil {
    log.Fatal("Configuration not suitable for production:", err)
}
```

## Environment Variable Reference

### Required Variables

| Variable | Example | Description |
|----------|---------|-------------|
| `EPHEMOS_SERVICE_NAME` | `"payment-service"` | Unique service identifier |
| `EPHEMOS_TRUST_DOMAIN` | `"prod.company.com"` | SPIFFE trust domain |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EPHEMOS_SPIFFE_SOCKET` | `/tmp/spire-agent/public/api.sock` | SPIRE agent socket path |
| `EPHEMOS_AUTHORIZED_CLIENTS` | `""` | Comma-separated SPIFFE IDs |
| `EPHEMOS_TRUSTED_SERVERS` | `""` | Comma-separated SPIFFE IDs |
| `EPHEMOS_REQUIRE_AUTHENTICATION` | `true` | Require authentication |
| `EPHEMOS_LOG_LEVEL` | `"info"` | Log level (debug, info, warn, error) |
| `EPHEMOS_DEBUG_ENABLED` | `false` | Enable debug mode |

### Security Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EPHEMOS_BIND_ADDRESS` | `:8443` | Server bind address |
| `EPHEMOS_TLS_MIN_VERSION` | `"1.3"` | Minimum TLS version |
| `EPHEMOS_CERT_ROTATION_THRESHOLD` | `"24h"` | Certificate rotation threshold |

## Production Security Checklist

### ✅ Before Deployment

- [ ] All secrets configured via environment variables
- [ ] No production values in version control
- [ ] Configuration validated with `IsProductionReady()`
- [ ] Trust domain is production domain (not example.org)
- [ ] Service name is production name (not demo/example)
- [ ] Debug mode disabled (`EPHEMOS_DEBUG_ENABLED=false`)
- [ ] Appropriate log level set (`EPHEMOS_LOG_LEVEL=warn`)
- [ ] Authorization lists populated (not empty)
- [ ] SPIFFE socket in secure location

### ✅ After Deployment

- [ ] Configuration loading works correctly
- [ ] Service can authenticate with SPIRE
- [ ] Authorization policies are enforced
- [ ] Logs show proper security events
- [ ] No sensitive data in logs
- [ ] Certificate rotation working

## Common Security Mistakes

### ❌ Don't Do This

```yaml
# NEVER commit production secrets
service:
  name: "payment-service"           # Reveals service topology
  domain: "prod.company.com"        # Exposes production domain

authorized_clients:
  - "spiffe://prod.company.com/api-gateway"    # Exposes architecture
  - "spiffe://prod.company.com/billing-*"      # Wildcards are dangerous
```

```bash
# NEVER set secrets in shell history
export EPHEMOS_SERVICE_NAME="payment-service"  # This goes in shell history!
```

### ✅ Do This Instead

```yaml
# Use environment variable references
service:
  name: "${EPHEMOS_SERVICE_NAME}"
  domain: "${EPHEMOS_TRUST_DOMAIN}"

authorized_clients: "${EPHEMOS_AUTHORIZED_CLIENTS}"
```

```bash
# Use secure methods for environment variables
# Method 1: External secret injection (Kubernetes, Docker secrets)
# Method 2: Vault/AWS Secrets Manager integration
# Method 3: Secure environment file (not in version control)
set -o allexport
source /secure/ephemos.env  # Not in version control
set +o allexport
```

## Testing Configuration

### Development Testing

```bash
# Use demo configuration for development
export EPHEMOS_SERVICE_NAME="test-service"
export EPHEMOS_TRUST_DOMAIN="test.local"
export EPHEMOS_DEBUG_ENABLED="true"

# Test with demo configuration
./bin/echo-server --config config/examples/echo-server.yaml
```

### Production Testing

```bash
# Test production configuration without starting service
go run cmd/config-validator/main.go --env-only

# Test configuration loading
go run -ldflags "-X main.configTest=true" cmd/your-service/main.go
```

## Troubleshooting

### Configuration Not Loading

1. **Check environment variables**:
   ```bash
   env | grep EPHEMOS_
   ```

2. **Validate configuration**:
   ```bash
   export EPHEMOS_SERVICE_NAME="test-service"
   export EPHEMOS_TRUST_DOMAIN="test.local"
   go run cmd/config-validator/main.go
   ```

3. **Check file permissions**:
   ```bash
   ls -la config/production.yaml  # Should be 600 or 640
   ```

### Production Validation Failures

1. **Example domain error**:
   ```
   Error: trust domain contains 'example.org' - not suitable for production
   ```
   **Solution**: Set `EPHEMOS_TRUST_DOMAIN` to your production domain

2. **Debug mode enabled**:
   ```
   Error: debug mode is enabled - should be disabled in production
   ```
   **Solution**: Set `EPHEMOS_DEBUG_ENABLED=false`

3. **Insecure socket path**:
   ```
   Error: SPIFFE socket should be in a secure directory
   ```
   **Solution**: Use `/run/spire/sockets/api.sock` or similar

## Migration Guide

### From File-Based to Environment-Based

1. **Audit current configuration**:
   ```bash
   # Find all configuration files
   find . -name "*.yaml" -o -name "*.yml" | grep -v examples
   ```

2. **Extract values to environment**:
   ```bash
   # Convert YAML values to environment variables
   # service.name -> EPHEMOS_SERVICE_NAME
   # service.domain -> EPHEMOS_TRUST_DOMAIN
   # etc.
   ```

3. **Validate migration**:
   ```bash
   # Test both old and new configuration methods
   go run cmd/config-validator/main.go --file config/old.yaml
   go run cmd/config-validator/main.go --env-only
   ```

4. **Remove configuration files**:
   ```bash
   # Move to templates
   mv config/production.yaml config/templates/production.yaml.template
   
   # Update .gitignore
   echo "config/production.yaml" >> .gitignore
   ```

## Support

- **Security Issues**: See [SECURITY.md](../.github/SECURITY.md)
- **Configuration Documentation**: [docs/security/CONFIGURATION_SECURITY.md](../docs/security/CONFIGURATION_SECURITY.md)
- **General Support**: Create an issue on GitHub

---

*Remember: Security is everyone's responsibility. When in doubt, use environment variables and external secret management.*