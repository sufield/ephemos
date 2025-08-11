# Ephemos - Transport-Agnostic Service Framework

[![CI/CD Pipeline](https://github.com/sufield/ephemos/actions/workflows/ci.yml/badge.svg)](https://github.com/sufield/ephemos/actions/workflows/ci.yml)
[![Security & Dependencies](https://github.com/sufield/ephemos/actions/workflows/security.yml/badge.svg)](https://github.com/sufield/ephemos/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sufield/ephemos)](https://goreportcard.com/report/github.com/sufield/ephemos)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/sufield/ephemos/badge)](https://securityscorecards.dev/viewer/?uri=github.com/sufield/ephemos)

Ephemos is a Go library that provides **transport-agnostic services** with identity-based authentication using SPIFFE/SPIRE. Write your services once with plain Go types, and run them over gRPC, HTTP, or any future transport without code changes.

## Features

- **🚀 Transport Agnostic**: Same service code runs on gRPC, HTTP, or future transports
- **🔧 Configuration Driven**: Switch transports via config, not code changes
- **📦 Domain First**: Write services with plain Go types - no protocol dependencies
- **🎯 Type Safe**: Generic `Mount[T]` API provides compile-time safety
- **🔌 Hexagonal Architecture**: Clean separation between domain and transport layers
- **🛡️ Identity Security**: SPIFFE/SPIRE integration with automatic certificate rotation
- **⚡ Zero Protocol Lock-in**: Never be tied to a specific transport protocol again

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/sufield/ephemos.git
cd ephemos

# Install dependencies
go mod download

# Run the 5-minute demo
make demo
```

### Server Usage (Transport-Agnostic)

```go
import (
	"context"
	"log"
	
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/pkg/ephemos"
)

// Your service implementation - pure domain logic, no transport concerns
type EchoService struct {
	name string
}

// Plain Go types - no gRPC, no HTTP, no protobuf
func (e *EchoService) Echo(ctx context.Context, message string) (string, error) {
	return fmt.Sprintf("[%s] Echo: %s", e.name, message), nil
}

func (e *EchoService) Ping(ctx context.Context) error {
	log.Printf("[%s] Ping received", e.name)
	return nil
}

func main() {
	ctx := context.Background()
	
	// Create transport-agnostic server (transport determined by config)
	server, err := ephemos.NewTransportServer(ctx, "config/service.yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	// Mount your service - works with gRPC, HTTP, or any transport
	echoService := &EchoService{name: "my-service"}
	if err := ephemos.Mount[ports.EchoService](server, echoService); err != nil {
		log.Fatal(err)
	}

	// Start server - transport determined by configuration
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}
```

### Configuration (Choose Your Transport)

The same service code works with different transports - just change the config:

**gRPC Transport:**
```yaml
service:
  name: "my-service"
  domain: "example.org"

transport:
  type: "grpc"
  address: ":50051"
  
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"
```

**HTTP Transport:**
```yaml
service:
  name: "my-service"
  domain: "example.org"

transport:
  type: "http"
  address: ":8080"
  
spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"
```

**The Magic**: Same service code, different transport - just change the config! 🎉

## Testing Your Service

### gRPC Transport
```bash
# Start your service with gRPC config
EPHEMOS_CONFIG=config/grpc.yaml go run main.go

# Test with grpc_cli
grpc_cli call localhost:50051 EchoService.Echo "message: 'Hello gRPC'"
```

### HTTP Transport  
```bash
# Start your service with HTTP config
EPHEMOS_CONFIG=config/http.yaml go run main.go

# Test with curl
curl -X POST http://localhost:8080/echoservice/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello HTTP"}'
```

## Key Benefits

### 🚀 **No Protocol Lock-in**
Switch from gRPC to HTTP, or add WebSocket support later - **no code changes needed**.

### 🧪 **Easy Testing**
Test your business logic without transport concerns:
```go
func TestEchoService(t *testing.T) {
    service := &EchoService{name: "test"}
    result, err := service.Echo(context.Background(), "test message")
    // Pure domain logic testing - no mocking gRPC or HTTP!
}
```

### 📈 **Evolution Ready**
Future transport support (WebSocket, NATS, etc.) requires no changes to your services.

## Service Implementation

### Domain-First Approach

Write your services using plain Go interfaces - no transport dependencies:

```go
// Define your service interface (or use built-in ones)
type MyCustomService interface {
    ProcessData(ctx context.Context, data string) (string, error)
    ValidateInput(ctx context.Context, input string) error
}

// Implement your service with pure business logic
type MyCustomServiceImpl struct {
    validator *SomeValidator
    processor *SomeProcessor
}

func (s *MyCustomServiceImpl) ProcessData(ctx context.Context, data string) (string, error) {
    // Pure business logic - no transport concerns!
    processed := s.processor.Transform(data)
    return processed, nil
}

func (s *MyCustomServiceImpl) ValidateInput(ctx context.Context, input string) error {
    return s.validator.Validate(input)
}

// Mount with type safety
ephemos.Mount[MyCustomService](server, &MyCustomServiceImpl{...})
```

### Built-in Service Interfaces

Ephemos provides common service interfaces you can implement:

```go
// Echo service for request-response patterns
type EchoService interface {
    Echo(ctx context.Context, message string) (string, error)
    Ping(ctx context.Context) error
}

// File service for binary data handling
type FileService interface {
    Upload(ctx context.Context, filename string, data io.Reader) error
    Download(ctx context.Context, filename string) (io.Reader, error)
    List(ctx context.Context, prefix string) ([]string, error)
}

// Health service for monitoring
type HealthService interface {
    Check(ctx context.Context, service string) (HealthStatus, error)
}
```

### Example Configurations

The `config/` folder contains example configurations:
- `config/ephemos.yaml` - General service configuration template
- `config/echo-server.yaml` - Echo server example (authorizes echo-client)
- `config/echo-client.yaml` - Echo client example (trusts echo-server)

## Architecture

Ephemos implements **hexagonal architecture** with **transport adapters**:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Your Service  │    │  Ephemos Core    │    │   Transport     │
│  (Domain Logic) │    │                  │    │   Adapters      │
├─────────────────┤    ├──────────────────┤    ├─────────────────┤
│ • Pure Go types │───▶│ • Mount[T] API   │───▶│ • gRPC Server   │
│ • No transport  │    │ • Type safety    │    │ • HTTP Handlers │
│   dependencies  │    │ • Generic design │    │ • Future: NATS  │
│ • Easy testing  │    │ • Config driven  │    │   WebSocket...  │
└─────────────────┘    └──────────────────┘    └─────────────────┘

Directory Structure:
ephemos/
├── pkg/ephemos/              # 🎯 Transport-agnostic public API
├── internal/core/ports/      # 🔌 Domain service interfaces  
├── internal/adapters/
│   ├── grpc/                # 📡 gRPC transport adapter
│   ├── http/                # 🌐 HTTP transport adapter
│   └── secondary/           # 🔧 SPIFFE, config adapters
└── examples/transport-agnostic/ # 📚 Complete examples
```

### Key Principles

1. **🎯 Domain First**: Write services with plain Go - no protocols
2. **🔌 Ports & Adapters**: Clean boundaries between business and transport  
3. **📦 Single Responsibility**: Each adapter handles one transport protocol
4. **🎨 Open/Closed**: Add new transports without changing existing code

## Demo

Try the **transport-agnostic demo** - same service, different transports:

```bash
# Run with gRPC transport
EPHEMOS_CONFIG=config/transport-grpc.yaml go run examples/transport-agnostic/main.go

# Test gRPC
grpc_cli call localhost:50051 EchoService.Echo "message: 'Hello gRPC'"

# In another terminal, run with HTTP transport  
EPHEMOS_CONFIG=config/transport-http.yaml go run examples/transport-agnostic/main.go

# Test HTTP
curl -X POST http://localhost:8080/echoserviceimpl/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello HTTP"}'
```

**The same service implementation runs on both transports!** 🎉

### Full SPIRE Integration Demo

For complete identity-based authentication with SPIRE:

```bash
make demo  # Complete SPIRE setup + authentication demo
```

## Identity-Based Authentication

Ephemos enforces **cryptographic identity authentication** using SPIFFE/SPIRE, replacing traditional API keys with short-lived X.509 certificates. Here's exactly how it works:

### How Identity Authentication is Enforced

#### 1. **Certificate-Based Authentication (Not API Keys)**

```go
// ❌ Traditional API Key Authentication:
// if request.APIKey != "secret-key-123" { return Unauthorized }

// ✅ Ephemos Identity Authentication:
// Authentication happens at TLS transport layer using X.509 certificates
server := ephemos.NewIdentityServer(ctx, "config.yaml")  // Automatic mTLS
client := ephemos.NewIdentityClient(ctx, "config.yaml")  // Automatic mTLS
```

**Key Differences:**
- **API Keys**: Plaintext secrets that can be intercepted or leaked
- **Ephemos**: Cryptographic certificates that expire in 1 hour and auto-rotate

#### 2. **Transport-Layer Security Enforcement**

Authentication occurs at the **TLS handshake level**, not in application code:

```go
// Connection establishment with identity verification:
conn, err := client.Connect("echo-server", "localhost:50051")
// ↑ This line performs:
// 1. mTLS handshake with both client and server certificates  
// 2. Certificate validation against SPIFFE trust bundle
// 3. SPIFFE ID verification (spiffe://example.org/echo-client)
// 4. Connection establishment ONLY if authentication succeeds

if err != nil {
    // Authentication failed - connection never established
    // Possible errors:
    // - "certificate signed by unknown authority" 
    // - "authentication handshake failed"
    // - "transport: Error while dialing"
    log.Fatal("Authentication failed:", err)
}

// If we reach here, identity authentication succeeded
response, err := echoClient.Echo(ctx, &EchoRequest{Message: "Hello"})
```

#### 3. **Automatic Identity Verification**

Every connection automatically verifies:

| **Verification Step** | **What's Checked** | **Enforcement Point** |
|----------------------|-------------------|---------------------|
| **Certificate Validity** | Not expired, properly signed | TLS handshake |
| **Trust Chain** | Issued by trusted SPIRE CA | TLS handshake |
| **SPIFFE ID** | Matches expected service identity | TLS handshake |
| **Mutual Authentication** | Both client AND server present certificates | TLS handshake |

#### 4. **Service-Level Authorization**

Beyond authentication, Ephemos enforces **service-level authorization**:

```yaml
# config/echo-server.yaml
service:
  name: "echo-server"
  domain: "example.org"

# Only these services can connect to this server:
authorized_clients:
  - "echo-client"        # ✅ Allowed
  - "payment-service"    # ✅ Allowed
  # "malicious-service"  # ❌ Denied - not in list
```

**Enforcement Flow:**
```
1. TLS Handshake: "I'm spiffe://example.org/mystery-service"
2. Certificate Valid: ✅ Certificate is cryptographically valid
3. Authorization Check: ❌ "mystery-service" not in authorized_clients
4. Connection Rejected: Transport-level failure before app code runs
```

#### 5. **Zero Application Code Changes**

The beauty of Ephemos is that **identity enforcement is transparent**:

```go
// Your service code remains clean - no auth logic needed:
func (s *EchoService) Echo(ctx context.Context, message string) (string, error) {
    // No need to check API keys, validate tokens, or verify certificates
    // If this function is called, identity authentication already succeeded
    
    return fmt.Sprintf("Echo: %s", message), nil
}

// Authentication enforcement happens automatically in:
// - ephemos.NewIdentityServer() - sets up mTLS server
// - ephemos.NewIdentityClient() - sets up mTLS client  
// - client.Connect() - performs identity verification
```

#### 6. **Authentication Failure Scenarios**

When authentication fails, connections are rejected **before** application code runs:

```bash
# What happens with invalid/expired certificates:

$ echo-client  # Tries to connect with invalid cert
❌ Error: transport: authentication handshake failed
❌ Error: connection error: x509: certificate signed by unknown authority  
❌ Error: rpc error: code = Unavailable desc = connection error

# Server logs show:
# "TLS handshake failed: certificate verification failed"
# 
# Echo service code is NEVER executed - rejected at transport layer
```

#### 7. **Automatic Certificate Rotation**

Identity certificates automatically rotate without service downtime:

```go
// Your service code never changes:
server := ephemos.NewIdentityServer(ctx, "config.yaml")

// But behind the scenes:
// - Hour 1: Certificate A (expires 2:00 PM) 
// - Hour 1.5: SPIRE issues Certificate B (expires 3:00 PM)
// - Hour 2: Automatic switchover to Certificate B
// - Service continues running without interruption
// - Old connections gracefully drain, new connections use new cert
```

### Security Guarantees

✅ **Cryptographic Authentication**: X.509 certificates, not plaintext secrets  
✅ **Short-Lived Credentials**: 1-hour expiration limits breach impact  
✅ **Automatic Rotation**: No manual certificate management  
✅ **Mutual Authentication**: Both client and server verify each other  
✅ **Transport-Layer Security**: Authentication before application logic  
✅ **Service Authorization**: Fine-grained access control  
✅ **Zero Trust**: Every connection authenticated, nothing assumed  

### Demo: See Authentication in Action

```bash
make demo  # Watch authentication succeed AND fail in real-time
```

The demo shows:
1. **✅ Success**: echo-client authenticates and connects to echo-server
2. **❌ Failure**: After deregistration, same client is rejected
3. **🔍 Transparency**: Your application code never changes

## Development

```bash
# Generate protobuf code
make proto

# Build everything
make build

# Run tests
make test

# Build examples
make examples

# Clean artifacts
make clean
```

### Development Workflow

1. **First time setup:**
   ```bash
   make check-requirements  # Verify all tools are installed
   make proto              # Generate protobuf code
   make build              # Build CLI and library
   ```

2. **After modifying .proto files:**
   ```bash
   make proto              # Regenerate protobuf code
   make build              # Rebuild
   ```

3. **Regular development:**
   ```bash
   make test               # Run tests
   make lint               # Lint code
   make fmt                # Format code
   ```

## Documentation

The project documentation is organized in the `docs/` directory:

- **[Contributing](docs/contributing/)** - Guidelines for contributing to the project
- **[Demo](docs/demo/)** - Interactive demos and examples
- **[Deployment](docs/deployment/)** - Production deployment guides
- **[Development](docs/development/)** - Development guides and API documentation
- **[Security](docs/security/)** - Security architecture and threat models
- **[FAQ](docs/FAQ.md)** - Frequently asked questions

For a complete documentation index, see [docs/README.md](docs/README.md).

## Requirements

### System Requirements
- **Go 1.23 or later** (Go 1.24.5+ recommended)
- Protocol Buffers compiler (protoc) 
- Ubuntu 24 (for demo scripts)
- SPIRE 1.8+ (automatically installed by demo)

**Note**: Go versions 1.22 and earlier are not supported due to missing standard library packages (`slices`, `maps`, `log/slog`, `math/rand/v2`).

### Installing Protocol Buffers Compiler

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install -y protobuf-compiler
```

**Manual Installation (Linux):**
```bash
# Download latest release
wget https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-linux-x86_64.zip
unzip protoc-25.1-linux-x86_64.zip -d protoc
sudo cp protoc/bin/protoc /usr/local/bin/
sudo cp -r protoc/include/* /usr/local/include/
```

**Go Protocol Buffer Tools:**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Quick Setup
For a complete development environment setup:
```bash
make check-requirements  # Check what's missing
make install-tools       # Install all prerequisites (Ubuntu 24)
```

## How It Works

1. **Service Registration**: Services are registered with SPIRE using the Ephemos CLI
2. **Automatic Identity**: On startup, services automatically obtain their identity from SPIRE
3. **mTLS Setup**: All TLS configuration is handled transparently
4. **Certificate Rotation**: Certificates are automatically renewed before expiration
5. **Peer Verification**: Services verify each other's identities based on configuration

## License

MIT License

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to submit Pull Requests, report issues, and contribute to the project.