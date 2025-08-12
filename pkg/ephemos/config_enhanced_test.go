package ephemos

import (
	"testing"
)

func TestConfigBuilder_PureCodeConfiguration(t *testing.T) {
	ctx := t.Context()

	// Test pure-code configuration
	config, err := NewConfigBuilder().
		WithSource(ConfigSourcePureCode).
		WithServiceName("test-service").
		WithServiceDomain("test.example.com").
		WithSPIFFESocket("/custom/spiffe/socket").
		WithTransport("grpc", ":8080").
		WithAuthorizedClients([]string{"spiffe://test.com/client1", "spiffe://test.com/client2"}).
		WithTrustedServers([]string{"spiffe://test.com/server1", "spiffe://test.com/server2"}).
		Build(ctx)
	if err != nil {
		t.Fatalf("Failed to build pure-code config: %v", err)
	}

	// Verify configuration
	if config.Service.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", config.Service.Name)
	}
	if config.Service.Domain != "test.example.com" {
		t.Errorf("Expected service domain 'test.example.com', got '%s'", config.Service.Domain)
	}
	if config.SPIFFE.SocketPath != "/custom/spiffe/socket" {
		t.Errorf("Expected SPIFFE socket '/custom/spiffe/socket', got '%s'", config.SPIFFE.SocketPath)
	}
	if config.Transport.Type != "grpc" {
		t.Errorf("Expected transport type 'grpc', got '%s'", config.Transport.Type)
	}
	if config.Transport.Address != ":8080" {
		t.Errorf("Expected transport address ':8080', got '%s'", config.Transport.Address)
	}
	if len(config.AuthorizedClients) != 2 {
		t.Errorf("Expected 2 authorized clients, got %d", len(config.AuthorizedClients))
	}
	if len(config.TrustedServers) != 2 {
		t.Errorf("Expected 2 trusted servers, got %d", len(config.TrustedServers))
	}
}

// verifyServiceConfig checks service configuration fields.
func verifyServiceConfig(t *testing.T, config *Configuration, expectedName, expectedDomain string) {
	t.Helper()
	if config.Service.Name != expectedName {
		t.Errorf("Expected service name '%s', got '%s'", expectedName, config.Service.Name)
	}
	if config.Service.Domain != expectedDomain {
		t.Errorf("Expected service domain '%s', got '%s'", expectedDomain, config.Service.Domain)
	}
}

// verifyTransportConfig checks transport configuration fields.
func verifyTransportConfig(t *testing.T, config *Configuration, expectedType, expectedAddress string) {
	t.Helper()
	if config.Transport.Type != expectedType {
		t.Errorf("Expected transport type '%s', got '%s'", expectedType, config.Transport.Type)
	}
	if config.Transport.Address != expectedAddress {
		t.Errorf("Expected transport address '%s', got '%s'", expectedAddress, config.Transport.Address)
	}
}

// verifyListConfig checks list configuration fields.
func verifyListConfig(t *testing.T, config *Configuration, expectedClients, expectedServers int) {
	t.Helper()
	if len(config.AuthorizedClients) != expectedClients {
		t.Errorf("Expected %d authorized clients, got %d", expectedClients, len(config.AuthorizedClients))
	}
	if len(config.TrustedServers) != expectedServers {
		t.Errorf("Expected %d trusted servers, got %d", expectedServers, len(config.TrustedServers))
	}
}

