# Ephemos Examples

This directory contains comprehensive examples and documentation for building services with the Ephemos library. All examples are standalone templates that developers can copy and customize for their own services.

## Quick Start

1. **🚀 New users**: Start with [Getting Started](#getting-started) for a complete tutorial
2. **📄 Copy templates**: Use the [`proto/`](proto/) directory as your starting template  
3. **🖥️ Reference examples**: See [`echo-server/`](echo-server/) and [`echo-client/`](echo-client/) for working implementations

## Directory Structure

```
examples/
├── README.md            # 📋 This comprehensive guide
├── proto/               # 📄 Protocol buffer templates
│   ├── echo.proto       # Example service definition
│   ├── echo.pb.go       # Generated protobuf code
│   ├── echo_grpc.pb.go  # Generated gRPC interfaces
│   ├── client.go        # Generic client wrapper patterns
│   ├── registrar.go     # Service registrar implementation
│   └── README.md        # Template usage guide
├── echo-server/         # 🖥️ Complete server example
│   └── main.go
└── echo-client/         # 📱 Complete client example
    └── main.go
```

## What Ephemos Provides

Ephemos is an identity-based authentication library for Go services using SPIFFE/SPIRE:

- **Automatic mTLS**: All service communication secured with mutual TLS
- **Identity-based auth**: Services authenticate using cryptographic identities, not passwords
- **Zero-config security**: Certificate management handled automatically
- **Service-agnostic**: Works with any gRPC service
- **Production-ready**: Comprehensive error handling, logging, and resource management

---

# Getting Started

This complete tutorial walks you through building your first identity-based service using the Ephemos library.

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (`protoc`)
- SPIRE server and agent (for production)

### Install Protocol Buffers Tools

```bash
# Install protoc
sudo apt install protobuf-compiler  # Ubuntu/Debian
# OR
brew install protobuf               # macOS

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Step 1: Set Up Your Project

Create a new Go project:

```bash
mkdir my-service
cd my-service
go mod init github.com/yourcompany/my-service

# Add Ephemos dependency
go get github.com/sufield/ephemos
```

## Step 2: Copy Example Templates

Copy the proto templates from this examples directory:

```bash
# Copy the entire proto directory as your starting point
cp -r /path/to/ephemos/examples/proto ./proto

# Your project structure should now look like:
# my-service/
# ├── go.mod
# └── proto/
#     ├── echo.proto      # Rename and modify this
#     ├── client.go       # Modify for your service
#     └── registrar.go    # Modify for your service
```

## Step 3: Define Your Service

### 3.1 Create Your Protocol Buffer Definition

Rename and modify `proto/echo.proto` to `proto/my-service.proto`:

```protobuf
syntax = "proto3";

package myservice;

option go_package = "github.com/yourcompany/my-service/proto";

// Define your service interface
service MyService {
  rpc ProcessData(DataRequest) returns (DataResponse);
  rpc GetStatus(StatusRequest) returns (StatusResponse);
}

// Define your message types
message DataRequest {
  string input = 1;
  int32 priority = 2;
}

message DataResponse {
  string output = 1;
  bool success = 2;
}

message StatusRequest {
  string component = 1;
}

message StatusResponse {
  string status = 1;
  int64 timestamp = 2;
}
```

### 3.2 Generate Go Code

```bash
cd proto
protoc --go_out=. --go-grpc_out=. my-service.proto

# This generates:
# - my-service.pb.go      (message types)
# - my-service_grpc.pb.go (service interfaces)
```

## Step 4: Create Service Components

### 4.1 Update the Service Registrar

Modify `proto/registrar.go`:

```go
package proto

import (
    "google.golang.org/grpc"
)

// MyServiceRegistrar implements the ServiceRegistrar interface for Ephemos
type MyServiceRegistrar struct {
    server MyServiceServer
}

// NewMyServiceRegistrar creates a registrar for your service
func NewMyServiceRegistrar(server MyServiceServer) *MyServiceRegistrar {
    return &MyServiceRegistrar{
        server: server,
    }
}

// Register registers your service with the gRPC server
func (r *MyServiceRegistrar) Register(grpcServer *grpc.Server) {
    RegisterMyServiceServer(grpcServer, r.server)
}
```

### 4.2 Update the Client Wrapper

Modify `proto/client.go` to replace EchoClient with MyServiceClient:

```go
package proto

import (
    "context"
    "fmt"
    "strings"
    "google.golang.org/grpc"
)

// Keep the generic Client[T] as-is - it works for any service

// MyServiceClient wraps the generated client with additional functionality
type MyServiceClient struct {
    *Client[MyServiceClient]
}

// NewMyServiceClient creates a new client for your service
func NewMyServiceClient(conn *grpc.ClientConn) (*MyServiceClient, error) {
    client, err := NewClient(conn, NewMyServiceClient)
    if err != nil {
        return nil, fmt.Errorf("failed to create my service client: %w", err)
    }
    
    return &MyServiceClient{Client: client}, nil
}

// ProcessData calls your service with validation and error handling
func (c *MyServiceClient) ProcessData(ctx context.Context, input string, priority int32) (*DataResponse, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    if strings.TrimSpace(input) == "" {
        return nil, fmt.Errorf("input cannot be empty")
    }
    
    req := &DataRequest{
        Input:    strings.TrimSpace(input),
        Priority: priority,
    }
    
    resp, err := c.client.ProcessData(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to process data: %w", err)
    }
    
    return resp, nil
}

// GetStatus calls the status endpoint
func (c *MyServiceClient) GetStatus(ctx context.Context, component string) (*StatusResponse, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    req := &StatusRequest{Component: component}
    resp, err := c.client.GetStatus(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to get status: %w", err)
    }
    
    return resp, nil
}
```

## Step 5: Implement Your Server

Create `cmd/server/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/sufield/ephemos/pkg/ephemos"
    "github.com/yourcompany/my-service/proto"
)

