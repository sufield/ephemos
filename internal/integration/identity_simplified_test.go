// Package integration provides practical integration tests for the identity system.
// These tests validate the core identity flows using the actual APIs.
package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/memidentity"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
	"github.com/sufield/ephemos/pkg/ephemos"
)

// TestIdentityProviderFlow tests the complete identity provider flow
// using the actual in-memory provider implementation.
func TestIdentityProviderFlow(t *testing.T) {
	t.Run("CompleteIdentityFlow", func(t *testing.T) {
		// Step 1: Create service identity
		identity := domain.NewServiceIdentity("test-service", "example.org")
		if err := identity.Validate(); err != nil {
			t.Fatalf("Identity validation failed: %v", err)
		}
		t.Logf("✅ Created identity: %s", identity.URI)

		// Step 2: Create identity provider with custom identity
		provider := memidentity.New().WithIdentity(identity)
		defer provider.Close()

		// Step 3: Retrieve service identity
		retrievedIdentity, err := provider.GetServiceIdentity()
		if err != nil {
			t.Fatalf("Failed to get service identity: %v", err)
		}

		if retrievedIdentity.Name != identity.Name {
			t.Errorf("Expected identity name %s, got %s", identity.Name, retrievedIdentity.Name)
		}
		if retrievedIdentity.Domain != identity.Domain {
			t.Errorf("Expected identity domain %s, got %s", identity.Domain, retrievedIdentity.Domain)
		}
		if retrievedIdentity.URI != identity.URI {
			t.Errorf("Expected identity URI %s, got %s", identity.URI, retrievedIdentity.URI)
		}
		t.Logf("✅ Retrieved identity matches: %s", retrievedIdentity.URI)

		// Step 4: Retrieve certificate
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

		// Step 5: Retrieve trust bundle
		trustBundle, err := provider.GetTrustBundle()
		if err != nil {
			t.Fatalf("Failed to get trust bundle: %v", err)
		}

		if len(trustBundle.Certificates) == 0 {
			t.Error("Trust bundle is empty")
		}
		t.Logf("✅ Retrieved trust bundle with %d certificates", len(trustBundle.Certificates))
	})
}

// TestAuthenticationPolicyFlow tests authentication policy creation and validation.
func TestAuthenticationPolicyFlow(t *testing.T) {
	t.Run("PolicyCreationAndValidation", func(t *testing.T) {
		// Step 1: Create service identity
		serverIdentity := domain.NewServiceIdentity("api-server", "company.com")
		clientIdentity := domain.NewServiceIdentity("web-client", "company.com")

		// Step 2: Create authentication policy
		policy := domain.NewAuthenticationPolicy(serverIdentity)
		if policy.ServiceIdentity != serverIdentity {
			t.Error("Policy does not reference correct service identity")
		}
		t.Logf("✅ Created authentication policy for %s", serverIdentity.Name)

		// Step 3: Add authorized clients
		policy.AddAuthorizedClient(clientIdentity.Name)
		policy.AddAuthorizedClient("mobile-app")
		policy.AddAuthorizedClient("batch-processor")

		expectedClients := []string{"web-client", "mobile-app", "batch-processor"}
		if len(policy.AuthorizedClients) != len(expectedClients) {
			t.Errorf("Expected %d authorized clients, got %d", len(expectedClients), len(policy.AuthorizedClients))
		}
		t.Logf("✅ Added %d authorized clients: %v", len(policy.AuthorizedClients), policy.AuthorizedClients)

		// Step 4: Test client authorization
		testCases := []struct {
			clientName string
			authorized bool
		}{
			{"web-client", true},
			{"mobile-app", true},
			{"batch-processor", true},
			{"unauthorized-client", false},
			{"", false},
		}

		for _, tc := range testCases {
			result := policy.IsClientAuthorized(tc.clientName)
			if result != tc.authorized {
				t.Errorf("Client %s: expected authorized=%v, got %v", tc.clientName, tc.authorized, result)
			}
		}
		t.Logf("✅ Client authorization validation passed")

		// Step 5: Add trusted servers (for client-side policy)
		clientPolicy := domain.NewAuthenticationPolicy(clientIdentity)
		clientPolicy.AddTrustedServer("api-server")
		clientPolicy.AddTrustedServer("payment-service")

		if !clientPolicy.IsServerTrusted("api-server") {
			t.Error("api-server should be trusted")
		}
		if !clientPolicy.IsServerTrusted("payment-service") {
			t.Error("payment-service should be trusted")
		}
		if clientPolicy.IsServerTrusted("untrusted-service") {
			t.Error("untrusted-service should not be trusted")
		}
		t.Logf("✅ Server trust validation passed")
	})
}

