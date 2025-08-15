// Package services provides core business logic services.
package services

import (
	"fmt"
	"sync"
	"time"

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
//   - ValidateServiceIdentity() explicitly checks certificate validity and expiry
//   - Validates SPIFFE ID matches expected service identity
//   - Warns when certificates are near expiry (within 1 hour)
//   - Invalid/expired certificates cause immediate connection failure
//
// 4. AUTHORIZATION ENFORCEMENT:
//   - Checks 'authorized_clients' config for server-side authorization
//   - Checks 'trusted_servers' config for client-side authorization  
//   - Creates AuthorizationPolicy when rules are configured
//   - Falls back to trust domain authorization when no explicit rules
//   - Unauthorized services are rejected at transport layer
//
// 5. CERTIFICATE ROTATION:
//   - Caches certificates with TTL to avoid excessive fetching
//   - Automatically refreshes expired cached certificates
//   - Validates certificates on each fetch to ensure freshness
//
// Security Properties:
// - Zero Trust: Every connection requires valid certificate authentication
// - Transport Layer: Authentication enforced below application code
// - Short-Lived: Certificates expire in 1 hour and auto-rotate
// - Mutual: Both client and server authenticate each other
// - Cryptographic: Uses X.509 certificates, not plaintext secrets
//
// It handles certificate management, identity validation, and secure connection establishment.
// The service caches validated identities and certificates for performance and thread-safety.
type IdentityService struct {
	identityProvider  ports.IdentityProvider
	transportProvider ports.TransportProvider
	config            *ports.Configuration
	cachedIdentity    *domain.ServiceIdentity
	
	// Certificate caching for rotation support
	cachedCert       *domain.Certificate
	cachedBundle     *domain.TrustBundle
	certCachedAt     time.Time
	bundleCachedAt   time.Time
	cacheTTL         time.Duration
	
	mu               sync.RWMutex
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
		cacheTTL:          30 * time.Minute, // Cache certificates for 30 minutes (half of typical 1-hour SPIFFE cert lifetime)
	}, nil
}

// CreateServerIdentity creates a server with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a configured server ready for service registration.
func (s *IdentityService) CreateServerIdentity() (ports.ServerPort, error) {
	s.mu.RLock()
	identity := s.cachedIdentity
	s.mu.RUnlock()

	cert, err := s.getCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name(), err)
	}

	// Explicitly validate certificate as described in architecture documentation
	if err := s.ValidateServiceIdentity(cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed for service %s: %w", identity.Name(), err)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name(), err)
	}

	// Create authentication policy with authorization based on configuration
	var policy *domain.AuthenticationPolicy
	if len(s.config.Service.AuthorizedClients) > 0 || len(s.config.Service.TrustedServers) > 0 {
		// Use authorization policy when rules are configured
		policy = domain.NewAuthorizationPolicy(identity, s.config.Service.AuthorizedClients, s.config.Service.TrustedServers)
	} else {
		// Fall back to authentication-only policy
		policy = domain.NewAuthenticationPolicy(identity)
	}

	server, err := s.transportProvider.CreateServer(cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create server transport for service %s: %w", identity.Name(), err)
	}

	return server, nil
}

// CreateClientIdentity creates a client connection with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a client ready for establishing secure connections to servers.
func (s *IdentityService) CreateClientIdentity() (ports.ClientPort, error) {
	s.mu.RLock()
	identity := s.cachedIdentity
	s.mu.RUnlock()

	cert, err := s.getCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name(), err)
	}

	// Explicitly validate certificate as described in architecture documentation
	if err := s.ValidateServiceIdentity(cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed for service %s: %w", identity.Name(), err)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name(), err)
	}

	// Create authentication policy with authorization based on configuration
	var policy *domain.AuthenticationPolicy
	if len(s.config.Service.AuthorizedClients) > 0 || len(s.config.Service.TrustedServers) > 0 {
		// Use authorization policy when rules are configured
		policy = domain.NewAuthorizationPolicy(identity, s.config.Service.AuthorizedClients, s.config.Service.TrustedServers)
	} else {
		// Fall back to authentication-only policy
		policy = domain.NewAuthenticationPolicy(identity)
	}

	client, err := s.transportProvider.CreateClient(cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client transport for service %s: %w", identity.Name(), err)
	}

	return client, nil
}

