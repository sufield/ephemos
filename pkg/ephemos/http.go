// Package ephemos provides identity-based authentication for backend services.
package ephemos

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

// Authorizer validates peer certificates during mTLS handshake.
// This is needed for HTTP authentication functionality.
type Authorizer = tlsconfig.Authorizer

// AuthorizeAny returns an Authorizer that accepts any valid SPIFFE certificate.
// This provides basic SPIFFE identity validation for authentication.
func AuthorizeAny() Authorizer {
	return tlsconfig.AuthorizeAny()
}


// HTTPClientConfig configures an HTTP client with SPIFFE mTLS.
type HTTPClientConfig struct {
	// IdentityService provides certificates and trust bundles.
	IdentityService IdentityService

	// Authorizer validates peer certificates.
	// If nil, AuthorizeAny() is used.
	Authorizer Authorizer

	// TrustDomain restricts which trust domains are accepted.
	// If empty, any trust domain is accepted.
	TrustDomain string

	// Timeout specifies the timeout for HTTP requests.
	// If zero, DefaultClientTimeout is used.
	Timeout time.Duration

	// MaxIdleConns controls the maximum number of idle connections.
	// If zero, 100 is used.
	MaxIdleConns int

	// IdleConnTimeout is the maximum amount of time an idle connection will remain idle.
	// If zero, 90 seconds is used.
	IdleConnTimeout time.Duration
}

// NewHTTPClient creates an HTTP client configured with SPIFFE mTLS.
// This is the primary way contrib middleware should create HTTP clients.
//
// Example:
//
//	config := &ephemos.HTTPClientConfig{
//	    IdentityService: identityService,
//	    Authorizer: ephemos.AuthorizeMemberOf("prod.company.com"),
//	    Timeout: 30 * time.Second,
//	}
//	client, err := ephemos.NewHTTPClient(config)
//	if err != nil {
//	    return err
//	}
//	resp, err := client.Get("https://service.prod.company.com/api/data")
func NewHTTPClient(config *HTTPClientConfig) (*http.Client, error) {
	if config.IdentityService == nil {
		return nil, fmt.Errorf("identity service is required")
	}

	// Set defaults
	if config.Authorizer == nil {
		config.Authorizer = AuthorizeAny()
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultClientTimeout
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = 90 * time.Second
	}

	// Create TLS config
	tlsConfig, err := NewTLSConfig(config.IdentityService, config.Authorizer, config.TrustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create HTTP transport
	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          config.MaxIdleConns,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}

	// Create HTTP client
	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}, nil
}

// NewTLSConfig creates a TLS configuration for SPIFFE mTLS.
// This is a lower-level function used when you need direct control over TLS configuration.
//
// Example:
//
//	tlsConfig, err := ephemos.NewTLSConfig(
//	    identityService,
//	    ephemos.AuthorizeMemberOf("prod.company.com"),
//	    "prod.company.com",
//	)
//	if err != nil {
//	    return err
//	}
//	transport := &http.Transport{TLSClientConfig: tlsConfig}
func NewTLSConfig(identityService IdentityService, authorizer Authorizer, trustDomain string) (*tls.Config, error) {
	if identityService == nil {
		return nil, fmt.Errorf("identity service is required")
	}

	// Create SVID source adapter
	svidSource := &svidSourceAdapter{identityService: identityService}

	// Create bundle source adapter
	var bundleSource x509bundle.Source
	if trustDomain != "" {
		td, err := spiffeid.TrustDomainFromString(trustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain %q: %w", trustDomain, err)
		}
		bundleSource = &bundleSourceAdapter{
			identityService:       identityService,
			restrictedTrustDomain: td,
		}
	} else {
		bundleSource = &bundleSourceAdapter{
			identityService: identityService,
		}
	}

	// Use go-spiffe to create mTLS config
	tlsConfig := tlsconfig.MTLSClientConfig(svidSource, bundleSource, authorizer)

	// Ensure TLS 1.3 minimum
	tlsConfig.MinVersion = tls.VersionTLS13

	return tlsConfig, nil
}

// svidSourceAdapter adapts IdentityService to x509svid.Source
type svidSourceAdapter struct {
	identityService IdentityService
}

// GetX509SVID implements x509svid.Source
func (s *svidSourceAdapter) GetX509SVID() (*x509svid.SVID, error) {
	cert, err := s.identityService.GetCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	if cert.Cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}

	// Build certificate chain
	var certChain []*x509.Certificate
	certChain = append(certChain, cert.Cert)
	certChain = append(certChain, cert.Chain...)

	// Ensure private key implements crypto.Signer
	signer, ok := cert.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key does not implement crypto.Signer")
	}

	// Extract SPIFFE ID from certificate
	spiffeID, err := extractSPIFFEIDFromCert(cert.Cert)
	if err != nil {
		return nil, fmt.Errorf("failed to extract SPIFFE ID: %w", err)
	}

	return &x509svid.SVID{
		ID:           spiffeID,
		Certificates: certChain,
		PrivateKey:   signer,
	}, nil
}

