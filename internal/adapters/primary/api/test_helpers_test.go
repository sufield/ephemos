//go:build test

package api

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestService is a simple gRPC service for testing purposes.
// It provides real functionality rather than mocks.
type TestService struct {
	mu         sync.RWMutex
	callCount  int
	lastInput  string
	shouldFail bool
	failCode   codes.Code
}

// NewTestService creates a new test service with configurable behavior.
func NewTestService() *TestService {
	return &TestService{
		failCode: codes.Internal,
	}
}

// SetShouldFail configures the service to fail with the specified error code.
func (s *TestService) SetShouldFail(shouldFail bool, code codes.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shouldFail = shouldFail
	s.failCode = code
}

// GetCallCount returns the number of times the service was called.
func (s *TestService) GetCallCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.callCount
}

// GetLastInput returns the last input received by the service.
func (s *TestService) GetLastInput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastInput
}

// Reset clears the service state for fresh test runs.
func (s *TestService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount = 0
	s.lastInput = ""
	s.shouldFail = false
	s.failCode = codes.Internal
}

// TestMethod implements a simple test RPC method that honors context and preserves gRPC status codes.
func (s *TestService) TestMethod(ctx context.Context, req *TestRequest) (*TestResponse, error) {
	// Fast context check
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	s.mu.Lock()
	s.callCount++
	s.lastInput = req.GetInput()
	shouldFail, code := s.shouldFail, s.failCode
	s.mu.Unlock()

	if shouldFail {
		return nil, status.Errorf(code, "simulated failure")
	}

	return &TestResponse{Output: "processed: " + req.GetInput()}, nil
}

// TestServiceServer defines the interface for the test service
type TestServiceServer interface {
	TestMethod(context.Context, *TestRequest) (*TestResponse, error)
}

// Compile-time check: TestService implements TestServiceServer
var _ TestServiceServer = (*TestService)(nil)

// Service descriptor and handler for TestService.TestMethod.
var _TestService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.TestService",
	HandlerType: (*TestServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "TestMethod",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(TestRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(TestServiceServer).TestMethod(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/api.TestService/TestMethod",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(TestServiceServer).TestMethod(ctx, req.(*TestRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
}

// TestServiceRegistrar is a real implementation of ServiceRegistrar for testing.
// It registers a real gRPC service using grpc.ServiceRegistrar interface.
type TestServiceRegistrar struct {
	service       *TestService
	registered    bool
	registerCount int
	mu            sync.Mutex
}

// NewTestServiceRegistrar creates a new test service registrar.
func NewTestServiceRegistrar(service *TestService) *TestServiceRegistrar {
	if service == nil {
		service = NewTestService()
	}
	return &TestServiceRegistrar{
		service: service,
	}
}

// Register registers the test service with any grpc.ServiceRegistrar.
func (r *TestServiceRegistrar) Register(s grpc.ServiceRegistrar) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s == nil {
		return
	}
	s.RegisterService(&_TestService_serviceDesc, r.service)
	r.registered = true
	r.registerCount++
}

// IsRegistered returns whether the service has been registered.
func (r *TestServiceRegistrar) IsRegistered() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registered
}

// GetRegisterCount returns how many times Register was called.
func (r *TestServiceRegistrar) GetRegisterCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerCount
}

// Reset clears the registrar state for fresh test runs.
func (r *TestServiceRegistrar) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registered = false
	r.registerCount = 0
}

// GetService returns the underlying test service.
func (r *TestServiceRegistrar) GetService() *TestService {
	return r.service
}

// TestRequest represents a test request message for gRPC testing.
type TestRequest struct {
	Input string
}

// GetInput returns the input field of the test request.
func (r *TestRequest) GetInput() string {
	if r != nil {
		return r.Input
	}
	return ""
}

// TestResponse represents a test response message for gRPC testing.
type TestResponse struct {
	Output string
}

// GetOutput returns the output field of the test response.
func (r *TestResponse) GetOutput() string {
	if r != nil {
		return r.Output
	}
	return ""
}
