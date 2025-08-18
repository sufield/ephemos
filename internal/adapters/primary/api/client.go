// Package api provides high-level client and server APIs for secure SPIFFE-based communication.
package api

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// ClientOption configures client creation.
type ClientOption func(*clientOpts)

type clientOpts struct {
	transportProvider ports.TransportProvider
	authorizer        tlsconfig.Authorizer
	trustDomain       spiffeid.TrustDomain
}

// WithTransportProvider sets the transport provider for the client.
func WithTransportProvider(p ports.TransportProvider) ClientOption {
	return func(o *clientOpts) { o.transportProvider = p }
}

// WithAuthorizer sets the authorizer for peer verification.
func WithAuthorizer(a tlsconfig.Authorizer) ClientOption {
	return func(o *clientOpts) { o.authorizer = a }
}

// WithTrustDomain sets the trust domain for authorization.
func WithTrustDomain(td spiffeid.TrustDomain) ClientOption {
	return func(o *clientOpts) { o.trustDomain = td }
}

// Client provides a high-level API for connecting to SPIFFE-secured services.
type Client struct {
	identityService *services.IdentityService
	domainClient    ports.ClientPort
	trustDomain     spiffeid.TrustDomain
	authorizer      tlsconfig.Authorizer
	mu              sync.Mutex
}

// NewClient creates a new identity client with dependency injection.
// The caller must provide all dependencies through options to avoid internal adapter imports.
func NewClient(identityProvider ports.IdentityProvider, cfg *ports.Configuration, opts ...ClientOption) (*Client, error) {
	var o clientOpts
	for _, f := range opts {
		f(&o)
	}

	if o.transportProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "transportProvider",
			Message: "transport provider cannot be nil",
		}
	}

	// Use provided authorizer and trust domain from options
	return IdentityClient(identityProvider, o.transportProvider, cfg, o.authorizer, o.trustDomain)
}

// IdentityClient creates a new identity client with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func IdentityClient(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	cfg *ports.Configuration,
	authorizer tlsconfig.Authorizer,
	trustDomain spiffeid.TrustDomain,
) (*Client, error) {
	if cfg == nil {
		return nil, &errors.ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	if identityProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "identityProvider",
			Value:   nil,
			Message: "identity provider cannot be nil",
		}
	}

	if transportProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "transportProvider",
			Value:   nil,
			Message: "transport provider cannot be nil",
		}
	}

	identityService, err := services.NewIdentityService(
		identityProvider,
		transportProvider,
		cfg,
		nil, // default validator
		nil, // no-op metrics
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &Client{
		identityService: identityService,
		trustDomain:     trustDomain,
		authorizer:      authorizer,
	}, nil
}

// Connect establishes a secure connection to a remote service using SPIFFE identities.
func (c *Client) Connect(ctx context.Context, serviceNameStr, addressStr string) (*ClientConnection, error) {
	// Input validation
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	// Use ServiceName value object for validation
	serviceName, err := domain.NewServiceName(serviceNameStr)
	if err != nil {
		return nil, &errors.ValidationError{
			Field:   "serviceName",
			Value:   serviceNameStr,
			Message: fmt.Sprintf("invalid service name: %v", err),
		}
	}

	// Use ServiceAddress value object for validation
	address, err := domain.NewServiceAddress(addressStr)
	if err != nil {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   addressStr,
			Message: fmt.Sprintf("invalid service address: %v", err),
		}
	}

	// Thread-safe connection initialization
	c.mu.Lock()
	if c.domainClient == nil {
		client, err := c.identityService.CreateClientIdentity()
		if err != nil {
			c.mu.Unlock()
			return nil, fmt.Errorf("failed to create client identity: %w", err)
		}
		c.domainClient = client
	}
	c.mu.Unlock()

	domainConn, err := c.domainClient.Connect(serviceName.Value(), address.Value())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName.Value(), address.Value(), err)
	}

	// Extract the underlying gRPC connection
	grpcConn, ok := domainConn.GetClientConnection().(*grpc.ClientConn)
	if !ok {
		return nil, fmt.Errorf("unexpected connection type from domain client")
	}

	// Use deterministic, config-driven authorizer
	authorizer := buildAuthorizer(serviceName.Value(), c.trustDomain)
	if authorizer == nil {
		// Fall back to client's configured authorizer if available
		authorizer = c.authorizer
		if authorizer == nil {
			return nil, fmt.Errorf("no authorizer configured for service %s", serviceName.Value())
		}
	}

	return &ClientConnection{
		conn:            grpcConn,
		domainConn:      domainConn,
		identityService: c.identityService,
		authorizer:      authorizer,
		trustDomain:     c.trustDomain,
	}, nil
}

// Close cleans up the client resources and closes any connections.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.domainClient != nil {
		if err := c.domainClient.Close(); err != nil {
			return fmt.Errorf("failed to close domain client: %w", err)
		}
		c.domainClient = nil
	}

	return nil
}

// CertificateProvider provides access to certificates and trust bundles for HTTP clients
type CertificateProvider interface {
	GetCertificate() (*domain.Certificate, error)
	GetTrustBundle() (*domain.TrustBundle, error)
}

// ClientConnection represents a secure client connection to a remote service.
type ClientConnection struct {
	conn            *grpc.ClientConn
	domainConn      ports.ConnectionPort
	identityService CertificateProvider
	authorizer      tlsconfig.Authorizer
	trustDomain     spiffeid.TrustDomain

	tlsOnce sync.Once
	tlsCfg  *tls.Config
	tlsErr  error
}

// Close terminates the client connection and cleans up resources.
func (c *ClientConnection) Close() error {
	if c.domainConn != nil {
		if err := c.domainConn.Close(); err != nil {
			return fmt.Errorf("failed to close domain connection: %w", err)
		}
	}
	return nil
}