// MyServer implements your service business logic
type MyServer struct {
    proto.UnimplementedMyServiceServer
}

// ProcessData implements your main business logic
func (s *MyServer) ProcessData(ctx context.Context, req *proto.DataRequest) (*proto.DataResponse, error) {
    if req == nil {
        return nil, fmt.Errorf("request cannot be nil")
    }
    
    slog.Info("Processing data", "input", req.Input, "priority", req.Priority)
    
    // Your business logic here
    output := fmt.Sprintf("Processed: %s (priority: %d)", req.Input, req.Priority)
    
    return &proto.DataResponse{
        Output:  output,
        Success: true,
    }, nil
}

// GetStatus provides service health information
func (s *MyServer) GetStatus(ctx context.Context, req *proto.StatusRequest) (*proto.StatusResponse, error) {
    if req == nil {
        return nil, fmt.Errorf("request cannot be nil")
    }
    
    slog.Info("Status request", "component", req.Component)
    
    return &proto.StatusResponse{
        Status:    "healthy",
        Timestamp: time.Now().Unix(),
    }, nil
}

func main() {
    // Setup
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })))
    
    // Create identity-aware server
    server, err := ephemos.NewIdentityServer(ctx, os.Getenv("EPHEMOS_CONFIG"))
    if err != nil {
        slog.Error("Failed to create server", "error", err)
        os.Exit(1)
    }
    defer server.Close()
    
    // Register your service
    serviceImpl := &MyServer{}
    registrar := proto.NewMyServiceRegistrar(serviceImpl)
    
    if err := server.RegisterService(ctx, registrar); err != nil {
        slog.Error("Failed to register service", "error", err)
        os.Exit(1)
    }
    
    // Setup listener
    port := os.Getenv("PORT")
    if port == "" {
        port = "50051"
    }
    
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        slog.Error("Failed to listen", "port", port, "error", err)
        os.Exit(1)
    }
    defer lis.Close()
    
    // Graceful shutdown
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-shutdown
        slog.Info("Shutdown signal received")
        cancel()
    }()
    
    slog.Info("Server starting", "port", port, "service", "my-service")
    
    if err := server.Serve(ctx, lis); err != nil {
        if ctx.Err() != nil {
            slog.Info("Server stopped gracefully")
        } else {
            slog.Error("Server error", "error", err)
            os.Exit(1)
        }
    }
}
```

## Step 6: Implement Your Client

Create `cmd/client/main.go`:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/sufield/ephemos/pkg/ephemos"
    "github.com/yourcompany/my-service/proto"
)

func main() {
    // Setup
    slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })))
    
    ctx := context.Background()
    
    // Create identity-aware client
    client, err := ephemos.NewIdentityClient(ctx, os.Getenv("EPHEMOS_CONFIG"))
    if err != nil {
        slog.Error("Failed to create client", "error", err)
        os.Exit(1)
    }
    defer client.Close()
    
    // Connect to your service
    serverAddr := os.Getenv("SERVER_ADDR")
    if serverAddr == "" {
        serverAddr = "localhost:50051"
    }
    
    conn, err := client.Connect(ctx, "my-service", serverAddr)
    if err != nil {
        slog.Error("Failed to connect", "error", err)
        os.Exit(1)
    }
    defer conn.Close()
    
    // Create service client
    serviceClient, err := proto.NewMyServiceClient(conn.GetClientConnection())
    if err != nil {
        slog.Error("Failed to create service client", "error", err)
        os.Exit(1)
    }
    defer serviceClient.Close()
    
    // Make requests
    slog.Info("Connected to server", "address", serverAddr)
    
    // Test ProcessData
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    resp1, err := serviceClient.ProcessData(ctx, "test data", 1)
    if err != nil {
        slog.Error("ProcessData failed", "error", err)
        os.Exit(1)
    }
    slog.Info("ProcessData response", "output", resp1.Output, "success", resp1.Success)
    
    // Test GetStatus
    resp2, err := serviceClient.GetStatus(ctx, "main")
    if err != nil {
        slog.Error("GetStatus failed", "error", err)
        os.Exit(1)
    }
    slog.Info("GetStatus response", "status", resp2.Status, "timestamp", resp2.Timestamp)
    
    slog.Info("Client completed successfully")
}
```