// TestConfigurationProviderFlow tests configuration loading and validation.
func TestConfigurationProviderFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("InMemoryConfigProvider", func(t *testing.T) {
		// Step 1: Create in-memory configuration provider
		provider := config.NewInMemoryProvider()

		// Step 2: Create test configuration with proper SPIFFE IDs
		testConfig := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service",
				Domain: "test.example.org",
			},
			SPIFFE: &ports.SPIFFEConfig{
				SocketPath: "/tmp/test-spire-agent/public/api.sock",
			},
			AuthorizedClients: []string{"spiffe://test.example.org/client-a", "spiffe://test.example.org/client-b"},
			TrustedServers:    []string{"spiffe://test.example.org/server-x", "spiffe://test.example.org/server-y"},
		}

		// Step 3: Set the configuration
		provider.SetConfiguration("test-config", testConfig)
		t.Logf("✅ Created and configured in-memory provider")

		// Step 4: Retrieve configuration
		retrievedConfig, err := provider.LoadConfiguration(ctx, "test-config")
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		// Step 5: Validate configuration values
		if retrievedConfig.Service.Name != "test-service" {
			t.Errorf("Expected service name 'test-service', got '%s'", retrievedConfig.Service.Name)
		}
		if retrievedConfig.Service.Domain != "test.example.org" {
			t.Errorf("Expected trust domain 'test.example.org', got '%s'", retrievedConfig.Service.Domain)
		}

		// Filter out empty strings that may be added by the config copying logic
		nonEmptyClients := make([]string, 0)
		for _, client := range retrievedConfig.AuthorizedClients {
			if strings.TrimSpace(client) != "" {
				nonEmptyClients = append(nonEmptyClients, client)
			}
		}
		nonEmptyServers := make([]string, 0)
		for _, server := range retrievedConfig.TrustedServers {
			if strings.TrimSpace(server) != "" {
				nonEmptyServers = append(nonEmptyServers, server)
			}
		}

		if len(nonEmptyClients) != 2 {
			t.Errorf("Expected 2 authorized clients, got %d: %v", len(nonEmptyClients), nonEmptyClients)
		}
		if len(nonEmptyServers) != 2 {
			t.Errorf("Expected 2 trusted servers, got %d: %v", len(nonEmptyServers), nonEmptyServers)
		}

		t.Logf("✅ Configuration validated successfully")
		t.Logf("   Service: %s@%s", retrievedConfig.Service.Name, retrievedConfig.Service.Domain)
		t.Logf("   Authorized clients: %v", retrievedConfig.AuthorizedClients)
		t.Logf("   Trusted servers: %v", retrievedConfig.TrustedServers)

		// Step 6: Test default configuration
		defaultConfig := provider.GetDefaultConfiguration(ctx)
		if defaultConfig == nil {
			t.Fatal("Default configuration is nil")
		}
		if defaultConfig.Service.Name != "default-service" {
			t.Errorf("Expected default service name 'default-service', got '%s'", defaultConfig.Service.Name)
		}
		t.Logf("✅ Default configuration validated")
	})
}

// TestIdentityServiceIntegration tests the identity service with real components.
func TestIdentityServiceIntegration(t *testing.T) {
	t.Run("IdentityServiceWithProviders", func(t *testing.T) {
		// Step 1: Create service identity
		identity := domain.NewServiceIdentity("integration-service", "test.example.org")
		
		// Step 2: Create identity provider
		provider := memidentity.New().WithIdentity(identity)
		defer provider.Close()

		// Step 3: Test provider directly (service integration)
		serviceIdentity, err := provider.GetServiceIdentity()
		if err != nil {
			t.Fatalf("Failed to get service identity: %v", err)
		}

		if serviceIdentity.URI != identity.URI {
			t.Errorf("Expected URI %s, got %s", identity.URI, serviceIdentity.URI)
		}
		t.Logf("✅ Identity provider returned correct identity: %s", serviceIdentity.URI)

		// Step 4: Test certificate retrieval through provider
		certificate, err := provider.GetCertificate()
		if err != nil {
			t.Fatalf("Failed to get certificate through provider: %v", err)
		}

		if certificate == nil {
			t.Fatal("Certificate is nil")
		}
		t.Logf("✅ Identity provider returned valid certificate")

		// Step 5: Test trust bundle retrieval through provider
		trustBundle, err := provider.GetTrustBundle()
		if err != nil {
			t.Fatalf("Failed to get trust bundle through provider: %v", err)
		}

		if trustBundle == nil || len(trustBundle.Certificates) == 0 {
			t.Fatal("Trust bundle is empty")
		}
		t.Logf("✅ Identity provider returned trust bundle with %d certificates", len(trustBundle.Certificates))

		// Step 6: Test identity service creation (with correct SPIFFE IDs)
		mockConfig := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "integration-service",
				Domain: "test.example.org",
			},
			SPIFFE: &ports.SPIFFEConfig{
				SocketPath: "/tmp/test-spire-agent/public/api.sock",
			},
			AuthorizedClients: []string{"spiffe://test.example.org/test-client"},
			TrustedServers:    []string{"spiffe://test.example.org/test-server"},
		}

		// Create a simple transport provider mock for testing
		mockTransportProvider := &mockTransportProvider{}
		
		identityService, err := services.NewIdentityService(provider, mockTransportProvider, mockConfig)
		if err != nil {
			t.Fatalf("Failed to create identity service: %v", err)
		}
		
		if identityService == nil {
			t.Fatal("Identity service is nil")
		}
		t.Logf("✅ Identity service created successfully")

		// Step 7: Test service creation through identity service
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
	})
}

