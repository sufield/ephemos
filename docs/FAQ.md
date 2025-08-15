# Ephemos FAQ

## General Questions

### What is Ephemos?

Ephemos is a Go library that provides identity-based authentication for backend services using SPIFFE/SPIRE. It replaces plaintext API keys with mutual TLS (mTLS) authentication while abstracting away all the complexity of SPIFFE/SPIRE configuration.

### Why use Ephemos instead of API keys?

- **Security**: X.509 certificates are cryptographically verifiable, unlike plaintext API keys
- **Short-lived**: Certificates expire in 1 hour, limiting exposure if compromised
- **Automatic rotation**: No manual certificate management required
- **Identity-based**: Services authenticate based on cryptographic identity, not shared secrets
- **Zero-trust**: Each service interaction is mutually authenticated

### How simple is it to use?

Extremely simple - just two lines of code for developers:

**Server:**
```go
server := ephemos.IdentityServer(ctx, configPath)
server.ListenAndServe(ctx)
```

**Client:**
```go
client := ephemos.IdentityClient(ctx, configPath)
conn, _ := client.Connect(ctx, "service-name", "localhost:50051")
```

**Service Registration** (CLI-only, for administrators):
```bash
ephemos-cli register --name my-service --domain company.com
```

## Technical Questions

### Does Ephemos use X.509 certificates for authentication?

Yes, Ephemos uses X.509 certificates for authentication through SPIFFE's implementation:

1. **SPIFFE X.509 SVIDs**: Uses SPIFFE X.509-SVID (Secure Verifiable Identity Documents) which are X.509 certificates with SPIFFE IDs in the Subject Alternative Name (SAN) extension.

2. **mTLS Authentication**: Certificates enable mutual TLS authentication:
   - Each service gets an X.509 certificate from SPIRE
   - Certificates contain the service's SPIFFE ID (e.g., `spiffe://example.org/echo-server`)
   - Both client and server present certificates during TLS handshake

3. **Certificate Structure**: 
   ```go
   type Certificate struct {
       Cert       *x509.Certificate    // The X.509 certificate
       PrivateKey interface{}          // Private key for the certificate
       Chain      []*x509.Certificate // Certificate chain
   }
   ```

4. **Automatic Rotation**: X.509 certificates are automatically rotated by SPIRE:
   - Default validity: 1 hour
   - SPIRE handles renewal before expiration
   - Services get new certificates transparently

5. **Trust Verification**: Services verify peer certificates against the trust bundle containing root CA certificates.

### What happens when authentication fails?

In the context of using the go-spiffe SDK with SPIFFE and SPIRE, identity-based authentication relies on workloads obtaining and presenting valid SPIFFE Verifiable Identity Documents (SVIDs), typically X.509-SVIDs for mTLS connections or JWT-SVIDs for other authentication flows. These SVIDs are issued by the SPIRE Agent after successful workload attestation, which verifies the workload's identity against pre-registered selectors (e.g., process ID, Kubernetes pod details) defined in registration entries on the SPIRE Server.

If authentication fails—such as when a service is not registered with the SPIRE Agent (meaning no matching registration entry exists, so attestation fails and no SVID can be issued)—the following occurs:

- The workload cannot fetch a valid SVID from the SPIFFE Workload API.
- In practical usage with the go-spiffe SDK (e.g., during mTLS setup via functions like `spiffetls.Dial` or `spiffetls.Listen`), this results in an error being returned, preventing the secure connection from being established. For instance, the TLS handshake fails if the peer cannot present or validate an SVID against the trust bundle, leading to a connection refusal or termination.
- No further processing of the request proceeds, as the identity cannot be verified.

Your understanding is correct: no authorization is performed. SPIFFE and SPIRE primarily handle authentication (AuthN: verifying *who* the workload is via SVIDs), while authorization (AuthZ: deciding *what* the authenticated workload can do) is a separate concern typically implemented on top of successful authentication, often using external tools like Open Policy Agent (OPA) or application-level policies based on the verified SPIFFE ID. If authentication fails, the process does not reach the authorization stage, as there is no trusted identity to authorize.

**Typical Error Messages when authentication fails:**
- `"transport: authentication handshake failed"`
- `"connection error: desc = \"transport: Error while dialing dial tcp: x509: certificate signed by unknown authority\""`
- `"rpc error: code = Unavailable desc = connection error"`

