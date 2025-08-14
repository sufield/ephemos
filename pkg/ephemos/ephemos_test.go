package ephemos_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/sufield/ephemos/pkg/ephemos"
)

// TestRegistrationTracker tracks registrations for testing purposes.
// This is a real implementation, not a mock.
type TestRegistrationTracker struct {
	mu            sync.Mutex
	registered    bool
	registerCount int
	lastServer    interface{}
}

// NewTestRegistrationTracker creates a new registration tracker.
func NewTestRegistrationTracker() *TestRegistrationTracker {
	return &TestRegistrationTracker{}
}

// RegisterFunction returns a function that can be used with ephemos.NewServiceRegistrar.
func (t *TestRegistrationTracker) RegisterFunction() func(interface{}) {
	return func(server interface{}) {
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
		registerFunc func(interface{})
		expectNil    bool
	}{
		{
			name:         "valid registration function",
			registerFunc: func(s interface{}) { tracker.RegisterFunction()(s) },
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
		setupFunc  func() (*TestRegistrationTracker, func(interface{}))
		shouldCall bool
	}{
		{
			name: "calls registration function",
			setupFunc: func() (*TestRegistrationTracker, func(interface{})) {
				tracker := NewTestRegistrationTracker()
				fn := func(s interface{}) { tracker.RegisterFunction()(s) }
				return tracker, fn
			},
			shouldCall: true,
		},
		{
			name: "handles nil registration function gracefully",
			setupFunc: func() (*TestRegistrationTracker, func(interface{})) {
				return NewTestRegistrationTracker(), nil
			},
			shouldCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker, registerFunc := tt.setupFunc()

			registrar := ephemos.NewServiceRegistrar(registerFunc)
			mockServer := "mock-transport-server"

			// Should not panic
			registrar.Register(mockServer)

			if tracker.IsRegistered() != tt.shouldCall {
				t.Errorf("Register() called registration function = %v, want %v", tracker.IsRegistered(), tt.shouldCall)
			}
		})
	}
}

func TestGenericServiceRegistrar_Integration(t *testing.T) {
	// Integration test showing real usage pattern
	mockServer := "mock-transport-server"

	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(func(s interface{}) { tracker.RegisterFunction()(s) })

	registrar.Register(mockServer)

	if !tracker.IsRegistered() {
		t.Error("Registration function was not called during integration test")
	}

	if tracker.GetRegisterCount() != 1 {
		t.Errorf("Expected registration count to be 1, got %d", tracker.GetRegisterCount())
	}
}

func TestGenericServiceRegistrar_MultipleRegistrations(t *testing.T) {
	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(func(s interface{}) { tracker.RegisterFunction()(s) })

	// Register with multiple servers
	for i := 0; i < 3; i++ {
		mockServer := fmt.Sprintf("mock-server-%d", i)
		registrar.Register(mockServer)
	}

	if tracker.GetRegisterCount() != 3 {
		t.Errorf("Expected 3 registrations, got %d", tracker.GetRegisterCount())
	}
}

func BenchmarkNewServiceRegistrar(b *testing.B) {
	tracker := NewTestRegistrationTracker()
	registerFunc := func(s interface{}) { tracker.RegisterFunction()(s) }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ephemos.NewServiceRegistrar(registerFunc)
	}
}

func BenchmarkGenericServiceRegistrar_Register(b *testing.B) {
	tracker := NewTestRegistrationTracker()
	registrar := ephemos.NewServiceRegistrar(func(s interface{}) { tracker.RegisterFunction()(s) })
	mockServer := "benchmark-server"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registrar.Register(mockServer)
	}
}

func ExampleNewServiceRegistrar() {
	// Example showing how to use the generic registrar
	// (This would normally be in your main application code)

	// Create your service implementation
	// serviceImpl := &MyServiceImpl{}

	// Register using the generic registrar - no boilerplate needed!
	registrar := ephemos.NewServiceRegistrar(func(s interface{}) {
		// In a real implementation, you would register your transport service here:
		// For HTTP: httpServer.RegisterHandlers(serviceImpl)
		// For other transports: register according to transport requirements
	})

	// Use with Ephemos server
	// server.RegisterService(ctx, registrar)
	_ = registrar // Avoid unused variable in example

	// Output:
}
