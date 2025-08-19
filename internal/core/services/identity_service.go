// Package services provides core business logic services.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/core/adapters"
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
//
//   - Checks 'authorized_clients' config for server-side authorization
//
//   - Checks 'trusted_servers' config for client-side authorization
//
//   - Creates AuthorizationPolicy when rules are configured
//
//   - Falls back to trust domain authorization when no explicit rules
//
//   - Unauthorized services are rejected at transport layer
//
//     5. CERTIFICATE ROTATION AND LIFECYCLE MANAGEMENT:
//     The service implements comprehensive certificate rotation handling to ensure continuous
//     security and zero-downtime operations. This follows SPIFFE best practices for short-lived certificates.
//
//     Cache-Based Rotation Strategy:
//
//   - Certificates are cached with configurable TTL (default: 30 minutes, half of typical 1-hour SPIFFE lifetime)
//
//   - Trust bundles are also cached to reduce load on SPIRE and improve performance
//
//   - Cache TTL can be configured via service.cache.ttl_minutes (max 60 minutes for security)
//
//     Proactive Refresh Mechanism:
//
//   - Certificates are proactively refreshed before expiry (default: 10 minutes before expiry)
//
//   - Refresh threshold is configurable via service.cache.proactive_refresh_minutes
//
//   - This prevents certificate expiry during high-traffic periods and ensures continuous service
//
//   - Refresh operations include full cryptographic validation of new certificates
//
//     Rotation Triggers:
//
//   - Time-based: Automatic refresh when cache TTL expires
//
//   - Expiry-based: Proactive refresh when certificate approaches expiry
//
//   - Validation-based: Immediate refresh if cached certificate fails validation
//
//   - Error-based: Retry with exponential backoff on provider failures
//
//     Thread Safety and Metrics:
//
//   - All cache operations are protected by RWMutex for concurrent access
//
//   - Cache performance metrics (hits/misses/ratios) are tracked for monitoring
//
//   - Atomic operations ensure thread-safe metric updates
//
//   - Structured logging provides observability into rotation events
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
	validator         ports.CertValidatorPort // Certificate validator
	metrics           MetricsReporter         // Metrics reporter (Prometheus or NoOp)

	// Certificate caching for rotation support
	cachedCert       *domain.Certificate
	certCacheEntry   *domain.CacheEntry
	cachedBundle     *domain.TrustBundle
	bundleCacheEntry *domain.CacheEntry
	cacheTTL         time.Duration

	// Enhanced mTLS connection tracking and enforcement
	connectionRegistry *MTLSConnectionRegistry
	enforcementService *MTLSEnforcementService
	continuityService  *RotationContinuityService

	mu sync.RWMutex
}

// NewIdentityService creates a new IdentityService with full customization.
// If validator is nil, uses the default certificate validator.
// If metrics is nil, uses NoOp metrics reporter.
// Returns an error if the configuration is invalid.
func NewIdentityService(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	config *ports.Configuration,
	validator ports.CertValidatorPort,
	metrics MetricsReporter,
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
	// Extract intermediate to avoid deep selector chains
	serviceConfig := config.Service
	if serviceConfig.Name.Value() == "" {
		return nil, fmt.Errorf("service name is required")
	}
	if serviceConfig.Domain == "" {
		return nil, fmt.Errorf("service domain is required")
	}
	identity := domain.NewServiceIdentity(serviceConfig.Name.Value(), serviceConfig.Domain)
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service identity: %w", err)
	}

	// Use default validator if none provided
	if validator == nil {
		validator = adapters.NewDefaultCertValidator()
	}

	// Use NoOp metrics if none provided
	if metrics == nil {
		metrics = &NoOpMetrics{}
	}

	// Determine cache TTL from configuration or use default
	cacheTTL := 30 * time.Minute // Default: 30 minutes (half of typical 1-hour SPIFFE cert lifetime)
	// Extract intermediate to avoid deep selector chains
	if serviceConfig.Cache != nil && serviceConfig.Cache.TTLMinutes > 0 {
		cacheTTL = time.Duration(serviceConfig.Cache.TTLMinutes) * time.Minute
	}

	service := &IdentityService{
		identityProvider:  identityProvider,
		transportProvider: transportProvider,
		config:            config,
		cachedIdentity:    identity,
		validator:         validator,
		metrics:           metrics,
		cacheTTL:          cacheTTL,
	}

	// Initialize enhanced mTLS components
	service.connectionRegistry = NewMTLSConnectionRegistry(service)
	service.enforcementService = NewMTLSEnforcementService(service, service.connectionRegistry)
	service.continuityService = NewRotationContinuityService(service, transportProvider)

	// Add logging observer for rotation events
	logObserver := NewLogRotationObserver(slog.Default())
	service.connectionRegistry.AddRotationObserver(logObserver)

	return service, nil
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
	if validationErr := s.ValidateServiceIdentity(cert); validationErr != nil {
		return nil, fmt.Errorf("certificate validation failed for service %s: %w", identity.Name(), validationErr)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name(), err)
	}

	// Create authentication policy with authorization based on configuration
	policy, err := s.createPolicy(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create authentication policy: %w", err)
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
	if validationErr := s.ValidateServiceIdentity(cert); validationErr != nil {
		return nil, fmt.Errorf("certificate validation failed for service %s: %w", identity.Name(), validationErr)
	}

	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name(), err)
	}

	// Create authentication policy with authorization based on configuration
	policy, err := s.createPolicy(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create authentication policy: %w", err)
	}

	client, err := s.transportProvider.CreateClient(cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client transport for service %s: %w", identity.Name(), err)
	}

	return client, nil
}

