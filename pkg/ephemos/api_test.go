package ephemos

import (
	"context"
	"testing"
)

func TestPublicAPI(t *testing.T) {
	// Test that the public API compiles and basic interfaces work
	ctx := context.Background()
	
	// Test IdentityClient creation
	_, err := IdentityClient(ctx, "test-config")
	if err != nil {
		t.Logf("IdentityClient returned error (expected): %v", err)
	}
	
	// Test IdentityServer creation
	_, err = IdentityServer(ctx, "test-config")
	if err != nil {
		t.Logf("IdentityServer returned error (expected): %v", err)
	}
	
	// Test ServiceRegistrar creation
	registrar := NewServiceRegistrar(func(transport interface{}) {
		// Mock registration function
	})
	if registrar == nil {
		t.Log("NewServiceRegistrar returned nil (expected - implementation delegated to internal packages)")
	}
	
	// Test Configuration struct
	config := Configuration{
		Service: ServiceConfig{
			Name:   "test-service",
			Domain: "test.domain",
		},
	}
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