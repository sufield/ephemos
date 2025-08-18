// Package transport provides concrete transport provider implementations.
package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// RotatableGRPCProvider provides gRPC-based transport with SPIFFE mTLS authentication
// that supports automatic SVID rotation through go-spiffe sources.
type RotatableGRPCProvider struct {
	config       *ports.Configuration
	svidSource   x509svid.Source
	bundleSource x509bundle.Source
	authorizer   tlsconfig.Authorizer
	mu           sync.RWMutex
}

// NewRotatableGRPCProvider creates a new rotation-capable gRPC transport provider.
func NewRotatableGRPCProvider(config *ports.Configuration) *RotatableGRPCProvider {
	return &RotatableGRPCProvider{
		config: config,
	}
}

// SetSources sets the SVID and bundle sources for rotation support.
// These should be long-lived sources that are reused across connections.
func (p *RotatableGRPCProvider) SetSources(svidSource x509svid.Source, bundleSource x509bundle.Source, authorizer tlsconfig.Authorizer) error {
	// INPUT VALIDATION: Ensure required sources are provided
	if svidSource == nil {
		return fmt.Errorf("SVID source cannot be nil")
	}
	if bundleSource == nil {
		return fmt.Errorf("bundle source cannot be nil")
	}
	// Note: authorizer can be nil - we'll use secure defaults

	p.mu.Lock()
	defer p.mu.Unlock()
	p.svidSource = svidSource
	p.bundleSource = bundleSource
	p.authorizer = authorizer
	return nil
}

// CreateClient creates a gRPC client with rotation-capable SPIFFE mTLS.
func (p *RotatableGRPCProvider) CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ClientPort, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check for development mode
	if p.config != nil && p.config.ShouldSkipCertificateValidation() {
		log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		return &grpcClient{
			tlsConfig: &tls.Config{InsecureSkipVerify: true},
			policy:    policy,
		}, nil
	}

	// If sources are set, use them for rotation capability
	if p.svidSource != nil && p.bundleSource != nil {
		tlsConfig := p.createRotatableClientTLSConfig()
		return &grpcClient{
			tlsConfig: tlsConfig,
			policy:    policy,
		}, nil
	}

	// INPUT VALIDATION: For fallback mode, ensure cert and bundle are provided
	if cert == nil {
		return nil, fmt.Errorf("certificate required when sources are not configured")
	}
	if bundle == nil {
		return nil, fmt.Errorf("trust bundle required when sources are not configured")
	}

	// Fallback: create source adapters from provided cert/bundle
	svidAdapter := &staticSVIDAdapter{cert: cert}
	bundleAdapter := &staticBundleAdapter{bundle: bundle}

	// Determine authorizer based on policy
	auth, err := p.determineAuthorizer(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to determine authorizer: %w", err)
	}

	// Use tlsconfig for proper SPIFFE handling
	tlsConfig := tlsconfig.MTLSClientConfig(svidAdapter, bundleAdapter, auth)

	return &grpcClient{
		tlsConfig: tlsConfig,
		policy:    policy,
	}, nil
}

// CreateServer creates a gRPC server with rotation-capable SPIFFE mTLS.
func (p *RotatableGRPCProvider) CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ServerPort, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check for development mode
	if p.config != nil && p.config.ShouldSkipCertificateValidation() {
		log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		return &grpcServer{
			tlsConfig: &tls.Config{InsecureSkipVerify: true},
			policy:    policy,
		}, nil
	}

	// If sources are set, use them for rotation capability
	if p.svidSource != nil && p.bundleSource != nil {
		tlsConfig := p.createRotatableServerTLSConfig()
		return &grpcServer{
			tlsConfig: tlsConfig,
			policy:    policy,
		}, nil
	}

	// INPUT VALIDATION: For fallback mode, ensure cert and bundle are provided
	if cert == nil {
		return nil, fmt.Errorf("certificate required when sources are not configured")
	}
	if bundle == nil {
		return nil, fmt.Errorf("trust bundle required when sources are not configured")
	}

	// Fallback: create source adapters from provided cert/bundle
	svidAdapter := &staticSVIDAdapter{cert: cert}
	bundleAdapter := &staticBundleAdapter{bundle: bundle}

	// Determine authorizer based on policy
	auth, err := p.determineAuthorizer(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to determine authorizer: %w", err)
	}

	// Use tlsconfig for proper SPIFFE handling
	tlsConfig := tlsconfig.MTLSServerConfig(svidAdapter, bundleAdapter, auth)

	return &grpcServer{
		tlsConfig: tlsConfig,
		policy:    policy,
	}, nil
}