// bundleSourceAdapter adapts IdentityService to x509bundle.Source
type bundleSourceAdapter struct {
	identityService       IdentityService
	restrictedTrustDomain spiffeid.TrustDomain
}

// GetX509BundleForTrustDomain implements x509bundle.Source
func (b *bundleSourceAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Enforce trust domain restriction if configured
	if !b.restrictedTrustDomain.IsZero() && td != b.restrictedTrustDomain {
		return nil, fmt.Errorf("trust domain %s not allowed, restricted to %s", td, b.restrictedTrustDomain)
	}

	trustBundle, err := b.identityService.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	if len(trustBundle.Certificates) == 0 {
		return nil, fmt.Errorf("trust bundle is empty")
	}

	// Create bundle for the requested trust domain
	bundle := x509bundle.FromX509Authorities(td, trustBundle.Certificates)
	return bundle, nil
}

// extractSPIFFEIDFromCert extracts SPIFFE ID from certificate URI SAN
func extractSPIFFEIDFromCert(cert *x509.Certificate) (spiffeid.ID, error) {
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			id, err := spiffeid.FromURI(uri)
			if err == nil {
				return id, nil
			}
		}
	}
	return spiffeid.ID{}, fmt.Errorf("no valid SPIFFE ID found in certificate URI SANs")
}

// HTTPTransportConfig provides fine-grained control over HTTP transport configuration.
type HTTPTransportConfig struct {
	// Base configuration
	HTTPClientConfig

	// Transport-specific settings
	MaxConnsPerHost       int
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
	DisableCompression    bool
	ForceAttemptHTTP2     bool
}

// NewHTTPTransport creates an HTTP transport with SPIFFE mTLS.
// This provides more control than NewHTTPClient for advanced use cases.
//
// Example:
//
//	config := &ephemos.HTTPTransportConfig{
//	    HTTPClientConfig: ephemos.HTTPClientConfig{
//	        IdentityService: identityService,
//	        Authorizer: ephemos.AuthorizeMemberOf("prod.company.com"),
//	    },
//	    MaxConnsPerHost: 10,
//	    ForceAttemptHTTP2: true,
//	}
//	transport, err := ephemos.NewHTTPTransport(config)
//	if err != nil {
//	    return err
//	}
//	client := &http.Client{Transport: transport}
func NewHTTPTransport(config *HTTPTransportConfig) (*http.Transport, error) {
	if config.IdentityService == nil {
		return nil, fmt.Errorf("identity service is required")
	}

	// Set defaults
	if config.Authorizer == nil {
		config.Authorizer = AuthorizeAny()
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = 90 * time.Second
	}

	// Create TLS config
	tlsConfig, err := NewTLSConfig(config.IdentityService, config.Authorizer, config.TrustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create HTTP transport with all settings
	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		MaxIdleConns:          config.MaxIdleConns,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     config.DisableKeepAlives,
		DisableCompression:    config.DisableCompression,
		ForceAttemptHTTP2:     config.ForceAttemptHTTP2,
	}

	return transport, nil
}

// NewServerTLSConfig creates a TLS configuration for HTTPS servers with SPIFFE mTLS.
// This enables servers to authenticate clients using SPIFFE identities.
// Note: This is for authentication only. Authorization is out of scope.
//
// Example:
//
//	tlsConfig, err := ephemos.NewServerTLSConfig(
//	    identityService,
//	    ephemos.AuthorizeAny(), // Accept any valid SPIFFE identity
//	)
//	if err != nil {
//	    return err
//	}
//	server := &http.Server{
//	    Addr:      ":8443",
//	    Handler:   router,
//	    TLSConfig: tlsConfig,
//	}
//	server.ListenAndServeTLS("", "")
func NewServerTLSConfig(identityService IdentityService, authorizer Authorizer) (*tls.Config, error) {
	if identityService == nil {
		return nil, fmt.Errorf("identity service is required")
	}

	// Set default authorizer if none provided
	if authorizer == nil {
		authorizer = AuthorizeAny()
	}

	// Create SVID source adapter
	svidSource := &svidSourceAdapter{identityService: identityService}

	// Create bundle source adapter (no trust domain restriction for server)
	bundleSource := &bundleSourceAdapter{
		identityService: identityService,
	}

	// Use go-spiffe to create mTLS server config
	tlsConfig := tlsconfig.MTLSServerConfig(svidSource, bundleSource, authorizer)

	// Ensure TLS 1.3 minimum
	tlsConfig.MinVersion = tls.VersionTLS13

	return tlsConfig, nil
}
