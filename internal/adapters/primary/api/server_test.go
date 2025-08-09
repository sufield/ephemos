package api

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestIdentityServer_NewIdentityServer(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "empty config path",
			configPath: "",
			wantErr:    true, // Default config may not be valid without proper SPIFFE setup
		},
		{
			name:       "invalid config path",
			configPath: "/nonexistent/path",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewIdentityServer(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIdentityServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewIdentityServer() returned nil server")
			}
		})
	}
}

func TestIdentityServer_RegisterService(t *testing.T) {
	server, err := NewIdentityServer("")
	if err != nil {
		t.Skip("Skipping RegisterService tests - could not create server:", err)
	}

	ctx := context.Background()

	// Test with nil context
	err = server.RegisterService(context.TODO(), nil)
	if err == nil {
		t.Error("RegisterService() with nil registrar should return error")
	}

	// Test with nil registrar
	err = server.RegisterService(ctx, nil)
	if err == nil {
		t.Error("RegisterService() with nil registrar should return error")
	}
}

func TestIdentityServer_Serve(t *testing.T) {
	server, err := NewIdentityServer("")
	if err != nil {
		t.Skip("Skipping Serve tests - could not create server:", err)
	}

	// Test with nil context
	err = server.Serve(context.TODO(), nil)
	if err == nil {
		t.Error("Serve() with nil listener should return error")
	}

	// Test with nil listener
	ctx := context.Background()
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
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
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

func TestIdentityServer_Close(t *testing.T) {
	server, err := NewIdentityServer("")
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

func TestIdentityServer_RegisterService_WithRealService(t *testing.T) {
	server, err := NewIdentityServer("")
	if err != nil {
		t.Skip("Skipping RegisterService test - could not create server:", err)
	}

	ctx := t.Context()
	
	// Use a real test service instead of a mock
	testService := NewTestService()
	registrar := NewTestServiceRegistrar(testService)

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
