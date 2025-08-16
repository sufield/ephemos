// Package integration provides practical integration tests for the identity system.
// These tests validate the core identity flows using the actual APIs.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/memidentity"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// TestIdentityProviderFlow tests the complete identity provider flow
// using the actual in-memory provider implementation.
func TestIdentityProviderFlow(t *testing.T) {
	t.Run("CompleteIdentityFlow", func(t *testing.T) {
		identity := testCreateServiceIdentity(t)
		provider := testCreateProvider(t, identity)
		defer provider.Close()

		testValidateRetrievedIdentity(t, provider, identity)
		testValidateCertificate(t, provider)
		testValidateTrustBundle(t, provider)
	})
}

func testCreateServiceIdentity(t *testing.T) *domain.ServiceIdentity {
	t.Helper()
	identity := domain.NewServiceIdentity("test-service", "example.org")
	if err := identity.Validate(); err != nil {
		t.Fatalf("Identity validation failed: %v", err)
	}
	t.Logf("✅ Created identity: %s", identity.URI())
	return identity
}

func testCreateProvider(_ *testing.T, identity *domain.ServiceIdentity) ports.IdentityProvider {
	provider := memidentity.New().WithIdentity(identity)
	return provider
}

func testValidateRetrievedIdentity(t *testing.T, provider ports.IdentityProvider, identity *domain.ServiceIdentity) {
	t.Helper()
	retrievedIdentity, err := provider.GetServiceIdentity()
	if err != nil {
		t.Fatalf("Failed to get service identity: %v", err)
	}

	if retrievedIdentity.Name() != identity.Name() {
		t.Errorf("Expected identity name %s, got %s", identity.Name(), retrievedIdentity.Name())
	}
	if retrievedIdentity.Domain() != identity.Domain() {
		t.Errorf("Expected identity domain %s, got %s", identity.Domain(), retrievedIdentity.Domain())
	}
	if retrievedIdentity.URI() != identity.URI() {
		t.Errorf("Expected identity URI %s, got %s", identity.URI(), retrievedIdentity.URI())
	}
	t.Logf("✅ Retrieved identity matches: %s", retrievedIdentity.URI())
}

func testValidateCertificate(t *testing.T, provider ports.IdentityProvider) {
	t.Helper()
	certificate, err := provider.GetCertificate()
	if err != nil {
		t.Fatalf("Failed to get certificate: %v", err)
	}

	if certificate.Cert == nil {
		t.Error("Certificate is nil")
	}
	if certificate.PrivateKey == nil {
		t.Error("Private key is nil")
	}
	if len(certificate.Chain) == 0 {
		t.Error("Certificate chain is empty")
	}
	t.Logf("✅ Retrieved certificate with %d certificates in chain", len(certificate.Chain))
}

func testValidateTrustBundle(t *testing.T, provider ports.IdentityProvider) {
	t.Helper()
	trustBundle, err := provider.GetTrustBundle()
	if err != nil {
		t.Fatalf("Failed to get trust bundle: %v", err)
	}

	if len(trustBundle.Certificates) == 0 {
		t.Error("Trust bundle is empty")
	}
	t.Logf("✅ Retrieved trust bundle with %d certificates", len(trustBundle.Certificates))
}

// TestAuthenticationPolicyFlow tests authentication policy creation.
func TestAuthenticationPolicyFlow(t *testing.T) {
	t.Run("PolicyCreation", func(t *testing.T) {
		// Step 1: Create service identity
		serverIdentity := domain.NewServiceIdentity("api-server", "company.com")
		clientIdentity := domain.NewServiceIdentity("web-client", "company.com")

		// Step 2: Create authentication policy
		policy := domain.NewAuthenticationPolicy(serverIdentity)
		if policy.ServiceIdentity != serverIdentity {
			t.Error("Policy does not reference correct service identity")
		}
		t.Logf("✅ Created authentication policy for %s", serverIdentity.Name)

		// Step 3: Create client-side policy
		clientPolicy := domain.NewAuthenticationPolicy(clientIdentity)
		if clientPolicy.ServiceIdentity != clientIdentity {
			t.Error("Client policy does not reference correct service identity")
		}
		t.Logf("✅ Created client authentication policy for %s", clientIdentity.Name)
	})
}

