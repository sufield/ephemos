# Configuration Security Guide

## Overview

This guide provides comprehensive security practices for configuring Ephemos in production environments. Proper configuration management is critical for maintaining security in identity-based authentication systems.

## Table of Contents

1. [Security Principles](#security-principles)
2. [Configuration Sources](#configuration-sources)
3. [Production Configuration Management](#production-configuration-management)
4. [Environment Variables](#environment-variables)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [Secret Management](#secret-management)
7. [Configuration Validation](#configuration-validation)
8. [Security Hardening](#security-hardening)
9. [Monitoring and Auditing](#monitoring-and-auditing)
10. [Common Vulnerabilities](#common-vulnerabilities)

## Security Principles

### 1. Zero Trust Configuration
- **Never commit production secrets** to version control
- **Assume configurations are compromised** - use defense in depth
- **Rotate configuration credentials** regularly
- **Validate all configuration inputs** before use

### 2. Least Privilege
- **Minimal permissions** for configuration access
- **Role-based access control** for configuration management
- **Separate environments** with isolated configurations
- **Audit all configuration changes**

### 3. Defense in Depth
- **Multiple validation layers** for configuration
- **Encrypted storage** for sensitive configurations
- **Runtime validation** of configuration integrity
- **Fallback to secure defaults** when possible

## Configuration Sources

Ephemos supports multiple configuration sources with security precedence:

### Priority Order (Highest to Lowest)
1. **Environment Variables** (Most secure - runtime injection)
2. **External Secret Management** (HashiCorp Vault, AWS Secrets Manager, etc.)
3. **Kubernetes Secrets** (For containerized deployments)
4. **Configuration Files** (Least secure - use only for non-sensitive data)

### Security Assessment by Source

| Source | Security Level | Use Case | Secrets Allowed |
|--------|---------------|----------|-----------------|
| Environment Variables | High | Production | Yes |
| External Secrets | High | Enterprise | Yes |
| Kubernetes Secrets | Medium-High | Container deployments | Yes |
| Config Files | Low | Development/Templates | No |

## Production Configuration Management

### 1. File-Based Configuration (Non-Sensitive Only)

**✅ Safe for Production:**
```yaml
# production.yaml - NO SECRETS
service:
  name: "${EPHEMOS_SERVICE_NAME}"
  domain: "${EPHEMOS_TRUST_DOMAIN}"

spiffe:
  socketPath: "${EPHEMOS_SPIFFE_SOCKET}"

# Use environment variable references only
```

**❌ Never in Production:**
```yaml
# NEVER DO THIS - secrets in config files
service:
  name: "payment-service"
  domain: "prod.company.com"  # Exposes production topology
  
api_keys:
  database: "super-secret-key"  # NEVER commit secrets
  external_api: "prod-api-token-123"  # Security violation
```

### 2. Directory Structure for Multi-Environment

```
config/
├── templates/              # ✅ Version controlled templates
│   ├── base.yaml           # Base configuration template
│   ├── server.yaml.template
│   └── client.yaml.template
├── examples/               # ✅ Demo configurations only
│   ├── echo-client.yaml    # Development examples
│   └── echo-server.yaml
└── .gitignore             # Ensures production configs ignored

# Never in version control:
config/production/          # ❌ Blocked by .gitignore
config/staging/            # ❌ Blocked by .gitignore
config/customers/          # ❌ Blocked by .gitignore
```

## Environment Variables

### 1. Core Configuration Variables

```bash
# Service Identity (Required)
export EPHEMOS_SERVICE_NAME="payment-service"
export EPHEMOS_TRUST_DOMAIN="prod.company.com"

# SPIFFE Configuration (Required)
export EPHEMOS_SPIFFE_SOCKET="/run/spire/agent/api.sock"

# Authorization (Optional)
export EPHEMOS_AUTHORIZED_CLIENTS="spiffe://prod.company.com/api-gateway,spiffe://prod.company.com/billing-service"
export EPHEMOS_TRUSTED_SERVERS="spiffe://prod.company.com/database-service"

# Security Settings (Optional)
export EPHEMOS_REQUIRE_AUTHENTICATION="true"
export EPHEMOS_LOG_LEVEL="warn"  # Reduce information disclosure
export EPHEMOS_ENABLE_AUDIT_LOGGING="true"
```

### 2. Advanced Security Variables

```bash
# Network Security
export EPHEMOS_BIND_ADDRESS="127.0.0.1:8443"  # Restrict bind interface
export EPHEMOS_TLS_MIN_VERSION="1.3"          # Enforce TLS 1.3+
export EPHEMOS_CIPHER_SUITES="TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256"

# Certificate Security
export EPHEMOS_CERT_ROTATION_THRESHOLD="24h"   # Rotate certificates daily
export EPHEMOS_CERT_VALIDATION_STRICT="true"   # Strict certificate validation
export EPHEMOS_REVOCATION_CHECK_ENABLED="true" # Enable certificate revocation checks

# Monitoring Security
export EPHEMOS_METRICS_ENDPOINT=""              # Disable metrics in production
export EPHEMOS_HEALTH_CHECK_ENDPOINT="/internal/health"  # Internal health checks only
export EPHEMOS_DEBUG_ENDPOINTS_ENABLED="false" # Disable debug endpoints
```

### 3. Environment Variable Validation

```go
// Example validation in application startup
func validateProductionConfig() error {
    requiredEnvVars := []string{
        "EPHEMOS_SERVICE_NAME",
        "EPHEMOS_TRUST_DOMAIN",
        "EPHEMOS_SPIFFE_SOCKET",
    }
    
    for _, envVar := range requiredEnvVars {
        if value := os.Getenv(envVar); value == "" {
            return fmt.Errorf("required environment variable %s not set", envVar)
        }
    }
    
    // Validate trust domain is not example/demo domain
    trustDomain := os.Getenv("EPHEMOS_TRUST_DOMAIN")
    if strings.Contains(trustDomain, "example.org") || strings.Contains(trustDomain, "localhost") {
        return fmt.Errorf("production trust domain cannot contain example or localhost")
    }
    
    return nil
}
```

## Kubernetes Deployment

### 1. Kubernetes Secrets

```yaml
# ephemos-config-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: ephemos-config
  namespace: production
type: Opaque
data:
  # Base64 encoded values
  service-name: <base64-encoded-service-name>
  trust-domain: <base64-encoded-trust-domain>
  spiffe-socket: <base64-encoded-socket-path>
```

### 2. Secure Deployment Configuration

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
spec:
  template:
    spec:
      serviceAccountName: payment-service-sa  # Dedicated service account
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534  # nobody
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: payment-service
        image: payment-service:v1.2.3
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
          readOnlyRootFilesystem: true
        env:
        - name: EPHEMOS_SERVICE_NAME
          valueFrom:
            secretKeyRef:
              name: ephemos-config
              key: service-name
        - name: EPHEMOS_TRUST_DOMAIN
          valueFrom:
            secretKeyRef:
              name: ephemos-config
              key: trust-domain
        - name: EPHEMOS_SPIFFE_SOCKET
          value: "/run/spire/sockets/api.sock"
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: DirectoryOrCreate
```

### 3. Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ephemos-network-policy
spec:
  podSelector:
    matchLabels:
      app: payment-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: api-gateway
    ports:
    - protocol: TCP
      port: 8443
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: database-service
    ports:
    - protocol: TCP
      port: 5432
```

## Secret Management

### 1. HashiCorp Vault Integration

```go
// vault-config.go
type VaultConfig struct {
    Address string
    Token   string
    Path    string
}

func LoadConfigFromVault(vaultConfig VaultConfig) (*Config, error) {
    client, err := vault.NewClient(&vault.Config{
        Address: vaultConfig.Address,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }
    
    client.SetToken(vaultConfig.Token)
    
    secret, err := client.Logical().Read(vaultConfig.Path)
    if err != nil {
        return nil, fmt.Errorf("failed to read secret from vault: %w", err)
    }
    
    return parseConfigFromSecret(secret.Data)
}
```

### 2. AWS Secrets Manager Integration

```go
// aws-secrets.go
func LoadConfigFromAWSSecrets(secretArn string) (*Config, error) {
    sess := session.Must(session.NewSession())
    svc := secretsmanager.New(sess)
    
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretArn),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get secret from AWS: %w", err)
    }
    
    return parseConfigFromJSON(*result.SecretString)
}
```

## Configuration Validation

### 1. Production Readiness Checks

```go
// production-validation.go
func ValidateProductionConfig(cfg *Config) error {
    var errors []string
    
    // Check for demo/development values
    if strings.Contains(cfg.Service.Domain, "example") {
        errors = append(errors, "trust domain contains 'example' - not suitable for production")
    }
    
    if strings.Contains(cfg.Service.Domain, "localhost") {
        errors = append(errors, "trust domain contains 'localhost' - not suitable for production")
    }
    
    // Validate SPIFFE socket security
    if !strings.HasPrefix(cfg.SPIFFE.SocketPath, "/run/") && !strings.HasPrefix(cfg.SPIFFE.SocketPath, "/var/run/") {
        errors = append(errors, "SPIFFE socket should be in /run or /var/run for security")
    }
    
    // Check authorization configuration
    if len(cfg.AuthorizedClients) == 0 && cfg.Service.Name != "public-api" {
        errors = append(errors, "no authorized clients specified - potential security risk")
    }
    
    // Validate SPIFFE IDs format
    for _, client := range cfg.AuthorizedClients {
        if !strings.HasPrefix(client, "spiffe://") {
            errors = append(errors, fmt.Sprintf("invalid SPIFFE ID format: %s", client))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("production validation failed: %s", strings.Join(errors, "; "))
    }
    
    return nil
}
```

### 2. Runtime Security Checks

```go
// runtime-security.go
func PerformRuntimeSecurityChecks() error {
    // Check file permissions on socket
    socketPath := os.Getenv("EPHEMOS_SPIFFE_SOCKET")
    if socketPath != "" {
        info, err := os.Stat(socketPath)
        if err == nil {
            mode := info.Mode()
            if mode.Perm() > 0660 {
                return fmt.Errorf("SPIFFE socket has overly permissive permissions: %o", mode.Perm())
            }
        }
    }
    
    // Verify we're not running as root
    if os.Geteuid() == 0 {
        return fmt.Errorf("service should not run as root user")
    }
    
    // Check for debug flags in production
    if os.Getenv("EPHEMOS_DEBUG_ENABLED") == "true" {
        return fmt.Errorf("debug mode should not be enabled in production")
    }
    
    return nil
}
```

## Security Hardening

### 1. File System Security

```bash
# Set restrictive permissions on configuration directories
chmod 750 /etc/ephemos
chmod 640 /etc/ephemos/*.yaml
chown ephemos:ephemos /etc/ephemos/*.yaml

# Ensure SPIFFE socket has proper permissions
chmod 660 /run/spire/sockets/api.sock
chown spire-agent:ephemos /run/spire/sockets/api.sock
```

### 2. Process Security

```bash
# Run service with dedicated user
useradd -r -s /bin/false -M ephemos-user
usermod -a -G spiffe ephemos-user

# Start service with security options
systemctl edit ephemos-service
```

```ini
# /etc/systemd/system/ephemos-service.service.d/security.conf
[Service]
User=ephemos-user
Group=ephemos-user
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true
```

### 3. Network Security

```go
// network-security.go
func CreateSecureServer() *grpc.Server {
    // TLS 1.3 minimum
    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS13,
        CipherSuites: []uint16{
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
        PreferServerCipherSuites: true,
        CurvePreferences: []tls.CurveID{
            tls.CurveP384,
            tls.X25519,
        },
    }
    
    creds := credentials.NewTLS(tlsConfig)
    
    return grpc.NewServer(
        grpc.Creds(creds),
        grpc.KeepaliveParams(keepalive.ServerParameters{
            MaxConnectionIdle:     15 * time.Second,
            MaxConnectionAge:      30 * time.Second,
            MaxConnectionAgeGrace: 5 * time.Second,
            Time:                  5 * time.Second,
            Timeout:               1 * time.Second,
        }),
        grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
            MinTime:             5 * time.Second,
            PermitWithoutStream: false,
        }),
    )
}
```

## Monitoring and Auditing

### 1. Configuration Change Auditing

```go
// audit-logging.go
func LogConfigurationChange(operation, field, oldValue, newValue string) {
    auditLog := logrus.WithFields(logrus.Fields{
        "event_type":    "configuration_change",
        "operation":     operation,
        "field":         field,
        "old_value_hash": hashValue(oldValue),
        "new_value_hash": hashValue(newValue),
        "user_id":       getCurrentUser(),
        "timestamp":     time.Now().UTC(),
        "source_ip":     getClientIP(),
    })
    
    auditLog.Info("Configuration modified")
}

func hashValue(value string) string {
    hasher := sha256.New()
    hasher.Write([]byte(value))
    return hex.EncodeToString(hasher.Sum(nil))
}
```

### 2. Security Metrics

```go
// security-metrics.go
var (
    configValidationFailures = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ephemos_config_validation_failures_total",
            Help: "Number of configuration validation failures",
        },
        []string{"validation_type", "service"},
    )
    
    unauthorizedConfigAccess = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ephemos_unauthorized_config_access_total",
            Help: "Number of unauthorized configuration access attempts",
        },
        []string{"source_ip", "service"},
    )
)
```

### 3. Alerting

```yaml
# prometheus-alerts.yaml
groups:
- name: ephemos-security
  rules:
  - alert: EphemosConfigValidationFailure
    expr: increase(ephemos_config_validation_failures_total[5m]) > 0
    labels:
      severity: warning
    annotations:
      summary: "Ephemos configuration validation failed"
      description: "Service {{ $labels.service }} failed configuration validation"
      
  - alert: EphemosUnauthorizedConfigAccess
    expr: increase(ephemos_unauthorized_config_access_total[1m]) > 0
    labels:
      severity: critical
    annotations:
      summary: "Unauthorized access to Ephemos configuration"
      description: "Unauthorized access attempt from {{ $labels.source_ip }}"
```

## Common Vulnerabilities

### 1. Information Disclosure

**❌ Vulnerable:**
```yaml
# Exposes internal topology
authorized_clients:
  - "spiffe://prod.company.com/internal-payment-processor"
  - "spiffe://prod.company.com/fraud-detection-service"
  - "spiffe://prod.company.com/regulatory-compliance-service"
```

**✅ Secure:**
```yaml
# Use environment variables or external secrets
authorized_clients: "${EPHEMOS_AUTHORIZED_CLIENTS}"
```

### 2. Path Traversal

**❌ Vulnerable:**
```yaml
spiffe:
  socketPath: "${USER_SOCKET_PATH}"  # User-controlled input
```

**✅ Secure:**
```yaml
spiffe:
  socketPath: "/run/spire/sockets/api.sock"  # Fixed, secure path
```

### 3. Injection Attacks

**❌ Vulnerable:**
```go
// User input directly in SPIFFE ID
spiffeID := fmt.Sprintf("spiffe://%s/%s", trustDomain, userInput)
```

**✅ Secure:**
```go
// Validate and sanitize input
if !regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(userInput) {
    return fmt.Errorf("invalid service name format")
}
spiffeID := fmt.Sprintf("spiffe://%s/%s", trustDomain, userInput)
```

### 4. Privilege Escalation

**❌ Vulnerable:**
```yaml
# Overly broad authorization
authorized_clients:
  - "spiffe://prod.company.com/*"  # Allows any service
```

**✅ Secure:**
```yaml
# Specific service authorization
authorized_clients:
  - "spiffe://prod.company.com/api-gateway"
  - "spiffe://prod.company.com/billing-service"
```

## Best Practices Summary

### ✅ Do This
- Use environment variables for all sensitive configuration
- Implement configuration validation in production
- Use dedicated service accounts and users
- Enable audit logging for configuration changes
- Implement network policies and firewalls
- Use external secret management systems
- Rotate secrets regularly
- Monitor for unauthorized access

### ❌ Never Do This
- Commit production secrets to version control
- Use demo/example values in production
- Run services as root
- Disable certificate validation
- Use wildcards in authorization lists
- Log sensitive configuration values
- Use HTTP for configuration endpoints
- Share configuration between environments

## Emergency Response

### Configuration Compromise Response

1. **Immediate Actions** (0-30 minutes):
   - Rotate all compromised credentials
   - Revoke compromised certificates
   - Update authorization lists to block compromised services
   - Enable additional monitoring

2. **Investigation** (30 minutes - 4 hours):
   - Identify scope of compromise
   - Check audit logs for unauthorized changes
   - Verify integrity of configuration systems
   - Document timeline of events

3. **Recovery** (4-24 hours):
   - Deploy new configurations with rotated secrets
   - Update monitoring and alerting
   - Conduct post-incident review
   - Update security procedures

### Contact Information

- **Security Team**: security@company.com
- **On-Call Engineer**: +1-XXX-XXX-XXXX
- **Incident Response**: incident-response@company.com

---

*This document should be reviewed quarterly and updated with new security threats and best practices.*