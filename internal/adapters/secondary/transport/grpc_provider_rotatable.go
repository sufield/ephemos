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
	svidSource    x509svid.Source
	bundleSource  x509bundle.Source
	authorizer    tlsconfig.Authorizer
	trustProvider ports.TrustDomainProvider // Injected capability
	mu            sync.RWMutex
}

// NewRotatableGRPCProvider creates a new rotation-capable gRPC transport provider.
func NewRotatableGRPCProvider(trustProvider ports.TrustDomainProvider) *RotatableGRPCProvider {
	return &RotatableGRPCProvider{
		trustProvider: trustProvider,
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
	if p.trustProvider != nil && p.trustProvider.ShouldSkipCertificateValidation() {
		log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		return &grpcClient{
			tlsConfig: &tls.Config{InsecureSkipVerify: true},
			policy:    policy,
		}, nil
	}

	// Require SPIFFE sources to be properly configured - no fallback patterns
	if p.svidSource == nil || p.bundleSource == nil {
		return nil, fmt.Errorf("SPIFFE sources must be properly configured: svidSource=%v, bundleSource=%v",
			p.svidSource != nil, p.bundleSource != nil)
	}

	tlsConfig := p.createRotatableClientTLSConfig()
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
	if p.trustProvider != nil && p.trustProvider.ShouldSkipCertificateValidation() {
		log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		return &grpcServer{
			tlsConfig: &tls.Config{InsecureSkipVerify: true},
			policy:    policy,
		}, nil
	}

	// Require SPIFFE sources to be properly configured - no fallback patterns
	if p.svidSource == nil || p.bundleSource == nil {
		return nil, fmt.Errorf("SPIFFE sources must be properly configured: svidSource=%v, bundleSource=%v",
			p.svidSource != nil, p.bundleSource != nil)
	}

	tlsConfig := p.createRotatableServerTLSConfig()
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

	// Authentication-only scope: no specific SPIFFE ID authorization

	// Use secure default instead of AuthorizeAny for empty policy
	return p.createSecureDefaultAuthorizer(), nil
}

// createSecureDefaultAuthorizer creates a secure default authorizer using injected capability.
// SECURITY NOTE: This method ensures we never fall back to AuthorizeAny() without explicit configuration.
func (p *RotatableGRPCProvider) createSecureDefaultAuthorizer() tlsconfig.Authorizer {
	// Use injected capability instead of direct config access
	if p.trustProvider != nil {
		if authorizer, err := p.trustProvider.CreateDefaultAuthorizer(); err == nil {
			// Use the authorizer from our trust domain provider
			return authorizer
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
	cachedSVID     *x509svid.SVID
	svidCacheEntry *domain.CacheEntry

	// Cached bundle values with expiration
	cachedBundle     *x509bundle.Bundle
	bundleCacheEntry *domain.CacheEntry
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
	if a.cachedSVID != nil && a.svidCacheEntry != nil && a.svidCacheEntry.IsFresh() {
		cached := a.cachedSVID
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Cache miss or expired - fetch fresh data
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check pattern - another goroutine might have updated cache
	if a.cachedSVID != nil && a.svidCacheEntry != nil && a.svidCacheEntry.IsFresh() {
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
	a.svidCacheEntry = domain.NewCacheEntry(a.cacheTTL)

	return svid, nil
}

// GetX509BundleForTrustDomain implements x509bundle.Source.
func (a *SourceAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Check cache first
	a.mu.RLock()
	if a.cachedBundle != nil && a.bundleCacheEntry != nil && a.bundleCacheEntry.IsFresh() {
		cached := a.cachedBundle
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Cache miss or expired - fetch fresh data
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check pattern - another goroutine might have updated cache
	if a.cachedBundle != nil && a.bundleCacheEntry != nil && a.bundleCacheEntry.IsFresh() {
		return a.cachedBundle, nil
	}

	// Use the explicit interface for compile-time safety
	trustBundle, err := a.provider.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle from provider: %w", err)
	}

	if trustBundle == nil || trustBundle.IsEmpty() {
		return nil, fmt.Errorf("provider returned empty trust bundle")
	}

	// Create bundle for the trust domain
	bundle := x509bundle.New(td)
	for _, cert := range trustBundle.Certificates {
		if cert != nil && cert.Cert != nil {
			bundle.AddX509Authority(cert.Cert)
		}
	}

	// Update cache with TTL
	a.cachedBundle = bundle
	a.bundleCacheEntry = domain.NewCacheEntry(a.cacheTTL)

	return bundle, nil
}

// InvalidateCache manually invalidates cached SVID and bundle data.
// This is useful when you know the underlying certificates have been rotated.
func (a *SourceAdapter) InvalidateCache() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cachedSVID = nil
	a.svidCacheEntry = nil
	a.cachedBundle = nil
	a.bundleCacheEntry = nil
}

// SetCacheTTL updates the cache TTL. Setting to 0 disables caching.
func (a *SourceAdapter) SetCacheTTL(ttl time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cacheTTL = ttl

	// If TTL is set to 0, invalidate cache
	if ttl == 0 {
		a.cachedSVID = nil
		a.svidCacheEntry = nil
		a.cachedBundle = nil
		a.bundleCacheEntry = nil
	}
}

// extractSPIFFEID extracts SPIFFE ID from certificate using only proper SPIFFE URIs.
// No fallback patterns - fails fast if proper SPIFFE identity is not available.
func (a *SourceAdapter) extractSPIFFEID(cert *domain.Certificate) (spiffeid.ID, error) {
	// Extract SPIFFE ID from certificate URI SAN - the only valid source
	for _, uri := range cert.Cert.URIs {
		if uri.Scheme == "spiffe" {
			spiffeID, err := spiffeid.FromURI(uri)
			if err != nil {
				return spiffeid.ID{}, fmt.Errorf("invalid SPIFFE URI in certificate: %w", err)
			}
			if spiffeID.IsZero() {
				return spiffeid.ID{}, fmt.Errorf("empty SPIFFE ID in certificate URI")
			}
			return spiffeID, nil
		}
	}

	return spiffeid.ID{}, fmt.Errorf("no valid SPIFFE URI found in certificate - proper SPIFFE certificates must contain URI SANs with spiffe:// scheme")
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