// GetClientConnection returns the underlying gRPC client connection.
func (c *ClientConnection) GetClientConnection() *grpc.ClientConn {
	return c.conn
}

// HTTPClient returns an HTTP client configured with SPIFFE certificate authentication.
// This creates a new HTTP client that uses the same SPIFFE certificates and trust bundle
// as the gRPC connection for secure HTTP communication.
func (c *ClientConnection) HTTPClient() (*http.Client, error) {
	if c.domainConn == nil {
		return nil, fmt.Errorf("no domain connection available")
	}

	tlsCfg, err := c.extractTLSConfig(c.trustDomain)
	if err != nil {
		return nil, fmt.Errorf("tls config: %w", err)
	}

	tr := &http.Transport{
		TLSClientConfig:       tlsCfg,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}, nil
}

// buildAuthorizer creates a deterministic, config-driven authorizer.
// If caller passes a full SPIFFE ID, enforce exact match.
// Otherwise, require membership in the configured trust domain.
func buildAuthorizer(serviceName string, td spiffeid.TrustDomain) tlsconfig.Authorizer {
	// If caller passes a full SPIFFE ID, enforce exact match.
	if strings.HasPrefix(serviceName, "spiffe://") {
		if id, err := spiffeid.FromString(serviceName); err == nil {
			return tlsconfig.AuthorizeID(id)
		}
	}
	// Otherwise, require membership in the configured trust domain.
	return tlsconfig.AuthorizeMemberOf(td)
}

// extractTLSConfig extracts TLS configuration using official go-spiffe tlsconfig
// This provides the same SPIFFE certificate authentication for HTTP as gRPC uses
// Results are cached using sync.Once for performance.
func (c *ClientConnection) extractTLSConfig(trustDomain spiffeid.TrustDomain) (*tls.Config, error) {
	c.tlsOnce.Do(func() {
		// Create adapters for go-spiffe interfaces
		svidSource, err := c.createSVIDSource()
		if err != nil {
			c.tlsErr = fmt.Errorf("failed to create SVID source for SPIFFE authentication: %w", err)
			return
		}

		bundleSource, err := c.createBundleSourceForTrustDomain(trustDomain)
		if err != nil {
			c.tlsErr = fmt.Errorf("failed to create bundle source for SPIFFE authentication: %w", err)
			return
		}

		// Use official go-spiffe tlsconfig for mutual TLS with SPIFFE authentication
		// This provides the standard SPIFFE peer verification with proper authorizers
		tlsConfig := tlsconfig.MTLSClientConfig(svidSource, bundleSource, c.authorizer)

		// Ensure TLS 1.3 minimum version (go-spiffe uses 1.2 by default)
		tlsConfig.MinVersion = tls.VersionTLS13

		c.tlsCfg = tlsConfig
	})

	if c.tlsErr != nil {
		return nil, c.tlsErr
	}

	return c.tlsCfg, nil
}

// svidSourceAdapter adapts the domain certificate to go-spiffe x509svid.Source
type svidSourceAdapter struct {
	identityService CertificateProvider
}

// GetX509SVID implements x509svid.Source interface
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

	// Create X509SVID from domain certificate
	svid := &x509svid.SVID{
		ID:           spiffeID,
		Certificates: certChain,
		PrivateKey:   signer,
	}

	return svid, nil
}

// bundleSourceAdapter adapts the domain trust bundle to go-spiffe x509bundle.Source
type bundleSourceAdapter struct {
	identityService       CertificateProvider
	restrictedTrustDomain spiffeid.TrustDomain
}

// GetX509BundleForTrustDomain implements x509bundle.Source interface
// This enforces trust domain isolation by only serving bundles for the configured domain
func (b *bundleSourceAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Enforce trust domain isolation - only serve bundles for our configured domain
	if !b.restrictedTrustDomain.IsZero() && td != b.restrictedTrustDomain {
		return nil, fmt.Errorf("trust domain %s not allowed, restricted to %s", td, b.restrictedTrustDomain)
	}

	trustBundle, err := b.identityService.GetTrustBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	if trustBundle.IsEmpty() {
		return nil, fmt.Errorf("trust bundle is empty")
	}

	// Create bundle for the requested trust domain
	bundle := x509bundle.FromX509Authorities(td, trustBundle.RawCertificates())
	return bundle, nil
}

// createSVIDSource creates an x509svid.Source adapter
func (c *ClientConnection) createSVIDSource() (x509svid.Source, error) {
	if c.identityService == nil {
		return nil, fmt.Errorf("no identity service available")
	}

	return &svidSourceAdapter{
		identityService: c.identityService,
	}, nil
}

// createBundleSource creates an x509bundle.Source adapter without trust domain restrictions
func (c *ClientConnection) createBundleSource() (x509bundle.Source, error) {
	if c.identityService == nil {
		return nil, fmt.Errorf("no identity service available")
	}

	return &bundleSourceAdapter{
		identityService: c.identityService,
	}, nil
}

// createBundleSourceForTrustDomain creates an x509bundle.Source adapter with trust domain enforcement
func (c *ClientConnection) createBundleSourceForTrustDomain(td spiffeid.TrustDomain) (x509bundle.Source, error) {
	if c.identityService == nil {
		return nil, fmt.Errorf("no identity service available")
	}

	return &bundleSourceAdapter{
		identityService:       c.identityService,
		restrictedTrustDomain: td,
	}, nil
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
	// Return error if no valid SPIFFE ID found
	return spiffeid.ID{}, fmt.Errorf("no valid SPIFFE ID found in certificate URI SANs")
}
