package ephemos

import (
	"context"
	"net"
	"net/http"
)

// Configuration represents the service configuration.
// This is a public interface that abstracts internal configuration details.
type Configuration interface {
	// Validate checks if the configuration is valid.
	Validate() error
	
	// IsProductionReady checks if the configuration is suitable for production use.
	IsProductionReady() error
}

// Dialer creates authenticated connections to remote services.
type Dialer interface {
	// Connect establishes a connection to the target service with authentication.
	Connect(ctx context.Context, serviceName, target string) (ClientConnection, error)
	
	// Close releases any resources held by the dialer.
	Close() error
}

// ClientConnection represents a connection to a service.
type ClientConnection interface {
	// HTTPClient returns an HTTP client for this connection.
	HTTPClient() (*http.Client, error)
	
	// Close closes the connection.
	Close() error
}

// AuthenticatedServer provides authenticated server capabilities.
type AuthenticatedServer interface {
	// HTTPHandler returns an HTTP handler with authentication middleware.
	HTTPHandler() http.Handler
	
	// Serve starts serving on the provided listener.
	Serve(ctx context.Context, listener net.Listener) error
	
	// Close gracefully shuts down the server.
	Close() error
	
	// Addr returns the server's listening address.
	Addr() net.Addr
}

// ConfigLoader defines an interface for loading configuration from various sources.
// This allows for custom configuration loading strategies beyond simple file paths.
type ConfigLoader interface {
	// LoadConfiguration loads configuration from the specified source.
	// The source parameter can be a file path, URL, or other identifier
	// depending on the implementation.
	// The context allows for cancellation and deadlines during loading.
	LoadConfiguration(ctx context.Context, source string) (Configuration, error)
}