// Package api provides high-level client and server APIs for secure SPIFFE-based communication.
package api

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// Client provides a high-level API for connecting to SPIFFE-secured services.
type Client struct {
	identityService *services.IdentityService
	domainClient    ports.ClientPort
	mu              sync.Mutex
}

// NewClientFromConfig creates a new identity client from configuration path.
// Handles all provider creation internally to hide implementation details from public API.
func NewClientFromConfig(ctx context.Context, configPath string) (*Client, error) {
	// Load configuration
	configProvider := config.NewFileProvider()
	cfg, err := configProvider.LoadConfiguration(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create identity provider
	identityProvider, err := spiffe.NewProvider(cfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}

	// Create default transport provider
	transportProvider := transport.NewGRPCProvider(cfg)
	
	return IdentityClient(identityProvider, transportProvider, cfg)
}

// NewClient creates a new identity client with minimal dependencies.
// Uses default transport provider internally to hide implementation details from public API.
func NewClient(identityProvider ports.IdentityProvider, cfg *ports.Configuration) (*Client, error) {
	// Create default transport provider internally
	transportProvider := transport.NewGRPCProvider(cfg)
	
	return IdentityClient(identityProvider, transportProvider, cfg)
}

// IdentityClient creates a new identity client with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func IdentityClient(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	cfg *ports.Configuration,
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &Client{
		identityService: identityService,
	}, nil
}

// Connect establishes a secure connection to a remote service using SPIFFE identities.
func (c *Client) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	// Input validation
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	if strings.TrimSpace(serviceName) == "" {
		return nil, &errors.ValidationError{
			Field:   "serviceName",
			Value:   serviceName,
			Message: "service name cannot be empty or whitespace",
		}
	}

	if strings.TrimSpace(address) == "" {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address cannot be empty or whitespace",
		}
	}

	// Validate address format (host:port)
	if _, _, err := net.SplitHostPort(address); err != nil {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address must be in format 'host:port'",
		}
	}

	serviceName = strings.TrimSpace(serviceName)
	address = strings.TrimSpace(address)

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

	domainConn, err := c.domainClient.Connect(serviceName, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}

	// Extract the underlying gRPC connection
	grpcConn, ok := domainConn.GetClientConnection().(*grpc.ClientConn)
	if !ok {
		return nil, fmt.Errorf("unexpected connection type from domain client")
	}

	return &ClientConnection{
		conn:            grpcConn,
		domainConn:      domainConn,
		identityService: c.identityService,
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
func (c *ClientConnection) HTTPClient() *http.Client {
	if c.domainConn == nil {
		// Return basic HTTP client as fallback
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Extract SPIFFE certificates and trust configuration from the domain connection
	// This uses the same security context as the gRPC connection
	tlsConfig, err := c.extractTLSConfig()
	if err != nil {
		// Log error and return basic client as fallback
		// In production, this should use structured logging
		fmt.Printf("Warning: Failed to extract TLS config for HTTP client: %v\n", err)
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create HTTP transport with SPIFFE certificate authentication
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		// Configure connection pooling and timeouts
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limit redirects for security
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

// extractTLSConfig extracts TLS configuration using official go-spiffe tlsconfig
// This provides the same SPIFFE certificate authentication for HTTP as gRPC uses
func (c *ClientConnection) extractTLSConfig() (*tls.Config, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("no gRPC connection available")
	}

	// Get the connection state from the gRPC connection
	state := c.conn.GetState()
	if state == connectivity.Shutdown {
		return nil, fmt.Errorf("gRPC connection is shut down")
	}

	// Create adapters for go-spiffe interfaces
	svidSource, err := c.createSVIDSource()
	if err != nil {
		// Fallback to basic secure TLS configuration
		return c.createSecureTLSConfig(), nil
	}

	bundleSource, err := c.createBundleSource()
	if err != nil {
		// Fallback to basic secure TLS configuration
		return c.createSecureTLSConfig(), nil
	}

	// Use official go-spiffe tlsconfig for mutual TLS with SPIFFE authentication
	// This provides the standard SPIFFE peer verification with proper authorizers
	authorizer := tlsconfig.AuthorizeAny() // For now, authorize any valid SPIFFE ID
	tlsConfig := tlsconfig.MTLSClientConfig(svidSource, bundleSource, authorizer)

	// Ensure TLS 1.3 minimum version (go-spiffe uses 1.2 by default)
	tlsConfig.MinVersion = tls.VersionTLS13

	return tlsConfig, nil
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

	// Create X509SVID from domain certificate
	svid := &x509svid.SVID{
		ID:           extractSPIFFEIDFromCert(cert.Cert),
		Certificates: certChain,
		PrivateKey:   signer,
	}

	return svid, nil
}

// bundleSourceAdapter adapts the domain trust bundle to go-spiffe x509bundle.Source  
type bundleSourceAdapter struct {
	identityService CertificateProvider
	trustDomain     spiffeid.TrustDomain
}

// GetX509BundleForTrustDomain implements x509bundle.Source interface
func (b *bundleSourceAdapter) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
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

// createSVIDSource creates an x509svid.Source adapter
func (c *ClientConnection) createSVIDSource() (x509svid.Source, error) {
	if c.identityService == nil {
		return nil, fmt.Errorf("no identity service available")
	}

	return &svidSourceAdapter{
		identityService: c.identityService,
	}, nil
}

// createBundleSource creates an x509bundle.Source adapter
func (c *ClientConnection) createBundleSource() (x509bundle.Source, error) {
	if c.identityService == nil {
		return nil, fmt.Errorf("no identity service available")
	}

	// For now, use a default trust domain. In production, this should be configurable
	trustDomain, err := spiffeid.TrustDomainFromString("example.org")
	if err != nil {
		return nil, fmt.Errorf("failed to create trust domain: %w", err)
	}

	return &bundleSourceAdapter{
		identityService: c.identityService,
		trustDomain:     trustDomain,
	}, nil
}

// extractSPIFFEIDFromCert extracts SPIFFE ID from certificate URI SAN
func extractSPIFFEIDFromCert(cert *x509.Certificate) spiffeid.ID {
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			id, err := spiffeid.FromURI(uri)
			if err == nil {
				return id
			}
		}
	}
	// Return zero value if no valid SPIFFE ID found
	return spiffeid.ID{}
}


// createSecureTLSConfig creates a secure TLS configuration as fallback
func (c *ClientConnection) createSecureTLSConfig() *tls.Config {
	return &tls.Config{
		// Use system certificate pool as base
		RootCAs: nil, // nil means use system root CAs
		
		// Enable certificate verification
		InsecureSkipVerify: false,
		
		// Set minimum TLS version
		MinVersion: tls.VersionTLS13,
		
		// Configure cipher suites for security
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}
}