**Security Benefit**: Unauthorized clients cannot establish connections, providing defense in depth.

### What does the echo server respond to echo client requests?

The echo server responds with two fields:

1. **Message**: The exact same message that the client sent (echoing it back)
2. **From**: The string "echo-server" to identify the responder

**Example:**
When the client sends:
```go
Message: "Hello from echo-client!"
```

The server responds:
```go
Message: "Hello from echo-client!"  // Same message echoed back
From: "echo-server"                  // Identifies the responder
```

**Client output:**
```
Response: Hello from echo-client! (from: echo-server)
```

This demonstrates that:
- The server successfully received and processed the request
- mTLS authentication worked (otherwise connection would fail)
- Both services can identify themselves in communication

## Architecture Questions

### Why is service registration CLI-only and not part of the public API?

This addresses a fundamental **chicken and egg security problem** in authentication systems.

#### **The Security Paradox**

```
Client needs SPIFFE identity → to authenticate to registration service
    ↓
But client gets SPIFFE identity → by registering with SPIRE
    ↓  
How does client authenticate → to register itself? 🤔
```

#### **Bad Solutions (Security Anti-Patterns)**

❌ **API Keys in Configuration Files**
```yaml
# This defeats the entire purpose of identity-based authentication!
registration:
  api_key: "sk_live_51H..."  # Plaintext secret - exactly what we're trying to avoid
```

❌ **Pre-shared Secrets**
```go
// Doesn't scale, creates key management nightmare
client.RegisterSelf("shared-secret-123")
```

❌ **Open Registration Endpoint**
```go
// Anyone can register malicious services!
http.POST("/register", serviceInfo)  // No authentication required
```

❌ **Self-Registration with Trust-on-First-Use**
```go
// How do we verify this is a legitimate service?
service.RegisterSelf("payment-service")  // Could be malicious impersonator
```

#### **The Correct Solution: Administrative Registration**

**Service registration is an infrastructure operation, not an application operation.**

```bash
# 🔐 SECURE: Admin registers service out-of-band
ephemos-cli register \
    --name payment-service \
    --domain company.com \
    --selector k8s:ns:production \
    --selector k8s:sa:payment-service

# ✅ SIMPLE: Service uses pre-registered identity
server := ephemos.IdentityServer(ctx, configPath)  // Gets registered identity
client := ephemos.IdentityClient(ctx, configPath)  // Gets registered identity
```

#### **Who Can Register Services?**

✅ **Platform Administrators** - Direct SPIRE server access  
✅ **Infrastructure as Code** - Terraform, Helm charts with proper credentials  
✅ **CI/CD Pipelines** - Authenticated deployment systems  
✅ **Kubernetes Operators** - Service account permissions  
✅ **Bastion Hosts** - Secure admin workstations with VPN

#### **From Where?**

✅ **Secure Networks** - Admin VPN, private networks  
✅ **Control Plane** - Kubernetes API server, management clusters  
✅ **Deployment Systems** - Authenticated CI/CD environments  
✅ **Operations Centers** - SOC workstations with proper access controls

#### **Benefits of This Architecture**

🛡️ **Zero Trust** - No service can self-register or register others  
📋 **Audit Trail** - All registrations logged and auditable  
🔒 **Principle of Least Privilege** - Only authenticated admins can register services  
⚖️ **Separation of Concerns** - Infrastructure operations separate from application logic  
🏗️ **Scalable** - Works in containerized and orchestrated environments

#### **Example: Production Workflow**

```bash
# 1. 👨‍💼 Admin registers service (one-time, secure)
ephemos-cli register --config payment-service.yaml

# 2. 🏗️ Service deploys and automatically gets identity
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
spec:
  template:
    spec:
      serviceAccountName: payment-service  # Maps to SPIRE registration
      containers:
      - name: payment-service
        image: company/payment-service:v1.2.3
        # Service automatically gets spiffe://company.com/payment-service identity
```

```go
// 3. 👩‍💻 Developer code is simple and secure
func main() {
    // Automatically uses pre-registered identity
    server := ephemos.IdentityServer(ctx, "config.yaml")
    server.ListenAndServe(ctx)  // Only authorized clients can connect
}
```

This solves the bootstrapping problem by separating **identity provisioning** (admin/CLI) from **identity usage** (developer/API).

### What architecture does Ephemos use?