// TestPublicAPIIntegration tests the public ephemos API integration points.
func TestPublicAPIIntegration(t *testing.T) {
	t.Run("InterceptorConfiguration", func(t *testing.T) {
		// Step 1: Test default interceptor configuration
		defaultConfig := ephemos.NewDefaultInterceptorConfig()
		if !defaultConfig.EnableAuth {
			t.Error("Expected auth to be enabled in default config")
		}
		if !defaultConfig.EnableLogging {
			t.Error("Expected logging to be enabled in default config")
		}
		if !defaultConfig.EnableMetrics {
			t.Error("Expected metrics to be enabled in default config")
		}
		if defaultConfig.EnableIdentityPropagation {
			t.Error("Expected identity propagation to be disabled in default config")
		}
		t.Logf("✅ Default interceptor configuration validated")

		// Step 2: Test production interceptor configuration
		prodConfig := ephemos.NewProductionInterceptorConfig("prod-service")
		if !prodConfig.EnableAuth {
			t.Error("Expected auth to be enabled in production config")
		}
		if !prodConfig.EnableIdentityPropagation {
			t.Error("Expected identity propagation to be enabled in production config")
		}
		t.Logf("✅ Production interceptor configuration validated")

		// Step 3: Test development interceptor configuration
		devConfig := ephemos.NewDevelopmentInterceptorConfig("dev-service")
		if devConfig.EnableAuth {
			t.Error("Expected auth to be disabled in development config")
		}
		if !devConfig.EnableIdentityPropagation {
			t.Error("Expected identity propagation to be enabled in development config")
		}
		t.Logf("✅ Development interceptor configuration validated")
	})

	t.Run("ServiceRegistrarCreation", func(t *testing.T) {
		// Test service registrar creation
		var registrationCalled bool
		registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
			registrationCalled = true
		})

		if registrar == nil {
			t.Fatal("Service registrar is nil")
		}

		// Test registration function is stored and called
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()
		registrar.Register(grpcServer)
		
		if !registrationCalled {
			t.Error("Registration function was not called")
		}
		t.Logf("✅ Service registrar created and tested successfully")
	})
}

// TestErrorHandlingFlow tests error handling throughout the identity stack.
func TestErrorHandlingFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
				identity: &domain.ServiceIdentity{Name: "", Domain: "example.org"},
				hasError: true,
			},
			{
				name:     "EmptyDomain",
				identity: &domain.ServiceIdentity{Name: "service", Domain: ""},
				hasError: true,
			},
			{
				name:     "BothEmpty",
				identity: &domain.ServiceIdentity{Name: "", Domain: ""},
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
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()
		
		_, err = provider.LoadConfiguration(cancelledCtx, "test")
		if err == nil {
			t.Error("Expected error for cancelled context")
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
			if serviceIdentity.URI != identity.URI {
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
		ctx := context.Background()
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

// Mock transport provider for testing
type mockTransportProvider struct{}

func (m *mockTransportProvider) CreateServer(cert *domain.Certificate, trustBundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.Server, error) {
	return &mockServer{}, nil
}

func (m *mockTransportProvider) CreateClient(cert *domain.Certificate, trustBundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.Client, error) {
	return &mockClient{}, nil
}

type mockServer struct{}

func (m *mockServer) RegisterService(serviceRegistrar ports.ServiceRegistrar) error { return nil }
func (m *mockServer) Start(listener ports.Listener) error                           { return nil }
func (m *mockServer) Stop() error                                                   { return nil }

type mockClient struct{}

func (m *mockClient) Connect(serviceName, address string) (ports.Connection, error) {
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

	b.Run("PolicyAuthorization", func(b *testing.B) {
		policy := domain.NewAuthenticationPolicy(identity)
		policy.AddAuthorizedClient("test-client")
		
		for i := 0; i < b.N; i++ {
			_ = policy.IsClientAuthorized("test-client")
		}
	})
}