# Ephemos Production Deployment Guide

## Overview

In production, Ephemos integrates with existing SPIFFE/SPIRE infrastructure to provide identity-based authentication for services. This guide explains various deployment architectures and best practices.

## Deployment Architectures

### 1. Kubernetes Deployment (Most Common)

```yaml
# SPIRE is typically deployed as a DaemonSet
# Each node runs a SPIRE agent
# Your services connect to the local agent
```

**Architecture:**
```
┌─────────────────────────────────────────┐
│            Kubernetes Cluster           │
├─────────────────────────────────────────┤
│  Master Node                            │
│  ┌────────────────┐                     │
│  │  SPIRE Server  │                     │
│  └────────────────┘                     │
├─────────────────────────────────────────┤
│  Worker Node 1                          │
│  ┌──────────────┐  ┌─────────────────┐ │
│  │ SPIRE Agent  │  │  Your Service   │ │
│  │ (DaemonSet)  │←─│  with Ephemos   │ │
│  └──────────────┘  └─────────────────┘ │
├─────────────────────────────────────────┤
│  Worker Node 2                          │
│  ┌──────────────┐  ┌─────────────────┐ │
│  │ SPIRE Agent  │  │  Your Service   │ │
│  │ (DaemonSet)  │←─│  with Ephemos   │ │
│  └──────────────┘  └─────────────────┘ │
└─────────────────────────────────────────┘
```

**Deployment Steps:**

1. **Deploy SPIRE Server:**
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: spire-server
  namespace: spire
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spire-server
  template:
    spec:
      containers:
      - name: spire-server
        image: ghcr.io/spiffe/spire-server:1.8.7
        args:
          - -config
          - /run/spire/config/server.conf
        volumeMounts:
        - name: spire-config
          mountPath: /run/spire/config
          readOnly: true
        - name: spire-data
          mountPath: /run/spire/data
```

2. **Deploy SPIRE Agent (DaemonSet):**
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: spire-agent
  namespace: spire
spec:
  selector:
    matchLabels:
      app: spire-agent
  template:
    spec:
      hostPID: true
      hostNetwork: true
      containers:
      - name: spire-agent
        image: ghcr.io/spiffe/spire-agent:1.8.7
        args:
          - -config
          - /run/spire/config/agent.conf
        volumeMounts:
        - name: spire-config
          mountPath: /run/spire/config
          readOnly: true
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: DirectoryOrCreate
```

3. **Deploy Your Service with Ephemos:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-server
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: echo-server
        image: your-registry/echo-server:latest
        env:
        - name: EPHEMOS_CONFIG
          value: /config/ephemos.yaml
        - name: SPIFFE_ENDPOINT_SOCKET
          value: unix:///run/spire/sockets/agent.sock
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true
        - name: config
          mountPath: /config
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: Directory
      - name: config
        configMap:
          name: echo-server-config
```

### 2. VM/Bare Metal Deployment

**Architecture:**
```
┌──────────────────────────────┐
│     SPIRE Server Node        │
│  ┌────────────────────────┐  │
│  │    SPIRE Server        │  │
│  │  (systemd service)     │  │
│  └────────────────────────┘  │
└──────────────────────────────┘
           ↑
           │ TCP 8081
           │
┌──────────────────────────────┐
│    Application Server 1      │
│  ┌────────────────────────┐  │
│  │    SPIRE Agent         │  │
│  │  (systemd service)     │  │
│  └───────┬────────────────┘  │
│          │ Unix Socket       │
│  ┌───────▼────────────────┐  │
│  │   Your Service         │  │
│  │   with Ephemos         │  │
│  └────────────────────────┘  │
└──────────────────────────────┘
```

**Deployment Steps:**

1. **Install SPIRE Server (on dedicated node):**
```bash
# Install SPIRE
curl -s -N -L https://github.com/spiffe/spire/releases/download/v1.8.7/spire-1.8.7-linux-x86_64-glibc.tar.gz | tar xz
sudo mv spire-1.8.7 /opt/spire

# Configure SPIRE Server
cat > /opt/spire/conf/server.conf <<EOF
server {
  bind_address = "0.0.0.0"
  bind_port = "8081"
  trust_domain = "your-company.com"
  data_dir = "/opt/spire/data"
  log_level = "INFO"
}

plugins {
  DataStore "sql" {
    plugin_data {
      database_type = "postgres"
      connection_string = "dbname=spire user=spire host=postgres.internal"
    }
  }
  
  NodeAttestor "join_token" {
    plugin_data {}
  }
  
  KeyManager "aws_kms" {
    plugin_data {
      region = "us-west-2"
      key_id = "arn:aws:kms:..."
    }
  }
}
EOF

