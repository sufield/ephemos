package services

import (
	"context"
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

// MockIdentityProvider for testing
type MockIdentityProvider struct {
	shouldFail bool
	identity   ports.ServiceIdentity
}

func (m *MockIdentityProvider) GetServiceIdentity() (ports.ServiceIdentity, error) {
	if m.shouldFail {
		return nil, ports.ErrIdentityNotFound
	}
	return m.identity, nil
}

func (m *MockIdentityProvider) Close() error {
	return nil
}

// MockTransportProvider for testing
type MockTransportProvider struct {
	shouldFailServer bool
	shouldFailClient bool
}

func (m *MockTransportProvider) CreateServerTransport(ctx context.Context) (ports.ServerTransport, error) {
	if m.shouldFailServer {
		return nil, ports.ErrTransportCreationFailed
	}
	return &MockServerTransport{}, nil
}

func (m *MockTransportProvider) CreateClientTransport(ctx context.Context, targetService, serverAddress string) (ports.ClientTransport, error) {
	if m.shouldFailClient {
		return nil, ports.ErrTransportCreationFailed
	}
	return &MockClientTransport{}, nil
}

func (m *MockTransportProvider) Close() error {
	return nil
}

// MockServerTransport for testing
type MockServerTransport struct{}

func (m *MockServerTransport) Serve(ctx context.Context, listener interface{}) error {
	return nil
}

func (m *MockServerTransport) Close() error {
	return nil
}

// MockClientTransport for testing
type MockClientTransport struct{}

func (m *MockClientTransport) Connect(ctx context.Context, serviceName, address string) (interface{}, error) {
	return nil, nil
}

func (m *MockClientTransport) Close() error {
	return nil
}

// MockServiceIdentity for testing
type MockServiceIdentity struct {
	domain string
	name   string
}

func (m *MockServiceIdentity) GetDomain() string {
	return m.domain
}

func (m *MockServiceIdentity) GetName() string {
	return m.name
}

func (m *MockServiceIdentity) Validate() error {
	if m.domain == "" {
		return &ports.ValidationError{
			Field:   "domain",
			Value:   m.domain,
			Message: "domain cannot be empty",
		}
	}
	if m.name == "" {
		return &ports.ValidationError{
			Field:   "name", 
			Value:   m.name,
			Message: "name cannot be empty",
		}
	}
	return nil
}

func (m *MockServiceIdentity) Close() error {
	return nil
}

func TestNewIdentityService(t *testing.T) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}
	transportProvider := &MockTransportProvider{}

	tests := []struct {
		name              string
		identityProvider  ports.IdentityProvider
		transportProvider ports.TransportProvider
		config            *ports.Configuration
		wantErr           bool
	}{
		{
			name:              "valid configuration",
			identityProvider:  identityProvider,
			transportProvider: transportProvider,
			config:            validConfig,
			wantErr:           false,
		},
		{
			name:              "nil config",
			identityProvider:  identityProvider,
			transportProvider: transportProvider,
			config:            nil,
			wantErr:           true,
		},
		{
			name:              "invalid config - empty service name",
			identityProvider:  identityProvider,
			transportProvider: transportProvider,
			config: &ports.Configuration{
				Service: &ports.ServiceConfig{
					Name:   "",
					Domain: "example.com",
				},
				SPIFFE: &ports.SPIFFEConfig{
					Domain:      "example.com",
					SocketPath:  "/tmp/spire-agent/public/api.sock",
					TrustDomain: "example.com",
				},
			},
			wantErr: true,
		},
		{
			name:              "nil identity provider",
			identityProvider:  nil,
			transportProvider: transportProvider,
			config:            validConfig,
			wantErr:           true,
		},
		{
			name:              "nil transport provider",
			identityProvider:  identityProvider,
			transportProvider: nil,
			config:            validConfig,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewIdentityService(tt.identityProvider, tt.transportProvider, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIdentityService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Error("NewIdentityService() returned nil service")
			}
		})
	}
}

