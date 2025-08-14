# Proto Templates

This directory contains protocol buffer definitions that serve as templates for building your own services with Ephemos.

## Files Overview

### Template Files
- **`echo.proto`** - Example service definition showing gRPC service structure
- **`README.md`** - This documentation file

### Generated Code Location
The Echo example's generated protobuf code and wrapper implementations are located in:
- `github.com/sufield/ephemos/pkg/ephemos/proto` - Generated code and client wrappers

This demonstrates the recommended pattern where:
- **Templates** (like `echo.proto`) stay in `examples/proto/` for copying
- **Generated code** goes in your project's appropriate package structure

## Using These Templates

### 1. Copy as Starting Point

Copy this entire directory to your project:

```bash
cp -r /path/to/ephemos/examples/proto ./proto
cd proto
```

### 2. Customize for Your Service

#### Update Protocol Definition

Rename and modify `echo.proto`:

```bash
mv echo.proto my-service.proto
```

Edit the content:

```protobuf
syntax = "proto3";

package myservice;
option go_package = "github.com/yourcompany/my-service/proto";

service MyService {
  rpc ProcessData(DataRequest) returns (DataResponse);
}

message DataRequest {
  string input = 1;
}

message DataResponse {
  string output = 1;
  bool success = 2;
}
```

#### Generate Go Code

```bash
protoc --go_out=. --go-grpc_out=. my-service.proto
```

This creates:
- `my-service.pb.go` (message types)
- `my-service_grpc.pb.go` (service interfaces)

#### Update Service Registrar

Modify `registrar.go`:

```go
type MyServiceRegistrar struct {
    server MyServiceServer
}

func NewMyServiceRegistrar(server MyServiceServer) *MyServiceRegistrar {
    return &MyServiceRegistrar{
        server: server,
    }
}

func (r *MyServiceRegistrar) Register(grpcServer *grpc.Server) {
    RegisterMyServiceServer(grpcServer, r.server)
}
```

#### Update Client Wrapper

Modify `client.go` to replace EchoClient:

```go
type MyServiceClient struct {
    *Client[MyServiceClient]
}

func NewMyServiceClient(conn *grpc.ClientConn) (*MyServiceClient, error) {
    client, err := NewClient(conn, NewMyServiceClient)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }
    
    return &MyServiceClient{Client: client}, nil
}

func (c *MyServiceClient) ProcessData(ctx context.Context, input string) (*DataResponse, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    req := &DataRequest{Input: input}
    resp, err := c.client.ProcessData(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to process data: %w", err)
    }
    
    return resp, nil
}
```

## Template Components Explained

### Generic Client Pattern

The `Client[T]` wrapper provides a reusable pattern:

```go
type Client[T any] struct {
    client T
    conn   *grpc.ClientConn
}
```

**Benefits:**
- Type-safe at compile time
- Consistent connection management
- Error handling and validation
- Easy to test and mock

### Service Registration Pattern

The `ServiceRegistrar` interface allows Ephemos to work with any service:

```go
type ServiceRegistrar interface {
    Register(*grpc.Server)
}
```

**Why this works:**
- Keeps Ephemos service-agnostic
- Clean separation of concerns
- Standard pattern across all services

### Error Handling Pattern

Consistent error handling throughout:

```go
if err != nil {
    return nil, fmt.Errorf("failed to [operation]: %w", err)
}
```

**Best practices:**
- Always wrap errors with context
- Use structured error types when appropriate
- Validate inputs before processing

## Development Workflow

### 1. Proto-First Development

1. Define your service interface in `.proto`
2. Generate Go code with `protoc`
3. Implement business logic
4. Create client wrappers for convenience

### 2. Iterative Design

1. Start with simple service definition
2. Add methods as needed
3. Use backward-compatible changes when possible
4. Version your API when breaking changes are needed

### 3. Testing Strategy

1. Unit test business logic separately
2. Integration test with real gRPC connections
3. Contract test client-server compatibility

## Example: Converting Echo to Your Service

Here's a step-by-step example of adapting the Echo templates:

### Step 1: Define Your Service

```protobuf
// user-service.proto
syntax = "proto3";

package userservice;
option go_package = "github.com/company/user-service/proto";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  int64 created_at = 4;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message GetUserRequest {
  string id = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
}
```

### Step 2: Update Registrar

```go
// registrar.go
package proto

import "google.golang.org/grpc"

type UserServiceRegistrar struct {
    server UserServiceServer
}

func NewUserServiceRegistrar(server UserServiceServer) *UserServiceRegistrar {
    return &UserServiceRegistrar{
        server: server,
    }
}

func (r *UserServiceRegistrar) Register(grpcServer *grpc.Server) {
    RegisterUserServiceServer(grpcServer, r.server)
}
```

### Step 3: Create Client Wrapper

```go
// client.go - keep the generic Client[T], add UserServiceClient
type UserServiceClient struct {
    *Client[UserServiceClient]
}

func NewUserServiceClient(conn *grpc.ClientConn) (*UserServiceClient, error) {
    client, err := NewClient(conn, NewUserServiceClient)
    if err != nil {
        return nil, fmt.Errorf("failed to create user service client: %w", err)
    }
    
    return &UserServiceClient{Client: client}, nil
}

func (c *UserServiceClient) CreateUser(ctx context.Context, name, email string) (*User, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    if strings.TrimSpace(name) == "" {
        return nil, fmt.Errorf("name cannot be empty")
    }
    
    if strings.TrimSpace(email) == "" {
        return nil, fmt.Errorf("email cannot be empty")
    }
    
    req := &CreateUserRequest{
        Name:  strings.TrimSpace(name),
        Email: strings.TrimSpace(email),
    }
    
    user, err := c.client.CreateUser(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    return user, nil
}

func (c *UserServiceClient) GetUser(ctx context.Context, id string) (*User, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    if strings.TrimSpace(id) == "" {
        return nil, fmt.Errorf("user ID cannot be empty")
    }
    
    req := &GetUserRequest{Id: strings.TrimSpace(id)}
    user, err := c.client.GetUser(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to get user %s: %w", id, err)
    }
    
    return user, nil
}

func (c *UserServiceClient) ListUsers(ctx context.Context, pageSize int32, pageToken string) (*ListUsersResponse, error) {
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    req := &ListUsersRequest{
        PageSize:  pageSize,
        PageToken: pageToken,
    }
    
    resp, err := c.client.ListUsers(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to list users: %w", err)
    }
    
    return resp, nil
}
```

## Best Practices

### 1. Protocol Design

- Use clear, descriptive names
- Include version information in package names
- Design for backward compatibility
- Use standard gRPC patterns (pagination, error handling)

### 2. Client Wrappers

- Always validate inputs
- Provide convenient method signatures
- Add retry logic where appropriate
- Include proper error context

### 3. Service Registration

- Keep registrars simple and focused
- Follow the established pattern
- Include proper documentation

### 4. Code Organization

- Keep generated code with proto definitions
- Separate business logic from transport concerns
- Use consistent naming conventions
- Include examples in documentation

## Next Steps

1. **Copy these templates** to start your project
2. **Customize the proto definition** for your domain
3. **Implement business logic** in your server
4. **Create client applications** using the wrappers
5. **Deploy with Ephemos** for automatic identity-based authentication

For more detailed guidance, see:
- `../GETTING_STARTED.md` - Step-by-step service creation
- `../ARCHITECTURE.md` - Design patterns and best practices
- `../DEPLOYMENT.md` - Production deployment guide