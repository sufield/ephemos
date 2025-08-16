// Package ports defines stable interfaces for the core business capabilities.
// These ports provide the hexagonal architecture boundary between the public API
// and the internal implementation adapters.
package ports

import (
	"context"
	"net"
	"net/http"
)

// Dialer provides authenticated connection establishment capabilities.
// Implementations handle the underlying authentication protocol (e.g., SPIFFE/SPIRE).
type Dialer interface {
	// Connect establishes an authenticated connection to the specified service.
	// The serviceName is used for identity verification and authorization.
	// The address specifies where to connect (host:port format).
	Connect(ctx context.Context, serviceName, address string) (Conn, error)
	
	// Close releases any resources held by the dialer.
	// Must be safe to call multiple times.
	Close() error
}

// Conn represents an authenticated connection to a service.
// Provides access to protocol-specific clients (HTTP, gRPC, etc.).
type Conn interface {
	// HTTPClient returns an HTTP client configured for this authenticated connection.
	// The client will automatically include authentication credentials in requests.
	HTTPClient() (*http.Client, error)
	
	// Close closes the connection and releases associated resources.
	// Must be safe to call multiple times.
	Close() error
}

// AuthenticatedServer provides authenticated server hosting capabilities.
// Implementations handle the underlying authentication protocol (e.g., SPIFFE/SPIRE).
type AuthenticatedServer interface {
	// Serve starts serving requests on the provided listener.
	// The server will automatically verify client authentication.
	// Blocks until the context is cancelled or an error occurs.
	Serve(ctx context.Context, lis net.Listener) error
	
	// Close gracefully shuts down the server.
	// Must be safe to call multiple times.
	Close() error
	
	// Addr returns the network address the server is listening on.
	// Returns nil if the server is not currently listening.
	Addr() net.Addr
}