package ephemos_test

import (
	"sync"
	"testing"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/pkg/ephemos"
)

// TestRegistrationTracker tracks registrations for testing purposes.
// This is a real implementation, not a mock.
type TestRegistrationTracker struct {
	mu            sync.Mutex
	registered    bool
	registerCount int
	lastServer    *grpc.Server
}

// NewTestRegistrationTracker creates a new registration tracker.
func NewTestRegistrationTracker() *TestRegistrationTracker {
	return &TestRegistrationTracker{}
}

// RegisterFunction returns a function that can be used with ephemos.NewServiceRegistrar.
func (t *TestRegistrationTracker) RegisterFunction() func(*grpc.Server) {
	return func(server *grpc.Server) {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.registered = true
		t.registerCount++
		t.lastServer = server
	}
}

// IsRegistered returns whether registration occurred.
func (t *TestRegistrationTracker) IsRegistered() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.registered
}

// GetRegisterCount returns the number of times registration occurred.
func (t *TestRegistrationTracker) GetRegisterCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.registerCount
}

// Reset resets the tracker state.
func (t *TestRegistrationTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.registered = false
	t.registerCount = 0
	t.lastServer = nil
}

func TestNewServiceRegistrar(t *testing.T) {
	tracker := NewTestRegistrationTracker()

	tests := []struct {
		name         string
		registerFunc func(*grpc.Server)
		expectNil    bool
	}{
		{
			name:         "valid registration function",
			registerFunc: tracker.RegisterFunction(),
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
			registrar := ephemos.NewServiceRegistrar(tt.registerFunc)

			if (registrar == nil) != tt.expectNil {
				t.Errorf("ephemos.NewServiceRegistrar() = %v, expectNil = %v", registrar == nil, tt.expectNil)
			}

			// registrar is already of type ephemos.ServiceRegistrar, so interface is implemented by design
		})
	}
}

func TestGenericServiceRegistrar_Register(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() (*TestRegistrationTracker, func(*grpc.Server))
		shouldCall bool
	}{
		{
			name: "calls registration function",
			setupFunc: func() (*TestRegistrationTracker, func(*grpc.Server)) {
				tracker := NewTestRegistrationTracker()
				fn := tracker.RegisterFunction()
				return tracker, fn
			},
			shouldCall: true,
		},
		{
			name: "handles nil registration function gracefully",
			setupFunc: func() (*TestRegistrationTracker, func(*grpc.Server)) {
				return NewTestRegistrationTracker(), nil
			},
			shouldCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker, registerFunc := tt.setupFunc()

			registrar := ephemos.NewServiceRegistrar(registerFunc)
			grpcServer := grpc.NewServer()
			defer grpcServer.Stop()

			// Should not panic
			registrar.Register(grpcServer)

			if tracker.IsRegistered() != tt.shouldCall {
				t.Errorf("Register() called registration function = %v, want %v", tracker.IsRegistered(), tt.shouldCall)
			}
		})
	}
}

func TestGenericServiceRegistrar_Integration(t *testing.T) {
	// Integration test showing real usage pattern
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(tracker.RegisterFunction())

	registrar.Register(grpcServer)

	if !tracker.IsRegistered() {
		t.Error("Registration function was not called during integration test")
	}

	if tracker.GetRegisterCount() != 1 {
		t.Errorf("Expected registration count to be 1, got %d", tracker.GetRegisterCount())
	}
}

func TestGenericServiceRegistrar_MultipleRegistrations(t *testing.T) {
	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(tracker.RegisterFunction())

	// Register with multiple servers
	for i := 0; i < 3; i++ {
		server := grpc.NewServer()
		registrar.Register(server)
		server.Stop()
	}

	if tracker.GetRegisterCount() != 3 {
		t.Errorf("Expected 3 registrations, got %d", tracker.GetRegisterCount())
	}
}

func BenchmarkNewServiceRegistrar(b *testing.B) {
	tracker := NewTestRegistrationTracker()
	registerFunc := tracker.RegisterFunction()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ephemos.NewServiceRegistrar(registerFunc)
	}
}

func BenchmarkGenericServiceRegistrar_Register(b *testing.B) {
	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(tracker.RegisterFunction())
	server := grpc.NewServer()
	defer server.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registrar.Register(server)
	}
}

func ExampleNewServiceRegistrar() {
	// Example showing how to use the generic registrar
	// (This would normally be in your main application code)

	// Create your service implementation
	// serviceImpl := &MyServiceImpl{}

	// Register using the generic registrar - no boilerplate needed!
	registrar := ephemos.NewServiceRegistrar(func(_ *grpc.Server) {
		// In a real implementation, you would register your gRPC service here:
		// proto.RegisterMyServiceServer(s, serviceImpl)
	})

	// Use with Ephemos server
	// server.RegisterService(ctx, registrar)
	_ = registrar // Avoid unused variable in example

	// Output:
}
