package app

import (
	"errors"

	"github.com/sufield/ephemos/internal/domain"
)

// ErrTransportCreationFailed is returned when transport creation fails.
var ErrTransportCreationFailed = errors.New("failed to create transport")

// Server represents a secure server abstraction without framework dependencies.
type Server interface {
	// RegisterService registers a service with the server
	RegisterService(serviceRegistrar ServiceRegistrar) error
	// Start begins listening on the provided listener
	Start(listener Listener) error
	// Stop gracefully shuts down the server
	Stop() error
}

// Client represents a secure client abstraction without framework dependencies.
type Client interface {
	// Connect establishes a connection to a service
	Connect(serviceName, address string) (Connection, error)
	// Close releases client resources
	Close() error
}

// Connection represents a connection to a service.
type Connection interface {
	// GetClientConnection returns the underlying connection for service clients
	GetClientConnection() interface{}
	// Close closes the connection
	Close() error
}

// Listener represents a network listener abstraction.
type Listener interface {
	// Accept waits for and returns the next connection
	Accept() (interface{}, error)
	// Close closes the listener
	Close() error
	// Addr returns the listener's network address
	Addr() string
}

// ServiceRegistrar abstracts service registration without framework dependencies.
type ServiceRegistrar interface {
	// Register registers the service with the provided server
	Register(server interface{})
}

// TransportProvider provides secure transport without framework dependencies.
type TransportProvider interface {
	CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (Server, error)
	CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (Client, error)
}