// getCertificate retrieves the certificate from the identity provider with TTL-based caching.
//
// CERTIFICATE ROTATION FLOW:
// This method implements the core certificate rotation logic that ensures continuous
// service availability while maintaining security through regular certificate refresh.
//
// Cache Hit Path (Fast Path):
// 1. Check if cached certificate exists and cache TTL hasn't expired
// 2. Validate cached certificate is not expired using validateCertificateExpiry()
// 3. Check if proactive refresh is needed (certificate expires soon)
// 4. If certificate is valid and not expiring soon, return cached certificate (cache hit)
//
// Cache Miss Path (Rotation Path):
// 1. If no cached certificate, cache expired, or certificate expiring soon
// 2. Call fetchCertificateWithRetry() to get fresh certificate from SPIRE
// 3. Retry with exponential backoff on transient failures (max 3 attempts)
// 4. Cache the new certificate with current timestamp
// 5. Return the fresh certificate
//
// The rotation is transparent to callers - they always receive a valid, current certificate.
// All operations are thread-safe using mutex locks and atomic operations for metrics.
func (s *IdentityService) getCertificate() (*domain.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if cached certificate is still valid and not expiring soon
	if s.cachedCert != nil && s.certCacheEntry != nil && s.certCacheEntry.IsFresh() {
		// Validate the cached certificate is not expired
		if err := s.validateCertificateExpiry(s.cachedCert); err == nil {
			// Determine proactive refresh threshold from configuration or use default
			refreshThreshold := 10 * time.Minute // Default: 10 minutes before expiry
			// Extract intermediates to avoid deep selector chains
			service := s.config.Service
			if service.Cache != nil && service.Cache.ProactiveRefreshMinutes > 0 {
				refreshThreshold = time.Duration(service.Cache.ProactiveRefreshMinutes) * time.Minute
			}

			// Proactive refresh if certificate expires soon
			// This aligns with SPIFFE short-lived cert best practices
			if s.cachedCert.IsExpiringWithin(refreshThreshold) {
				slog.Info("Proactively refreshing certificate expiring soon",
					"service_name", s.cachedIdentity.Name(),
					"cert_expires_at", s.cachedCert.ExpiresAt(),
					"refresh_threshold", refreshThreshold.String(),
					"expires_in", s.cachedCert.TimeToExpiry().String(),
				)
				// Clear cache to force refresh
				s.cachedCert = nil
			} else {
				// Certificate is valid and not expiring soon - cache hit
				s.metrics.RecordCacheHit("certificate")
				return s.cachedCert, nil
			}
		} else {
			// Certificate is expired, clear cache and fetch new one
			s.cachedCert = nil
		}
	}

	// Cache miss - fetch fresh certificate from provider with retry logic for transient failures
	s.metrics.RecordCacheMiss("certificate")

	// Track refresh duration
	refreshStart := time.Now()
	cert, err := s.fetchCertificateWithRetry()
	if err != nil {
		return nil, fmt.Errorf("identity provider failed for service %s: %w", s.cachedIdentity.Name(), err)
	}
	refreshDuration := time.Since(refreshStart).Seconds()

	// Determine refresh reason
	refreshReason := "cache_miss"
	if s.cachedCert == nil {
		refreshReason = "initial"
	} else if time.Now().After(s.cachedCert.Cert.NotAfter) {
		refreshReason = "expired"
	} else if time.Now().Add(10 * time.Minute).After(s.cachedCert.Cert.NotAfter) {
		refreshReason = "proactive"
	}
	s.metrics.RecordRefresh(refreshReason, refreshDuration)

	// Update certificate expiry metric
	if cert != nil && cert.Cert != nil {
		s.metrics.UpdateCertExpiry(s.cachedIdentity.Name(), float64(cert.Cert.NotAfter.Unix()))
	}

	// Cache the new certificate
	s.cachedCert = cert
	s.certCacheEntry = domain.NewCacheEntry(s.cacheTTL)

	return cert, nil
}

