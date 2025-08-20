// Package application provides use cases that orchestrate domain logic.
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// AuthenticationService handles authentication-related operations using port abstractions.
// This service orchestrates identity and trust bundle operations through injected ports,
// ensuring proper validation and invariant enforcement.
type AuthenticationService struct {
	identityProvider ports.IdentityProviderPort
	bundleProvider   ports.BundleProviderPort
	logger           *slog.Logger

	// Configuration
	expiryThreshold time.Duration // How soon before expiry to consider renewal
	maxRetries      int           // Maximum retry attempts for operations
}

// AuthenticationServiceConfig provides configuration for the AuthenticationService.
type AuthenticationServiceConfig struct {
	IdentityProvider ports.IdentityProviderPort
	BundleProvider   ports.BundleProviderPort
	Logger           *slog.Logger
	ExpiryThreshold  time.Duration
	MaxRetries       int
}

// NewAuthenticationService creates a new AuthenticationService with the provided configuration.
func NewAuthenticationService(config AuthenticationServiceConfig) (*AuthenticationService, error) {
	// Validate required dependencies
	if config.IdentityProvider == nil {
		return nil, fmt.Errorf("identity provider is required")
	}
	if config.BundleProvider == nil {
		return nil, fmt.Errorf("bundle provider is required")
	}

	// Set defaults
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	expiryThreshold := config.ExpiryThreshold
	if expiryThreshold == 0 {
		expiryThreshold = 5 * time.Minute // Default to 5 minutes before expiry
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3 // Default to 3 retries
	}

	return &AuthenticationService{
		identityProvider: config.IdentityProvider,
		bundleProvider:   config.BundleProvider,
		logger:           logger,
		expiryThreshold:  expiryThreshold,
		maxRetries:       maxRetries,
	}, nil
}

// GetValidatedSVID retrieves and validates the current service SVID.
// This method enforces invariants before returning the SVID to ensure it's valid and trusted.
func (s *AuthenticationService) GetValidatedSVID(ctx context.Context) (*x509svid.SVID, error) {
	s.logger.Debug("retrieving validated SVID")

	// Get SVID from provider
	svid, err := s.identityProvider.GetSVID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SVID: %w", err)
	}

	// Enforce invariant: SVID must not be nil
	if svid == nil {
		return nil, fmt.Errorf("identity provider returned nil SVID")
	}

	// Enforce invariant: SVID must have certificates
	if len(svid.Certificates) == 0 {
		return nil, fmt.Errorf("SVID has no certificates")
	}

	// Check if identity is expiring soon
	expiresAt := svid.Certificates[0].NotAfter
	if time.Until(expiresAt) < s.expiryThreshold {
		s.logger.Warn("SVID is expiring soon",
			"expires_at", expiresAt,
			"threshold", s.expiryThreshold)

		// Attempt to refresh the identity
		if err := s.refreshIdentityWithRetry(ctx); err != nil {
			s.logger.Error("failed to refresh expiring identity", "error", err)
			// Continue with existing identity if refresh fails
		} else {
			// Get the refreshed identity
			svid, err = s.identityProvider.GetSVID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get refreshed SVID: %w", err)
			}
		}
	}

	// Get trust bundle for validation
	trustBundle, err := s.bundleProvider.GetTrustBundle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	// Create domain certificate for validation
	cert, err := domain.NewCertificate(
		svid.Certificates[0],
		svid.PrivateKey,
		svid.Certificates[1:],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate from SVID: %w", err)
	}

	// Validate certificate against trust bundle
	if err := s.bundleProvider.ValidateCertificateAgainstBundle(ctx, cert); err != nil {
		return nil, fmt.Errorf("SVID validation against trust bundle failed: %w", err)
	}
	
	// Use trustBundle to avoid unused variable warning
	_ = trustBundle

	s.logger.Debug("successfully retrieved and validated SVID",
		"spiffe_id", svid.ID.String(),
		"valid_until", expiresAt)

	return svid, nil
}

// CreateAuthenticatedConnection creates a connection with proper authentication.
// This method ensures all authentication requirements are met before establishing a connection.
func (s *AuthenticationService) CreateAuthenticatedConnection(ctx context.Context, targetService string) (*AuthenticatedConnection, error) {
	s.logger.Debug("creating authenticated connection", "target", targetService)

	// Get validated SVID
	svid, err := s.GetValidatedSVID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validated SVID: %w", err)
	}

	// Create certificate from SVID
	cert, err := domain.NewCertificate(
		svid.Certificates[0],
		svid.PrivateKey,
		svid.Certificates[1:],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate from SVID: %w", err)
	}

	// Get trust bundle for the target's domain
	targetDomain, err := s.extractTrustDomain(targetService)
	if err != nil {
		return nil, fmt.Errorf("failed to extract trust domain from target: %w", err)
	}

	spiffeTrustDomain := targetDomain.ToSpiffeTrustDomain()

	x509Bundle, err := s.bundleProvider.GetTrustBundleForDomain(ctx, spiffeTrustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for domain %s: %w", targetDomain, err)
	}

	// Convert SDK bundle to domain bundle
	trustBundle, err := domain.NewTrustBundle(x509Bundle.X509Authorities())
	if err != nil {
		return nil, fmt.Errorf("failed to convert trust bundle: %w", err)
	}

	// Authentication-only scope: no specific target identity parsing needed

	// Create authentication policy
	policy := &domain.AuthenticationPolicy{
		TrustDomain: targetDomain,
		RequireAuth: true,
		// Authentication-only scope: no specific identity authorization
	}

	return &AuthenticatedConnection{
		Certificate:   cert,
		TrustBundle:   trustBundle,
		Policy:        policy,
		TargetService: targetService,
	}, nil
}

