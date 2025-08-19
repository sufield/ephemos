# Port Abstraction Analysis

## 🔍 Infrastructure Leakage Issues Found

After inspecting the port interfaces in `/internal/core/ports/`, several violations of hexagonal architecture principles were discovered where infrastructure concerns leak into the core domain through port interfaces.

## 🚨 Issues Identified

### 1. HTTP Client Infrastructure Leakage
**File:** `internal/core/ports/client.go`  
**Issue:** `Conn.HTTPClient() (*http.Client, error)`  
**Problem:** Exposes `net/http.Client` directly in port interface  

**Current Problematic Code:**
```go
type Conn interface {
    // HTTPClient returns an HTTP client configured for this authenticated connection.
    // The client will automatically include authentication credentials in requests.
    HTTPClient() (*http.Client, error) // ❌ Leaks net/http.Client
    
    Close() error
}
```

**Impact:** 
- Core domain is coupled to net/http package
- Violates dependency inversion principle
- Makes testing harder (requires mocking net/http.Client)
- Prevents using alternative HTTP implementations

### 2. Network Listener Infrastructure Leakage
**File:** `internal/core/ports/client.go`  
**Issue:** `AuthenticatedServer.Serve(ctx context.Context, lis net.Listener) error`  
**Problem:** Exposes `net.Listener` directly in port interface

**Current Problematic Code:**
```go
type AuthenticatedServer interface {
    // Serve starts serving requests on the provided listener.
    Serve(ctx context.Context, lis net.Listener) error // ❌ Leaks net.Listener
    
    // Addr returns the network address the server is listening on.
    Addr() net.Addr // ❌ Leaks net.Addr
    
    Close() error
}
```

**Impact:**
- Core domain is coupled to net package
- Server implementations must use specific network types
- Harder to test with mock listeners

### 3. Transport Connection Infrastructure Leakage
**File:** `internal/core/ports/transport.go`  
**Issue:** Multiple infrastructure types exposed  
**Problems:** 
- `ServerPort.Start(listener net.Listener) error` - Exposes `net.Listener`
- `ConnectionPort.AsNetConn() net.Conn` - Exposes `net.Conn`

**Current Problematic Code:**
```go
type ServerPort interface {
    RegisterService(serviceRegistrar ServiceRegistrarPort) error
    Start(listener net.Listener) error // ❌ Leaks net.Listener
    Stop() error
}

type ConnectionPort interface {
    GetClientConnection() interface{}
    AsNetConn() net.Conn // ❌ Leaks net.Conn
    Close() error
}
```

**Impact:**
- Transport abstractions are not properly abstracted
- Infrastructure concerns leak into core domain

## ✅ Proper Abstraction Examples

### Abstracted HTTP Client Interface
```go
// HTTPRequest represents an HTTP request abstraction that doesn't leak net/http.
type HTTPRequest struct {
    Method  string
    URL     string
    Headers map[string]string
    Body    io.Reader
}

// HTTPResponse represents an HTTP response abstraction that doesn't leak net/http.
type HTTPResponse struct {
    StatusCode int
    Headers    map[string]string
    Body       io.ReadCloser
}

// HTTPClient provides authenticated HTTP client capabilities without leaking net/http types.
type HTTPClient interface {
    // Do executes an HTTP request with authentication credentials automatically included.
    Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
    
    // Close releases resources held by the HTTP client.
    Close() error
}

// Properly abstracted connection interface
type Conn interface {
    // HTTPClient returns an HTTP client configured for this authenticated connection.
    HTTPClient() (HTTPClient, error) // ✅ Uses abstraction
    
    Close() error
}
```

### Abstracted Network Listener Interface
```go
// NetworkListener provides an abstraction for network listening without leaking net.Listener.
type NetworkListener interface {
    io.Closer
    // Accept waits for and returns the next connection
    Accept() (io.ReadWriteCloser, error)
    // Addr returns the listener's network address
    Addr() string
}

// Properly abstracted server interface
type AuthenticatedServer interface {
    // Serve starts serving requests on the provided listener abstraction.
    Serve(ctx context.Context, listener NetworkListener) error // ✅ Uses abstraction
    
    // Addr returns the network address the server is listening on.
    Addr() string // ✅ Uses string instead of net.Addr
    
    Close() error
}
```

### Abstracted Connection Interface
```go
type ConnectionPort interface {
    GetClientConnection() interface{}
    // AsReadWriteCloser safely converts the connection to io.ReadWriteCloser if possible
    AsReadWriteCloser() io.ReadWriteCloser // ✅ Uses standard library interface
    Close() error
}
```

## 🔧 Migration Strategy

### Phase 1: Non-Breaking Additions
1. Create new abstracted interfaces alongside existing ones
2. Add suffix like `V2` or `Abstracted` to new interfaces
3. Implement adapters that bridge between old and new interfaces

### Phase 2: Gradual Migration
1. Update adapters to implement both old and new interfaces
2. Migrate internal usage to new interfaces
3. Add deprecation warnings to old interfaces

### Phase 3: Breaking Change Migration
1. Remove old interfaces after all usage is migrated
2. Rename new interfaces to canonical names
3. Update all implementations and tests

## 🧪 Testing Benefits

Proper abstractions enable better testing:

```go
// With abstraction - easy to test
type mockHTTPClient struct{}
func (m *mockHTTPClient) Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
    return &HTTPResponse{StatusCode: 200}, nil
}

// With infrastructure leakage - harder to test
// Must mock entire net/http.Client with complex transport setup
```

## 📊 Impact Assessment

### Current Infrastructure Dependencies in Ports:
- ❌ `net/http` package (1 interface, 1 method)
- ❌ `net` package (3 interfaces, 4 methods)
- ❌ Direct type exposure instead of behavioral contracts

### After Proper Abstraction:
- ✅ Only `io` and `context` from standard library
- ✅ Behavioral contracts instead of concrete types
- ✅ Easy mocking and testing
- ✅ True hexagonal architecture compliance

## 🎯 Recommendation

**Priority: High** - These infrastructure leaks violate core architectural principles and should be addressed. The current leakage makes the core domain coupled to specific infrastructure implementations, reducing testability and flexibility.

**Suggested Approach:** Implement Phase 1 (non-breaking additions) first to provide proper abstractions, then gradually migrate existing code.