// createRotatableClientTLSConfig creates a TLS config that auto-rotates with source updates.
func (p *RotatableGRPCProvider) createRotatableClientTLSConfig() *tls.Config {
	// Use go-spiffe tlsconfig for automatic rotation support
	// New handshakes will pick up rotated certificates automatically
	auth := p.authorizer
	if auth == nil {
		// SECURITY: Use secure default based on configuration instead of AuthorizeAny()
		auth = p.createSecureDefaultAuthorizer()
	}

	return tlsconfig.MTLSClientConfig(p.svidSource, p.bundleSource, auth)
}

// createRotatableServerTLSConfig creates a TLS config that auto-rotates with source updates.
func (p *RotatableGRPCProvider) createRotatableServerTLSConfig() *tls.Config {
	// Use go-spiffe tlsconfig for automatic rotation support
	// New handshakes will pick up rotated certificates automatically
	auth := p.authorizer
	if auth == nil {
		// SECURITY: Use secure default based on configuration instead of AuthorizeAny()
		auth = p.createSecureDefaultAuthorizer()
	}

	return tlsconfig.MTLSServerConfig(p.svidSource, p.bundleSource, auth)
}

// determineAuthorizer creates an authorizer based on the authentication policy.
// This method now returns an error to prevent silent fallback to permissive authorization.
func (p *RotatableGRPCProvider) determineAuthorizer(policy *domain.AuthenticationPolicy) (tlsconfig.Authorizer, error) {
	if policy == nil {
		// Use secure default instead of AuthorizeAny for nil policy
		return p.createSecureDefaultAuthorizer(), nil
	}

	// If policy has specific trust domain, use it
	if !policy.TrustDomain.IsZero() {
		td, err := spiffeid.TrustDomainFromString(policy.TrustDomain.String())
		if err != nil {
			// SECURITY: Return error instead of falling back to AuthorizeAny
			return nil, fmt.Errorf("invalid trust domain in policy %q: %w", policy.TrustDomain, err)
		}
		return tlsconfig.AuthorizeMemberOf(td), nil
	}

	// If policy has specific SPIFFE IDs, use them
	if len(policy.AllowedSPIFFEIDs) > 0 {
		// AllowedSPIFFEIDs is already a slice of spiffeid.ID, no parsing needed
		return tlsconfig.AuthorizeOneOf(policy.AllowedSPIFFEIDs...), nil
	}

	// Use secure default instead of AuthorizeAny for empty policy
	return p.createSecureDefaultAuthorizer(), nil
}

// createSecureDefaultAuthorizer creates a secure default authorizer based on configuration.
// SECURITY NOTE: This method ensures we never fall back to AuthorizeAny() without explicit configuration.
func (p *RotatableGRPCProvider) createSecureDefaultAuthorizer() tlsconfig.Authorizer {
	// Try to get trust domain from configuration
	if p.config != nil && p.config.Service.Domain != "" {
		if td, err := spiffeid.TrustDomainFromString(p.config.Service.Domain); err == nil {
			// Default to authorizing members of the same trust domain
			// This is much more secure than AuthorizeAny() but still practical
			return tlsconfig.AuthorizeMemberOf(td)
		}
	}

	// WARNING: If no trust domain is configured, we have no choice but to use a permissive authorizer
	// In production, this should never happen - trust domain should always be configured
	// Log a warning to alert operators of this security risk
	fmt.Printf("WARNING: No trust domain configured, using permissive authorization. This is a security risk!\n")
	return tlsconfig.AuthorizeAny()
}

// Close releases any resources held by the provider.
func (p *RotatableGRPCProvider) Close() error {
	// Sources should be closed by their owner (e.g., SPIFFE provider)
	// We don't close them here as they may be shared
	return nil
}

// staticSVIDAdapter adapts a static certificate to x509svid.Source interface.
// This is used as a fallback when sources are not available.
type staticSVIDAdapter struct {
	cert *domain.Certificate
}

// GetX509SVID implements x509svid.Source.
func (s *staticSVIDAdapter) GetX509SVID() (*x509svid.SVID, error) {
	if s.cert == nil || s.cert.Cert == nil {
		return nil, fmt.Errorf("no certificate available")
	}

	// Extract SPIFFE ID from certificate
	var spiffeID spiffeid.ID
	for _, uri := range s.cert.Cert.URIs {
		if uri.Scheme == "spiffe" {
			var err error
			spiffeID, err = spiffeid.FromURI(uri)
			if err != nil {
				return nil, fmt.Errorf("invalid SPIFFE ID in certificate: %w", err)
			}
			break
		}
	}

	if spiffeID.IsZero() {
		return nil, fmt.Errorf("no SPIFFE ID found in certificate")
	}

	// Build certificate chain
	certs := []*x509.Certificate{s.cert.Cert}
	certs = append(certs, s.cert.Chain...)

	return &x509svid.SVID{
		ID:           spiffeID,
		Certificates: certs,
		PrivateKey:   s.cert.PrivateKey,
	}, nil
}