# Create systemd service
sudo systemctl enable --now spire-server
```

2. **Install SPIRE Agent (on each application server):**
```bash
# Install SPIRE Agent
curl -s -N -L https://github.com/spiffe/spire/releases/download/v1.8.7/spire-1.8.7-linux-x86_64-glibc.tar.gz | tar xz
sudo mv spire-1.8.7 /opt/spire

# Configure SPIRE Agent
cat > /opt/spire/conf/agent.conf <<EOF
agent {
  data_dir = "/opt/spire/data"
  log_level = "INFO"
  server_address = "spire-server.internal"
  server_port = "8081"
  socket_path = "/run/spire/sockets/agent.sock"
  trust_domain = "your-company.com"
}

plugins {
  NodeAttestor "aws_iid" {
    plugin_data {
      proof_type = "aws"
    }
  }
  
  WorkloadAttestor "unix" {
    plugin_data {}
  }
  
  WorkloadAttestor "docker" {
    plugin_data {}
  }
}
EOF

# Create systemd service
sudo systemctl enable --now spire-agent
```

3. **Deploy Your Service:**
```bash
# Your service configuration
cat > /etc/ephemos/config.yaml <<EOF
service:
  name: "api-gateway"
  domain: "your-company.com"
  
spiffe:
  socket_path: "/run/spire/sockets/agent.sock"
  
authorized_clients:
  - "spiffe://your-company.com/web-frontend"
  - "spiffe://your-company.com/mobile-app"
EOF

# Run your service
EPHEMOS_CONFIG=/etc/ephemos/config.yaml ./your-service
```

### 3. Cloud-Native Deployment (AWS ECS/Fargate)

```
┌─────────────────────────────────────┐
│           AWS Account               │
├─────────────────────────────────────┤
│  ECS Cluster                        │
│  ┌─────────────────────────────┐    │
│  │  SPIRE Server (ECS Service) │    │
│  └─────────────────────────────┘    │
│                                     │
│  ┌─────────────────────────────┐    │
│  │  Task Definition             │    │
│  │  ┌───────────────────────┐  │    │
│  │  │  SPIRE Agent Sidecar  │  │    │
│  │  └───────────────────────┘  │    │
│  │  ┌───────────────────────┐  │    │
│  │  │  Your Application     │  │    │
│  │  │  with Ephemos         │  │    │
│  │  └───────────────────────┘  │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

## Service Registration Strategies

### 1. Static Registration (Simple)
```bash
# Pre-register services during deployment
ephemos register --name service-a --domain company.com --selector unix:uid:1000
ephemos register --name service-b --domain company.com --selector docker:label:app:service-b
```

### 2. Dynamic Registration (CI/CD)
```yaml
# GitLab CI/CD example
deploy:
  script:
    - ephemos register --name $CI_PROJECT_NAME --domain company.com
    - kubectl apply -f k8s/
```

### 3. Automated Registration (Kubernetes Operator)
```yaml
# Using SPIRE Controller Manager
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: example
spec:
  spiffeIDTemplate: "spiffe://{{ .TrustDomain }}/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app: your-service
```

## Configuration Management

### 1. Environment Variables
```bash
export EPHEMOS_SERVICE_NAME=api-gateway
export EPHEMOS_DOMAIN=company.com
export EPHEMOS_SOCKET_PATH=/run/spire/sockets/agent.sock
export EPHEMOS_AUTHORIZED_CLIENTS=spiffe://company.com/frontend,spiffe://company.com/mobile
```

### 2. Configuration Files
```yaml
# /etc/ephemos/config.yaml
service:
  name: "{{ .Env.SERVICE_NAME }}"
  domain: "{{ .Env.DOMAIN }}"
  
spiffe:
  socket_path: "{{ .Env.SPIFFE_ENDPOINT_SOCKET }}"
  
authorized_clients: {{ .Env.AUTHORIZED_CLIENTS | split "," }}
```

### 3. Secret Management (HashiCorp Vault)
```hcl
# Store configuration in Vault
path "secret/ephemos/*" {
  capabilities = ["read"]
}
```

## High Availability Considerations

### SPIRE Server HA Setup
```
┌────────────────┐     ┌────────────────┐     ┌────────────────┐
│ SPIRE Server 1 │────│ SPIRE Server 2 │────│ SPIRE Server 3 │
└────────────────┘     └────────────────┘     └────────────────┘
         │                      │                      │
         └──────────────────────┼──────────────────────┘
                                │
                        ┌───────▼────────┐
                        │   PostgreSQL   │
                        │    (RDS/HA)    │
                        └────────────────┘
```

### Load Balancing
```nginx
upstream spire_servers {
    server spire-1.internal:8081;
    server spire-2.internal:8081;
    server spire-3.internal:8081;
}
```

