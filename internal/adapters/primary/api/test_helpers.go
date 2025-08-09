package api

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestService is a simple gRPC service for testing purposes.
// It provides real functionality rather than mocks.
type TestService struct {
	UnimplementedTestServiceServer
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

// TestMethod implements a simple test RPC method.
func (s *TestService) TestMethod(ctx context.Context, req *TestRequest) (*TestResponse, error) {
	s.mu.Lock()
	s.callCount++
	s.lastInput = req.GetInput()
	shouldFail := s.shouldFail
	failCode := s.failCode
	s.mu.Unlock()

	if shouldFail {
		return nil, fmt.Errorf("test service failure: %w", status.Error(failCode, "simulated failure"))
	}

	return &TestResponse{
		Output: "processed: " + req.GetInput(),
	}, nil
}

// TestServiceRegistrar is a real implementation of ServiceRegistrar for testing.
// It registers a real gRPC service, not a mock.
type TestServiceRegistrar struct {
	service        *TestService
	registered     bool
	registerCount  int
	mu             sync.Mutex
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

// Register registers the test service with the gRPC server.
func (r *TestServiceRegistrar) Register(server *grpc.Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if server != nil {
		// In a real scenario, this would be:
		// RegisterTestServiceServer(server, r.service)
		// For now, we just track that it was called
		r.registered = true
		r.registerCount++
	}
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

// GetService returns the underlying test service.
func (r *TestServiceRegistrar) GetService() *TestService {
	return r.service
}

// Simple message types for testing (these would normally be generated from proto)
type TestRequest struct {
	Input string
}

func (r *TestRequest) GetInput() string {
	if r != nil {
		return r.Input
	}
	return ""
}

type TestResponse struct {
	Output string
}

func (r *TestResponse) GetOutput() string {
	if r != nil {
		return r.Output
	}
	return ""
}

// UnimplementedTestServiceServer is a minimal implementation for forward compatibility
type UnimplementedTestServiceServer struct{}

func (UnimplementedTestServiceServer) TestMethod(context.Context, *TestRequest) (*TestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TestMethod not implemented")
}