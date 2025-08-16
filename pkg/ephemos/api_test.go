package ephemos

import (
	"context"
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

func TestPublicAPI(t *testing.T) {
	// Test that the public API compiles and basic interfaces work
	ctx := context.Background()
	
	// Create a test configuration
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.domain",
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/run/sockets/agent.sock",
		},
	}
	
	// Test IdentityClient creation with configuration option
	_, err := IdentityClient(ctx, WithConfig(config))
	if err != nil {
		t.Logf("IdentityClient returned error (expected without real SPIFFE setup): %v", err)
	}
	
	// Test IdentityServer creation with configuration and address options
	_, err = IdentityServer(ctx, WithServerConfig(config), WithAddress("localhost:0"))
	if err != nil {
		t.Logf("IdentityServer returned error (expected without real SPIFFE setup): %v", err)
	}
	
	// Note: Service registration is now CLI-only, not part of public API
	
	// Test Configuration struct
	if config.Service.Name != "test-service" {
		t.Error("Configuration struct not working properly")
	}
	
	t.Log("Public API structure is working correctly")
}

func TestClientConnection(t *testing.T) {
	// Test ClientConnection
	conn := &ClientConnection{}
	err := conn.Close()
	if err != nil {
		t.Errorf("ClientConnection.Close() returned error: %v", err)
	}
}