## Step 7: Configuration

Create an `ephemos.yaml` configuration file:

```yaml
service:
  name: "my-service"
  domain: "example.org"

spiffe:
  socket_path: "/tmp/spire-agent/public/api.sock"

# For servers: specify which clients can connect
authorized_clients:
  - "spiffe://example.org/my-client"

# For clients: specify which servers to trust (optional)
trusted_servers:
  - "spiffe://example.org/my-service"
```

## Step 8: Build and Test

### 8.1 Update go.mod

```bash
go mod tidy
```

### 8.2 Build

```bash
# Build server
go build -o bin/server cmd/server/main.go

# Build client
go build -o bin/client cmd/client/main.go
```

### 8.3 Test Locally

```bash
# Terminal 1 - Start server
EPHEMOS_CONFIG=ephemos.yaml ./bin/server

# Terminal 2 - Run client
EPHEMOS_CONFIG=ephemos.yaml ./bin/client
```

---

# Echo Service Example

The Echo service demonstrates a complete working implementation:

### Run the Examples

```bash
# Terminal 1 - Start server
cd examples/echo-server  
go run main.go

# Terminal 2 - Run client
cd examples/echo-client
go run main.go
```

### What it Demonstrates

- ✅ **Identity-based authentication** with SPIFFE/SPIRE
- ✅ **Automatic mTLS** for all service communication  
- ✅ **Generic client patterns** that work with any gRPC service
- ✅ **Production-ready patterns** with proper error handling and logging
- ✅ **Resource management** with cleanup and graceful shutdown
- ✅ **Clean architecture** separating business logic from transport concerns