Ephemos follows strict hexagonal (ports and adapters) architecture:

```
ephemos/
├── internal/
│   ├── core/              # Domain logic (no external dependencies)
│   │   ├── domain/        # Business entities and value objects
│   │   ├── ports/         # Interface definitions
│   │   └── services/      # Domain services
│   └── adapters/
│       ├── primary/       # Inbound adapters (API, CLI)
│       └── secondary/     # Outbound adapters (SPIFFE, gRPC, config)
└── pkg/ephemos/          # Public API
```

- Domain core has zero external dependencies
- Dependencies flow: adapters → ports → domain
- Clean interfaces (ports) define boundaries
- Dependency inversion properly implemented

### Application Layer vs Domain Layer Clarification

The codebase separates business concerns into distinct layers following Clean Architecture principles:

#### **Domain Layer** (`internal/domain/`, `internal/core/domain/`)

**Purpose:** Pure business entities and rules

**Contains:**
- **Entities**: `ServiceIdentity`, `Certificate`, `TrustBundle` 
- **Value Objects**: Authentication policies, configuration values
- **Domain Logic**: Validation rules, business constraints
- **No Dependencies**: Only standard library imports

**Characteristics:**
- 📍 **Pure business concepts**
- 📍 **No framework dependencies** 
- 📍 **No I/O operations**
- 📍 **Immutable by design when possible**

```go
// Domain entity - pure business concept
type ServiceIdentity struct {
    Name   string
    Domain string  
    URI    string
}

func (s *ServiceIdentity) Validate() error {
    // Pure business rules
}
```

#### **Application Layer** (`internal/app/`, `internal/core/services/`)

**Purpose:** Use case orchestration and business workflows

**Contains:**
- **Application Services**: `IdentityService` 
- **Use Cases**: "Create server identity", "Establish secure connection"
- **Workflow Orchestration**: Coordinates domain objects + ports
- **Port Interfaces**: Defines contracts for external systems

**Characteristics:**
- 📍 **Orchestrates domain objects**
- 📍 **Defines use case workflows**
- 📍 **Manages state and caching**
- 📍 **Depends on domain layer**
- 📍 **Uses ports for external dependencies**

```go
// Application service - orchestrates use cases
type IdentityService struct {
    identityProvider  IdentityProvider  // Port
    config           *Configuration     // Domain value object
    cachedIdentity   *domain.ServiceIdentity // Domain entity
}

func (s *IdentityService) CreateServerIdentity() (*domain.ServiceIdentity, error) {
    // Use case workflow: validate config, get identity, cache result
}
```

#### **Key Distinction:**

| **Domain Layer** | **Application Layer** |
|------------------|----------------------|
| **What the business IS** | **What the business DOES** |
| Entities, Values, Rules | Use Cases, Workflows |
| Pure, no side effects | Coordinates I/O via ports |
| Framework-agnostic | Framework-agnostic |
| No external dependencies | Uses domain + ports |

### How does certificate rotation work?

Certificate rotation is handled automatically and transparently:

1. **SPIRE Management**: SPIRE server issues short-lived certificates (1 hour validity)
2. **Automatic Renewal**: go-spiffe library automatically renews certificates before expiration
3. **Transparent to Application**: Services continue operating without interruption
4. **Zero Downtime**: New certificates are obtained and used seamlessly
5. **Background Process**: Rotation happens in background threads

## Configuration Questions

### What configuration is required?

Minimal configuration in `ephemos.yaml`:

```yaml
service:
  name: "your-service"     # Required: service identifier
  domain: "example.org"    # Required: trust domain

# Optional - has sensible defaults
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"

# For servers: specify authorized clients
authorized_clients:
  - "allowed-client-1"
  - "allowed-client-2"

# For clients: specify trusted servers (optional)
trusted_servers:
  - "trusted-server"
```

### Does Ephemos automatically register itself with SPIRE?

**No, Ephemos does NOT automatically register itself with SPIRE.** Services must be explicitly registered before they can obtain identities.

**Why No Auto-Registration?**
- **Security**: Prevents unauthorized services from self-registering with SPIRE
- **Control**: Administrators control which services get which identities
- **SPIRE Design**: SPIRE requires explicit registration entries for security compliance

**What Ephemos Provides:**

1. **Manual Registration Tool**: 
   ```bash
   # Register a service with SPIRE (one-time setup)
   ephemos register --name echo-server --domain example.org
   ```