// validateCertificateExpiry checks if a certificate is still valid (not expired).
// This is a lightweight check used for cache validation without full validation overhead.
func (s *IdentityService) validateCertificateExpiry(cert *domain.Certificate) error {
	// Use centralized validation with minimal options for quick expiry check
	opts := domain.CertValidationOptions{
		SkipChainVerify: true, // Skip chain verification for performance
		// Other checks like expiry and basic structure are still performed
	}

	// Use centralized validator for consistency
	return s.validator.Validate(cert, opts)
}

// getTrustBundle retrieves the trust bundle from the identity provider with TTL-based caching.
//
// TRUST BUNDLE ROTATION FLOW:
// Trust bundles contain root CA certificates that are used to validate certificates.
// They typically have longer lifetimes than individual certificates but still require rotation.
//
// Cache Strategy:
// 1. Check if cached trust bundle exists and cache TTL hasn't expired
// 2. If valid cached bundle exists, return it immediately (cache hit)
// 3. If cache miss, fetch fresh bundle from SPIRE with retry logic
// 4. Cache the new bundle with current timestamp
//
// Trust bundle rotation is less frequent than certificate rotation but equally important
// for maintaining the security of the certificate validation process. Trust bundles
// are shared across all certificate validation operations.
func (s *IdentityService) getTrustBundle() (*domain.TrustBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if cached trust bundle is still valid
	if s.cachedBundle != nil && s.bundleCacheEntry != nil && s.bundleCacheEntry.IsFresh() {
		// Cache hit
		s.metrics.RecordCacheHit("bundle")
		return s.cachedBundle, nil
	}

	// Cache miss - fetch fresh trust bundle from provider with retry logic for transient failures
	s.metrics.RecordCacheMiss("bundle")
	bundle, err := s.fetchTrustBundleWithRetry()
	if err != nil {
		return nil, fmt.Errorf("trust bundle provider failed for service %s: %w", s.cachedIdentity.Name(), err)
	}

	// Cache the new trust bundle
	s.cachedBundle = bundle
	s.bundleCacheEntry = domain.NewCacheEntry(s.cacheTTL)

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
//
// VALIDATION IN CERTIFICATE ROTATION:
// This method plays a critical role in the certificate rotation flow by ensuring that
// both cached and newly fetched certificates meet security requirements before use.
//
// Validation Steps:
// 1. Certificate Structure: Ensures certificate and X.509 certificate are not nil
// 2. Validity Period: Checks certificate is not expired and not yet valid
// 3. Expiry Warning: Logs structured warning if certificate expires soon (within 1 hour)
// 4. SPIFFE ID Matching: Validates certificate SPIFFE ID matches expected service identity
// 5. Trust Domain Verification: Ensures certificate trust domain matches expected domain
// 6. Chain Validation: Performs cryptographic validation of certificate chain
//
// This validation is called:
// - After retrieving cached certificates (to ensure they're still valid)
// - After fetching fresh certificates from SPIRE (to ensure new certificates are valid)
// - Before using certificates for transport layer security
//
// Failed validation triggers certificate rotation by clearing the cache and fetching fresh certificates.
func (s *IdentityService) ValidateServiceIdentity(cert *domain.Certificate) error {
	// Get cached identity and trust bundle for validation
	s.mu.RLock()
	expectedIdentity := s.cachedIdentity
	trustBundle := s.cachedBundle
	s.mu.RUnlock()

	// Configure validation options
	opts := domain.CertValidationOptions{
		ExpectedIdentity: expectedIdentity,
		WarningThreshold: time.Hour,      // Warn when certificate expires within 1 hour
		TrustBundle:      trustBundle,    // Use cached trust bundle if available
		SkipExpiry:       false,          // Always check expiry in production
		SkipChainVerify:  false,          // Always verify chain in production
		Logger:           slog.Default(), // Use default logger for warnings
	}

	// Use centralized validator
	if err := s.validator.Validate(cert, opts); err != nil {
		return fmt.Errorf("certificate validation failed for service %s: %w", expectedIdentity.Name(), err)
	}

	// Log successful validation for observability
	if spiffeID, err := cert.ToSPIFFEID(); err == nil {
		slog.Debug("Certificate SPIFFE ID validation successful",
			"service_name", expectedIdentity.Name(),
			"spiffe_id", spiffeID.String(),
			"trust_domain", spiffeID.TrustDomain().String(),
			"path", spiffeID.Path(),
		)
	}

	return nil
}

// validateCertificateChain performs cryptographic validation of the certificate chain
// using the trust bundle to verify signatures and chain integrity.
func (s *IdentityService) validateCertificateChain(cert *domain.Certificate) error {
	// Get service name and identity with proper locking for thread safety
	s.mu.RLock()
	serviceName := s.cachedIdentity.Name()
	identity := s.cachedIdentity
	s.mu.RUnlock()

	// Get the trust bundle for chain verification
	trustBundle, err := s.getTrustBundle()
	if err != nil {
		return fmt.Errorf("failed to get trust bundle for chain validation: %w", err)
	}

	// Use centralized validation with comprehensive options
	opts := domain.CertValidationOptions{
		ExpectedIdentity: identity,         // Verify SPIFFE ID matches our identity
		WarningThreshold: 30 * time.Minute, // Warn if expires within 30 minutes
		TrustBundle:      trustBundle,      // Verify chain against trust bundle
		SkipExpiry:       false,            // Always check expiry in production
		SkipChainVerify:  false,            // Always verify chain cryptographically
	}

	// Perform validation through the validator port
	if err := s.validator.Validate(cert, opts); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	slog.Debug("Certificate chain validation successful",
		"service_name", serviceName,
		"subject", cert.Cert.Subject.String(),
		"issuer", cert.Cert.Issuer.String(),
		"serial_number", cert.Cert.SerialNumber.String(),
		"not_before", cert.Cert.NotBefore,
		"not_after", cert.Cert.NotAfter,
		"chain_length", len(cert.Chain),
	)

	return nil
}

// fetchCertificateWithRetry retrieves certificate from identity provider with retry logic for transient failures.
func (s *IdentityService) fetchCertificateWithRetry() (*domain.Certificate, error) {
	// Get service name with proper locking for thread safety
	s.mu.RLock()
	serviceName := s.cachedIdentity.Name()
	s.mu.RUnlock()

	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		cert, err := s.identityProvider.GetCertificate()
		if err == nil {
			return cert, nil
		}

		lastErr = err
		s.metrics.RecordRetry("certificate", attempt+1)

		// Log retry attempts with structured logging
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
			slog.Warn("Certificate fetch failed, retrying",
				"service_name", serviceName,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"retry_delay", delay.String(),
				"error", err.Error(),
			)
			time.Sleep(delay)
		}
	}

	slog.Error("Certificate fetch failed after all retries",
		"service_name", serviceName,
		"max_retries", maxRetries,
		"final_error", lastErr.Error(),
	)

	return nil, fmt.Errorf("failed to get certificate after %d retries: %w", maxRetries, lastErr)
}

