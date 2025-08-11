// Package services provides core business logic services.
package services

import (
	"fmt"
	"sync"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// IdentityService manages service identities and enforces authentication at the transport layer.
//
// IDENTITY AUTHENTICATION ENFORCEMENT ARCHITECTURE:
// This service is the core component responsible for enforcing identity-based authentication
// in Ephemos. It coordinates between SPIFFE/SPIRE identity providers and transport layers
// to ensure all connections are cryptographically authenticated.
//
// Authentication Enforcement Flow:
// 1. SERVICE IDENTITY CREATION:
//   - CreateServerIdentity() obtains server's SPIFFE certificate from SPIRE
//   - CreateClientIdentity() obtains client's SPIFFE certificate from SPIRE
//   - Certificates contain SPIFFE IDs (e.g., spiffe://example.org/echo-server)
//
// 2. TRANSPORT-LAYER INTEGRATION:
//   - EstablishSecureConnection() configures mTLS transport with certificates
//   - Transport layer (gRPC/HTTP) handles TLS handshake and certificate verification
//   - Authentication happens BEFORE any application code runs
//
// 3. CERTIFICATE VALIDATION:
//   - ValidateServiceIdentity() ensures certificates are valid and not expired
//   - Certificates are automatically verified against SPIRE trust bundle
//   - Invalid/expired certificates cause immediate connection failure
//
// 4. AUTHORIZATION ENFORCEMENT:
//   - Service checks 'authorized_clients' config for server connections
//   - Service checks 'trusted_servers' config for client connections
//   - Unauthorized services are rejected at transport layer
//
// Security Properties:
// - Zero Trust: Every connection requires valid certificate authentication
// - Transport Layer: Authentication enforced below application code
// - Short-Lived: Certificates expire in 1 hour and auto-rotate
// - Mutual: Both client and server authenticate each other
// - Cryptographic: Uses X.509 certificates, not plaintext secrets
//
// It handles certificate management, identity validation, and secure connection establishment.
// The service caches validated identities for performance and thread-safety.
type IdentityService struct {
	identityProvider  ports.IdentityProvider
	transportProvider ports.TransportProvider
	config            *ports.Configuration
	cachedIdentity    *domain.ServiceIdentity
	mu                sync.RWMutex
}

// NewIdentityService creates a new IdentityService with the provided configuration.
// The configuration is validated and cached during initialization for better performance.
// Returns an error if the configuration is invalid.
func NewIdentityService(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	config *ports.Configuration,
) (*IdentityService, error) {
	if config == nil {
		return nil, &errors.ValidationError{
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
func (s *IdentityService) CreateServerIdentity() (ports.Server, error) {
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
func (s *IdentityService) CreateClientIdentity() (ports.Client, error) {
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
	cert, err := s.identityProvider.GetCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return cert, nil
}

// getTrustBundle retrieves the trust bundle from the identity provider.
func (s *IdentityService) getTrustBundle() (*domain.TrustBundle, error) {
	// In a pure domain service, we delegate to the port without exposing context
	// The adapter layer will handle context management
	bundle, err := s.identityProvider.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}
	return bundle, nil
}