2. **Automatic Identity Retrieval** (after registration):
   ```go
   // Once registered, these automatically retrieve SPIFFE identities:
   server := ephemos.IdentityServer(ctx, "config.yaml") 
   client := ephemos.IdentityClient(ctx, "config.yaml")
   ```

**Registration Workflow:**
```bash
# Step 1: MANUAL - Register the service with SPIRE
ephemos register --config service.yaml

# Step 2: AUTOMATIC - Service retrieves its identity when started
server := ephemos.IdentityServer(ctx, "service.yaml")  # Gets registered identity
client := ephemos.IdentityClient(ctx, "client.yaml")   # Gets registered identity
```

**Demo Script Registration:**
The demo script handles registration manually using SPIRE CLI:
```bash
# Demo does this for you:
sudo spire-server entry create \
    -spiffeID spiffe://example.org/echo-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:uid:1000 \
    -ttl 3600
```

**Summary:**
- ✅ **Registration**: Manual (required first step)
- ✅ **Identity Retrieval**: Automatic (after registration)
- ✅ **Certificate Rotation**: Automatic (handled by SPIRE)
- ✅ **mTLS Authentication**: Automatic (handled by Ephemos)

### How do I register services?

Use the Ephemos CLI for one-time registration:

```bash
# Register a service with SPIRE
ephemos register --config ephemos.yaml
```

This creates the necessary SPIRE registration entries for the service.

### Does Ephemos provide only CLI for registering services? Does Ephemos need to register itself with SPIRE?

**Ephemos provides a CLI for service registration, but Ephemos itself does NOT need to register as a workload with SPIRE.**

#### Registration Architecture Overview

**Ephemos is a framework/library, not a separate service:**
- Ephemos code runs **within your service process**
- Each **developer service** needs its own SPIRE registration
- **No separate Ephemos daemon** requiring registration
- Ephemos provides tools to **help register your services**

#### What Ephemos Provides for Registration

**1. CLI Registration Tool**

Ephemos includes a production CLI (`cmd/ephemos-cli`) with registration commands:

```bash
# Register using config file (recommended)
ephemos register --config service.yaml

# Register using command-line flags
ephemos register --name payment-service --domain prod.company.com
ephemos register --name echo-server --domain example.org --selector unix:uid:1000
```

**CLI Components:**
- **Binary**: `cmd/ephemos-cli/main.go` - Production CLI tool
- **Register Command**: `internal/cli/register.go` - Registration interface
- **Registrar Logic**: `internal/adapters/primary/cli/registrar.go` - Core implementation

**2. Programmatic Registration**

The CLI uses the `internal/adapters/primary/cli/registrar.go` component that can also be used programmatically:

```go
// For advanced use cases or custom tooling
registrar := cli.NewRegistrar(configProvider, registrarConfig)
err := registrar.RegisterService(ctx, "service.yaml")
```

#### Registration Process Detail

**Manual Registration (Security-Required, One-Time):**
```bash
# Step 1: Administrator or developer runs this
ephemos register --name payment-service --domain prod.company.com

# What happens under the hood:
# - CLI validates service name and domain
# - Calls: spire-server entry create \
#     -spiffeID spiffe://prod.company.com/payment-service \
#     -parentID spiffe://prod.company.com/spire-agent \
#     -selector unix:uid:1000 \
#     -ttl 3600
```

**Automatic Identity Retrieval (Runtime):**
```go
// Step 2: Service code automatically retrieves registered identity
server := ephemos.IdentityServer(ctx, "config.yaml")
// - Connects to SPIRE Agent via Unix socket
// - Gets X.509-SVID certificate for spiffe://prod.company.com/payment-service
// - Sets up mTLS with automatic certificate rotation
// - Ready to authenticate clients
```

#### Why No Auto-Registration?

**Security by Design:**
- **Prevents rogue services** from self-registering with SPIRE
- **Administrative control** - only authorized personnel decide service identities
- **SPIRE compliance** - SPIRE requires explicit registration for security
- **Zero Trust principle** - no automatic trust is granted

**SPIFFE/SPIRE Standard:**
- Registration must be **explicit and authorized**
- Follows **principle of least privilege**
- Supports **audit trails** for identity management
- Enables **policy enforcement** before service deployment

#### Complete Registration Workflow

