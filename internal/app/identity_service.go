// Package app provides application use cases and orchestration logic.
package app

import (
	"fmt"
	"sync"

	"github.com/sufield/ephemos/internal/domain"
)

// IdentityService manages service identities and provides authenticated transport.
// It handles certificate management, identity validation, and secure connection establishment.
// The service caches validated identities for performance and thread-safety.
type IdentityService struct {
	identityProvider  IdentityProvider
	transportProvider TransportProvider
	config            *Configuration
	cachedIdentity    *domain.ServiceIdentity
	mu                sync.RWMutex
}

// NewIdentityService creates a new IdentityService with the provided configuration.
// The configuration is validated and cached during initialization for better performance.
// Returns an error if the configuration is invalid.
func NewIdentityService(
	identityProvider IdentityProvider,
	transportProvider TransportProvider,
	config *Configuration,
) (*IdentityService, error) {
	if config == nil {
		return nil, &ValidationError{
			Field:   "config",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create and validate identity during initialization
	identity := domain.NewServiceIdentity(config.Service.Name, config.Service.Domain)
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service identity: %w", err)
	}

	return &IdentityService{
		identityProvider:  identityProvider,
		transportProvider: transportProvider,
		config:            config,
		cachedIdentity:    identity,
	}, nil
}

// CreateServerIdentity creates a server with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a configured server ready for service registration.
func (s *IdentityService) CreateServerIdentity() (Server, error) {
	s.mu.RLock()
	identity := s.cachedIdentity
	config := s.config
	s.mu.RUnlock()

	cert, err := s.getCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name, err)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name, err)
	}

	policy := domain.NewAuthenticationPolicy(identity)
	for _, client := range config.AuthorizedClients {
		policy.AddAuthorizedClient(client)
	}

	server, err := s.transportProvider.CreateServer(cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create server transport for service %s: %w", identity.Name, err)
	}

	return server, nil
}

// CreateClientIdentity creates a client connection with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a client ready for establishing secure connections to servers.
func (s *IdentityService) CreateClientIdentity() (Client, error) {
	s.mu.RLock()
	identity := s.cachedIdentity
	config := s.config
	s.mu.RUnlock()

	cert, err := s.getCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name, err)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name, err)
	}

	policy := domain.NewAuthenticationPolicy(identity)
	for _, server := range config.TrustedServers {
		policy.AddTrustedServer(server)
	}

	client, err := s.transportProvider.CreateClient(cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client transport for service %s: %w", identity.Name, err)
	}

	return client, nil
}

// getCertificate retrieves the certificate from the identity provider.
func (s *IdentityService) getCertificate() (*domain.Certificate, error) {
	// In a pure domain service, we delegate to the port without exposing context
	// The adapter layer will handle context management
	return s.identityProvider.GetCertificate()
}

// getTrustBundle retrieves the trust bundle from the identity provider.
func (s *IdentityService) getTrustBundle() (*domain.TrustBundle, error) {
	// In a pure domain service, we delegate to the port without exposing context
	// The adapter layer will handle context management
	return s.identityProvider.GetTrustBundle()
}