// TestConfigurationProviderFlow tests configuration loading and validation.
func TestConfigurationProviderFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	t.Run("InMemoryConfigProvider", func(t *testing.T) {
		provider, testConfig := testSetupConfigProvider(t)
		retrievedConfig := testLoadAndValidateConfig(ctx, t, provider, testConfig)
		testValidateDefaultConfig(ctx, t, provider)
		_ = retrievedConfig // Use the variable to avoid unused variable warning
	})
}

func testSetupConfigProvider(t *testing.T) (*config.InMemoryProvider, *ports.Configuration) {
	t.Helper()
	provider := config.NewInMemoryProvider()

	testConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.example.org",
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/test-spire-agent/public/api.sock",
		},
	}

	provider.SetConfiguration("test-config", testConfig)
	t.Logf("✅ Created and configured in-memory provider")
	return provider, testConfig
}

func testLoadAndValidateConfig(
	ctx context.Context, t *testing.T, provider *config.InMemoryProvider, testConfig *ports.Configuration,
) *ports.Configuration {
	t.Helper()
	retrievedConfig, err := provider.LoadConfiguration(ctx, "test-config")
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	testValidateServiceConfig(t, retrievedConfig, testConfig)
	testLogConfigurationDetails(t, retrievedConfig)
	return retrievedConfig
}

func testValidateServiceConfig(t *testing.T, retrieved, expected *ports.Configuration) {
	t.Helper()
	if retrieved.Service.Name != expected.Service.Name {
		t.Errorf("Expected service name '%s', got '%s'", expected.Service.Name, retrieved.Service.Name)
	}
	if retrieved.Service.Domain != expected.Service.Domain {
		t.Errorf("Expected trust domain '%s', got '%s'", expected.Service.Domain, retrieved.Service.Domain)
	}
}

func testLogConfigurationDetails(t *testing.T, config *ports.Configuration) {
	t.Helper()
	t.Logf("✅ Configuration validated successfully")
	t.Logf("   Service: %s@%s", config.Service.Name, config.Service.Domain)
}

func testValidateDefaultConfig(ctx context.Context, t *testing.T, provider *config.InMemoryProvider) {
	t.Helper()
	defaultConfig := provider.GetDefaultConfiguration(ctx)
	if defaultConfig == nil {
		t.Fatal("Default configuration is nil")
	}
	if defaultConfig.Service.Name != "default-service" {
		t.Errorf("Expected default service name 'default-service', got '%s'", defaultConfig.Service.Name)
	}
	t.Logf("✅ Default configuration validated")
}

// TestIdentityServiceIntegration tests the identity service with real components.
func TestIdentityServiceIntegration(t *testing.T) {
	t.Run("IdentityServiceWithProviders", func(t *testing.T) {
		identity, provider := testSetupIdentityProvider(t)
		defer provider.Close()

		testValidateProviderDirectly(t, provider, identity)
		identityService := testCreateIdentityService(t, provider)
		testValidateServiceCreation(t, identityService)
	})
}

func testSetupIdentityProvider(t *testing.T) (*domain.ServiceIdentity, ports.IdentityProvider) {
	t.Helper()
	identity := domain.NewServiceIdentity("integration-service", "test.example.org")
	provider := memidentity.New().WithIdentity(identity)
	return identity, provider
}

func testValidateProviderDirectly(t *testing.T, provider ports.IdentityProvider, identity *domain.ServiceIdentity) {
	t.Helper()
	serviceIdentity, err := provider.GetServiceIdentity()
	if err != nil {
		t.Fatalf("Failed to get service identity: %v", err)
	}

	if serviceIdentity.URI() != identity.URI() {
		t.Errorf("Expected URI %s, got %s", identity.URI(), serviceIdentity.URI())
	}
	t.Logf("✅ Identity provider returned correct identity: %s", serviceIdentity.URI())

	testValidateProviderCertificate(t, provider)
	testValidateProviderTrustBundle(t, provider)
}

func testValidateProviderCertificate(t *testing.T, provider ports.IdentityProvider) {
	t.Helper()
	certificate, err := provider.GetCertificate()
	if err != nil {
		t.Fatalf("Failed to get certificate through provider: %v", err)
	}

	if certificate == nil {
		t.Fatal("Certificate is nil")
	}
	t.Logf("✅ Identity provider returned valid certificate")
}