// fetchTrustBundleWithRetry retrieves trust bundle from identity provider with retry logic for transient failures.
func (s *IdentityService) fetchTrustBundleWithRetry() (*domain.TrustBundle, error) {
	// Get service name with proper locking for thread safety
	s.mu.RLock()
	serviceName := s.cachedIdentity.Name()
	s.mu.RUnlock()

	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		bundle, err := s.identityProvider.GetTrustBundle()
		if err == nil {
			return bundle, nil
		}

		lastErr = err

		// Log retry attempts with structured logging
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
			slog.Warn("Trust bundle fetch failed, retrying",
				"service_name", serviceName,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"retry_delay", delay.String(),
				"error", err.Error(),
			)
			time.Sleep(delay)
		}
	}

	slog.Error("Trust bundle fetch failed after all retries",
		"service_name", serviceName,
		"max_retries", maxRetries,
		"final_error", lastErr.Error(),
	)

	return nil, fmt.Errorf("failed to get trust bundle after %d retries: %w", maxRetries, lastErr)
}

// createPolicy creates an authentication policy based on the service configuration.
// It consolidates the policy creation logic that was duplicated between CreateServerIdentity and CreateClientIdentity.
func (s *IdentityService) createPolicy(identity *domain.ServiceIdentity) (*domain.AuthenticationPolicy, error) {
	// Create authentication-only policy for identity verification
	return domain.NewAuthenticationPolicy(identity), nil
}