---

# Architecture Guide

## Overview

Ephemos follows a clean architecture pattern that separates concerns between:

- **Identity Management**: Handled by Ephemos library
- **Transport Security**: Automatic mTLS with SPIFFE/SPIRE
- **Business Logic**: Your service implementation
- **Protocol Definition**: gRPC service interfaces

## Key Patterns

### 1. Service Registration Pattern

```go
// 1. Define your service implementation
type MyServer struct {
    proto.UnimplementedMyServiceServer
}

// 2. Create a registrar that implements ServiceRegistrar
type MyServiceRegistrar struct {
    server MyServiceServer
}

func (r *MyServiceRegistrar) Register(grpcServer *grpc.Server) {
    RegisterMyServiceServer(grpcServer, r.server)
}

// 3. Register with Ephemos
server := ephemos.NewIdentityServer(ctx, configPath)
registrar := NewMyServiceRegistrar(&MyServer{})
server.RegisterService(ctx, registrar)
```

### 2. Generic Client Pattern

```go
// Generic wrapper for any gRPC service
type Client[T any] struct {
    client T
    conn   *grpc.ClientConn
}

// Service-specific wrapper
type MyServiceClient struct {
    *Client[MyServiceClient]
}
```

### 3. Recommended Project Structure

```
my-service/
├── cmd/
│   ├── server/
│   │   └── main.go          # Server entry point
│   └── client/
│       └── main.go          # Client entry point (optional)
├── proto/
│   ├── my-service.proto     # Service definition
│   ├── my-service.pb.go     # Generated messages
│   ├── my-service_grpc.pb.go # Generated gRPC code
│   ├── client.go            # Client wrapper
│   └── registrar.go         # Service registrar
├── internal/
│   ├── service/
│   │   └── service.go       # Business logic
│   ├── repository/
│   │   └── repository.go    # Data access
│   └── middleware/
│       └── logging.go       # Cross-cutting concerns
├── config/
│   ├── local.yaml           # Local development
│   ├── staging.yaml         # Staging environment
│   └── production.yaml      # Production environment
├── go.mod
├── go.sum
└── README.md
```

---

# Deployment Guide

## Local Development

### SPIRE Setup

Create SPIRE server config `/opt/spire/conf/server.conf`:

```hcl
server {
    bind_address = "127.0.0.1"
    bind_port = "8081"
    trust_domain = "example.org"
    data_dir = "/opt/spire/data/server"
    log_level = "DEBUG"
}

plugins {
    DataStore "sql" {
        plugin_data {
            database_type = "sqlite3"
            connection_string = "/opt/spire/data/server/datastore.sqlite3"
        }
    }

    NodeAttestor "join_token" {
        plugin_data {}
    }

    KeyManager "disk" {
        plugin_data {
            keys_path = "/opt/spire/data/server/keys.json"
        }
    }
}
```

Create SPIRE agent config `/opt/spire/conf/agent.conf`:

```hcl
agent {
    data_dir = "/opt/spire/data/agent"
    log_level = "DEBUG"
    server_address = "127.0.0.1"
    server_port = "8081"
    socket_path = "/tmp/spire-agent/public/api.sock"
    trust_domain = "example.org"
}

plugins {
    NodeAttestor "join_token" {
        plugin_data {}
    }

    KeyManager "disk" {
        plugin_data = {
            directory = "/opt/spire/data/agent"
        }
    }

    WorkloadAttestor "unix" {
        plugin_data {}
    }
}
```

### Start SPIRE

```bash
# Terminal 1 - Start SPIRE server
sudo spire-server run -config /opt/spire/conf/server.conf

# Terminal 2 - Create join token
TOKEN=$(sudo spire-server token generate -spiffeID spiffe://example.org/node)

# Terminal 3 - Start SPIRE agent
sudo spire-agent run -config /opt/spire/conf/agent.conf -joinToken $TOKEN
```

### Register Services