// getCertificate retrieves the certificate from the identity provider with TTL-based caching.
func (s *IdentityService) getCertificate() (*domain.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if cached certificate is still valid
	if s.cachedCert != nil && time.Since(s.certCachedAt) < s.cacheTTL {
		// Validate the cached certificate is not expired
		if err := s.validateCertificateExpiry(s.cachedCert); err == nil {
			return s.cachedCert, nil
		}
		// Certificate is expired, clear cache and fetch new one
		s.cachedCert = nil
	}
	
	// Fetch fresh certificate from provider
	cert, err := s.identityProvider.GetCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	
	// Cache the new certificate
	s.cachedCert = cert
	s.certCachedAt = time.Now()
	
	return cert, nil
}

// validateCertificateExpiry checks if a certificate is still valid (not expired).
func (s *IdentityService) validateCertificateExpiry(cert *domain.Certificate) error {
	if cert == nil || cert.Cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	
	now := time.Now()
	if now.After(cert.Cert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}
	
	return nil
}

// getTrustBundle retrieves the trust bundle from the identity provider with TTL-based caching.
func (s *IdentityService) getTrustBundle() (*domain.TrustBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if cached trust bundle is still valid
	if s.cachedBundle != nil && time.Since(s.bundleCachedAt) < s.cacheTTL {
		return s.cachedBundle, nil
	}
	
	// Fetch fresh trust bundle from provider
	bundle, err := s.identityProvider.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}
	
	// Cache the new trust bundle
	s.cachedBundle = bundle
	s.bundleCachedAt = time.Now()
	
	return bundle, nil
}

// GetCertificate retrieves the certificate from the identity provider.
// This is exposed for use by HTTP client connections to get SPIFFE certificates.
func (s *IdentityService) GetCertificate() (*domain.Certificate, error) {
	return s.getCertificate()
}

// GetTrustBundle retrieves the trust bundle from the identity provider.
// This is exposed for use by HTTP client connections to get SPIFFE trust bundles.
func (s *IdentityService) GetTrustBundle() (*domain.TrustBundle, error) {
	return s.getTrustBundle()
}

// ValidateServiceIdentity validates that a certificate is valid, not expired, and matches expected identity.
// This ensures certificates are cryptographically valid and within their validity period.
func (s *IdentityService) ValidateServiceIdentity(cert *domain.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	if cert.Cert == nil {
		return fmt.Errorf("X.509 certificate is nil")
	}

	// Check certificate validity period
	now := time.Now()
	if now.Before(cert.Cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (NotBefore: %v, now: %v)", 
			cert.Cert.NotBefore, now)
	}

	if now.After(cert.Cert.NotAfter) {
		return fmt.Errorf("certificate has expired (NotAfter: %v, now: %v)", 
			cert.Cert.NotAfter, now)
	}

	// Warn if certificate expires within the next hour (SPIFFE certs typically have 1-hour TTL)
	if now.Add(time.Hour).After(cert.Cert.NotAfter) {
		// In production, this should use structured logging
		fmt.Printf("Warning: Certificate expires soon (NotAfter: %v, now: %v)\n", 
			cert.Cert.NotAfter, now)
	}

	// Validate SPIFFE ID matches expected service identity
	s.mu.RLock()
	expectedSPIFFEID := s.cachedIdentity.URI()
	s.mu.RUnlock()

	// Extract SPIFFE ID from certificate
	var certSPIFFEID string
	for _, uri := range cert.Cert.URIs {
		if uri.Scheme == "spiffe" {
			certSPIFFEID = uri.String()
			break
		}
	}

	if certSPIFFEID == "" {
		return fmt.Errorf("certificate contains no SPIFFE ID in URI SANs")
	}

	if certSPIFFEID != expectedSPIFFEID {
		return fmt.Errorf("certificate SPIFFE ID mismatch: expected %s, got %s", 
			expectedSPIFFEID, certSPIFFEID)
	}

	return nil
}