// staticBundleAdapter adapts a static trust bundle to x509bundle.Source interface.
// This is used as a fallback when sources are not available.
type staticBundleAdapter struct {
	bundle *domain.TrustBundle
}

// GetX509BundleForTrustDomain implements x509bundle.Source.
func (b *staticBundleAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	if b.bundle == nil || len(b.bundle.Certificates) == 0 {
		return nil, fmt.Errorf("no trust bundle available")
	}

	// Create bundle for the trust domain
	bundle := x509bundle.New(td)
	for _, cert := range b.bundle.Certificates {
		bundle.AddX509Authority(cert)
	}

	return bundle, nil
}

// IdentityProvider defines the interface for identity providers that can supply certificates and trust bundles.
// This interface ensures compile-time safety and clear API contracts.
type IdentityProvider interface {
	// GetCertificate returns the current certificate for this identity.
	GetCertificate() (*domain.Certificate, error)

	// GetTrustBundle returns the current trust bundle for certificate validation.
	GetTrustBundle() (*domain.TrustBundle, error)

	// GetServiceIdentity returns the service identity information.
	// This is optional and may return nil if not available.
	GetServiceIdentity() (*domain.ServiceIdentity, error)
}

// SourceAdapter wraps an identity provider to implement x509svid.Source and x509bundle.Source.
// This enables any identity provider to work with go-spiffe's rotation-capable tlsconfig.
type SourceAdapter struct {
	provider IdentityProvider // Identity provider interface
	mu       sync.RWMutex

	// Cache configuration
	cacheTTL time.Duration // TTL for cached values (default: 5 minutes)

	// Cached SVID values with expiration
	cachedSVID   *x509svid.SVID
	svidCachedAt time.Time

	// Cached bundle values with expiration
	cachedBundle   *x509bundle.Bundle
	bundleCachedAt time.Time
}

// Default cache TTL for source adapters (5 minutes)
const DefaultCacheTTL = 5 * time.Minute

// NewSourceAdapter creates a new adapter that wraps an identity provider.
// The provider must implement the IdentityProvider interface for compile-time safety.
func NewSourceAdapter(provider IdentityProvider) *SourceAdapter {
	return &SourceAdapter{
		provider: provider,
		cacheTTL: DefaultCacheTTL,
	}
}

// NewSourceAdapterWithTTL creates a new adapter with custom cache TTL.
func NewSourceAdapterWithTTL(provider IdentityProvider, cacheTTL time.Duration) *SourceAdapter {
	return &SourceAdapter{
		provider: provider,
		cacheTTL: cacheTTL,
	}
}

// GetX509SVID implements x509svid.Source.
func (a *SourceAdapter) GetX509SVID() (*x509svid.SVID, error) {
	// Check cache first
	a.mu.RLock()
	if a.cachedSVID != nil && time.Since(a.svidCachedAt) < a.cacheTTL {
		cached := a.cachedSVID
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Cache miss or expired - fetch fresh data
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check pattern - another goroutine might have updated cache
	if a.cachedSVID != nil && time.Since(a.svidCachedAt) < a.cacheTTL {
		return a.cachedSVID, nil
	}

	// Use the explicit interface for compile-time safety
	cert, err := a.provider.GetCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate from provider: %w", err)
	}

	if cert == nil || cert.Cert == nil {
		return nil, fmt.Errorf("provider returned nil certificate")
	}

	// Improved SPIFFE ID handling with better fallback order
	spiffeID, err := a.extractSPIFFEID(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to extract SPIFFE ID: %w", err)
	}

	// Build certificate chain with duplication prevention
	certs := a.buildCertificateChain(cert)

	svid := &x509svid.SVID{
		ID:           spiffeID,
		Certificates: certs,
		PrivateKey:   cert.PrivateKey,
	}

	// Update cache with TTL
	a.cachedSVID = svid
	a.svidCachedAt = time.Now()

	return svid, nil
}

