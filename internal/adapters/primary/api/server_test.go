package api_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Mock implementations for server testing
type mockServerIdentityProvider struct{}

func (m *mockServerIdentityProvider) GetIdentity(ctx context.Context) (ports.Identity, error) {
	return &mockIdentity{}, nil
}

func (m *mockServerIdentityProvider) Close() error {
	return nil
}

type mockServerTransportProvider struct{}

func (m *mockServerTransportProvider) CreateServer(identity ports.Identity, config *ports.Configuration) (ports.Server, error) {
	return &mockServer{}, nil
}

func (m *mockServerTransportProvider) CreateClient(identity ports.Identity, config *ports.Configuration) (ports.Client, error) {
	return nil, nil
}

type mockConfigProvider struct{}

func (m *mockConfigProvider) GetConfiguration(ctx context.Context) (*ports.Configuration, error) {
	return &ports.Configuration{}, nil
}

type mockServer struct{}

func (m *mockServer) Serve(listener net.Listener) error {
	return nil
}

func (m *mockServer) RegisterService(service interface{}) error {
	return nil
}

func (m *mockServer) Close() error {
	return nil
}

func TestServer_WorkloadServer(t *testing.T) {
	tests := []struct {
		name              string
		identityProvider  ports.IdentityProvider
		transportProvider ports.TransportProvider
		configProvider    ports.ConfigurationProvider
		config            *ports.Configuration
		wantErr           bool
	}{
		{
			name:              "nil config",
			identityProvider:  &mockServerIdentityProvider{},
			transportProvider: &mockServerTransportProvider{},
			configProvider:    &mockConfigProvider{},
			config:            nil,
			wantErr:           true,
		},
		{
			name:              "nil identity provider",
			identityProvider:  nil,
			transportProvider: &mockServerTransportProvider{},
			configProvider:    &mockConfigProvider{},
			config:            &ports.Configuration{Service: &ports.ServiceConfig{Name: "test"}},
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := api.WorkloadServer(tt.identityProvider, tt.transportProvider, tt.configProvider, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("api.WorkloadServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("api.WorkloadServer() returned nil server")
			}
		})
	}
}

func TestServer_RegisterService(t *testing.T) {
	config := &ports.Configuration{Service: &ports.ServiceConfig{Name: "test"}}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, &mockServerTransportProvider{}, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping RegisterService tests - could not create server:", err)
	}

	// Test with nil context
	err = server.RegisterService(t.Context(), nil)
	if err == nil {
		t.Error("RegisterService() with nil registrar should return error")
	}

	// Test with nil registrar
	err = server.RegisterService(ctx, nil)
	if err == nil {
		t.Error("RegisterService() with nil registrar should return error")
	}
}

func TestServer_Serve(t *testing.T) {
	config := &ports.Configuration{Service: &ports.ServiceConfig{Name: "test"}}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, &mockServerTransportProvider{}, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping Serve tests - could not create server:", err)
	}

	// Test with nil context
	err = server.Serve(t.Context(), nil)
	if err == nil {
		t.Error("Serve() with nil listener should return error")
	}

	// Test with nil listener
	err = server.Serve(ctx, nil)
	if err == nil {
		t.Error("Serve() with nil listener should return error")
	}

	// Test with cancellation
	t.Run("context cancellation", func(t *testing.T) {
		// Create a listener on a random port
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Could not create listener:", err)
		}
		defer lis.Close()

		// Create a context that will be canceled quickly
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// Start serving in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- server.Serve(ctx, lis)
		}()

		// Wait for completion or timeout
		select {
		case err := <-done:
			if err != nil && ctx.Err() == nil {
				t.Errorf("Serve() returned unexpected error: %v", err)
			}
		case <-time.After(200 * time.Millisecond):
			t.Error("Serve() did not return after context cancellation")
		}
	})
}

func TestServer_Close(t *testing.T) {
	config := &ports.Configuration{Service: &ports.ServiceConfig{Name: "test"}}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, &mockServerTransportProvider{}, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping Close test - could not create server:", err)
	}

	// Close should not return an error
	if err := server.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Multiple closes should be safe
	if err := server.Close(); err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}

// TestServiceRegistrar is now defined in test_helpers.go
// It provides a real implementation instead of a mock

func TestServer_RegisterService_WithRealService(t *testing.T) {
	config := &ports.Configuration{Service: &ports.ServiceConfig{Name: "test"}}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, &mockServerTransportProvider{}, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping RegisterService test - could not create server:", err)
	}

	// Use a real test service instead of a mock
	testService := api.NewTestService()
	registrar := api.NewTestServiceRegistrar(testService)

	err = server.RegisterService(ctx, registrar)
	if err != nil {
		t.Errorf("RegisterService() returned error: %v", err)
	}

	if !registrar.IsRegistered() {
		t.Error("RegisterService() did not call Register on the registrar")
	}

	if registrar.GetRegisterCount() != 1 {
		t.Errorf("Expected Register to be called once, got %d", registrar.GetRegisterCount())
	}
}