func testValidateProviderTrustBundle(t *testing.T, provider ports.IdentityProvider) {
	t.Helper()
	trustBundle, err := provider.GetTrustBundle()
	if err != nil {
		t.Fatalf("Failed to get trust bundle through provider: %v", err)
	}

	if trustBundle == nil || len(trustBundle.Certificates) == 0 {
		t.Fatal("Trust bundle is empty")
	}
	t.Logf("✅ Identity provider returned trust bundle with %d certificates", len(trustBundle.Certificates))
}

func testCreateIdentityService(t *testing.T, provider ports.IdentityProvider) *services.IdentityService {
	t.Helper()
	mockConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "integration-service",
			Domain: "test.example.org",
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/test-spire-agent/public/api.sock",
		},
	}

	mockTransportProvider := &mockTransportProvider{}
	identityService, err := services.NewIdentityService(provider, mockTransportProvider, mockConfig)
	if err != nil {
		t.Fatalf("Failed to create identity service: %v", err)
	}

	if identityService == nil {
		t.Fatal("Identity service is nil")
	}
	t.Logf("✅ Identity service created successfully")
	return identityService
}

func testValidateServiceCreation(t *testing.T, identityService *services.IdentityService) {
	t.Helper()
	server, err := identityService.CreateServerIdentity()
	if err != nil {
		t.Fatalf("Failed to create server identity: %v", err)
	}
	if server == nil {
		t.Fatal("Server is nil")
	}
	t.Logf("✅ Server identity created through service")

	client, err := identityService.CreateClientIdentity()
	if err != nil {
		t.Fatalf("Failed to create client identity: %v", err)
	}
	if client == nil {
		t.Fatal("Client is nil")
	}
	t.Logf("✅ Client identity created through service")
}

// TestPublicAPIIntegration tests the public ephemos API integration points.
func TestPublicAPIIntegration(t *testing.T) {
	t.Run("InterceptorConfiguration", func(t *testing.T) {
		testInterceptorConfigurations(t)
	})

}

func testInterceptorConfigurations(t *testing.T) {
	t.Helper()
	// TODO: Implement interceptor configuration functions in public API
	t.Skip("Interceptor configuration functions not yet implemented")
}

// TODO: Implement these interceptor configuration functions in public API
// func testDefaultInterceptorConfig(t *testing.T) { ... }
// func testProductionInterceptorConfig(t *testing.T) { ... }  
// func testDevelopmentInterceptorConfig(t *testing.T) { ... }

// TestErrorHandlingFlow tests error handling throughout the identity stack.
func TestErrorHandlingFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	t.Run("IdentityValidationErrors", func(t *testing.T) {
		// Test invalid identities
		testCases := []struct {
			name     string
			identity *domain.ServiceIdentity
			hasError bool
		}{
			{
				name:     "ValidIdentity",
				identity: domain.NewServiceIdentity("valid-service", "example.org"),
				hasError: false,
			},
			{
				name:     "EmptyServiceName",
				identity: domain.NewServiceIdentity("", "example.org"),
				hasError: true,
			},
			{
				name:     "EmptyDomain",
				identity: domain.NewServiceIdentity("service", ""),
				hasError: true,
			},
			{
				name:     "BothEmpty",
				identity: domain.NewServiceIdentity("", ""),
				hasError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.identity.Validate()
				if tc.hasError && err == nil {
					t.Errorf("Expected error for %s, but got nil", tc.name)
				} else if !tc.hasError && err != nil {
					t.Errorf("Expected no error for %s, but got: %v", tc.name, err)
				}
			})
		}
		t.Logf("✅ Identity validation error handling validated")
	})

	t.Run("ProviderErrorHandling", func(t *testing.T) {
		// Create provider with valid identity
		identity := domain.NewServiceIdentity("error-test-service", "example.org")
		provider := memidentity.New().WithIdentity(identity)
		defer provider.Close()

		// Test that provider methods work correctly
		_, err := provider.GetServiceIdentity()
		if err != nil {
			t.Logf("Unexpected error from provider: %v", err)
		}

		// Test provider after close
		provider.Close()
		_, err = provider.GetServiceIdentity()
		if err == nil {
			t.Error("Expected error after closing provider")
		}
		t.Logf("✅ Provider error handling validated: %v", err)
	})

	t.Run("ConfigurationErrors", func(t *testing.T) {
		provider := config.NewInMemoryProvider()

		// Test loading non-existent configuration
		_, err := provider.LoadConfiguration(ctx, "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent configuration")
		}
		t.Logf("✅ Configuration error handling validated: %v", err)

		// Test context cancellation
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err = provider.LoadConfiguration(canceledCtx, "test")
		if err == nil {
			t.Error("Expected error for canceled context")
		}
		t.Logf("✅ Context cancellation handling validated: %v", err)
	})
}