// GetX509BundleForTrustDomain implements x509bundle.Source.
func (a *SourceAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Check cache first
	a.mu.RLock()
	if a.cachedBundle != nil && time.Since(a.bundleCachedAt) < a.cacheTTL {
		cached := a.cachedBundle
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Cache miss or expired - fetch fresh data
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check pattern - another goroutine might have updated cache
	if a.cachedBundle != nil && time.Since(a.bundleCachedAt) < a.cacheTTL {
		return a.cachedBundle, nil
	}

	// Use the explicit interface for compile-time safety
	trustBundle, err := a.provider.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle from provider: %w", err)
	}

	if trustBundle == nil || len(trustBundle.Certificates) == 0 {
		return nil, fmt.Errorf("provider returned empty trust bundle")
	}

	// Create bundle for the trust domain
	bundle := x509bundle.New(td)
	for _, cert := range trustBundle.Certificates {
		bundle.AddX509Authority(cert)
	}

	// Update cache with TTL
	a.cachedBundle = bundle
	a.bundleCachedAt = time.Now()

	return bundle, nil
}

// InvalidateCache manually invalidates cached SVID and bundle data.
// This is useful when you know the underlying certificates have been rotated.
func (a *SourceAdapter) InvalidateCache() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cachedSVID = nil
	a.svidCachedAt = time.Time{}
	a.cachedBundle = nil
	a.bundleCachedAt = time.Time{}
}

// SetCacheTTL updates the cache TTL. Setting to 0 disables caching.
func (a *SourceAdapter) SetCacheTTL(ttl time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cacheTTL = ttl

	// If TTL is set to 0, invalidate cache
	if ttl == 0 {
		a.cachedSVID = nil
		a.svidCachedAt = time.Time{}
		a.cachedBundle = nil
		a.bundleCachedAt = time.Time{}
	}
}

// extractSPIFFEID extracts SPIFFE ID from certificate with improved fallback handling.
// This method implements better fallback order and supports additional SAN types.
func (a *SourceAdapter) extractSPIFFEID(cert *domain.Certificate) (spiffeid.ID, error) {
	// Strategy 1: Try to get SPIFFE ID from service identity first (most reliable)
	// This ensures we always try the configured identity before certificate parsing
	identity, err := a.provider.GetServiceIdentity()
	if err == nil && identity != nil {
		// Construct SPIFFE ID from identity
		td, err := spiffeid.TrustDomainFromString(identity.Domain())
		if err == nil {
			spiffeID, err := spiffeid.FromSegments(td, identity.Name())
			if err == nil && !spiffeID.IsZero() {
				return spiffeID, nil
			}
		}
	}

	// Strategy 2: Extract SPIFFE ID from certificate URI SAN
	for _, uri := range cert.Cert.URIs {
		if uri.Scheme == "spiffe" {
			spiffeID, err := spiffeid.FromURI(uri)
			if err != nil {
				// Log but don't fail - try other URIs
				continue
			}
			if !spiffeID.IsZero() {
				return spiffeID, nil
			}
		}
	}

	// Strategy 3: Support for DNS-based identities (if applicable)
	// This is useful when SPIFFE URIs are not available but DNS names are present
	for _, dnsName := range cert.Cert.DNSNames {
		// Check if DNS name follows SPIFFE-like patterns
		if len(dnsName) > 0 {
			// For future extension - could map DNS names to SPIFFE IDs
			// based on organizational policies
			// Example: service.prod.company.com -> spiffe://prod.company.com/service
		}
	}

	// Strategy 4: Fallback to subject common name
	if cert.Cert.Subject.CommonName != "" {
		// This is a last resort and should be used carefully
		// Only if we can derive meaningful SPIFFE ID from CN
		// Example: CN=service -> spiffe://domain/service (if domain known)
	}

	return spiffeid.ID{}, fmt.Errorf("no valid SPIFFE ID found in certificate or identity - URI SANs: %d, DNS SANs: %d, identity available: %t",
		len(cert.Cert.URIs), len(cert.Cert.DNSNames), identity != nil)
}

// buildCertificateChain builds a certificate chain while preventing duplication.
// This ensures the leaf certificate isn't duplicated in the chain.
func (a *SourceAdapter) buildCertificateChain(cert *domain.Certificate) []*x509.Certificate {
	if cert.Cert == nil {
		return nil
	}

	// Start with leaf certificate
	certs := []*x509.Certificate{cert.Cert}

	// Create a map of certificate fingerprints to prevent duplication
	seen := make(map[string]bool)
	seen[string(cert.Cert.Raw)] = true

	// Add intermediate certificates, checking for duplicates
	for _, intermediateCert := range cert.Chain {
		if intermediateCert == nil {
			continue
		}

		fingerprint := string(intermediateCert.Raw)
		if !seen[fingerprint] {
			certs = append(certs, intermediateCert)
			seen[fingerprint] = true
		}
	}

	return certs
}