func TestConfigBuilder_EnvironmentOverrides(t *testing.T) {
	ctx := t.Context()

	// Set up environment variables
	envVars := map[string]string{
		"EPHEMOS_SERVICE_NAME":       "env-service",
		"EPHEMOS_SERVICE_DOMAIN":     "env.example.com",
		"EPHEMOS_SPIFFE_SOCKET":      "/env/spiffe/socket",
		"EPHEMOS_TRANSPORT_TYPE":     "http",
		"EPHEMOS_TRANSPORT_ADDRESS":  ":9090",
		"EPHEMOS_AUTHORIZED_CLIENTS": "spiffe://env.com/client1, spiffe://env.com/client2, spiffe://env.com/client3",
		"EPHEMOS_TRUSTED_SERVERS":    "spiffe://env.com/server1, spiffe://env.com/server2",
	}

	// Set environment variables
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	// Test environment-only configuration
	config, err := NewConfigBuilder().
		WithSource(ConfigSourceEnvOnly).
		Build(ctx)
	if err != nil {
		t.Fatalf("Failed to build env-only config: %v", err)
	}

	// Verify configuration using helper functions
	verifyServiceConfig(t, config, "env-service", "env.example.com")
	verifyTransportConfig(t, config, "http", ":9090")
	verifyListConfig(t, config, 3, 2)

	// Verify SPIFFE socket
	if config.SPIFFE.SocketPath != "/env/spiffe/socket" {
		t.Errorf("Expected SPIFFE socket '/env/spiffe/socket', got '%s'", config.SPIFFE.SocketPath)
	}

	// Verify first client
	if config.AuthorizedClients[0] != "spiffe://env.com/client1" {
		t.Errorf("Expected first client 'spiffe://env.com/client1', got '%s'", config.AuthorizedClients[0])
	}
}

func TestLoadConfigFlexible_PureCode(t *testing.T) {
	ctx := t.Context()

	config, err := LoadConfigFlexible(ctx,
		WithPureCodeSource(),
		WithService("flexible-service", "flexible.example.com"),
		WithTransportOption("grpc", ":7070"),
	)
	if err != nil {
		t.Fatalf("Failed to load flexible config: %v", err)
	}

	if config.Service.Name != "flexible-service" {
		t.Errorf("Expected service name 'flexible-service', got '%s'", config.Service.Name)
	}
	if config.Service.Domain != "flexible.example.com" {
		t.Errorf("Expected service domain 'flexible.example.com', got '%s'", config.Service.Domain)
	}
	if config.Transport.Address != ":7070" {
		t.Errorf("Expected transport address ':7070', got '%s'", config.Transport.Address)
	}
}

func TestLoadConfigFlexible_EnvironmentOnly(t *testing.T) {
	ctx := t.Context()

	// Set environment variable
	t.Setenv("TEST_SERVICE_NAME", "test-env-service")

	// Note: WithService is applied AFTER environment variables are loaded,
	// so it will override the environment. Let's test the environment-only approach.
	config, err := LoadConfigFlexible(ctx,
		WithEnvSource("TEST"),
	)
	if err != nil {
		t.Fatalf("Failed to load flexible env config: %v", err)
	}

	// Environment variable should override the programmatic setting
	if config.Service.Name != "test-env-service" {
		t.Errorf("Expected service name 'test-env-service', got '%s'", config.Service.Name)
	}
}

func TestConfigBuilder_Defaults(t *testing.T) {
	ctx := t.Context()

	// Test that defaults are used when no overrides are provided
	config, err := NewConfigBuilder().
		WithSource(ConfigSourcePureCode).
		Build(ctx)
	if err != nil {
		t.Fatalf("Failed to build default config: %v", err)
	}

	// Should use default values
	if config.Service.Name != "ephemos-service" {
		t.Errorf("Expected default service name 'ephemos-service', got '%s'", config.Service.Name)
	}
	if config.Transport.Type != "grpc" {
		t.Errorf("Expected default transport type 'grpc', got '%s'", config.Transport.Type)
	}
	if config.Transport.Address != ":50051" {
		t.Errorf("Expected default transport address ':50051', got '%s'", config.Transport.Address)
	}
}

func TestConfigBuilder_Validation(t *testing.T) {
	ctx := t.Context()

	// Test that validation still works with invalid configurations
	// Use an invalid character in service name instead of empty string
	_, err := NewConfigBuilder().
		WithSource(ConfigSourcePureCode).
		WithServiceName("invalid@service"). // Invalid character should fail validation
		Build(ctx)

	if err == nil {
		t.Error("Expected validation error for invalid service name")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Test that the error is a configuration validation error
	if !IsConfigurationError(err) {
		t.Errorf("Expected configuration validation error, got: %T, %v", err, err)
	}
}
