// Package application provides a factory for creating use case implementations
// that orchestrate domain logic according to hexagonal architecture principles.
package application

import (
	"context"
	"fmt"

	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// UseCaseFactory creates configured use case implementations.
// This factory encapsulates the complexity of use case setup and dependency injection.
type UseCaseFactory struct {
	config                 *ports.Configuration
	identityProvider       ports.IdentityProvider
	transportProvider      ports.TransportProvider
	configurationProvider  ports.ConfigurationProvider
}

// NewUseCaseFactory creates a new use case factory with the required dependencies.
func NewUseCaseFactory(
	config *ports.Configuration,
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	configurationProvider ports.ConfigurationProvider,
) (*UseCaseFactory, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if identityProvider == nil {
		return nil, fmt.Errorf("identity provider cannot be nil")
	}
	if transportProvider == nil {
		return nil, fmt.Errorf("transport provider cannot be nil")
	}

	return &UseCaseFactory{
		config:                config,
		identityProvider:      identityProvider,
		transportProvider:     transportProvider,
		configurationProvider: configurationProvider,
	}, nil
}

// CreateIdentityUseCase creates a configured identity use case.
func (f *UseCaseFactory) CreateIdentityUseCase(ctx context.Context) (IdentityUseCase, error) {
	// Create the underlying identity service
	identityService, err := services.NewIdentityService(
		f.identityProvider,
		f.transportProvider,
		f.config,
		nil, // Use default validator
		nil, // Use default metrics
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	// Wrap in use case interface
	return NewIdentityUseCase(identityService), nil
}

// CreateHealthUseCase creates a configured health monitoring use case.
// Note: This is a placeholder for future health use case implementation.
func (f *UseCaseFactory) CreateHealthUseCase(ctx context.Context) (HealthUseCase, error) {
	// TODO: Implement health use case when HealthMonitorService is refactored
	return nil, fmt.Errorf("health use case not yet implemented")
}

// CreateConfigurationUseCase creates a configured configuration management use case.
func (f *UseCaseFactory) CreateConfigurationUseCase(ctx context.Context) (ConfigurationUseCase, error) {
	if f.configurationProvider == nil {
		return nil, fmt.Errorf("configuration provider is required for configuration use case")
	}
	
	return &ConfigurationUseCaseImpl{
		provider: f.configurationProvider,
		config:   f.config,
	}, nil
}

// CreateRegistrationUseCase creates a configured service registration use case.
// Note: This is a placeholder for future registration use case implementation.
func (f *UseCaseFactory) CreateRegistrationUseCase(ctx context.Context) (RegistrationUseCase, error) {
	// TODO: Implement registration use case when CLI registration logic is refactored
	return nil, fmt.Errorf("registration use case not yet implemented")
}

// ConfigurationUseCaseImpl implements the ConfigurationUseCase interface.
type ConfigurationUseCaseImpl struct {
	provider ports.ConfigurationProvider
	config   *ports.Configuration
}

// LoadConfiguration loads and validates configuration from the specified source.
func (c *ConfigurationUseCaseImpl) LoadConfiguration(ctx context.Context, source string) (*ports.Configuration, error) {
	return c.provider.LoadConfiguration(ctx, source)
}

// ValidateConfiguration validates configuration without loading.
func (c *ConfigurationUseCaseImpl) ValidateConfiguration(ctx context.Context, config *ports.Configuration) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	return config.Validate()
}

// GetConfiguration retrieves current configuration.
func (c *ConfigurationUseCaseImpl) GetConfiguration(ctx context.Context) (*ports.Configuration, error) {
	return c.config, nil
}