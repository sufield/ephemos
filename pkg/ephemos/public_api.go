// Package ephemos provides identity-based authentication for backend services.
// It provides simple, business-focused APIs that hide all implementation complexity.
//
// The public API focuses on two core operations:
//   - IdentityServer(): Create a server that can verify client identities
//   - IdentityClient(): Create a client that can authenticate to servers
//
// Service registration and management are handled by CLI tools, not the public API.
package ephemos

import (
	"context"
	"fmt"
	"net"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
)

// ServiceIdentity represents a service's identity in business terms.
type ServiceIdentity interface {
	// GetName returns the service name
	GetName() string
	// GetDomain returns the trust domain  
	GetDomain() string
	// Validate checks if the identity is valid
	Validate() error
}

// Configuration represents the basic configuration needed for identity-based authentication.
type Configuration struct {
	// Service contains the core service identification settings.
	Service ServiceConfig `yaml:"service"`
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	// Name is the unique identifier for this service.
	Name string `yaml:"name"`
	// Domain is the trust domain for this service.
	Domain string `yaml:"domain,omitempty"`
}

// Client provides identity-based client functionality for connecting to services.
type Client interface {
	// Connect establishes an authenticated connection to the specified service.
	// The serviceName is used for identity verification and must be non-empty.
	// The address should be in the format "host:port" and must be non-empty.
	Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error)
	// Close releases any resources held by the client.
	Close() error
}

// ClientConnection represents an established connection to a service.
type ClientConnection struct {
	// Implementation details hidden - only used for resource management
	internalConn *api.ClientConnection
}

// Close closes the connection and releases resources.
func (c *ClientConnection) Close() error {
	if c.internalConn != nil {
		return c.internalConn.Close()
	}
	return nil
}

// Server provides identity-based server functionality for hosting services.
type Server interface {
	// ListenAndServe starts the server and serves requests.
	ListenAndServe(ctx context.Context) error
	// Close gracefully shuts down the server.
	Close() error
}


// IdentityClient creates a new identity client for connecting to services.
func IdentityClient(ctx context.Context, configPath string) (Client, error) {
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

	// Create client using internal API (transport provider created internally)
	internalClient, err := api.NewClient(identityProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &clientWrapper{client: internalClient}, nil
}

// clientWrapper adapts internal client to public Client interface.
type clientWrapper struct {
	client *api.Client
}

// Connect establishes an authenticated connection to the specified service.
func (c *clientWrapper) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	internalConn, err := c.client.Connect(ctx, serviceName, address)
	if err != nil {
		return nil, err
	}
	
	// Wrap the internal connection in the public API type
	return &ClientConnection{
		internalConn: internalConn,
	}, nil
}

// Close releases any resources held by the client.
func (c *clientWrapper) Close() error {
	return c.client.Close()
}

// IdentityServer creates a new identity server for hosting services.  
func IdentityServer(ctx context.Context, configPath string) (Server, error) {
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

	// Create server using internal API (transport provider created internally)
	internalServer, err := api.NewServer(identityProvider, configProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &serverWrapper{server: internalServer}, nil
}

// serverWrapper adapts internal server to public Server interface.
type serverWrapper struct {
	server *api.Server
}


// ListenAndServe starts the server and serves requests.
func (s *serverWrapper) ListenAndServe(ctx context.Context) error {
	// For now, use a default listener - in production this should be configurable
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()
	
	return s.server.Serve(ctx, listener)
}

// Close gracefully shuts down the server.
func (s *serverWrapper) Close() error {
	return s.server.Close()
}
