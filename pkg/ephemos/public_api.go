// Package ephemos provides identity-based authentication for backend services.
// It provides simple, business-focused APIs that hide all implementation complexity.
package ephemos

import (
	"context"
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
}

// Close closes the connection and releases resources.
func (c *ClientConnection) Close() error {
	// Implementation will be handled internally
	return nil
}

// Server provides identity-based server functionality for hosting services.
type Server interface {
	// RegisterService registers a service implementation with the server.
	RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error
	// ListenAndServe starts the server and serves requests.
	ListenAndServe(ctx context.Context) error
	// Close gracefully shuts down the server.
	Close() error
}

// ServiceRegistrar handles service registration (used by admin/CLI tools).
type ServiceRegistrar interface {
	// Register registers the service with the transport
	Register(transport interface{})
}

// IdentityClient creates a new identity client for connecting to services.
func IdentityClient(ctx context.Context, configPath string) (Client, error) {
	// Implementation will use internal packages
	return createInternalClient(ctx, configPath)
}

// IdentityServer creates a new identity server for hosting services.  
func IdentityServer(ctx context.Context, configPath string) (Server, error) {
	// Implementation will use internal packages
	return createInternalServer(ctx, configPath)
}

// NewServiceRegistrar creates a service registrar for admin/CLI use.
func NewServiceRegistrar(registerFunc func(interface{})) ServiceRegistrar {
	return &serviceRegistrar{registerFunc: registerFunc}
}

// Internal implementations - these will be moved to use internal packages

type serviceRegistrar struct {
	registerFunc func(interface{})
}

func (r *serviceRegistrar) Register(transport interface{}) {
	if r.registerFunc != nil {
		r.registerFunc(transport)
	}
}

// Internal client implementation - will use internal packages
type internalClient struct{}

func (c *internalClient) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	// Will be implemented using internal transport/config packages
	return &ClientConnection{}, nil
}

func (c *internalClient) Close() error {
	return nil
}

func createInternalClient(ctx context.Context, configPath string) (Client, error) {
	return &internalClient{}, nil
}

// Internal server implementation - will use internal packages  
type internalServer struct{}

func (s *internalServer) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	// Will be implemented using internal transport packages
	return nil
}

func (s *internalServer) ListenAndServe(ctx context.Context) error {
	// Will be implemented using internal transport/shutdown packages
	return nil
}

func (s *internalServer) Close() error {
	return nil
}

func createInternalServer(ctx context.Context, configPath string) (Server, error) {
	return &internalServer{}, nil
}