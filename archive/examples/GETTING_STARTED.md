# Getting Started with Ephemos

This tutorial walks you through building your first identity-based service using the Ephemos library. By the end, you'll have a secure client-server application with automatic mTLS authentication.

## Table of Contents
- [What You'll Build](#what-youll-build)
- [Prerequisites](#prerequisites)
- [Quick Setup](#quick-setup)
- [Step 1: Set Up Your Environment](#step-1-set-up-your-environment)
- [Step 2: Create Your First Service](#step-2-create-your-first-service)
- [Step 3: Build a Client](#step-3-build-a-client)
- [Step 4: Set Up SPIRE](#step-4-set-up-spire)
- [Step 5: Register Services](#step-5-register-services)
- [Step 6: Run Your Services](#step-6-run-your-services)
- [Step 7: Verify Security](#step-7-verify-security)
- [Next Steps](#next-steps)

## What You'll Build

You'll create two services that communicate securely:

```
┌─────────────────┐    mTLS over gRPC    ┌─────────────────┐
│   Echo Client   │ ◄─────────────────► │   Echo Server   │
│                 │                     │                 │
│ SPIFFE ID:      │                     │ SPIFFE ID:      │
│ .../echo-client │                     │ .../echo-server │
└─────────────────┘                     └─────────────────┘
        │                                       │
        └─────────── SPIRE Agent ────────────────┘
                         │
                   ┌──────────┐
                   │   SPIRE  │
                   │  Server  │
                   └──────────┘
```

**Key Features:**
- ✅ Automatic mTLS encryption
- ✅ Certificate rotation (no manual management)
- ✅ Identity-based authorization
- ✅ No hardcoded secrets or passwords

## Prerequisites

- **Go 1.23+**: `go version` should show 1.23 or later
- **Docker**: For running SPIRE infrastructure
- **Basic gRPC knowledge**: Understanding of protocol buffers and gRPC

### Quick Environment Check

```bash
# Check Go version
go version

# Check Docker
docker --version

# Check if ports are available
lsof -i :8080 -i :8081  # Should return no results
```

## Quick Setup

For the impatient, run the automated demo:

```bash
git clone https://github.com/sufield/ephemos.git
cd ephemos
make demo
```

This starts SPIRE, builds the examples, and runs them automatically. Continue reading for the step-by-step tutorial.

## Step 1: Set Up Your Environment

### 1.1 Clone and Build Ephemos

```bash
# Clone the repository
git clone https://github.com/sufield/ephemos.git
cd ephemos

# Install dependencies
go mod download

# Build everything
make all
```

### 1.2 Verify Build

```bash
# Check that binaries were created
ls -la bin/
# Expected output:
# ephemos (CLI tool)
# echo-server (example server)
# echo-client (example client)
```

### 1.3 Explore the Code

```bash
# Look at the example server
cat examples/echo-server/main.go

# Look at the example client  
cat examples/echo-client/main.go

# Look at the protocol definition
cat examples/proto/echo.proto
```

## Step 2: Create Your First Service

Let's create a simple "Hello World" service from scratch.

### 2.1 Create Project Structure

```bash
# Create a new directory for your service
mkdir -p my-first-service/{server,client,proto}
cd my-first-service
```

### 2.2 Define Your Service

Create `proto/hello.proto`:

```protobuf
syntax = "proto3";
option go_package = "github.com/yourname/my-first-service/proto";

package hello;

service HelloService {
    rpc SayHello(HelloRequest) returns (HelloResponse);
}

message HelloRequest {
    string name = 1;
}

message HelloResponse {
    string message = 1;
    string server_identity = 2;  // Will contain SPIFFE ID
}
```

### 2.3 Generate gRPC Code

```bash
# Generate Go code from proto
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/hello.proto
```

### 2.4 Create the Server

Create `server/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    
    "google.golang.org/grpc"
    
    "github.com/sufield/ephemos/pkg/ephemos"
    pb "github.com/yourname/my-first-service/proto"
)

type server struct {
    pb.UnimplementedHelloServiceServer
    identity *ephemos.ServiceIdentity
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
    // Get client identity from context (provided by Ephemos)
    clientIdentity := ephemos.GetIdentityFromContext(ctx)
    
    log.Printf("Hello request from %s (client: %s)", req.Name, clientIdentity.URI)
    
    return &pb.HelloResponse{
        Message:        fmt.Sprintf("Hello %s! This is a secure service.", req.Name),
        ServerIdentity: s.identity.URI,
    }, nil
}

func main() {
    // Load Ephemos configuration
    config, err := ephemos.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Create Ephemos client
    client, err := ephemos.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create Ephemos client: %v", err)
    }
    defer client.Close()
    
    // Get our service identity
    identity, err := client.GetIdentity(context.Background())
    if err != nil {
        log.Fatalf("Failed to get identity: %v", err)
    }
    
    log.Printf("Server identity: %s", identity.URI)
    
    // Create gRPC server with Ephemos TLS
    tlsConfig, err := client.GetTLSConfig(context.Background())
    if err != nil {
        log.Fatalf("Failed to get TLS config: %v", err)
    }
    
    grpcServer := grpc.NewServer(grpc.Creds(tlsConfig))
    
    // Register our service
    pb.RegisterHelloServiceServer(grpcServer, &server{identity: identity})
    
    // Listen and serve
    listen, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    log.Println("Server starting on :8080")
    if err := grpcServer.Serve(listen); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### 2.5 Create the Client

Create `client/main.go`:

```go
package main

import (
    "context"
    "log"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    
    "github.com/sufield/ephemos/pkg/ephemos"
    pb "github.com/yourname/my-first-service/proto"
)

func main() {
    // Load Ephemos configuration
    config, err := ephemos.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Create Ephemos client
    client, err := ephemos.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create Ephemos client: %v", err)
    }
    defer client.Close()
    
    // Get our identity
    identity, err := client.GetIdentity(context.Background())
    if err != nil {
        log.Fatalf("Failed to get identity: %v", err)
    }
    
    log.Printf("Client identity: %s", identity.URI)
    
    // Get TLS credentials
    tlsConfig, err := client.GetTLSConfig(context.Background())
    if err != nil {
        log.Fatalf("Failed to get TLS config: %v", err)
    }
    
    // Connect to server
    conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()
    
    // Create service client
    helloClient := pb.NewHelloServiceClient(conn)
    
    // Make requests
    for i := 0; i < 5; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        
        response, err := helloClient.SayHello(ctx, &pb.HelloRequest{
            Name: fmt.Sprintf("User%d", i+1),
        })
        
        if err != nil {
            log.Fatalf("SayHello failed: %v", err)
        }
        
        log.Printf("Response: %s (from %s)", response.Message, response.ServerIdentity)
        
        cancel()
        time.Sleep(1 * time.Second)
    }
}
```

## Step 3: Configuration Files

### 3.1 Server Configuration

Create `server/config.yaml`:

```yaml
service:
  name: hello-server
  domain: example.org

spiffe:
  socket_path: /tmp/spire-agent/public/api.sock

# Only allow our client to connect
authorized_clients:
  - spiffe://example.org/hello-client
```

### 3.2 Client Configuration

Create `client/config.yaml`:

```yaml
service:
  name: hello-client
  domain: example.org

spiffe:
  socket_path: /tmp/spire-agent/public/api.sock

# Trust our server
trusted_servers:
  - spiffe://example.org/hello-server
```

## Step 4: Set Up SPIRE

### 4.1 Start SPIRE Infrastructure

Use Docker Compose to run SPIRE:

```bash
# From the ephemos directory
cd scripts/demo

# Start SPIRE server and agent
./start-spire.sh
```

This creates:
- SPIRE server on port 8081
- SPIRE agent with Unix socket at `/tmp/spire-agent/public/api.sock`

### 4.2 Verify SPIRE is Running

```bash
# Check SPIRE server
docker ps | grep spire-server

# Check SPIRE agent
ls -la /tmp/spire-agent/public/api.sock

# View logs
docker logs spire-server
docker logs spire-agent
```

## Step 5: Register Services

Services need to be registered with SPIRE before they can get identities.

### 5.1 Register Server

```bash
# Register hello-server service
docker exec -it spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://example.org/hello-server \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:user:$(id -u)

# Verify registration
docker exec -it spire-server /opt/spire/bin/spire-server entry show
```

### 5.2 Register Client

```bash
# Register hello-client service
docker exec -it spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://example.org/hello-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:user:$(id -u)
```

### 5.3 Alternative: Use Ephemos CLI

```bash
# Use the built-in registration tool
cd my-first-service

# Register server
../bin/ephemos register \
    --config server/config.yaml \
    --selector unix:user:$(id -u)

# Register client  
../bin/ephemos register \
    --config client/config.yaml \
    --selector unix:user:$(id -u)
```

## Step 6: Run Your Services

### 6.1 Build Your Services

```bash
# Build server
cd server
go mod init my-first-service
go mod tidy
go build -o hello-server .

# Build client
cd ../client
go mod init my-first-service
go mod tidy  
go build -o hello-client .
```

### 6.2 Start the Server

```bash
cd server
./hello-server
# Expected output:
# Server identity: spiffe://example.org/hello-server
# Server starting on :8080
```

### 6.3 Run the Client

In a new terminal:

```bash
cd client
./hello-client
# Expected output:
# Client identity: spiffe://example.org/hello-client
# Response: Hello User1! This is a secure service. (from spiffe://example.org/hello-server)
# Response: Hello User2! This is a secure service. (from spiffe://example.org/hello-server)
# ...
```

## Step 7: Verify Security

### 7.1 Test Identity Verification

The logs should show both services authenticating each other:

```bash
# Server logs show client identity
grep "Hello request from" server.log

# Client successfully verifies server identity
grep "Response:" client.log
```

### 7.2 Test Network Security

```bash
# Try connecting without proper identity (should fail)
curl https://localhost:8080
# Expected: Connection refused or certificate error

# Monitor encrypted traffic
sudo tcpdump -i lo port 8080
# You'll see encrypted TLS traffic, not plaintext
```

### 7.3 Examine Certificates

```bash
# View the certificate chain
echo | openssl s_client -connect localhost:8080 -servername hello-server 2>/dev/null | openssl x509 -noout -text

# Look for SPIFFE ID in Subject Alternative Names
```

## Step 8: Add Authorization

Let's add fine-grained authorization to the server.

### 8.1 Update Server Code

Modify `server/main.go`:

```go
func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
    // Get client identity
    clientIdentity := ephemos.GetIdentityFromContext(ctx)
    
    // Authorization check
    if !s.isAuthorized(clientIdentity.URI) {
        return nil, status.Errorf(codes.PermissionDenied, 
            "client %s not authorized", clientIdentity.URI)
    }
    
    log.Printf("Authorized request from %s", clientIdentity.URI)
    
    return &pb.HelloResponse{
        Message:        fmt.Sprintf("Hello %s! You are authorized.", req.Name),
        ServerIdentity: s.identity.URI,
    }, nil
}

func (s *server) isAuthorized(clientID string) bool {
    authorizedClients := []string{
        "spiffe://example.org/hello-client",
        "spiffe://example.org/admin-client",
    }
    
    for _, authorized := range authorizedClients {
        if clientID == authorized {
            return true
        }
    }
    return false
}
```

### 8.2 Test Authorization

```bash
# Rebuild and restart server
cd server && go build -o hello-server . && ./hello-server

# Client should still work
cd client && ./hello-client

# Register unauthorized client
docker exec -it spire-server /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://example.org/bad-client \
    -parentID spiffe://example.org/spire-agent \
    -selector unix:user:$(id -u)

# Try connecting as unauthorized client (modify client config and test)
```

## Next Steps

Congratulations! You've built your first secure service with Ephemos. Here's what to explore next:

### 🔧 Advanced Features

1. **Configuration Management**
   - Environment-based configs
   - Dynamic configuration reloading
   - Configuration validation

2. **Service Discovery**
   - Integration with service registries
   - Health check endpoints
   - Load balancing strategies

3. **Observability**
   - Metrics collection
   - Distributed tracing
   - Structured logging

### 📚 Learn More

- **[Architecture Guide](ARCHITECTURE.md)**: Deep dive into Ephemos internals
- **[Deployment Guide](DEPLOYMENT.md)**: Production deployment patterns  
- **[Troubleshooting](TROUBLESHOOTING.md)**: Common issues and solutions
- **[Examples](proto/)**: Copy-paste templates for your services

### 🚀 Production Checklist

Before deploying to production:

- [ ] Replace `example.org` with your actual trust domain
- [ ] Use production-grade SPIRE deployment (not Docker)  
- [ ] Implement proper authorization policies
- [ ] Set up monitoring and alerting
- [ ] Configure certificate rotation policies
- [ ] Test failure scenarios (network partitions, certificate expiry)
- [ ] Security review of service implementations

### 💡 Tips and Best Practices

1. **Always validate client identities** in your service handlers
2. **Use structured logging** to track identity and authorization events
3. **Monitor certificate expiration** to prevent service outages
4. **Test with certificate rotation** to ensure services handle updates gracefully
5. **Implement graceful shutdown** to properly close SPIRE connections

### 🤝 Get Help

- **GitHub Issues**: Report bugs or request features
- **Discussions**: Ask questions and share experiences
- **Documentation**: Comprehensive guides and API reference

---

You now have the foundation to build secure, identity-based services with Ephemos. The same patterns you learned here scale to complex microservice architectures with hundreds of services.