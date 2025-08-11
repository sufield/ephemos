# Ephemos - Identity-Based Authentication for Go Services

Ephemos is a Go library that provides identity-based authentication for backend services using SPIFFE/SPIRE, replacing plaintext API keys with mTLS. It abstracts away all SPIFFE/SPIRE complexity, making identity-based authentication as simple as using API keys.

## Features

- **Simple API**: One-line setup for both servers and clients
- **No Plaintext API Keys**: No more leaking plaintext secrets
- **Abstraction**: No SPIFFE/SPIRE terminology exposed to developers
- **Automatic Certificate Rotation**: Transparent handling of certificate lifecycle
- **Elegant Architecture**: Hexagonal architecture with proper separation of concerns
- **Ubuntu 24 Optimized**: Scripts and configurations optimized for Ubuntu 24

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

### Server Usage

```go
import (
	"context"
	"net"
	"github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/examples/proto"
)

ctx := context.Background()

// Create identity-based server with config
server, err := ephemos.NewIdentityServer(ctx, "config/echo-server.yaml")
if err != nil {
	log.Fatal(err)
}
defer server.Close()

// Register service using the generic registrar (recommended - no boilerplate)
serviceRegistrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
	proto.RegisterEchoServiceServer(s, &EchoServer{})
})
server.RegisterService(ctx, serviceRegistrar)

// Start listening - completely abstracted
lis, _ := net.Listen("tcp", ":50051")
server.Serve(ctx, lis)
```

### Client Usage

```go
import (
	"context"
	"github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/examples/proto"
)

// Simple connection with identity
ctx := context.Background()
client, err := ephemos.NewIdentityClient(ctx, "config/echo-client.yaml")
if err != nil {
	log.Fatal(err)
}
defer client.Close()

conn, err := client.Connect(ctx, "echo-server", "localhost:50051")
if err != nil {
	log.Fatal(err)
}
defer conn.Close() // Always defer Close for proper cleanup

// Create service client - no gRPC details exposed
echoClient, err := proto.NewEchoClient(conn.GetClientConnection())
if err != nil {
	log.Fatal(err)
}
defer echoClient.Close()
```

## Configuration

Create an `ephemos.yaml` file in your project root or use the examples in the `config/` folder:

```yaml
service:
  name: "your-service"
  domain: "example.org"

# Optional SPIFFE configuration
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

## Service Registration

Ephemos offers two approaches for registering your gRPC services:

### Option 1: Generic Registrar (Recommended)

Use the built-in generic registrar - no boilerplate code required:

```go
// Works for any gRPC service - just one line!
registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
	proto.RegisterYourServiceServer(s, &YourServiceImpl{})
})
server.RegisterService(ctx, registrar)
```

### Option 2: Custom Registrar (Advanced)

For developers who want more control, you can create service-specific registrars:

```go
type YourServiceRegistrar struct {
	server proto.YourServiceServer
}

func NewYourServiceRegistrar(server proto.YourServiceServer) *YourServiceRegistrar {
	return &YourServiceRegistrar{server: server}
}

func (r *YourServiceRegistrar) Register(grpcServer *grpc.Server) {
	proto.RegisterYourServiceServer(grpcServer, r.server)
}

// Usage
registrar := NewYourServiceRegistrar(&YourServiceImpl{})
server.RegisterService(ctx, registrar)
```

**Recommendation**: Use the generic registrar unless you need custom registration logic or validation.

### Example Configurations

The `config/` folder contains example configurations:
- `config/ephemos.yaml` - General service configuration template
- `config/echo-server.yaml` - Echo server example (authorizes echo-client)
- `config/echo-client.yaml` - Echo client example (trusts echo-server)

## Architecture

Ephemos follows the hexagonal architecture:

```
ephemos/
├── internal/
│   ├── core/           # Domain logic (no external dependencies)
│   │   ├── domain/     # Business entities
│   │   ├── ports/      # Interface definitions
│   │   └── services/   # Domain services
│   └── adapters/
│       ├── primary/    # Inbound adapters (API, CLI)
│       └── secondary/  # Outbound adapters (SPIFFE, gRPC)
└── pkg/ephemos/        # Public API
```

## Demo

The included demo shows:
1. Starting SPIRE server and agent
2. Registering services with one command
3. Server starting with identity "echo-server"
4. Client successfully connecting using mTLS
5. Authentication failure when registration is removed

Run the complete demo in under 5 minutes:

```bash
make demo
```

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