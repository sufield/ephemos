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

Extremely simple - just two lines of code:

**Server:**
```go
server := ephemos.IdentityServer()
server.RegisterService(serviceRegistrar)
server.Serve(listener)
```

**Client:**
```go
client := ephemos.IdentityClient()
conn, _ := client.Connect("service-name", "localhost:50051")
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

Authentication failure occurs at the TLS transport layer, not at the application level:

1. **Connection Failure**: The client receives a connection/transport error because the mTLS handshake fails before any application messages are exchanged.

2. **Typical Error Messages**:
   - `"transport: authentication handshake failed"`
   - `"connection error: desc = \"transport: Error while dialing dial tcp: x509: certificate signed by unknown authority\""`
   - `"rpc error: code = Unavailable desc = connection error"`

3. **No Application Response**: The server's application code (like the Echo service) is never reached - the connection is rejected during the mTLS handshake.

4. **Security Benefit**: Unauthorized clients cannot establish connections, providing defense in depth.

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

### How do I register services?

Use the Ephemos CLI for one-time registration:

```bash
# Register a service with SPIRE
ephemos register --config ephemos.yaml
```

This creates the necessary SPIRE registration entries for the service.

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
server := ephemos.NewIdentityServer(ctx, configPath)
client := ephemos.NewIdentityClient(ctx, configPath)
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
make proto      # Generate protobuf code
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