// ValidatePeerIdentity validates a peer's identity during authentication.
// This method enforces security invariants for peer authentication.
func (s *AuthenticationService) ValidatePeerIdentity(ctx context.Context, peerCert *domain.Certificate, expectedIdentity string) error {
	s.logger.Debug("validating peer identity", "expected", expectedIdentity)

	// Enforce invariant: peer certificate must not be nil
	if peerCert == nil {
		return fmt.Errorf("peer certificate is nil")
	}

	// Validate certificate structure
	if err := peerCert.Validate(domain.CertValidationOptions{}); err != nil {
		return fmt.Errorf("peer certificate validation failed: %w", err)
	}

	// Validate certificate against trust bundle
	if err := s.bundleProvider.ValidateCertificateAgainstBundle(ctx, peerCert); err != nil {
		return fmt.Errorf("peer certificate not trusted: %w", err)
	}

	// Extract and validate SPIFFE ID
	spiffeID, err := peerCert.ToSPIFFEID()
	if err != nil {
		return fmt.Errorf("failed to extract SPIFFE ID from peer certificate: %w", err)
	}

	// Verify the SPIFFE ID matches expected identity
	if spiffeID.String() != expectedIdentity {
		return fmt.Errorf("peer identity mismatch: expected %s, got %s", expectedIdentity, spiffeID.String())
	}

	// Check certificate expiry
	if peerCert.Cert != nil && time.Now().After(peerCert.Cert.NotAfter) {
		return fmt.Errorf("peer certificate has expired")
	}

	s.logger.Debug("peer identity validated successfully", "identity", spiffeID.String())
	return nil
}

// RefreshCredentials refreshes both identity and trust bundle.
// This method ensures credentials are up-to-date for authentication operations.
func (s *AuthenticationService) RefreshCredentials(ctx context.Context) error {
	s.logger.Info("refreshing authentication credentials")

	// Refresh identity with retry logic
	if err := s.refreshIdentityWithRetry(ctx); err != nil {
		return fmt.Errorf("failed to refresh identity: %w", err)
	}

	// Refresh trust bundle with retry logic
	if err := s.refreshTrustBundleWithRetry(ctx); err != nil {
		return fmt.Errorf("failed to refresh trust bundle: %w", err)
	}

	s.logger.Info("successfully refreshed authentication credentials")
	return nil
}

// refreshIdentityWithRetry attempts to refresh identity with retry logic.
func (s *AuthenticationService) refreshIdentityWithRetry(ctx context.Context) error {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		if err := s.identityProvider.RefreshIdentity(ctx); err != nil {
			lastErr = err
			s.logger.Warn("identity refresh attempt failed",
				"attempt", i+1,
				"max_retries", s.maxRetries,
				"error", err)

			// Exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<uint(i)) * time.Second):
				continue
			}
		}
		return nil // Success
	}
	return fmt.Errorf("failed after %d retries: %w", s.maxRetries, lastErr)
}

// refreshTrustBundleWithRetry attempts to refresh trust bundle with retry logic.
func (s *AuthenticationService) refreshTrustBundleWithRetry(ctx context.Context) error {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		if err := s.bundleProvider.RefreshTrustBundle(ctx); err != nil {
			lastErr = err
			s.logger.Warn("trust bundle refresh attempt failed",
				"attempt", i+1,
				"max_retries", s.maxRetries,
				"error", err)

			// Exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<uint(i)) * time.Second):
				continue
			}
		}
		return nil // Success
	}
	return fmt.Errorf("failed after %d retries: %w", s.maxRetries, lastErr)
}

// extractTrustDomain extracts the trust domain from a service identifier.
func (s *AuthenticationService) extractTrustDomain(serviceIdentifier string) (domain.TrustDomain, error) {
	// Parse as SPIFFE ID to extract trust domain
	namespace, err := domain.NewIdentityNamespaceFromString(serviceIdentifier)
	if err != nil {
		return domain.TrustDomain{}, fmt.Errorf("failed to parse service identifier: %w", err)
	}

	return namespace.GetTrustDomain(), nil
}

// parseAsIdentityNamespace parses a string as an identity namespace.
func (s *AuthenticationService) parseAsIdentityNamespace(identifier string) (domain.IdentityNamespace, error) {
	return domain.NewIdentityNamespaceFromString(identifier)
}

// AuthenticatedConnection represents a connection with authentication information.
type AuthenticatedConnection struct {
	Certificate   *domain.Certificate
	TrustBundle   *domain.TrustBundle
	Policy        *domain.AuthenticationPolicy
	TargetService string
}
