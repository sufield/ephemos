// Package ephemos provides public interfaces for the ephemos framework.
package ephemos

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// WorkloadServer represents a server that manages workload identities.
type WorkloadServer interface {
	// Start starts the server on the given listener
	Start(ctx context.Context, listener net.Listener) error
	// Stop gracefully stops the server
	Stop() error
	// Close releases resources and shuts down the server
	Close() error
	// RegisterService registers a service with the server
	RegisterService(registrar ServiceRegistrar) error
}

// SPIFFEProvider provides SPIFFE-based identity and certificate management.
type SPIFFEProvider interface {
	// GetTLSConfig returns TLS configuration for the given context
	GetTLSConfig(ctx context.Context) (*tls.Config, error)
	// GetServiceIdentity returns the service's SPIFFE identity
	GetServiceIdentity() (ServiceIdentity, error)
	// Close releases resources
	Close() error
}

// ServiceIdentity represents a service's identity.
type ServiceIdentity interface {
	// GetName returns the service name
	GetName() string
	// GetDomain returns the trust domain
	GetDomain() string
	// GetSPIFFEID returns the full SPIFFE ID
	GetSPIFFEID() string
	// Validate checks if the identity is valid
	Validate() error
}

// TransportAdapter adapts between transport protocols and domain services.
type TransportAdapter interface {
	// Mount mounts a service implementation
	Mount(service interface{}) error
	// GetServer returns the underlying server
	GetServer() interface{}
}


// HTTPAdapter provides HTTP-specific transport adaptation.
type HTTPAdapter interface {
	TransportAdapter
	// ConfigureMiddleware sets up HTTP middleware
	ConfigureMiddleware(middleware ...func(http.Handler) http.Handler)
	// GetHTTPServer returns the underlying HTTP server
	GetHTTPServer() *http.Server
}

// Listener represents a network listener.
type Listener interface {
	// Accept waits for and returns the next connection
	Accept() (net.Conn, error)
	// Close closes the listener
	Close() error
	// Addr returns the listener's network address
	Addr() net.Addr
}
