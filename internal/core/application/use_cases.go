// Package application provides use cases that orchestrate domain logic
// according to hexagonal architecture principles. Use cases represent
// application-specific business rules and coordinate between domain
// entities and external adapters through ports.
//
// This layer:
// - Defines use case interfaces for application business rules
// - Orchestrates interactions between domain and ports
// - Manages application state and workflows
// - Provides the application's public interface to adapters
//
// Use cases are consumed by primary adapters (API, CLI) and consume
// secondary adapters (repositories, external services) through ports.
package application

import (
	"context"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// IdentityUseCase defines the application-level operations for identity management.
// This interface represents the core identity use cases that the application supports,
// abstracting the complexity of SPIFFE/SPIRE integration and certificate lifecycle management.
type IdentityUseCase interface {
	// CreateServerIdentity creates a server with identity-based authentication.
	// This use case orchestrates server creation with proper certificate validation
	// and authentication policy setup.
	CreateServerIdentity(ctx context.Context) (ports.ServerPort, error)

	// CreateClientIdentity creates a client connection with identity-based authentication.
	// This use case handles client certificate acquisition, validation, and
	// secure connection establishment.
	CreateClientIdentity(ctx context.Context) (ports.ClientPort, error)

	// ValidateServiceIdentity validates that a certificate is valid, not expired,
	// and matches expected identity. This use case ensures certificates meet
	// security requirements before use.
	ValidateServiceIdentity(ctx context.Context, cert *domain.Certificate) error

	// GetCertificate retrieves the service certificate for client connections.
	// This use case handles certificate caching, rotation, and validation.
	GetCertificate(ctx context.Context) (*domain.Certificate, error)

	// GetTrustBundle retrieves the trust bundle for certificate validation.
	// This use case manages trust bundle caching and refresh.
	GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error)
}

// HealthUseCase defines the application-level operations for health monitoring.
// This interface represents health check and monitoring use cases.
type HealthUseCase interface {
	// StartMonitoring begins health monitoring for the service.
	// This use case coordinates health checkers and reporting mechanisms.
	StartMonitoring(ctx context.Context) error

	// StopMonitoring stops health monitoring gracefully.
	StopMonitoring(ctx context.Context) error

	// GetHealthStatus retrieves current health status.
	GetHealthStatus(ctx context.Context) (*domain.HealthStatus, error)
}

// ConfigurationUseCase defines the application-level operations for configuration management.
// This interface represents configuration loading, validation, and management use cases.
type ConfigurationUseCase interface {
	// LoadConfiguration loads and validates configuration from the specified source.
	LoadConfiguration(ctx context.Context, source string) (*ports.Configuration, error)

	// ValidateConfiguration validates configuration without loading.
	ValidateConfiguration(ctx context.Context, config *ports.Configuration) error

	// GetConfiguration retrieves current configuration.
	GetConfiguration(ctx context.Context) (*ports.Configuration, error)
}

// RegistrationUseCase defines the application-level operations for service registration.
// This interface represents SPIRE service registration and management use cases.
type RegistrationUseCase interface {
	// RegisterService registers the service with SPIRE server.
	RegisterService(ctx context.Context, selectors []string) error

	// UnregisterService removes service registration from SPIRE.
	UnregisterService(ctx context.Context) error

	// GetRegistrationStatus checks service registration status.
	GetRegistrationStatus(ctx context.Context) (*domain.RegistrationStatus, error)
}
