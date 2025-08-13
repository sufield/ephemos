package ports

import (
	"errors"

	"github.com/sufield/ephemos/internal/core/domain"
)

// ErrTransportCreationFailed is returned when transport creation fails.
var ErrTransportCreationFailed = errors.New("failed to create transport")

// ServerPort represents a secure server abstraction without framework dependencies.
type ServerPort interface {
	// RegisterService registers a service with the server
	RegisterService(serviceRegistrar ServiceRegistrarPort) error
	// Start begins listening on the provided listener
	Start(listener ListenerPort) error
	// Stop gracefully shuts down the server
	Stop() error
}

// Server is an alias for ServerPort for backward compatibility
type Server = ServerPort

// ClientPort represents a secure client abstraction without framework dependencies.
type ClientPort interface {
	// Connect establishes a connection to a service
	Connect(serviceName, address string) (ConnectionPort, error)
	// Close releases client resources
	Close() error
}

// Client is an alias for ClientPort for backward compatibility
type Client = ClientPort

// ConnectionPort represents a connection to a service.
type ConnectionPort interface {
	// GetClientConnection returns the underlying connection for service clients
	GetClientConnection() interface{}
	// Close closes the connection
	Close() error
}

// Connection is an alias for ConnectionPort for backward compatibility
type Connection = ConnectionPort

// ListenerPort represents a network listener abstraction.
type ListenerPort interface {
	// Accept waits for and returns the next connection
	Accept() (interface{}, error)
	// Close closes the listener
	Close() error
	// Addr returns the listener's network address
	Addr() string
}

// Listener is an alias for ListenerPort for backward compatibility
type Listener = ListenerPort

// ServiceRegistrarPort abstracts service registration without framework dependencies.
type ServiceRegistrarPort interface {
	// Register registers the service with the provided server
	Register(server interface{})
}

// ServiceRegistrar is an alias for ServiceRegistrarPort for backward compatibility
type ServiceRegistrar = ServiceRegistrarPort

// TransportProvider provides secure transport without framework dependencies.
type TransportProvider interface {
	CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ServerPort, error)
	CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ClientPort, error)
}
