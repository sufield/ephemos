# Ephemos

**No more API keys or secrets between your services.** Ephemos handles service-to-service authentication automatically using certificates that rotate every hour - so you never have to manage, store, or worry about API keys again.

## The Problem with API Keys

Every microservice needs to authenticate to other services. The traditional approach uses API keys:

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

### ✅ After: No API Keys with Ephemos

**Time Server (no authentication code needed):**
```go
func (s *TimeService) GetTime(ctx context.Context, timezone string) (string, error) {
    // 🎉 No API key validation - authentication is automatic!
    // If this function runs, the client is already authenticated
    
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        return "", err
    }
    
    currentTime := time.Now().In(loc)
    return currentTime.Format("2006-01-02 15:04:05 MST"), nil
}

func main() {
    // Setup happens once - no secrets to manage
    server, _ := ephemos.NewServer("config/time-server.yaml")
    ephemos.Mount(server, &TimeService{})
    server.ListenAndServe() // Authentication handled automatically
}
```

**Time Client (no API keys to manage):**
```go
func main() {
    // No API keys, no secrets, no environment variables!
    client, _ := ephemos.NewClient("config/time-client.yaml")
    
    // Connect automatically authenticates using certificates
    conn, err := client.Connect("time-server")
    if err != nil {
        log.Fatal("Authentication failed:", err) // But no secrets involved
    }
    
    timeService := ephemos.NewTimeServiceClient(conn)
    currentTime, _ := timeService.GetTime(context.Background(), "UTC")
    fmt.Printf("Current time: %s\n", currentTime)
}
```

**What changed:**
- ✅ **Zero secrets in code** - no API keys anywhere
- ✅ **Zero secret management** - no environment variables or secret stores
- ✅ **Automatic authentication** - happens transparently 
- ✅ **Automatic rotation** - certificates refresh every hour
- ✅ **Simple failure mode** - connection either works or doesn't

## Configuration (No Secrets Here Either!)

Instead of managing API keys, you just configure which services can talk to each other:

**time-server.yaml** (server configuration):
```yaml
service:
  name: "time-server"

# Which services are allowed to connect to this server
authorized_clients:
  - "time-client"
  - "monitoring-service"
  # Any other service will be rejected automatically
```

**time-client.yaml** (client configuration):
```yaml
service:
  name: "time-client"

# Which services this client trusts to connect to  
trusted_services:
  - "time-server"
```

**No secrets in these config files!** Authentication happens using certificates that are automatically managed behind the scenes.

## Real-World Example

Let's see how a payment service calls a user service:

### ❌ Before: API Keys

```go
// payment-service calling user-service
func (p *PaymentService) ProcessPayment(userID string, amount float64) error {
    // 😫 Need to manage API key for user-service
    userAPIKey := os.Getenv("USER_SERVICE_API_KEY")
    
    req, _ := http.NewRequest("GET", 
        fmt.Sprintf("https://user-service/users/%s", userID), nil)
    req.Header.Set("Authorization", "Bearer " + userAPIKey)
    
    resp, err := http.DefaultClient.Do(req)
    if resp.StatusCode == 401 {
        return errors.New("user service rejected our API key")
    }
    
    // Process payment logic...
    return nil
}
```

### ✅ After: Ephemos

```go
// payment-service calling user-service  
func (p *PaymentService) ProcessPayment(userID string, amount float64) error {
    // 🎉 No API keys needed - authentication is automatic!
    user, err := p.userClient.GetUser(context.Background(), userID)
    if err != nil {
        return err // Authentication happens transparently
    }
    
    // Process payment logic...
    return nil
}
```

**Configuration:**
```yaml
# payment-service config
service:
  name: "payment-service"
trusted_services:
  - "user-service"

---
# user-service config  
service:
  name: "user-service"
authorized_clients:
  - "payment-service"
  - "admin-dashboard" 
  # payment-service is allowed, others are rejected
```

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
ephemos register service --name payment-service
ephemos register service --name user-service
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
  USER_SERVICE_API_KEY: dXNlci1zZXJ2aWNlLXNlY3JldA== # base64 encoded secret 😫
  PAYMENT_SERVICE_API_KEY: cGF5bWVudC1zZXJ2aWNlLXNlY3JldA==

---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: payment-service
        env:
        - name: USER_SERVICE_API_KEY
          valueFrom:
            secretKeyRef:
              name: api-keys
              key: USER_SERVICE_API_KEY # 😫 Managing secrets
```

**After (Kubernetes with Ephemos):**
```yaml
apiVersion: apps/v1  
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: payment-service
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

### 🧹 **Cleaner Code**
- No authentication logic cluttering your business methods
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
func TestProcessPayment(t *testing.T) {
    service := &PaymentService{} 
    err := service.ProcessPayment("user123", 100.00)
    // No need to mock API key validation!
}
```

## How It Works (Simple Version)

1. **Instead of API keys**, each service gets a unique certificate
2. **Certificates prove identity** - like a driver's license for services  
3. **Certificates rotate automatically** every hour (you never see them)
4. **Services authenticate each other** during connection, before your code runs
5. **Authorization is configured**, not coded - specify which services can talk to which

Think of it like **HTTPS for your internal services** - you don't manage the certificates, but you get automatic encryption and authentication.

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

- [Complete Examples](examples/) - Working client/server code
- [Migration Guide](docs/migration.md) - Step-by-step API key to Ephemos migration
- [Kubernetes Setup](docs/kubernetes.md) - Deploy without secret management

## Requirements

- Go 1.24+
- Linux/macOS (Windows coming soon)

---

**🎯 Bottom Line:** Replace API keys with automatic authentication. Your services prove who they are using certificates instead of secrets, and certificates rotate automatically every hour. No more secret management headaches.