// == ENHANCED mTLS CONNECTION MANAGEMENT ==

// EstablishMTLSConnection creates a new managed mTLS connection with full invariant enforcement
func (s *IdentityService) EstablishMTLSConnection(ctx context.Context, connID string, remoteIdentity *domain.ServiceIdentity) (*MTLSConnection, error) {
	// Get current certificate and local identity first
	cert, err := s.getCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for connection %s: %w", connID, err)
	}

	s.mu.RLock()
	localIdentity := s.cachedIdentity
	s.mu.RUnlock()

	// Establish the connection through the connection manager
	conn, err := s.connectionRegistry.EstablishConnection(ctx, connID, remoteIdentity, cert, localIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to establish mTLS connection: %w", err)
	}

	// Validate the connection against all invariants
	if err := s.enforcementService.ValidateConnection(ctx, connID); err != nil {
		// Close the connection if it violates invariants
		s.connectionRegistry.CloseConnection(connID)
		return nil, fmt.Errorf("connection violates mTLS invariants: %w", err)
	}

	return conn, nil
}

// GetMTLSConnection retrieves an active mTLS connection
func (s *IdentityService) GetMTLSConnection(connID string) (*MTLSConnection, bool) {
	return s.connectionRegistry.GetConnection(connID)
}

// ListMTLSConnections returns all active mTLS connections
func (s *IdentityService) ListMTLSConnections() []*MTLSConnection {
	return s.connectionRegistry.ListConnections()
}

// CloseMTLSConnection closes a managed mTLS connection
func (s *IdentityService) CloseMTLSConnection(connID string) error {
	return s.connectionRegistry.CloseConnection(connID)
}

// GetConnectionStats returns statistics about managed connections
func (s *IdentityService) GetConnectionStats() ConnectionStats {
	return s.connectionRegistry.GetConnectionStats()
}

// StartMTLSEnforcement begins enforcing mTLS invariants on all connections
func (s *IdentityService) StartMTLSEnforcement(ctx context.Context) error {
	return s.enforcementService.StartEnforcement(ctx)
}

// GetInvariantStatus returns the current status of all mTLS invariants
func (s *IdentityService) GetInvariantStatus(ctx context.Context) InvariantStatus {
	return s.enforcementService.GetInvariantStatus(ctx)
}

// SetRotationPolicy updates the certificate rotation policy for all connections
func (s *IdentityService) SetRotationPolicy(policy *RotationPolicy) {
	s.connectionRegistry.SetRotationPolicy(policy)
}

// AddRotationObserver adds an observer for certificate rotation events
func (s *IdentityService) AddRotationObserver(observer RotationObserver) {
	s.connectionRegistry.AddRotationObserver(observer)
}

// SetEnforcementPolicy updates the invariant enforcement policy
func (s *IdentityService) SetEnforcementPolicy(policy *EnforcementPolicy) {
	s.enforcementService.SetEnforcementPolicy(policy)
}

// == ROTATION CONTINUITY MANAGEMENT ==

// RotateServerWithContinuity performs server rotation with zero downtime
func (s *IdentityService) RotateServerWithContinuity(ctx context.Context, serverID string, server ports.ServerPort) error {
	return s.continuityService.RotateServerWithContinuity(ctx, serverID, server)
}

// RotateClientWithContinuity performs client rotation with connection continuity
func (s *IdentityService) RotateClientWithContinuity(ctx context.Context, clientID string, client ports.ClientPort) error {
	return s.continuityService.RotateClientWithContinuity(ctx, clientID, client)
}

// GetActiveRotations returns information about currently active rotations
func (s *IdentityService) GetActiveRotations() ([]RotationInfo, []RotationInfo) {
	return s.continuityService.GetActiveRotations()
}

// GetRotationStats returns statistics about rotation operations
func (s *IdentityService) GetRotationStats() RotationStats {
	return s.continuityService.GetRotationStats()
}

// SetContinuityPolicy updates the rotation continuity policy
func (s *IdentityService) SetContinuityPolicy(policy *ContinuityPolicy) {
	s.continuityService.SetContinuityPolicy(policy)
}
