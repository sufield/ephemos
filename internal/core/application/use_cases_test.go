package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/sufield/ephemos/internal/core/adapters"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// TestUseCaseFactoryCreation tests that the use case factory can be created.
func TestUseCaseFactoryCreation(t *testing.T) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}

	// Create mock providers
	identityProvider := &mockIdentityProvider{}
	transportProvider := &mockTransportProvider{}
	configProvider := &mockConfigProvider{}

	factory, err := NewUseCaseFactory(config, identityProvider, transportProvider, configProvider)
	require.NoError(t, err)
	assert.NotNil(t, factory)
}

// TestUseCaseFactoryValidation tests factory validation.
func TestUseCaseFactoryValidation(t *testing.T) {
	tests := []struct {
		name              string
		config            *ports.Configuration
		identityProvider  ports.IdentityProvider
		transportProvider ports.TransportProvider
		configProvider    ports.ConfigurationProvider
		expectError       bool
		errorContains     string
	}{
		{
			name:              "nil config",
			config:            nil,
			identityProvider:  &mockIdentityProvider{},
			transportProvider: &mockTransportProvider{},
			configProvider:    &mockConfigProvider{},
			expectError:       true,
			errorContains:     "configuration cannot be nil",
		},
		{
			name: "nil identity provider",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{Name: "test", Domain: "test.local"},
			},
			identityProvider:  nil,
			transportProvider: &mockTransportProvider{},
			configProvider:    &mockConfigProvider{},
			expectError:       true,
			errorContains:     "identity provider cannot be nil",
		},
		{
			name: "nil transport provider",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{Name: "test", Domain: "test.local"},
			},
			identityProvider:  &mockIdentityProvider{},
			transportProvider: nil,
			configProvider:    &mockConfigProvider{},
			expectError:       true,
			errorContains:     "transport provider cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewUseCaseFactory(tt.config, tt.identityProvider, tt.transportProvider, tt.configProvider)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, factory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, factory)
			}
		})
	}
}

// TestIdentityUseCaseImpl tests the identity use case implementation.
func TestIdentityUseCaseImpl(t *testing.T) {
	// Create a test identity service
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}

	identityService, err := services.NewIdentityService(
		&mockIdentityProvider{},
		&mockTransportProvider{},
		config,
		adapters.NewDefaultCertValidator(),
		nil, // Use default metrics
	)
	require.NoError(t, err)

	useCase := NewIdentityUseCase(identityService)
	assert.NotNil(t, useCase)
	
	// Basic smoke test - just ensure the use case wrapper works
	assert.NotNil(t, useCase)
}

// Mock implementations for testing

type mockIdentityProvider struct{}

func (m *mockIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	return domain.NewServiceIdentity("mock-service", "mock.local"), nil
}

func (m *mockIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	return nil, assert.AnError
}

func (m *mockIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	return nil, assert.AnError
}

func (m *mockIdentityProvider) Close() error {
	return nil
}

type mockTransportProvider struct{}

func (m *mockTransportProvider) CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ServerPort, error) {
	return nil, assert.AnError
}

func (m *mockTransportProvider) CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ClientPort, error) {
	return nil, assert.AnError
}

type mockConfigProvider struct{}

func (m *mockConfigProvider) LoadConfiguration(ctx context.Context, path string) (*ports.Configuration, error) {
	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "mock-service",
			Domain: "mock.local",
		},
	}, nil
}

func (m *mockConfigProvider) GetDefaultConfiguration(ctx context.Context) *ports.Configuration {
	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "default-service",
			Domain: "default.local",
		},
	}
}