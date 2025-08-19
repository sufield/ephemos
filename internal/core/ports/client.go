// Package ports defines stable interfaces for the core business capabilities.
// These ports provide the hexagonal architecture boundary between the public API
// and the internal implementation adapters.
package ports

import (
	"context"
)

// DialerPort provides authenticated connection establishment capabilities.
// Implementations handle the underlying authentication protocol (e.g., SPIFFE/SPIRE).
type DialerPort interface {
	// Connect establishes an authenticated connection to the specified service.
	// The serviceName is used for identity verification and authorization.
	// The address specifies where to connect (host:port format).
	Connect(ctx context.Context, serviceName, address string) (ConnPort, error)

	// Close releases any resources held by the dialer.
	// Must be safe to call multiple times.
	Close() error
}

// ConnPort represents an authenticated connection to a service.
// Provides access to protocol-specific clients (HTTP, gRPC, etc.).
type ConnPort interface {
	// HTTPClient returns an HTTP client configured for this authenticated connection.
	// The client will automatically include authentication credentials in requests.
	HTTPClient() (HTTPClientPort, error)

	// Close closes the connection and releases associated resources.
	// Must be safe to call multiple times.
	Close() error
}

// AuthenticatedServerPort provides authenticated server hosting capabilities.
// Implementations handle the underlying authentication protocol (e.g., SPIFFE/SPIRE).
type AuthenticatedServerPort interface {
	// Serve starts serving requests on the provided listener.
	// The server will automatically verify client authentication.
	// Blocks until the context is cancelled or an error occurs.
	Serve(ctx context.Context, listener NetworkListenerPort) error

	// Close gracefully shuts down the server.
	// Must be safe to call multiple times.
	Close() error

	// Addr returns the network address the server is listening on.
	// Returns empty string if the server is not currently listening.
	Addr() string
}
