package ephemos

import (
	"testing"

	"google.golang.org/grpc"
)

// Mock service interface for testing
type MockService interface {
	TestMethod() string
}

// Mock service implementation
type mockServiceImpl struct{}

func (m *mockServiceImpl) TestMethod() string {
	return "test"
}

// Mock registration function
var mockRegistered = false

func mockRegisterFunction(server *grpc.Server) {
	mockRegistered = true
}

func TestNewServiceRegistrar(t *testing.T) {
	tests := []struct {
		name         string
		registerFunc func(*grpc.Server)
		expectNil    bool
	}{
		{
			name:         "valid registration function",
			registerFunc: mockRegisterFunction,
			expectNil:    false,
		},
		{
			name:         "nil registration function",
			registerFunc: nil,
			expectNil:    false, // Should still create registrar, but won't register anything
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := NewServiceRegistrar(tt.registerFunc)

			if (registrar == nil) != tt.expectNil {
				t.Errorf("NewServiceRegistrar() = %v, expectNil = %v", registrar == nil, tt.expectNil)
			}

			// Test that it implements ServiceRegistrar interface
			if _, ok := registrar.(ServiceRegistrar); !ok {
				t.Error("NewServiceRegistrar() result does not implement ServiceRegistrar interface")
			}
		})
	}
}

func TestGenericServiceRegistrar_Register(t *testing.T) {
	tests := []struct {
		name         string
		registerFunc func(*grpc.Server)
		shouldCall   bool
	}{
		{
			name: "calls registration function",
			registerFunc: func(server *grpc.Server) {
				mockRegistered = true
			},
			shouldCall: true,
		},
		{
			name:         "handles nil registration function gracefully",
			registerFunc: nil,
			shouldCall:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockRegistered = false

			registrar := NewServiceRegistrar(tt.registerFunc)
			grpcServer := grpc.NewServer()
			defer grpcServer.Stop()

			// Should not panic
			registrar.Register(grpcServer)

			if mockRegistered != tt.shouldCall {
				t.Errorf("Register() called registration function = %v, want %v", mockRegistered, tt.shouldCall)
			}
		})
	}
}

func TestGenericServiceRegistrar_Integration(t *testing.T) {
	// Integration test showing real usage pattern
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	called := false
	registrar := NewServiceRegistrar(func(s *grpc.Server) {
		called = true
		// In real usage, this would be something like:
		// proto.RegisterYourServiceServer(s, &YourServiceImpl{})
	})

	registrar.Register(grpcServer)

	if !called {
		t.Error("Registration function was not called during integration test")
	}
}

func ExampleNewServiceRegistrar() {
	// Example showing how to use the generic registrar
	// (This would normally be in your main application code)
	
	// Create your service implementation
	// serviceImpl := &MyServiceImpl{}

	// Register using the generic registrar - no boilerplate needed!
	registrar := NewServiceRegistrar(func(s *grpc.Server) {
		// proto.RegisterMyServiceServer(s, serviceImpl)
	})

	// Use with Ephemos server
	// server.RegisterService(ctx, registrar)
	_ = registrar // Avoid unused variable in example
}