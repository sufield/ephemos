# Testing Without Mocks in Ephemos

This document explains our approach to testing without mocks, following Go best practices for creating maintainable, robust tests.

## Philosophy

Instead of using mocking frameworks that generate brittle tests tied to specific method signatures, we use real implementations with configurable behavior. This approach:

1. **Tests real code paths** - Increases confidence in system behavior
2. **Reduces maintenance** - No mock expectations to update when signatures change
3. **Improves readability** - Test setup is explicit and clear
4. **Follows Go idioms** - Leverages interfaces and dependency injection

## Strategies

### 1. In-Memory Implementations

For configuration providers and data stores, we use in-memory implementations that behave like real components:

```go
// InMemoryProvider stores configurations in memory
type InMemoryProvider struct {
    configs map[string]*Configuration
}

// Real implementation that can be configured for tests
func (p *InMemoryProvider) LoadConfiguration(path string) (*Configuration, error) {
    config, ok := p.configs[path]
    if !ok {
        return nil, fmt.Errorf("configuration not found")
    }
    return config, nil
}
```

**Example**: See `internal/adapters/secondary/config/inmemory_provider.go`

### 2. Test Services with Configurable Behavior

Instead of mocks, we create real test services that can simulate different scenarios:

```go
// TestEchoServer is a real implementation for testing
type TestEchoServer struct {
    callCount   int
    lastMessage string
    shouldFail  bool
}

func (s *TestEchoServer) Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
    s.callCount++
    s.lastMessage = req.Message
    
    // Simulate different behaviors based on input
    if req.Message == "error" {
        return nil, status.Error(codes.Internal, "simulated error")
    }
    
    return &EchoResponse{Message: req.Message}, nil
}
```


### 3. Registration Trackers

For testing service registration, we use trackers that monitor real behavior:

```go
// TestRegistrationTracker tracks registrations
type TestRegistrationTracker struct {
    registered    bool
    registerCount int
}

func (t *TestRegistrationTracker) RegisterFunction() func(*grpc.Server) {
    return func(server *grpc.Server) {
        t.registered = true
        t.registerCount++
    }
}
```

**Example**: See `pkg/ephemos/ephemos_test.go`

### 4. Real gRPC Servers with bufconn

For testing gRPC services, we use real servers with in-memory connections:

```go
func TestWithRealServer(t *testing.T) {
    // Create in-memory listener
    lis := bufconn.Listen(1024 * 1024)
    
    // Start real gRPC server
    s := grpc.NewServer()
    RegisterEchoServiceServer(s, &TestEchoServer{})
    go s.Serve(lis)
    
    // Create real client connection
    conn, _ := grpc.DialContext(ctx, "bufnet", 
        grpc.WithContextDialer(bufDialer),
        grpc.WithInsecure())
    
    // Test with real client
    client := NewEchoServiceClient(conn)
    resp, err := client.Echo(ctx, &EchoRequest{Message: "test"})
}
```

### 5. Error Simulation

We simulate errors through input parameters rather than mock expectations:

```go
// Configure behavior through input
if req.Message == "timeout" {
    time.Sleep(2 * time.Second)
    return nil, status.Error(codes.DeadlineExceeded, "simulated timeout")
}

if req.Message == "not-found" {
    return nil, status.Error(codes.NotFound, "simulated not found")
}
```

## Benefits

### 1. Maintainability
- No mock setup code to maintain
- Tests don't break when internal implementations change
- Clear, explicit test behavior

### 2. Confidence
- Tests exercise real code paths
- Catch integration issues early
- Behavior is predictable and debuggable

### 3. Simplicity
- No mocking framework to learn
- Standard Go code throughout
- Easy to understand test failures

## Migration Guide

When refactoring tests from mocks to real implementations:

### Step 1: Identify Mock Usage
```bash
grep -r "mock\|Mock\|stub\|Stub" **/*_test.go
```

### Step 2: Create Real Implementation
Replace mock with a configurable test implementation:

**Before (Mock):**
```go
type MockService struct {
    mock.Mock
}

func (m *MockService) DoSomething(input string) error {
    args := m.Called(input)
    return args.Error(0)
}

// Test setup
mockService := new(MockService)
mockService.On("DoSomething", "test").Return(nil)
```

**After (Real Implementation):**
```go
type TestService struct {
    callCount int
    lastInput string
    errorToReturn error
}

func (s *TestService) DoSomething(input string) error {
    s.callCount++
    s.lastInput = input
    
    if s.errorToReturn != nil {
        return s.errorToReturn
    }
    
    // Simulate behavior based on input
    if input == "error" {
        return errors.New("simulated error")
    }
    
    return nil
}

// Test setup
testService := &TestService{
    errorToReturn: nil, // or errors.New("test error") for error cases
}
```

### Step 3: Update Test Assertions

**Before:**
```go
mockService.AssertExpectations(t)
mockService.AssertCalled(t, "DoSomething", "test")
```

**After:**
```go
if testService.callCount != 1 {
    t.Errorf("expected 1 call, got %d", testService.callCount)
}
if testService.lastInput != "test" {
    t.Errorf("expected input 'test', got '%s'", testService.lastInput)
}
```

## Common Patterns

### Pattern 1: Configuration Provider
Use `InMemoryProvider` with pre-loaded configurations for different test scenarios.

### Pattern 2: Service with States
Create test services that track calls and can be configured to return different responses.

### Pattern 3: Error Injection
Use special input values (e.g., "error", "timeout") to trigger error conditions.

### Pattern 4: Integration Tests
Use real components with test configurations (e.g., in-memory databases, test servers).

## Best Practices

1. **Keep test implementations simple** - They should be obviously correct
2. **Use table-driven tests** - Define scenarios clearly with inputs and expected outputs
3. **Leverage interfaces** - Small, focused interfaces are easier to implement for testing
4. **Avoid over-abstraction** - Sometimes using the real component is simpler
5. **Document behavior** - Make it clear what each test implementation does

## Examples in Codebase

- **Configuration Testing**: `internal/adapters/primary/cli/registrar_test.go`
- **Registration Testing**: `pkg/ephemos/ephemos_test.go`
- **API Testing**: `internal/adapters/primary/api/server_test.go`

## Conclusion

By avoiding mocks and using real implementations with configurable behavior, we create tests that are:
- More maintainable
- More reliable
- Easier to understand
- More likely to catch real bugs

This approach aligns with Go's philosophy of simplicity and explicit behavior, resulting in a more robust test suite.