```bash
# Register your server
sudo spire-server entry create \
    -spiffeID spiffe://example.org/my-service \
    -parentID spiffe://example.org/node \
    -selector unix:path:/path/to/your/server

# Register your client
sudo spire-server entry create \
    -spiffeID spiffe://example.org/my-client \
    -parentID spiffe://example.org/node \
    -selector unix:path:/path/to/your/client
```

## Docker Deployment

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

# Runtime stage  
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/config ./config

EXPOSE 50051

CMD ["./server"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  spire-server:
    image: ghcr.io/spiffe/spire-server:1.8.0
    hostname: spire-server
    volumes:
      - ./spire/server:/opt/spire/conf/server
      - spire-data:/opt/spire/data/server
    command: ["-config", "/opt/spire/conf/server/server.conf"]
    ports:
      - "8081:8081"

  spire-agent:
    image: ghcr.io/spiffe/spire-agent:1.8.0
    hostname: spire-agent
    depends_on:
      - spire-server
    volumes:
      - ./spire/agent:/opt/spire/conf/agent
      - spire-socket:/tmp/spire-agent/public
      - /var/run/docker.sock:/var/run/docker.sock
    command: ["-config", "/opt/spire/conf/agent/agent.conf"]

  my-service:
    build: .
    depends_on:
      - spire-agent
    volumes:
      - spire-socket:/tmp/spire-agent/public:ro
      - ./config:/app/config:ro
    environment:
      - EPHEMOS_CONFIG=/app/config/docker.yaml
      - PORT=50051
    ports:
      - "50051:50051"

volumes:
  spire-data:
  spire-socket:
```

## Kubernetes Deployment

### Service Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
  labels:
    app: my-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-service
  template:
    metadata:
      labels:
        app: my-service
      annotations:
        spiffe.io/spiffe-id: "spiffe://example.org/my-service"
    spec:
      containers:
      - name: my-service
        image: my-service:latest
        ports:
        - containerPort: 50051
          protocol: TCP
        env:
        - name: PORT
          value: "50051"
        - name: EPHEMOS_CONFIG
          value: "/app/config/k8s.yaml"
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /tmp/spire-agent/public
          readOnly: true
        - name: config
          mountPath: /app/config
          readOnly: true
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: Directory
      - name: config
        configMap:
          name: my-service-config
---
apiVersion: v1
kind: Service
metadata:
  name: my-service-svc
spec:
  selector:
    app: my-service
  ports:
  - port: 50051
    targetPort: 50051
    protocol: TCP
  type: ClusterIP
```

---

# Troubleshooting Guide

## Common Issues

### 1. Connection Refused / Cannot Connect

#### Symptoms
- `connection refused` errors
- `context deadline exceeded` 
- Client cannot reach server

#### Solutions

1. **Server not running**
   ```bash
   # Check server process
   ps aux | grep my-server
   
   # Check server logs for startup errors
   tail -f /var/log/my-service.log
   ```

2. **Wrong port/address**
   ```bash
   # Verify configuration
   echo $SERVER_ADDR
   cat ephemos.yaml
   ```

3. **Firewall blocking connection**
   ```bash
   # Check firewall rules
   sudo iptables -L
   sudo ufw status
   
   # Open port
   sudo ufw allow 50051
   ```

### 2. SPIFFE ID Not Found

#### Symptoms
- `no SPIFFE ID found in certificate`
- `workload not registered`
- `no identity available`

#### Solutions

1. **SPIRE agent not running**
   ```bash
   # Start SPIRE agent
   sudo spire-agent run -config /opt/spire/conf/agent.conf
   
   # Check agent logs
   sudo journalctl -u spire-agent -f
   ```

2. **Workload not registered**
   ```bash
   # Register your service
   sudo spire-server entry create \
     -spiffeID spiffe://example.org/my-service \
     -parentID spiffe://example.org/node \
     -selector unix:path:/path/to/binary
   ```

### 3. Certificate Validation Errors

#### Symptoms
- `certificate signed by unknown authority`
- `certificate has expired`
- `certificate is valid for different hostname`

#### Solutions

1. **Trust bundle issues**
   ```bash
   # Get trust bundle
   sudo spire-server bundle show
   
   # Compare with what agent has
   curl --unix-socket /tmp/spire-agent/public/api.sock \
     http://localhost/v1/bundles
   ```

2. **Certificate expired**
   ```bash
   # Force certificate refresh
   curl -X POST --unix-socket /tmp/spire-agent/public/api.sock \
     http://localhost/v1/svids
   ```

### 4. Context/Timeout Issues

#### Symptoms
- `context deadline exceeded`
- `context canceled`
- Operations timing out

#### Solutions

1. **Increase timeout**
   ```go
   // For slow operations
   ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
   defer cancel()
   ```

2. **Add retry logic**
   ```go
   func callWithRetry(ctx context.Context, client *proto.MyServiceClient) error {
       for i := 0; i < 3; i++ {
           reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
           _, err := client.ProcessData(reqCtx, input, priority)
           cancel()
           
           if err == nil {
               return nil
           }
           
           if i < 2 { // Don't sleep on last attempt
               time.Sleep(time.Duration(i+1) * time.Second)
           }
       }
       return err
   }
   ```

## Debugging Tools

### Enable Debug Logging

```go
// In your main.go
import "log/slog"

func main() {
    // Enable debug logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    slog.SetDefault(logger)
    
    // Your service code...
}
```

### SPIRE Debug Commands

```bash
# Check agent health
spire-agent api fetch

# Get detailed workload info
spire-agent api fetch -write /tmp/svids.json
cat /tmp/svids.json | jq

# Check server entries
spire-server entry show -output json | jq

# Validate specific entry
spire-server entry show -spiffeID spiffe://example.org/my-service
```

---

# Using the Templates

## Quick Template Usage

```bash
# 1. Copy templates to your project
cp -r examples/proto ./proto

# 2. Customize for your service
cd proto
# - Edit echo.proto -> your-service.proto  
# - Update client.go and registrar.go
# - Run: protoc --go_out=. --go-grpc_out=. your-service.proto

# 3. Implement your server (follow echo-server pattern)
# 4. Implement your client (follow echo-client pattern)
```

## Template Components

- **`echo.proto`** - Service definition template
- **`client.go`** - Generic `Client[T]` wrapper + service-specific client
- **`registrar.go`** - Service registration for Ephemos
- **Generated files** - `echo.pb.go`, `echo_grpc.pb.go`

## Best Practices Demonstrated

### Security
- Automatic mTLS with SPIFFE/SPIRE
- Input validation and sanitization
- Proper error handling without information leakage
- Resource cleanup and lifecycle management

### Performance  
- Connection reuse and pooling patterns
- Context-based timeouts and cancellation
- Efficient resource management
- Minimal overhead identity verification

### Maintainability
- Clear separation of concerns
- Comprehensive error context
- Structured logging for observability
- Test-friendly architecture

---

# Next Steps

1. **Add more methods** to your service interface
2. **Implement middleware** for logging, metrics, etc.
3. **Add database integration** for persistent data
4. **Set up monitoring** and health checks
5. **Deploy with orchestration** (Kubernetes, Docker Compose)

## Common Patterns

### Error Handling
```go
resp, err := client.ProcessData(ctx, input, priority)
if err != nil {
    return fmt.Errorf("failed to process data: %w", err)
}
```

### Context with Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Structured Logging
```go
slog.Info("Processing request", 
    "user", userID, 
    "operation", "process_data",
    "priority", priority)
```

### Resource Cleanup
```go
defer func() {
    if err := resource.Close(); err != nil {
        slog.Warn("Failed to close resource", "error", err)
    }
}()
```

---

The examples demonstrate everything needed to build production-ready services with automatic identity-based authentication using Ephemos!