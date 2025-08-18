# Demo vs Production: Certificate Acquisition Differences

## Overview

The fundamental SPIFFE/SPIRE mechanism is the same, but the **attestation method** (how services prove their identity) and **registration process** differ significantly between demo and production environments.

## Demo Environment (Local Development)

### How Services Get Certificates in Demo

```
┌─────────────────────────────────────────────────┐
│                   Demo Setup                    │
├─────────────────────────────────────────────────┤
│                                                  │
│  1. Manual Registration:                        │
│     $ sudo spire-server entry create -spiffeID spiffe://example.org/echo-server -parentID spiffe://example.org/spire-agent -selector unix:uid:0  │
│                                                  │
│  2. Unix UID Attestation (uid:0/root):         │
│     Selector: unix:uid:0                        │
│                                                  │
│  3. Both services run as same user (root)      │
│     - Less secure                               │
│     - Simple for demo                           │
│                                                  │
│  4. Insecure Bootstrap:                         │
│     agent.insecure_bootstrap = true             │
│                                                  │
└─────────────────────────────────────────────────┘
```

**Demo Registration (Current):**
```bash
# Services registered with Unix UID selector
sudo spire-server entry create \
    -spiffeID spiffe://example.org/echo-server \
    -selector unix:uid:0 \  # Root user - NOT SECURE!
    -socketPath /tmp/spire-server/private/api.sock
```

**Problem with Demo Approach:**
- Any process running as root gets the certificate
- No real identity verification
- Fine for local testing, terrible for production

## Production Environment

### How Services Get Certificates in Production

```
┌─────────────────────────────────────────────────┐
│              Production Setup                   │
├─────────────────────────────────────────────────┤
│                                                  │
│  1. Automated Registration via CI/CD            │
│  2. Platform-Specific Attestation               │
│  3. Unique Identity per Workload                │
│  4. Secure Bootstrap with Trust Bundles         │
│                                                  │
└─────────────────────────────────────────────────┘
```

### Production Attestation Methods

#### 1. Kubernetes Attestation
```yaml
# Services identified by Kubernetes properties
apiVersion: v1
kind: ServiceAccount
metadata:
  name: echo-server
  namespace: production
---
# SPIRE registration
spire-server entry create \
    -spiffeID spiffe://company.com/ns/production/sa/echo-server \
    -selector k8s:ns:production \
    -selector k8s:sa:echo-server \
    -selector k8s:pod-label:app:echo-server
```

**How it works:**
```go
// When your service starts in a K8s pod:
// 1. SPIRE Agent verifies pod identity via Kubernetes API
// 2. Checks namespace, service account, labels
// 3. Issues certificate ONLY if all selectors match

// Your code doesn't change:
server := ephemos.IdentityServer(ctx, configPath)
// Ephemos handles getting the cert from SPIRE
```

#### 2. AWS EC2 Attestation
```bash
# Services identified by AWS Instance Identity Document
spire-server entry create \
    -spiffeID spiffe://company.com/aws/instance/echo-server \
    -selector aws_iid:instance-id:i-1234567890abcdef0 \
    -selector aws_iid:tag:Name:echo-server \
    -selector aws_iid:sg:sg-0123456789abcdef0
```

**How it works:**
```
EC2 Instance boots → SPIRE Agent reads AWS metadata → 
Verifies with AWS API → Issues certificate if authorized
```

#### 3. Docker Container Attestation
```bash
# Services identified by Docker properties
spire-server entry create \
    -spiffeID spiffe://company.com/docker/echo-server \
    -selector docker:label:app:echo-server \
    -selector docker:image:company/echo-server:v1.2.3
```

### Key Differences in Certificate Flow

#### Demo Flow (Current)
```
Service Starts
    ↓
Connects to SPIRE Agent (unix socket)
    ↓
Agent: "What's your UID?"
    ↓
Service: "I'm UID 0 (root)"
    ↓
Agent: "OK, here's certificate for echo-server"
    ↓
⚠️ PROBLEM: Any root process gets this cert!
```

#### Production Flow
```
Service Starts (in K8s pod)
    ↓
Connects to SPIRE Agent (unix socket)
    ↓
Agent: "Let me check your pod details..."
    ↓
Agent → Kubernetes API: "Tell me about this pod"
    ↓
K8s API: "Pod: echo-server-7d4b9, NS: prod, SA: echo-server"
    ↓
Agent checks registered entries
    ↓
Agent: "✓ Verified! Here's certificate for echo-server"
    ↓
✅ SECURE: Only the real echo-server pod gets this cert
```

## Configuration Differences

### Demo Config (Current)
```yaml
# config/echo-server.yaml
service:
  name: "echo-server"
  domain: "example.org"
  
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"  # Local temp path
  
authorized_clients:
  - "spiffe://example.org/echo-client"
```