// TestIdentityLifecycle tests the complete lifecycle of identity components.
func TestIdentityLifecycle(t *testing.T) {
	t.Run("ProviderLifecycle", func(t *testing.T) {
		// Step 1: Create and configure provider
		identity := domain.NewServiceIdentity("lifecycle-service", "test.org")
		provider := memidentity.New().WithIdentity(identity)

		// Step 2: Use provider multiple times
		for i := 0; i < 3; i++ {
			serviceIdentity, err := provider.GetServiceIdentity()
			if err != nil {
				t.Fatalf("Iteration %d: Failed to get service identity: %v", i, err)
			}
			if serviceIdentity.URI() != identity.URI() {
				t.Errorf("Iteration %d: URI mismatch", i)
			}
		}
		t.Logf("✅ Provider multiple usage validated")

		// Step 3: Close provider
		err := provider.Close()
		if err != nil {
			t.Errorf("Failed to close provider: %v", err)
		}

		// Step 4: Verify provider is closed
		_, err = provider.GetServiceIdentity()
		if err == nil {
			t.Error("Expected error after closing provider")
		}
		t.Logf("✅ Provider lifecycle completed successfully")
	})

	t.Run("ConfigurationLifecycle", func(t *testing.T) {
		ctx := t.Context()
		provider := config.NewInMemoryProvider()

		// Step 1: Test default configuration
		defaultConfig := provider.GetDefaultConfiguration(ctx)
		if defaultConfig == nil {
			t.Fatal("Default configuration is nil")
		}
		originalName := defaultConfig.Service.Name

		// Step 2: Update default configuration
		newDefault := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "updated-default",
				Domain: "updated.org",
			},
		}
		provider.SetDefaultConfiguration(newDefault)

		// Step 3: Verify update
		updatedDefault := provider.GetDefaultConfiguration(ctx)
		if updatedDefault.Service.Name != "updated-default" {
			t.Errorf("Expected updated name, got %s", updatedDefault.Service.Name)
		}
		if updatedDefault.Service.Name == originalName {
			t.Error("Configuration was not updated")
		}
		t.Logf("✅ Configuration lifecycle validated")
	})
}

// Mock transport provider for testing.
type mockTransportProvider struct{}

func (m *mockTransportProvider) CreateServer(
	_ *domain.Certificate, _ *domain.TrustBundle, _ *domain.AuthenticationPolicy,
) (ports.ServerPort, error) {
	return &mockServer{}, nil
}

func (m *mockTransportProvider) CreateClient(
	_ *domain.Certificate, _ *domain.TrustBundle, _ *domain.AuthenticationPolicy,
) (ports.ClientPort, error) {
	return &mockClient{}, nil
}

type mockServer struct{}

func (m *mockServer) RegisterService(_ ports.ServiceRegistrarPort) error { return nil }
func (m *mockServer) Start(_ ports.ListenerPort) error                   { return nil }
func (m *mockServer) Stop() error                                    { return nil }

type mockClient struct{}

func (m *mockClient) Connect(_, _ string) (ports.ConnectionPort, error) {
	return &mockConnection{}, nil
}
func (m *mockClient) Close() error { return nil }

type mockConnection struct{}

func (m *mockConnection) GetClientConnection() interface{} { return nil }
func (m *mockConnection) Close() error                     { return nil }

// BenchmarkIdentityOperations benchmarks key identity operations for performance testing.
func BenchmarkIdentityOperations(b *testing.B) {
	identity := domain.NewServiceIdentity("bench-service", "example.org")
	provider := memidentity.New().WithIdentity(identity)
	defer provider.Close()

	b.Run("GetServiceIdentity", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := provider.GetServiceIdentity()
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

	b.Run("GetCertificate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := provider.GetCertificate()
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

	b.Run("GetTrustBundle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := provider.GetTrustBundle()
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

	b.Run("IdentityValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testIdentity := domain.NewServiceIdentity("bench-service", "example.org")
			err := testIdentity.Validate()
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

}
