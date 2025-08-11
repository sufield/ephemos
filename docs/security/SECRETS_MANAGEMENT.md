# Secrets Management for Ephemos

## Overview

This document outlines comprehensive secrets management practices for Ephemos, covering detection, prevention, and secure handling of sensitive data throughout the development and deployment lifecycle.

## Table of Contents

1. [Secrets Detection & Prevention](#secrets-detection--prevention)
2. [Configuration Security](#configuration-security)
3. [Development Workflow](#development-workflow)
4. [Production Deployment](#production-deployment)
5. [Security Scanning Tools](#security-scanning-tools)
6. [Incident Response](#incident-response)

## Secrets Detection & Prevention

### Automated Security Scanning

Ephemos includes multiple layers of secrets detection:

```bash
# Complete security setup
make security-all

# Individual scans
make scan-secrets    # Gitleaks + git-secrets
make scan-trivy      # Vulnerability scanning
make audit-config-files  # Manual config audit
```

### Git Hooks Protection

**Pre-commit Hook** - Blocks commits containing secrets:
```bash
# Install security hooks
make setup-git-hooks

# Manual hook testing
.git/hooks/pre-commit
```

**Pre-push Hook** - Comprehensive security validation before push:
```bash
# Runs full security suite
.git/hooks/pre-push
```

### Detection Tools Configuration

**Gitleaks** (`.gitleaks.toml`):
- Custom rules for SPIFFE URIs, domains, socket paths
- Allowlists for demo/template values
- Base64 encoded secret detection

**Git-secrets** (`.secrets`):
- Pattern-based secret detection
- API key, token, and credential patterns
- Production domain detection

## Configuration Security

### Security Hierarchy (Highest to Lowest Priority)

| Method | Security Level | Production Use | Secrets Allowed |
|--------|---------------|---------------|-----------------|
| Environment Variables | 🔒 **High** | ✅ Recommended | ✅ Yes |
| External Secrets (Vault) | 🔒 **High** | ✅ Recommended | ✅ Yes |
| Kubernetes Secrets | ⚠️ **Medium** | ✅ Acceptable | ✅ Yes |
| Configuration Files | ❌ **Low** | ❌ Templates Only | ❌ No |

### Safe Configuration Patterns

**✅ SECURE - Environment Variables:**
```bash
# Production service identity
export EPHEMOS_SERVICE_NAME="payment-service"
export EPHEMOS_TRUST_DOMAIN="prod.company.com"
export EPHEMOS_AUTHORIZED_CLIENTS="spiffe://prod.company.com/api-gateway"
```

**✅ SECURE - Kubernetes Secrets:**
```yaml
# From base64-encoded secret
apiVersion: v1
kind: Secret
metadata:
  name: ephemos-secret
data:
  EPHEMOS_SERVICE_NAME: "cGF5bWVudC1zZXJ2aWNl"  # payment-service
  EPHEMOS_TRUST_DOMAIN: "cHJvZC5jb21wYW55LmNvbQ=="  # prod.company.com
```

**✅ SECURE - External Secret Management:**
```go
// HashiCorp Vault integration
config, err := ports.LoadFromVault(vaultClient, "secret/ephemos/payment-service")

// AWS Secrets Manager integration
config, err := ports.LoadFromAWSSecretsManager(session, "ephemos/payment-service")
```

**❌ INSECURE - Hardcoded in Files:**
```yaml
# NEVER DO THIS
service:
  name: "payment-service"
  domain: "prod.company.com"  # Real production domain in file!
authorized_clients:
  - "spiffe://prod.company.com/real-client"  # Production SPIFFE ID!
```

### Configuration Validation

**Production Readiness Validation:**
```bash
# Validate production configuration
./bin/config-validator --env-only --production --verbose

# Check for demo values
./bin/config-validator --config config.yaml --production
```

**Automatic Security Checks:**
- ❌ Blocks `example.org`, `localhost`, `demo-service` values
- ❌ Prevents debug mode in production
- ❌ Detects wildcard authorizations (`spiffe://domain/*`)
- ❌ Validates secure socket paths (`/run/spire/sockets/`)

## Development Workflow

### Secure Development Practices

**1. Repository Setup:**
```bash
# Clone and setup security
git clone <repository>
cd ephemos
make security-all  # Installs tools, hooks, runs scans
```

**2. Configuration Development:**
```bash
# Use demo configurations only
cp config/templates/production.yaml.template config/local.yaml
# Edit config/local.yaml with demo values
# NEVER commit production values
```

**3. Testing with Secrets:**
```bash
# Environment-only testing (secure)
export EPHEMOS_SERVICE_NAME="test-service"
export EPHEMOS_TRUST_DOMAIN="test.local"
./bin/config-validator --env-only

# File + environment override (acceptable for dev)
EPHEMOS_SERVICE_NAME="override-service" ./bin/config-validator --config config/local.yaml
```

### CI/CD Integration

**GitHub Actions Security Workflow:**
```yaml
name: Security Scan
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Install Security Tools
      run: make install-security-tools
      
    - name: Run Security Scans
      run: make ci-security
      
    - name: Run Trivy Scan
      run: make scan-trivy
```

## Production Deployment

### Kubernetes Deployment Security

**1. Secret Creation (Manual Process):**
```bash
# Create production secret (NEVER commit)
kubectl create secret generic ephemos-secret \
  --from-literal=EPHEMOS_SERVICE_NAME="payment-service" \
  --from-literal=EPHEMOS_TRUST_DOMAIN="prod.company.com" \
  --from-literal=EPHEMOS_AUTHORIZED_CLIENTS="spiffe://prod.company.com/api-gateway"

# Verify secret
kubectl describe secret ephemos-secret
```

**2. Deployment Configuration:**
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: ephemos-service
        envFrom:
        - secretRef:
            name: ephemos-secret  # Load all secrets as env vars
        securityContext:
          runAsNonRoot: true
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
```

**3. Security Validation:**
```bash
# Validate deployment configuration
kubectl exec -it deployment/ephemos-service -- ./bin/config-validator --env-only --production
```

### External Secret Management

**HashiCorp Vault:**
```go
// Vault secret retrieval
func LoadFromVault(client *api.Client, path string) (*Configuration, error) {
    secret, err := client.Logical().Read(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read from vault: %w", err)
    }
    
    config := &Configuration{}
    config.Service.Name = secret.Data["service_name"].(string)
    config.Service.Domain = secret.Data["trust_domain"].(string)
    
    return config, nil
}
```

**AWS Secrets Manager:**
```go
// AWS Secrets Manager integration
func LoadFromAWSSecretsManager(sess *session.Session, secretId string) (*Configuration, error) {
    svc := secretsmanager.New(sess)
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretId),
    })
    
    // Parse JSON secret
    var secretData map[string]string
    json.Unmarshal([]byte(*result.SecretString), &secretData)
    
    config := &Configuration{}
    config.Service.Name = secretData["service_name"]
    config.Service.Domain = secretData["trust_domain"]
    
    return config, nil
}
```

## Security Scanning Tools

### Tool Installation

```bash
# Install all security tools
make install-security-tools

# Individual tool installation
curl -sSfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin
curl -sSfL https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks_Linux_x86_64.tar.gz | sudo tar -xz -C /usr/local/bin
git clone https://github.com/awslabs/git-secrets.git && cd git-secrets && make install
```

### Scanning Commands

**Secret Detection:**
```bash
# Comprehensive secret scan
make scan-secrets

# Individual tool scans
gitleaks detect --source . --config .gitleaks.toml
git-secrets --scan --recursive .
```

**Vulnerability Scanning:**
```bash
# Trivy security scans
make scan-trivy

# Individual scans
trivy fs --severity HIGH,CRITICAL .
trivy config .
```

**Configuration Audit:**
```bash
# Manual config file audit
make audit-config-files

# Check for production values
rg -i "(prod|production|staging)" config/*.yaml
rg -v "example\.org" config/*.yaml | rg "[a-zA-Z0-9.-]+\.(com|net|org|io)"
```

### Security Scan Results

**Clean Scan Output Example:**
```bash
$ make scan-secrets
🔍 Scanning for secrets and sensitive data...
Running gitleaks scan...
✅ No leaks found
Running git-secrets scan...
✅ No secrets found
Manual config file audit...
✅ No obvious secrets found in config/
✅ No hardcoded production values found
✅ Only example domains found
```

**Security Issue Detection:**
```bash
$ make scan-secrets
🔍 Scanning for secrets and sensitive data...
Running gitleaks scan...
⚠️  Gitleaks found potential secrets
  config/production.yaml:3: hardcoded production domain 'prod.company.com'
  config/production.yaml:8: real SPIFFE ID 'spiffe://prod.company.com/payment'
```

## Incident Response

### Secret Exposure Response

**1. Immediate Actions:**
```bash
# Remove from repository
git rm config/production.yaml
git commit -m "Remove accidentally committed production config"

# Rotate exposed secrets
# - Change service names if exposed
# - Regenerate SPIFFE certificates
# - Update authorization lists
```

**2. Investigation:**
```bash
# Check git history for secrets
git log --all --full-history -- config/
gitleaks detect --log-opts="--all --full-history"

# Scan for related exposures
make scan-secrets
make scan-trivy
```

**3. Prevention:**
```bash
# Strengthen hooks
make setup-git-hooks

# Add additional patterns to .gitleaks.toml
# Update .secrets patterns
# Review .gitignore patterns
```

### Security Monitoring

**Continuous Monitoring:**
```bash
# Daily security scans
0 2 * * * cd /path/to/ephemos && make scan-secrets
0 3 * * * cd /path/to/ephemos && make scan-trivy

# Weekly comprehensive audit
0 4 * * 0 cd /path/to/ephemos && make security-all
```

**Alert Thresholds:**
- ⚠️ **Warning**: Demo values in production environment
- 🚨 **Critical**: Real secrets detected in repository
- 🔥 **Emergency**: Production credentials in public repository

## Best Practices Summary

### ✅ DO

- **Use environment variables** for all sensitive configuration
- **Enable security hooks** on all development machines
- **Run security scans** before every commit and push
- **Validate production configuration** before deployment
- **Use external secret management** for production
- **Regularly audit** configuration files and git history
- **Document secret handling** procedures for your team

### ❌ DON'T

- **Commit real secrets** to version control (ever!)
- **Hardcode production values** in configuration files
- **Use demo configurations** in production environments
- **Share production credentials** via chat or email
- **Skip security validation** in CI/CD pipelines
- **Ignore security scan warnings** without investigation
- **Store secrets** in issue trackers or documentation

### 🚨 Emergency Contacts

If you discover exposed secrets:

1. **Immediate**: Remove from repository and rotate credentials
2. **Contact**: Security team at security@sufield.com
3. **Document**: Create incident report with timeline and actions
4. **Review**: Strengthen preventive measures

---

**Remember: Security is everyone's responsibility. When in doubt, use environment variables and external secret management.**

*For additional security information:*
- **Configuration Security**: [CONFIGURATION_SECURITY.md](./CONFIGURATION_SECURITY.md)
- **Security Features**: [SECURITY_FEATURES.md](./SECURITY_FEATURES.md)
- **Vulnerability Policy**: [.github/SECURITY.md](../../.github/SECURITY.md)

*Last updated: August 2025*