### Production Config
```yaml
# config/echo-server.yaml
service:
  name: "echo-server"
  domain: "company.com"
  
spiffe:
  socket_path: "/run/spire/sockets/agent.sock"  # Standard production path
  # OR use environment variable:
  # socket_path: "${SPIFFE_ENDPOINT_SOCKET}"
  
authorized_clients:
  - "spiffe://company.com/ns/production/sa/web-frontend"
  - "spiffe://company.com/ns/production/sa/mobile-api"
  - "spiffe://company.com/aws/lambda/data-processor"
```

## Registration Process Differences

### Demo Registration (Manual)
```bash
# Current demo - manual registration
cd scripts/demo
sudo spire-server entry create -spiffeID spiffe://example.org/echo-server -parentID spiffe://example.org/spire-agent -selector unix:uid:0
```

### Production Registration (Automated)

#### Option 1: Kubernetes Operator
```yaml
# Automatic registration via CRD
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: echo-server
spec:
  spiffeIDTemplate: "spiffe://{{ .TrustDomain }}/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app: echo-server
  workloadSelectorTemplates:
    - "k8s:ns:{{ .PodMeta.Namespace }}"
    - "k8s:sa:{{ .PodSpec.ServiceAccountName }}"
```

#### Option 2: CI/CD Pipeline
```yaml
# GitLab CI example
deploy:
  stage: deploy
  script:
    # Register service during deployment
    - |
      spire-server entry create \
        -spiffeID spiffe://company.com/service/${CI_PROJECT_NAME} \
        -selector k8s:ns:${ENVIRONMENT} \
        -selector k8s:deployment:${CI_PROJECT_NAME}
    # Deploy to Kubernetes
    - kubectl apply -f k8s/
```

#### Option 3: Terraform/IaC
```hcl
resource "spire_entry" "echo_server" {
  spiffe_id = "spiffe://company.com/service/echo-server"
  parent_id = "spiffe://company.com/spire-agent"
  
  selectors = [
    "k8s:ns:production",
    "k8s:sa:echo-server",
    "k8s:container-name:echo-server"
  ]
  
  ttl = 3600
}
```

## Trust Bundle Differences

### Demo (Insecure Bootstrap)
```yaml
# agent.conf in demo
agent {
  insecure_bootstrap = true  # ⚠️ Agent trusts server without verification
}
```

### Production (Secure Bootstrap)
```yaml
# agent.conf in production
agent {
  trust_bundle_path = "/opt/spire/conf/bundle.crt"  # Pre-distributed trust bundle
  # OR
  trust_bundle_url = "https://spire.company.com/bundle"  # Fetch from secure endpoint
}
```

## What Stays the Same

**Important:** Your application code using Ephemos remains identical!

```go
// This code is EXACTLY the same in demo and production:

func main() {
    // Create identity-aware server
    server, err := ephemos.IdentityServer(ctx, configPath)
    if err != nil {
        log.Fatal(err)
    }
    
    // Register your service
    server.RegisterService(ctx, serviceRegistrar)
    
    // Start serving
    server.Serve(ctx, listener)
}
```

**Ephemos handles:**
- Finding the SPIRE agent socket
- Requesting the SVID (certificate)
- Handling rotation
- Setting up mTLS

**You just change:**
- The config file (different SPIFFE IDs)
- The deployment method (K8s vs local)
- The registration process (automated vs manual)

## Migration Path: Demo to Production

### Step 1: Update Configuration
```yaml
# Development config
service:
  domain: "${SPIFFE_TRUST_DOMAIN:-example.org}"
spiffe:
  socket_path: "${SPIFFE_ENDPOINT_SOCKET:-/tmp/spire-agent/public/api.sock}"
```

### Step 2: Update Registration
```bash
# Instead of manual registration, add to CI/CD:
if [ "$ENVIRONMENT" = "production" ]; then
  kubectl apply -f k8s/spiffe-id.yaml
else
  spire-server entry create -spiffeID spiffe://company.com/$SERVICE_NAME -parentID spiffe://company.com/spire-agent -selector k8s:ns:production
fi
```

### Step 3: Update Attestation
```yaml
# Move from unix:uid to platform-specific
# Development: unix:uid:1000
# Staging: docker:label:env:staging
# Production: k8s:ns:production
```

## Summary

| Aspect | Demo | Production |
|--------|------|------------|
| **Attestation** | Unix UID (weak) | Platform-specific (strong) |
| **Registration** | Manual CLI | Automated CI/CD |
| **Trust Bootstrap** | Insecure | Pre-distributed bundle |
| **Socket Path** | /tmp/spire-agent | /run/spire/sockets |
| **Identity Granularity** | User-based | Workload-based |
| **Certificate Lifetime** | 1 hour (default) | 30-60 minutes |
| **Your Code** | No change | No change |

## Key Takeaway

The beauty of Ephemos is that **your application code doesn't change** between demo and production. The differences are all in:
1. How services are registered (manual vs automated)
2. How identities are verified (unix UID vs K8s/AWS/etc)
3. Where SPIRE runs (local vs distributed)

But the core API remains the same:
```go
server := ephemos.IdentityServer(ctx, configPath)  // Same in demo and prod!
```