| Step | Component | Action | Method |
|------|-----------|--------|---------|
| 1 | **Admin/Developer** | Register service identity | `ephemos register --config service.yaml` |
| 2 | **Service Runtime** | Retrieve identity automatically | `ephemos.IdentityServer(ctx, "config.yaml")` |
| 3 | **SPIRE Agent** | Provide certificates | Automatic (background) |
| 4 | **Authentication** | mTLS between services | Automatic (transparent) |

#### Registration Requirements by Component

| Component | Registration Required | Method |
|-----------|----------------------|---------|
| **Your Services** (payment-service, user-service, etc.) | ✅ **Yes** | `ephemos register --config service.yaml` |
| **Ephemos Framework** | ❌ **No** | (Library code - runs in your services) |
| **SPIRE Server** | ❌ **No** | (Infrastructure component) |
| **SPIRE Agent** | ❌ **No** | (Infrastructure component) |

#### Example: Complete Service Registration

**1. Service Configuration (`payment-service.yaml`):**
```yaml
service:
  name: payment-service
  domain: prod.company.com

spiffe:
  socket_path: /run/spire/sockets/agent.sock

authorized_clients:
  - order-service
  - billing-service
```

**2. Registration (One-Time):**
```bash
# Creates SPIRE entry for payment-service
ephemos register --config payment-service.yaml
```

**3. Service Code (Runtime):**
```go
// payment-service main.go
func main() {
    ctx := context.Background()
    
    // Automatically gets spiffe://prod.company.com/payment-service identity
    server, err := ephemos.IdentityServer(ctx, "payment-service.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer server.Close()
    
    // Service registration is handled by CLI, not code
    // Simply start the server with automatic mTLS authentication
    server.ListenAndServe(ctx) // Only authorized clients can connect
}
```

**4. Client Usage:**
```go
// order-service connecting to payment-service
client, err := ephemos.IdentityClient(ctx, "order-service.yaml")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Automatic mTLS authentication using registered identities
conn, err := client.Connect(ctx, "payment-service", "payment-svc:50051")
if err != nil {
    log.Fatal(err) // Fails if order-service not in payment-service's authorized_clients
}

paymentClient := pb.NewPaymentServiceClient(conn.GetClientConnection())
```

#### Summary