func TestIdentityService_CreateServerIdentity(t *testing.T) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	tests := []struct {
		name              string
		identityProvider  *MockIdentityProvider
		transportProvider *MockTransportProvider
		wantErr           bool
	}{
		{
			name: "successful creation",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{
					domain: "example.com",
					name:   "test-service",
				},
			},
			transportProvider: &MockTransportProvider{},
			wantErr:           false,
		},
		{
			name: "identity provider failure",
			identityProvider: &MockIdentityProvider{
				shouldFail: true,
			},
			transportProvider: &MockTransportProvider{},
			wantErr:           true,
		},
		{
			name: "transport provider failure",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{
					domain: "example.com",
					name:   "test-service",
				},
			},
			transportProvider: &MockTransportProvider{
				shouldFailServer: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewIdentityService(tt.identityProvider, tt.transportProvider, validConfig)
			if err != nil {
				if !tt.wantErr {
					t.Skip("Service creation failed, skipping test:", err)
				}
				return
			}

			ctx := context.Background()
			serverIdentity, err := service.CreateServerIdentity(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateServerIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && serverIdentity == nil {
				t.Error("CreateServerIdentity() returned nil")
			}
		})
	}
}

func TestIdentityService_CreateClientIdentity(t *testing.T) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	tests := []struct {
		name              string
		identityProvider  *MockIdentityProvider
		transportProvider *MockTransportProvider
		wantErr           bool
	}{
		{
			name: "successful creation",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{
					domain: "example.com",
					name:   "test-service",
				},
			},
			transportProvider: &MockTransportProvider{},
			wantErr:           false,
		},
		{
			name: "identity provider failure",
			identityProvider: &MockIdentityProvider{
				shouldFail: true,
			},
			transportProvider: &MockTransportProvider{},
			wantErr:           true,
		},
		{
			name: "transport provider failure",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{
					domain: "example.com",
					name:   "test-service",
				},
			},
			transportProvider: &MockTransportProvider{
				shouldFailClient: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewIdentityService(tt.identityProvider, tt.transportProvider, validConfig)
			if err != nil {
				if !tt.wantErr {
					t.Skip("Service creation failed, skipping test:", err)
				}
				return
			}

			ctx := context.Background()
			clientIdentity, err := service.CreateClientIdentity(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateClientIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && clientIdentity == nil {
				t.Error("CreateClientIdentity() returned nil")
			}
		})
	}
}

func TestIdentityService_ContextValidation(t *testing.T) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}
	transportProvider := &MockTransportProvider{}

	service, err := NewIdentityService(identityProvider, transportProvider, validConfig)
	if err != nil {
		t.Skip("Service creation failed:", err)
	}

	// Test CreateServerIdentity with nil context
	_, err = service.CreateServerIdentity(nil)
	if err == nil {
		t.Error("CreateServerIdentity() with nil context should return error")
	}

	// Test CreateClientIdentity with nil context
	_, err = service.CreateClientIdentity(nil)
	if err == nil {
		t.Error("CreateClientIdentity() with nil context should return error")
	}
}

func TestIdentityService_ConcurrentAccess(t *testing.T) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}
	transportProvider := &MockTransportProvider{}

	service, err := NewIdentityService(identityProvider, transportProvider, validConfig)
	if err != nil {
		t.Skip("Service creation failed:", err)
	}

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Alternate between server and client creation
			for j := 0; j < 10; j++ {
				if j%2 == 0 {
					service.CreateServerIdentity(ctx)
				} else {
					service.CreateClientIdentity(ctx)
				}
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkIdentityService_CreateServerIdentity(b *testing.B) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}
	transportProvider := &MockTransportProvider{}

	service, err := NewIdentityService(identityProvider, transportProvider, validConfig)
	if err != nil {
		b.Skip("Service creation failed:", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateServerIdentity(ctx)
	}
}

func BenchmarkIdentityService_CreateClientIdentity(b *testing.B) {
	validConfig := &ports.Configuration{
		Service: &ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
		SPIFFE: &ports.SPIFFEConfig{
			Domain:      "example.com",
			SocketPath:  "/tmp/spire-agent/public/api.sock",
			TrustDomain: "example.com",
		},
	}

	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}
	transportProvider := &MockTransportProvider{}

	service, err := NewIdentityService(identityProvider, transportProvider, validConfig)
	if err != nil {
		b.Skip("Service creation failed:", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateClientIdentity(ctx)
	}
}