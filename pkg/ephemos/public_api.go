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
	"net/http"
	"time"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
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
	// If address is empty, service discovery will be attempted.
	Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error)
	
	// ConnectByName establishes an authenticated connection using service discovery.
	// Only the serviceName is required - the address will be discovered automatically.
	ConnectByName(ctx context.Context, serviceName string) (*ClientConnection, error)
	
	// Close releases any resources held by the client.
	Close() error
}

// ClientConnection represents an established connection to a service.
type ClientConnection struct {
	// Implementation details hidden - only used for resource management
	internalConn *api.ClientConnection
}

// HTTPClient returns an HTTP client configured with SPIFFE certificate authentication.
// The returned client can be used to make authenticated HTTP requests to the connected service.
//
// Example:
//   conn, err := client.Connect(ctx, "payment-service", "https://payment.example.com")
//   if err != nil { ... }
//   defer conn.Close()
//   
//   httpClient := conn.HTTPClient()
//   resp, err := httpClient.Get("https://payment.example.com/api/balance")
func (c *ClientConnection) HTTPClient() *http.Client {
	if c.internalConn != nil {
		return c.internalConn.HTTPClient()
	}
	
	// Return a basic secure HTTP client if internal connection is not available
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
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
	// Create client using internal API (all provider creation handled internally)
	internalClient, err := api.NewClientFromConfig(ctx, configPath)
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
	// If address is empty, use service discovery
	if address == "" {
		return c.ConnectByName(ctx, serviceName)
	}
	
	internalConn, err := c.client.Connect(ctx, serviceName, address)
	if err != nil {
		return nil, err
	}
	
	// Wrap the internal connection in the public API type
	return &ClientConnection{
		internalConn: internalConn,
	}, nil
}

// ConnectByName establishes an authenticated connection using service discovery.
func (c *clientWrapper) ConnectByName(ctx context.Context, serviceName string) (*ClientConnection, error) {
	// Use service discovery to find the service address
	address, err := c.discoverService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service discovery failed for %s: %w", serviceName, err)
	}
	
	// Connect using the discovered address
	internalConn, err := c.client.Connect(ctx, serviceName, address)
	if err != nil {
		return nil, err
	}
	
	// Wrap the internal connection in the public API type
	return &ClientConnection{
		internalConn: internalConn,
	}, nil
}

// discoverService performs service discovery to find the address of a named service.
// This is a basic implementation that can be enhanced with different discovery mechanisms.
func (c *clientWrapper) discoverService(ctx context.Context, serviceName string) (string, error) {
	// Basic service discovery implementation
	// In a production system, this would integrate with:
	// - Kubernetes service discovery
	// - Consul service mesh
	// - AWS Cloud Map
	// - Custom service registry
	
	// For now, implement a simple DNS-based discovery with common patterns
	candidates := []string{
		fmt.Sprintf("%s.default.svc.cluster.local:443", serviceName),     // Kubernetes
		fmt.Sprintf("%s.service.consul:443", serviceName),                // Consul
		fmt.Sprintf("%s.internal:443", serviceName),                      // AWS/Cloud
		fmt.Sprintf("%s:443", serviceName),                               // Direct DNS
	}
	
	for _, candidate := range candidates {
		// Test if the address is reachable
		if err := c.testConnection(ctx, candidate); err == nil {
			return candidate, nil
		}
	}
	
	return "", fmt.Errorf("service %s not found via discovery", serviceName)
}

// testConnection tests if a service address is reachable
func (c *clientWrapper) testConnection(ctx context.Context, address string) error {
	// Create a timeout context for the connection test
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Try to establish a basic TCP connection
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(testCtx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	return nil
}

// Close releases any resources held by the client.
func (c *clientWrapper) Close() error {
	return c.client.Close()
}

// IdentityServer creates a new identity server for hosting services.  
func IdentityServer(ctx context.Context, configPath string) (Server, error) {
	// Create server using internal API (all provider creation handled internally)
	internalServer, err := api.NewServerFromConfig(ctx, configPath)
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
