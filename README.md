# Ephemos

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/sufield/ephemos/badge)](https://securityscorecards.dev/viewer/?uri=github.com/sufield/ephemos)

**Core = SPIFFE identity + mTLS (gRPC-focused). HTTP support via contrib middleware for Chi/Gin.**

**No more plaintext API keys or secrets between your services.** Ephemos provides a lightweight core for SPIFFE-based service identity and gRPC authentication, with framework-specific HTTP extensions available through contrib modules.

## The Problem with API Keys

Every service needs to authenticate to other services. The traditional approach uses API keys:

### ❌ Before: API Keys Everywhere

**Time Server (has to validate API keys):**
```go
func (s *TimeService) GetTime(w http.ResponseWriter, r *http.Request) {
    // 😫 Every service method starts with API key validation
    apiKey := r.Header.Get("Authorization")
    if apiKey != "Bearer time-client-secret-abc123" {
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    // Your actual business logic buried under auth code
    timezone := r.URL.Query().Get("timezone")
    loc, _ := time.LoadLocation(timezone)
    currentTime := time.Now().In(loc)
    
    json.NewEncoder(w).Encode(map[string]string{
        "time": currentTime.Format("2006-01-02 15:04:05 MST"),
    })
}
```

**Time Client (has to manage API keys):**
```go
func main() {
    // 😫 Hard-coded secrets or environment variables
    apiKey := os.Getenv("TIME_SERVICE_API_KEY") // "time-client-secret-abc123"
    if apiKey == "" {
        log.Fatal("TIME_SERVICE_API_KEY not set")
    }
    
    req, _ := http.NewRequest("GET", "https://time-service:8080/time?timezone=UTC", nil)
    req.Header.Set("Authorization", "Bearer "+apiKey)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        log.Fatal("Request failed:", err)
    }
    // Handle response...
}
```

**Problems with this approach:**
- 🔑 API keys in code, environment variables, or config files
- 🔄 Manual rotation when keys get compromised
- 💾 Storing secrets securely (Kubernetes secrets, HashiCorp Vault, etc.)
- 🐛 Services break when keys expire or change
- 📋 Managing different keys for each service pair
- 🚨 Keys can be stolen, logged, or accidentally committed to git

### ✅ After: No API Keys with Ephemos (gRPC Core)

**Time Server (gRPC with automatic authentication):**
```go
func (s *TimeService) GetTime(ctx context.Context, req *pb.TimeRequest) (*pb.TimeResponse, error) {
    // 🎉 No API key validation - SPIFFE authentication is automatic!
    // If this function runs, the client is already authenticated via mTLS
    
    loc, err := time.LoadLocation(req.Timezone)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid timezone: %v", err)
    }
    
    currentTime := time.Now().In(loc)
    return &pb.TimeResponse{
        Time: currentTime.Format("2006-01-02 15:04:05 MST"),
    }, nil
}

func main() {
    // Core provides gRPC server with SPIFFE authentication
    config := &ports.Configuration{
        Service: ports.ServiceConfig{
            Name: "time-server",
            Domain: "prod.company.com",
        },
    }
    
    server, _ := ephemos.IdentityServer(ctx, 
        ephemos.WithServerConfig(config), 
        ephemos.WithAddress(":8080"))
    
    // Register your gRPC service
    pb.RegisterTimeServiceServer(server.GRPCServer(), &TimeService{})
    server.ListenAndServe(ctx) // SPIFFE mTLS handled automatically
}
```

**Time Client (gRPC with automatic authentication):**
```go
func main() {
    // Core provides gRPC client with SPIFFE authentication
    config := &ports.Configuration{
        Service: ports.ServiceConfig{
            Name: "time-client",
            Domain: "prod.company.com",
        },
    }
    
    client, _ := ephemos.IdentityClient(ctx, ephemos.WithConfig(config))
    
    // Connect automatically authenticates using SPIFFE certificates
    conn, err := client.Connect(ctx, "time-server.prod.company.com:8080")
    if err != nil {
        log.Fatal("SPIFFE authentication failed:", err) // But no secrets involved
    }
    
    timeService := pb.NewTimeServiceClient(conn.GRPCConn())
    response, _ := timeService.GetTime(ctx, &pb.TimeRequest{Timezone: "UTC"})
    fmt.Printf("Current time: %s\n", response.Time)
}
```

**HTTP Support via Contrib:**
For HTTP/REST APIs, use contrib middleware that consumes core primitives:
```go
// See contrib/middleware/chi/ or contrib/middleware/gin/
import "github.com/sufield/ephemos/contrib/middleware/chi"

r := chi.NewRouter()
r.Use(chimiddleware.SPIFFEAuth(authConfig)) // Uses core cert/bundle/authorizer
r.Get("/time", timeHandler) // Handler gets authenticated SPIFFE identity
```

**What changed:**
- ✅ **Zero secrets in code** - no API keys anywhere
- ✅ **Zero secret management** - no environment variables or secret stores
- ✅ **Automatic authentication** - SPIFFE mTLS happens transparently 
- ✅ **Automatic rotation** - certificates refresh every hour
- ✅ **Simple failure mode** - connection either works or doesn't
- ✅ **gRPC-first architecture** - Core focuses on gRPC, HTTP via contrib

## Architecture: Core + Contrib

Ephemos follows a modular architecture:

### 🏗️ **Core** (this repository)
- **SPIFFE identity management** - Service certificates and trust bundles
- **gRPC client/server** - Authenticated gRPC connections with automatic mTLS
- **Identity interceptors** - Authentication and identity propagation for gRPC
- **Configuration** - Service identity and trust domain management

**Core provides:** Certificates, trust bundles, authorizers, and gRPC connectivity.

### 🧩 **Contrib** (extensions for frameworks)  
- **HTTP middleware** - Chi, Gin, and other framework integrations
- **Examples and guides** - HTTP client examples using core primitives
- **Framework adapters** - Consume core certificates/bundles for HTTP authentication

**Contrib consumes:** Core primitives like `IdentityService.GetCertificate()`, `GetTrustBundle()`, and authorizers.

### 📁 **Repository Structure**
```
ephemos/                    # Core library
├── pkg/ephemos/           # Public API (gRPC-focused)
├── internal/              # Core implementation
└── contrib/               # Framework extensions
    ├── middleware/
    │   ├── chi/          # Chi router middleware  
    │   └── gin/          # Gin framework middleware
    ├── examples/         # HTTP client examples
    └── docs/             # HTTP integration guides
```

## Configuration (No Secrets Here Either!)

Instead of managing API keys, you just configure your service identity:

**time-server.yaml** (server configuration):
```yaml
service:
  name: "time-server"
```

**time-client.yaml** (client configuration):
```yaml
service:
  name: "time-client"
```

**No secrets in these config files!** Authentication happens using certificates that are automatically managed behind the scenes.


## Workflow Comparison: Admin Registration vs Dashboard Management

### ❌ Old Workflow: Dashboard + API Key Management

**For Developers (every time they need service-to-service communication):**
1. 🌐 Log into company dashboard/admin panel  
2. 🔑 Navigate to "API Keys" section
3. ➕ Create new API key for each service pair (e.g., "time-client-to-time-server")
4. 📋 Copy the generated key 
5. 💾 Store key in environment variables, Kubernetes secrets, or config management
6. 🔄 Repeat for every service that needs to talk to another service
7. 📅 Set up rotation schedules and alerts for key expiration
8. 🚨 Handle key rotation across all deployments when keys expire

**Problems:**
- Developers need dashboard access and training
- Keys proliferate rapidly (N×M keys for N services talking to M services)
- Manual rotation procedures
- Keys can be forgotten, logged, or mismanaged

### ✅ New Workflow: Admin Registration (One-Time Setup)

**For Administrators (one-time setup per service):**
```bash
# Admin registers each service once when it's created
ephemos register service --name time-client
ephemos register service --name time-server
```

**For Developers (zero setup needed):**
- ✅ **No dashboard login required**
- ✅ **No API keys to create or manage**  
- ✅ **No secrets to store or rotate**
- ✅ **Just write code and configure your service identity**

**Key Difference:**
- **Before:** Developers had to manage secrets for every service interaction
- **After:** Admin registers services once; developers just configure service identity in YAML

## Quick Start

### 1. Install Ephemos
```bash
go get github.com/sufield/ephemos
```

### 2. Replace Your API Key Code

**Instead of this:**
```go
// Old way with API keys
apiKey := os.Getenv("SERVICE_API_KEY")
req.Header.Set("Authorization", "Bearer " + apiKey)
```

**Write this:**
```go  
// New way with Ephemos
client, _ := ephemos.NewClient("config.yaml")
conn, _ := client.Connect("other-service")
service := ephemos.NewServiceClient(conn)
```

### 3. Set Up Service Identity (One Time)

Instead of generating API keys, register your services:
```bash
# One-time setup per service (like creating a database user)
ephemos register service --name time-client
ephemos register service --name time-server
```

### 4. Deploy Without Secrets

Your deployment files no longer need secret management:

**Before (Kubernetes with API keys):**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-keys
data:
  TIME_SERVICE_API_KEY: dGltZS1zZXJ2aWNlLXNlY3JldA== # base64 encoded secret 😫
  TIME_CLIENT_API_KEY: dGltZS1jbGllbnQtc2VjcmV0

---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: time-client
        env:
        - name: TIME_SERVICE_API_KEY
          valueFrom:
            secretKeyRef:
              name: api-keys
              key: TIME_SERVICE_API_KEY # 😫 Managing secrets
```

**After (Kubernetes with Ephemos):**
```yaml
apiVersion: apps/v1  
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: time-client
        # 🎉 No secrets needed!
        volumeMounts:
        - name: ephemos-config
          mountPath: /config
      volumes:
      - name: ephemos-config
        configMap:
          name: ephemos-config # Just regular config, no secrets
```

## Benefits for Developers

### 🧹 **Simpler Code**
- No authentication logic cluttering your business logic
- No API key management code
- No error handling for expired/invalid keys

### 🔒 **Better Security**  
- No secrets to accidentally commit to git
- No secrets in environment variables or config files
- No secrets to store in Kubernetes/Docker secrets
- Certificates automatically rotate every hour

### 🚀 **Easier Operations**
- No secret rotation procedures
- No "service X is down because API key expired" incidents  
- No managing different keys for different environments
- No secrets in CI/CD pipelines

### 🧪 **Simpler Testing**
```go
// Test your business logic without mocking authentication
func TestGetCurrentTime(t *testing.T) {
    client := &TimeClient{} 
    time, err := client.GetCurrentTime("UTC")
    // No need to mock API key validation!
}
```

## How It Works

**Like Caddy for your internal services** - just as Caddy automatically handles HTTPS certificates, Ephemos automatically handles service authentication:

- **Caddy**: Issues short-lived certificates (12 hours) from its local CA for HTTPS
- **Ephemos**: Issues short-lived certificates (1 hour) from SPIRE for service identity
- **Both**: Auto-rotate certificates before expiration - zero manual management

The key difference: Caddy secures browser-to-server, Ephemos secures service-to-service.

## Installation & Setup

```bash
# 1. Get Ephemos
go get github.com/sufield/ephemos

# 2. Install the identity system (one-time setup)
./scripts/setup-ephemos.sh

# 3. Register your services (like creating database users)
ephemos register service --name my-service
ephemos register service --name other-service

# 4. Replace your API key code with Ephemos code
# (see examples above)

# 5. Deploy without secrets! 
```

## Examples

### gRPC (Core)
- [Complete gRPC Examples](examples/) - Working client/server code using core
- [Migration Guide](docs/migration.md) - Step-by-step API key to Ephemos migration
- [Kubernetes Setup](docs/kubernetes.md) - Deploy without secret management

### HTTP (Contrib)
- [Chi Middleware](contrib/middleware/chi/) - SPIFFE authentication for Chi router
- [Gin Middleware](contrib/middleware/gin/) - SPIFFE authentication for Gin framework  
- [HTTP Client Examples](contrib/examples/) - Using core primitives with `net/http`
- [HTTP Integration Guide](contrib/docs/HTTP_CLIENT.md) - Detailed HTTP setup instructions

## Requirements

- Go 1.24+
- Linux/macOS (Windows coming soon)

---

**🎯 Bottom Line:** Replace API keys with automatic authentication. Your services prove who they are using certificates instead of secrets, and certificates rotate automatically every hour. No more secret management headaches.