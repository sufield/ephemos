package ephemos

import (
	"net"
	"time"

	"github.com/sufield/ephemos/internal/core/ports"
)

// ClientOption configures client creation behavior.
type ClientOption func(*clientOpts)

// clientOpts holds the configuration for client creation.
type clientOpts struct {
	Config  *ports.Configuration
	Loader  ConfigLoader
	Impl    ports.Dialer // direct injection for tests
	Timeout time.Duration
}

// WithConfig provides an in-memory configuration for the client.
// This is the preferred method for production use as it avoids file I/O.
func WithConfig(config *ports.Configuration) ClientOption {
	return func(opts *clientOpts) {
		opts.Config = config
	}
}

// WithConfigLoader provides a custom configuration loader.
// This allows for loading configuration from non-standard sources.
func WithConfigLoader(loader ConfigLoader) ClientOption {
	return func(opts *clientOpts) {
		opts.Loader = loader
	}
}

// WithDialer provides a custom Dialer implementation.
// This is primarily used for testing with mock implementations.
func WithDialer(dialer ports.Dialer) ClientOption {
	return func(opts *clientOpts) {
		opts.Impl = dialer
	}
}

// WithClientTimeout sets the default timeout for client operations.
// If not specified, a reasonable default timeout will be used.
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(opts *clientOpts) {
		opts.Timeout = timeout
	}
}

// ServerOption configures server creation behavior.
type ServerOption func(*serverOpts)

// serverOpts holds the configuration for server creation.
type serverOpts struct {
	Config   *ports.Configuration
	Loader   ConfigLoader
	Listener net.Listener
	Address  string
	Impl     ports.AuthenticatedServer // direct injection for tests
	Timeout  time.Duration
}

// WithServerConfig provides an in-memory configuration for the server.
// This is the preferred method for production use as it avoids file I/O.
func WithServerConfig(config *ports.Configuration) ServerOption {
	return func(opts *serverOpts) {
		opts.Config = config
	}
}

// WithServerConfigLoader provides a custom configuration loader for the server.
// This allows for loading configuration from non-standard sources.
func WithServerConfigLoader(loader ConfigLoader) ServerOption {
	return func(opts *serverOpts) {
		opts.Loader = loader
	}
}

// WithListener provides a specific network listener for the server.
// This is useful for tests and when you need precise control over the listening socket.
func WithListener(listener net.Listener) ServerOption {
	return func(opts *serverOpts) {
		opts.Listener = listener
	}
}

// WithAddress specifies the network address for the server to listen on.
// The address should be in the format "host:port".
// If not provided, the server will need a listener via WithListener.
func WithAddress(address string) ServerOption {
	return func(opts *serverOpts) {
		opts.Address = address
	}
}

// WithServerImpl provides a custom AuthenticatedServer implementation.
// This is primarily used for testing with mock implementations.
func WithServerImpl(impl ports.AuthenticatedServer) ServerOption {
	return func(opts *serverOpts) {
		opts.Impl = impl
	}
}

// WithServerTimeout sets the default timeout for server operations.
// If not specified, a reasonable default timeout will be used.
func WithServerTimeout(timeout time.Duration) ServerOption {
	return func(opts *serverOpts) {
		opts.Timeout = timeout
	}
}

// DialOption configures connection establishment behavior.
type DialOption func(*dialOpts)

// dialOpts holds the configuration for connection establishment.
type dialOpts struct {
	Timeout time.Duration
}

// WithDialTimeout sets the timeout for connection establishment.
// If not specified, the client's default timeout will be used.
func WithDialTimeout(timeout time.Duration) DialOption {
	return func(opts *dialOpts) {
		opts.Timeout = timeout
	}
}

// ConfigLoader defines an interface for loading configuration from various sources.
// This allows for custom configuration loading strategies beyond simple file paths.
type ConfigLoader interface {
	// LoadConfiguration loads configuration from the specified source.
	// The source parameter can be a file path, URL, or other identifier
	// depending on the implementation.
	LoadConfiguration(source string) (*ports.Configuration, error)
}
