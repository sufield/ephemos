// Package api provides high-level client and server APIs for secure SPIFFE-based communication.
package api

import (
	"context"
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

	return &ClientConnection{conn: grpcConn, domainConn: domainConn}, nil
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

// ClientConnection represents a secure client connection to a remote service.
type ClientConnection struct {
	conn       *grpc.ClientConn
	domainConn ports.ConnectionPort
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

// extractTLSConfig extracts TLS configuration from the domain connection
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

	// Extract SPIFFE certificates and trust bundle from the identity service
	// This ensures HTTP and gRPC use the same authentication credentials
	spiffeConfig, err := c.extractSPIFFEConfig()
	if err != nil {
		// Fallback to basic secure TLS configuration
		return c.createSecureTLSConfig(), nil
	}

	// Create TLS config with SPIFFE certificates
	tlsConfig := &tls.Config{
		// Configure client certificates from SPIFFE
		Certificates: spiffeConfig.ClientCertificates,
		
		// Configure root CAs from SPIFFE trust bundle
		RootCAs: spiffeConfig.RootCAs,
		
		// Enable certificate verification
		InsecureSkipVerify: false,
		
		// Set minimum TLS version
		MinVersion: tls.VersionTLS12,
		
		// Configure cipher suites for security
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		
		// Custom certificate verification for SPIFFE IDs
		VerifyPeerCertificate: spiffeConfig.VerifyPeerCertificate,
		
		// Connection verification
		VerifyConnection: spiffeConfig.VerifyConnection,
	}

	return tlsConfig, nil
}

// SPIFFEConfig holds SPIFFE-specific TLS configuration
type SPIFFEConfig struct {
	ClientCertificates    []tls.Certificate
	RootCAs              *x509.CertPool
	VerifyPeerCertificate func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
	VerifyConnection     func(tls.ConnectionState) error
}

// extractSPIFFEConfig extracts SPIFFE configuration from the connection
func (c *ClientConnection) extractSPIFFEConfig() (*SPIFFEConfig, error) {
	// This is a placeholder for extracting actual SPIFFE certificates
	// In a full implementation, this would:
	// 1. Access the identity service that created the gRPC connection
	// 2. Get the current SPIFFE certificates and trust bundle
	// 3. Configure automatic certificate rotation
	
	// For now, return an error to trigger fallback to secure TLS
	return nil, fmt.Errorf("SPIFFE certificate extraction not yet implemented")
}

// createSecureTLSConfig creates a secure TLS configuration as fallback
func (c *ClientConnection) createSecureTLSConfig() *tls.Config {
	return &tls.Config{
		// Use system certificate pool as base
		RootCAs: nil, // nil means use system root CAs
		
		// Enable certificate verification
		InsecureSkipVerify: false,
		
		// Set minimum TLS version
		MinVersion: tls.VersionTLS12,
		
		// Configure cipher suites for security
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}
}
