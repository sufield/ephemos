// Package ports defines properly abstracted interfaces that don't leak infrastructure concerns.
// This file demonstrates the correct way to define port interfaces for hexagonal architecture.
package ports

import (
	"context"
	"io"
)

// HTTPRequest represents an HTTP request abstraction that doesn't leak net/http.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    io.Reader
}

// HTTPResponse represents an HTTP response abstraction that doesn't leak net/http.
type HTTPResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       io.ReadCloser
}

// AbstractedHTTPClient provides authenticated HTTP client capabilities without leaking net/http types.
// This is the proper abstraction that should replace the current HTTPClient usage.
type AbstractedHTTPClient interface {
	// Do executes an HTTP request with authentication credentials automatically included.
	Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
	
	// Close releases resources held by the HTTP client.
	Close() error
}

// NetworkListener provides an abstraction for network listening without leaking net.Listener.
type NetworkListener interface {
	io.Closer
	// Accept waits for and returns the next connection
	Accept() (io.ReadWriteCloser, error)
	// Addr returns the listener's network address as a string
	Addr() string
}

// AbstractedConn represents an authenticated connection to a service without infrastructure leaks.
// This is the proper abstraction that should replace the current Conn interface.
type AbstractedConn interface {
	// HTTPClient returns an HTTP client configured for this authenticated connection.
	// The client will automatically include authentication credentials in requests.
	HTTPClient() (AbstractedHTTPClient, error) // ✅ Uses abstraction, not *http.Client
	
	// Close closes the connection and releases associated resources.
	// Must be safe to call multiple times.
	Close() error
}

// AbstractedDialer provides authenticated connection establishment without infrastructure leaks.
// This is the proper abstraction that should replace the current Dialer interface.
type AbstractedDialer interface {
	// Connect establishes an authenticated connection to the specified service.
	// The serviceName is used for identity verification and authorization.
	// The address specifies where to connect (host:port format).
	Connect(ctx context.Context, serviceName, address string) (AbstractedConn, error)
	
	// Close releases any resources held by the dialer.
	// Must be safe to call multiple times.
	Close() error
}

// AbstractedServer provides authenticated server hosting without infrastructure leaks.
// This is the proper abstraction that should replace the current AuthenticatedServer interface.
type AbstractedServer interface {
	// Serve starts serving requests on the provided listener abstraction.
	// The server will automatically verify client authentication.
	// Blocks until the context is cancelled or an error occurs.
	Serve(ctx context.Context, listener NetworkListener) error // ✅ Uses abstraction, not net.Listener
	
	// Close gracefully shuts down the server.
	// Must be safe to call multiple times.
	Close() error
	
	// Addr returns the network address the server is listening on.
	// Returns empty string if the server is not currently listening.
	Addr() string // ✅ Uses string, not net.Addr
}

// AbstractedConnectionPort represents a connection to a service without infrastructure leaks.
// This is the proper abstraction that should replace the current ConnectionPort interface.
type AbstractedConnectionPort interface {
	// GetClientConnection returns the underlying connection for service clients
	// Note: This still returns interface{} which is not ideal but maintains compatibility
	GetClientConnection() interface{}
	
	// AsReadWriteCloser safely converts the connection to io.ReadWriteCloser if possible
	// Returns nil if the connection doesn't support read/write operations
	AsReadWriteCloser() io.ReadWriteCloser // ✅ Uses standard library interface, not net.Conn
	
	// Close closes the connection
	Close() error
}

// AbstractedServerPort represents a secure server abstraction without infrastructure leaks.
// This is the proper abstraction that should replace the current ServerPort interface.
type AbstractedServerPort interface {
	// RegisterService registers a service with the server
	RegisterService(serviceRegistrar ServiceRegistrarPort) error
	
	// Start begins listening on the provided listener abstraction
	Start(listener NetworkListener) error // ✅ Uses abstraction, not net.Listener
	
	// Stop gracefully shuts down the server
	Stop() error
}

// AbstractedClientPort represents a secure client abstraction without infrastructure leaks.
// This is the proper abstraction that should replace the current ClientPort interface.
type AbstractedClientPort interface {
	// Connect establishes a connection to a service
	Connect(serviceName, address string) (AbstractedConnectionPort, error)
	
	// Close releases client resources
	Close() error
}