- ✅ **Service Registration**: Manual (required security step via CLI)
- ✅ **Identity Retrieval**: Automatic (handled by Ephemos APIs)
- ✅ **Certificate Rotation**: Automatic (handled by SPIRE)
- ✅ **mTLS Authentication**: Automatic (handled by Ephemos)
- ❌ **Ephemos Self-Registration**: Not needed (it's a library, not a service)

### Where should I put the config file?

Ephemos looks for configuration in these locations (in order):
1. `EPHEMOS_CONFIG` environment variable
2. `ephemos.yaml` (current directory)
3. `configs/ephemos.yaml`
4. `/etc/ephemos/ephemos.yaml`

## Deployment Questions

### How does certificate acquisition differ between demo and production?

The fundamental SPIFFE/SPIRE mechanism is the same, but the **attestation method** (how services prove their identity) differs significantly:

#### Demo Environment (Local Development)
- **Attestation**: Unix UID (`unix:uid:0` for root) - any root process gets the certificate
- **Registration**: Manual via CLI (`ephemos register --name service-name`)
- **Bootstrap**: Insecure (`insecure_bootstrap = true` in agent config)
- **Socket Path**: `/tmp/spire-agent/public/api.sock`
- **Security**: Weak - suitable only for local testing

**Demo Flow:**
```
Service → SPIRE Agent: "I'm UID 0 (root)"
Agent → Service: "Here's certificate for echo-server"
⚠️ Problem: Any root process gets this certificate!
```

#### Production Environment
- **Attestation**: Platform-specific (Kubernetes pods, AWS instances, Docker containers)
- **Registration**: Automated via CI/CD, operators, or IaC
- **Bootstrap**: Secure with pre-distributed trust bundles
- **Socket Path**: `/run/spire/sockets/agent.sock` (standard production path)
- **Security**: Strong - only the actual workload gets its certificate

**Production Flow (Kubernetes example):**
```
Service (in pod) → SPIRE Agent: "I need a certificate"
Agent → K8s API: "Tell me about this pod"
K8s API → Agent: "Pod: echo-server-7d4b9, Namespace: prod, ServiceAccount: echo-server"
Agent → Service: "✓ Verified! Here's certificate for echo-server"
✅ Only the real echo-server pod gets this certificate!
```

#### Production Attestation Methods

1. **Kubernetes**: 
   ```bash
   -selector k8s:ns:production
   -selector k8s:sa:echo-server
   -selector k8s:pod-label:app:echo-server
   ```

2. **AWS EC2**:
   ```bash
   -selector aws_iid:instance-id:i-1234567890
   -selector aws_iid:tag:Name:echo-server
   ```

3. **Docker**:
   ```bash
   -selector docker:label:app:echo-server
   -selector docker:image:company/echo-server:v1.2.3
   ```

#### What Stays the Same
Your application code using Ephemos remains **identical** in both environments:

```go
// This code works in BOTH demo and production:
server := ephemos.IdentityServer(ctx, configPath)
client := ephemos.IdentityClient(ctx, configPath)
```

The differences are only in:
- How services are registered (manual vs automated)
- How identities are verified (Unix UID vs platform attestation)
- Where SPIRE runs (local vs distributed)

### What are the system requirements?

- **Go Version**: 1.24.5 or later
- **Platform**: Linux/amd64 (Ubuntu 24 optimized)
- **SPIRE**: 1.8+ (automatically installed by demo scripts)
- **Dependencies**: All Go dependencies managed via go.mod

### How do I run the demo?

Simple one command demo:

```bash
make demo
```

This will:
1. Install SPIRE (if needed)
2. Start SPIRE server and agent
3. Register demo services
4. Run the 5-minute demonstration
5. Show authentication success and failure scenarios

### How do I deploy in production?

1. **Install SPIRE**: Use production-ready SPIRE deployment
2. **Register Services**: Use `ephemos register` for each service
3. **Configure Services**: Provide appropriate `ephemos.yaml` for each service
4. **Deploy**: Services automatically obtain identities on startup
5. **Monitor**: SPIRE handles certificate lifecycle automatically

## Development Questions

### How do I build the project?

```bash
# Build everything
make build

# Build specific components
make examples   # Build example applications
make test       # Run tests
make clean      # Clean artifacts
```

### How do I add a new service?

1. **Create configuration**: New `service-name.yaml` with service identity
2. **Register service**: Run `ephemos register --config service-name.yaml`
3. **Use in code**: Call `ephemos.IdentityServer()` or `ephemos.IdentityClient()`
4. **Configure authorization**: Add to `authorized_clients` or `trusted_servers` as needed

### Can I extend Ephemos?

Yes! The hexagonal architecture makes extension easy:

- **Add new identity providers**: Implement the `IdentityProvider` port
- **Add new transports**: Implement the `TransportProvider` port  
- **Add new config sources**: Implement the `ConfigurationProvider` port
- **Add new authentication policies**: Extend the domain models

All extensions integrate cleanly without changing existing code.

### How are the tests structured?

Ephemos follows a strict testing strategy that separates fast unit tests from integration tests:

#### **Fast & Pure Tests**

The core business logic is tested with extremely fast, deterministic unit tests:

**Characteristics:**
- ✅ **Sub-millisecond execution** (0.009-0.011s per package)
- ✅ **No I/O operations** - no network, filesystem, or database calls
- ✅ **No external dependencies** - only standard library and internal packages
- ✅ **Pure functions** - same input always produces same output
- ✅ **Table-driven design** - comprehensive test cases in structured format

**Coverage:**
- **100% of business logic branches** in domain and application layers
- **All validation rules** and edge cases
- **Error handling paths** and boundary conditions
- **Concurrent access patterns** where applicable

**Example Test Structure:**
```go
func TestServiceIdentity_Validate(t *testing.T) {
    tests := []struct {
        name        string
        serviceName string
        domain      string
        wantErr     bool
        errorMsg    string
    }{
        {
            name:        "valid identity",
            serviceName: "test-service",
            domain:      "example.com",
            wantErr:     false,
        },
        {
            name:        "empty service name",
            serviceName: "",
            domain:      "example.com", 
            wantErr:     true,
            errorMsg:    "service name cannot be empty",
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)
            err := identity.Validate()
            // ... assertions
        })
    }
}
```

**Test Metrics:**
- 📊 **2,232+ lines** of fast unit test code
- 📊 **5 core packages** with comprehensive coverage  
- 📊 **Benchmarks included** for performance-critical paths
- 📊 **Concurrent safety tests** for shared state

**Benefits:**
- **Instant feedback** - tests run in milliseconds
- **TDD-friendly** - can run thousands of times during development
- **Reliable** - no flaky tests due to timing or external services
- **Maintainable** - clear test cases document expected behavior

**Integration Tests (Separate):**
- **Slower tests** with real SPIRE integration in `examples/` 
- **End-to-end scenarios** testing full authentication flow
- **Network communication** between services
- **Certificate lifecycle** validation

This separation allows developers to run fast tests continuously while reserving slower integration tests for CI/CD pipelines.

## Troubleshooting

### Common issues and solutions:

**"Connection failed"**: Check that SPIRE agent is running and socket path is correct

**"Service not registered"**: Run `ephemos register --config your-config.yaml`

**"Permission denied"**: Ensure user has access to SPIRE socket (typically owned by root)

**"Certificate expired"**: SPIRE should auto-rotate, check SPIRE agent logs

**"Build errors"**: Run `go mod tidy` to ensure dependencies are correct

### Getting help:

- Check the logs from SPIRE server and agent
- Verify service registration with: `spire-server entry show`
- Ensure socket paths match between config and SPIRE setup
- Review the demo scripts for working examples

## Security Considerations

### Is Ephemos production-ready?

The library provides a solid foundation with:
- Industry-standard X.509 certificates
- SPIFFE/SPIRE for identity management
- Automatic certificate rotation
- Mutual TLS authentication

For production use, consider:
- Proper SPIRE server deployment and clustering
- Network security and firewall configurations
- Monitoring and alerting for certificate lifecycle
- Regular security updates for all components

### What about performance?

- **Low overhead**: mTLS adds minimal latency compared to plaintext
- **Efficient rotation**: Certificate renewal happens in background
- **Connection reuse**: gRPC connections are reused when possible
- **Optimized libraries**: Uses efficient go-spiffe implementation

The performance impact is negligible compared to the security benefits provided.

### Can I disable certificate validation for development?

**Yes, but only for development environments.**

Ephemos supports disabling certificate validation for local development through the `certificate_validation_disabled` configuration option:

#### **Development Configuration** (Allowed)
```yaml
service:
  name: "dev-service"
  domain: "dev.example.org"

agent:
  socketPath: "/run/sockets/agent.sock"

security:
  certificate_validation_disabled: true  # Development only - ignored in production
```

#### **Production Configuration** (Secure by default)
```yaml
service:
  name: "payment-service"  
  domain: "prod.company.com"

agent:
  socketPath: "/run/sockets/agent.sock"

# Certificate validation is enabled by default - no security section needed
```

#### **Security Enforcement**

Ephemos is **foolproof in production** and automatically enforces secure certificate validation:

```go
// Production environments ALWAYS validate certificates regardless of configuration
config := loadConfig("production.yaml") // Even if this sets certificate_validation_disabled: true

// Production override: Always returns false in production environments
isDisabled := config.GetEffectiveCertificateValidationDisabled() // false

// Production validation also catches configuration errors
err := config.IsProductionReady()
if err != nil {
    // Error: "certificate_validation_disabled is enabled - certificate validation must be enabled in production"
}
```

**Production Detection:**
- Production domains (not containing "dev", "test", "local", "example")
- Standard production paths (`/run/sockets/`, `/var/run/`)
- Environment variables: `NODE_ENV=production`, `ENVIRONMENT=production`, `STAGE=prod`

#### **Industry Best Practices**

This follows the same pattern as other successful Go projects, but with business-focused configuration:

- **Business-Focused Terminology**: Uses `certificate_validation_disabled` instead of implementation terms like `insecure_skip_verify`
- **Secure by Default**: Certificate validation enabled unless explicitly disabled
- **Development Productivity**: Simple flag to disable validation for local testing
- **Production Safety**: Automatic validation prevents insecure production deployments

#### **Why This Approach?**

✅ **Development Productivity** - Developers can quickly test locally without certificate setup  
✅ **Foolproof Production** - **Impossible** to disable certificate validation in production, even with misconfigured YAML  
✅ **Clear Intent** - Configuration explicitly shows when security is relaxed  
✅ **Audit Trail** - Security teams can easily identify and review insecure configurations  
✅ **Defense in Depth** - Multiple layers detect and prevent production security bypasses
