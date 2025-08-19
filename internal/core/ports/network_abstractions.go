// Package ports - Network abstractions that don't leak net package infrastructure types.
// These types provide proper abstractions for network operations in hexagonal architecture.
package ports

import (
	"context"
	"io"
)

// NetworkListener abstracts listening without net.Listener.
// This interface replaces direct usage of net.Listener in port signatures.
type NetworkListener interface {
	io.Closer

	// Accept returns the next connection as a generic ReadWriteCloser.
	// This supports deadlines/timeouts via io if needed in adapters.
	Accept() (io.ReadWriteCloser, error)

	// Addr returns the listening address as a string (e.g., "localhost:8080").
	// This avoids leaking net.Addr types into the core domain.
	Addr() string
}

// ClientTransport provides a typed interface for client connections.
// This replaces the vague GetClientConnection() interface{} pattern.
type ClientTransport interface {
	// Send data over the transport with context for cancellation.
	Send(ctx context.Context, data []byte) error

	// Receive data from the transport with context for cancellation.
	Receive(ctx context.Context) ([]byte, error)

	// Close the transport connection.
	Close() error
}