## Monitoring and Observability

### 1. Metrics (Prometheus)
```yaml
# SPIRE metrics
- job_name: 'spire-server'
  static_configs:
    - targets: ['spire-server:9988']
  
- job_name: 'spire-agent'
  static_configs:
    - targets: ['localhost:9989']
```

### 2. Logging (ELK Stack)
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "component": "ephemos",
  "message": "Successfully authenticated client",
  "spiffe_id": "spiffe://company.com/frontend",
  "service": "api-gateway"
}
```

### 3. Tracing (OpenTelemetry)
```go
// Ephemos automatically adds SPIFFE ID to trace context
span.SetAttributes(
    attribute.String("spiffe.id", identity.ID),
    attribute.String("spiffe.trust_domain", identity.TrustDomain),
)
```

## Security Best Practices

### 1. Network Segmentation
```yaml
# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: spire-agent-access
spec:
  podSelector:
    matchLabels:
      spire: agent
  ingress:
  - from:
    - podSelector:
        matchLabels:
          ephemos: enabled
```

### 2. RBAC Configuration
```yaml
# Limit who can register services
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: spire-registrar
rules:
- apiGroups: [""]
  resources: ["spiffeids"]
  verbs: ["create", "update"]
```

### 3. Certificate Rotation
```yaml
# SPIRE automatically rotates certificates
# Default SVID lifetime: 1 hour
# Ephemos handles rotation transparently
server:
  default_svid_ttl: "1h"
  ca_ttl: "24h"
```

## Troubleshooting in Production

### Common Issues and Solutions

1. **Service Can't Get SVID:**
```bash
# Check agent connectivity
spire-agent healthcheck -socketPath /run/spire/sockets/agent.sock

# Check registration
spire-server entry show -spiffeID spiffe://company.com/your-service
```

2. **Authentication Failures:**
```bash
# Enable debug logging
export EPHEMOS_LOG_LEVEL=debug

# Check authorized clients list
cat /etc/ephemos/config.yaml | grep authorized_clients
```

3. **Performance Issues:**
```bash
# Check SPIRE cache
spire-agent api fetch x509 -socketPath /run/spire/sockets/agent.sock

# Monitor socket connections
ss -x | grep agent.sock
```

## Migration from Traditional Auth

### Phase 1: Dual Auth Mode
```go
// Support both mTLS and API keys during migration
if ephemosEnabled {
    server = ephemos.IdentityServer(ctx, config)
} else {
    server = traditional.NewAPIKeyServer(config)
}
```

### Phase 2: Gradual Rollout
```yaml
# Feature flag for progressive rollout
features:
  ephemos_auth:
    enabled: true
    percentage: 25  # Start with 25% of traffic
```

### Phase 3: Complete Migration
```go
// Remove legacy auth code
server = ephemos.IdentityServer(ctx, config)
```

## Cost Optimization

### 1. Right-size SPIRE Infrastructure
- SPIRE Server: 2 vCPU, 4GB RAM typically sufficient for 1000s of workloads
- SPIRE Agent: 0.5 vCPU, 512MB RAM per node

### 2. Cache Configuration
```yaml
# Optimize cache to reduce API calls
agent:
  cached_entries_hint: 1000  # Adjust based on workload count
```

### 3. Connection Pooling
```go
// Ephemos automatically pools connections
// Configure max connections per service
maxConns: 100
```

## Compliance and Auditing

### 1. Audit Logs
```json
{
  "event": "service_authenticated",
  "timestamp": "2024-01-15T10:30:00Z",
  "client_spiffe_id": "spiffe://company.com/frontend",
  "server_spiffe_id": "spiffe://company.com/api",
  "result": "success"
}
```

### 2. Compliance Frameworks
- **SOC2**: Ephemos provides cryptographic identity verification
- **PCI DSS**: No credentials stored, automatic rotation
- **HIPAA**: End-to-end encryption with mTLS
- **FedRAMP**: FIPS 140-2 compatible with appropriate configuration

## Next Steps

1. **Plan your trust domain structure** (e.g., `company.com`, `prod.company.com`)
2. **Choose attestation method** (K8s, AWS, GCP, Azure, Unix)
3. **Design service naming convention** (e.g., `spiffe://company.com/env/service`)
4. **Set up monitoring and alerting**
5. **Create runbooks for common operations**
6. **Plan migration strategy from existing auth**

## Additional Resources

- [SPIFFE Production Planning](https://spiffe.io/docs/latest/planning/)
- [SPIRE Deployment Models](https://spire.io/docs/latest/deploying/)
- [Ephemos Examples](./examples/)
- [Security Considerations](./SECURITY.md)