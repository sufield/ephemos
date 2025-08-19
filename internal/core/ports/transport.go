package ports

import (
	"errors"
	"io"

	"github.com/sufield/ephemos/internal/core/domain"
)

// ErrTransportCreationFailed is returned when transport creation fails.
var ErrTransportCreationFailed = errors.New("failed to create transport")

// ServerPort represents a secure server abstraction without framework dependencies.
type ServerPort interface {
	// RegisterService registers a service with the server
	RegisterService(serviceRegistrar ServiceRegistrarPort) error
	// Start begins listening on the provided listener
	Start(listener NetworkListener) error
	// Stop gracefully shuts down the server
	Stop() error
}

// ClientPort represents a secure client abstraction without framework dependencies.
type ClientPort interface {
	// Connect establishes a connection to a service
	Connect(serviceName, address string) (ConnectionPort, error)
	// Close releases client resources
	Close() error
}

// ConnectionPort represents a connection to a service.
type ConnectionPort interface {
	// GetClientConnection returns the underlying connection for service clients
	GetClientConnection() interface{}
	// AsReadWriteCloser safely converts the connection to io.ReadWriteCloser if possible
	// Returns nil if the connection doesn't support read/write operations
	AsReadWriteCloser() io.ReadWriteCloser
	// Close closes the connection
	Close() error
}

// ServiceRegistrarPort abstracts service registration without framework dependencies.
type ServiceRegistrarPort interface {
	// Register registers the service with the provided server
	Register(server interface{})
}

// TransportProvider provides secure transport without framework dependencies.
type TransportProvider interface {
	CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ServerPort, error)
	CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ClientPort, error)
}
