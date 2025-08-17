// Package ephemos provides identity-based authentication for backend services.
// It provides simple, business-focused APIs that hide all implementation complexity.
//
// # Concepts
//
// Identity: Each service has a unique identity within a trust domain.
// Trust Domain: A security boundary that defines which services can communicate.
// Service Name: A human-readable identifier for the service within the trust domain.
//
// # Basic Usage
//
// Client example:
//
//	config := &ports.Configuration{
//	    Service: ports.ServiceConfig{
//	        Name: "my-service",
//	        Domain: "prod.company.com",
//	    },
//	}
//	client, err := IdentityClient(ctx, WithConfig(config))
//	if err != nil { return err }
//	defer client.Close()
//
//	conn, err := client.Connect(ctx, "payment-service", "payment.company.com:443")
//	if err != nil { return err }
//	defer conn.Close()
//
//	httpClient, err := conn.HTTPClient()
//	if err != nil { return err }
//	resp, err := httpClient.Get("https://payment.company.com/api/balance")
//
// Server example:
//
//	config := &ports.Configuration{
//	    Service: ports.ServiceConfig{
//	        Name: "payment-service",
//	        Domain: "prod.company.com",
//	    },
//	}
//	server, err := IdentityServer(ctx, WithServerConfig(config), WithAddress(":8080"))
//	if err != nil { return err }
//	defer server.Close()
//
//	if err := server.ListenAndServe(ctx); err != nil { return err }
//
// Service registration and management are handled by CLI tools, not the public API.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/factory"
)

// Default timeout values for operations
const (
	DefaultClientTimeout = 30 * time.Second
	DefaultServerTimeout = 30 * time.Second
	DefaultDialTimeout   = 10 * time.Second
)

// Client provides identity-based client functionality for connecting to services.
// All methods are safe for concurrent use by multiple goroutines.
type Client interface {
	// Connect establishes an authenticated connection to the specified target service.
	// The target should be in "host:port" format.
	// Options can be used to configure connection-specific behavior.
	Connect(ctx context.Context, target string, opts ...DialOption) (ClientConnection, error)

	// Close releases any resources held by the client.
	// It is safe to call Close multiple times.
	Close() error
}

// clientConn is the minimal behavior we need from the internal connection.
type clientConn interface {
	HTTPClient() (*http.Client, error)
	Close() error
}

// clientConnectionImpl represents an established authenticated connection to a service.
// Methods are safe for concurrent use by multiple goroutines.
type clientConnectionImpl struct {
	conn   clientConn
	mu     sync.RWMutex
	closed bool
}

// HTTPClient returns an HTTP client configured with SPIFFE certificate authentication.
// The returned client can be used to make authenticated HTTP requests to the connected service.
// Multiple calls return the same client instance.
func (c *clientConnectionImpl) HTTPClient() (*http.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrServerClosed
	}

	if c.conn == nil {
		return nil, ErrNoSPIFFEAuth
	}

	return c.conn.HTTPClient()
}

// Close closes the connection and releases resources.
// It is safe to call Close multiple times.
func (c *clientConnectionImpl) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil // idempotent
	}

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Server provides identity-based server functionality for hosting services.
// All methods are safe for concurrent use by multiple goroutines.
type Server interface {
	// ListenAndServe starts the server and serves requests.
	// Blocks until the context is cancelled or an error occurs.
	ListenAndServe(ctx context.Context) error

	// Close gracefully shuts down the server.
	// It is safe to call Close multiple times.
	Close() error

	// Addr returns the network address the server is listening on.
	// Returns nil if the server is not currently listening.
	Addr() net.Addr
}

// IdentityClient creates a new identity client for connecting to services.
// Configuration can be provided via options, with sensible defaults applied.
func IdentityClient(ctx context.Context, opts ...ClientOption) (Client, error) {
	// Apply options to build configuration
	options := &clientOpts{
		Timeout: DefaultClientTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}

	// If a direct implementation is provided (for testing), use it
	if options.Impl != nil {
		return &clientWrapper{
			dialer:  options.Impl,
			timeout: options.Timeout,
		}, nil
	}

	// Load configuration from options
	config, err := loadClientConfig(options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
	}

	// Create SPIFFE/SPIRE-backed dialer via factory
	dialer, err := factory.SPIFFEDialer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &clientWrapper{
		dialer:  dialer,
		timeout: options.Timeout,
	}, nil
}

// IdentityServer creates a new identity server for hosting services.
// Configuration can be provided via options, with explicit networking required.
func IdentityServer(ctx context.Context, opts ...ServerOption) (Server, error) {
	// Apply options to build configuration
	options := &serverOpts{
		Timeout: DefaultServerTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}

	// If a direct implementation is provided (for testing), use it
	if options.Impl != nil {
		return &serverWrapper{
			impl:     options.Impl,
			listener: options.Listener,
			address:  options.Address,
			timeout:  options.Timeout,
		}, nil
	}

	// Validate networking configuration
	if options.Listener == nil && options.Address == "" {
		return nil, fmt.Errorf("%w: either WithListener or WithAddress must be specified", ErrConfigInvalid)
	}

	// Load configuration from options
	config, err := loadServerConfig(options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
	}

	// Create SPIFFE/SPIRE-backed server via factory
	impl, err := factory.SPIFFEServer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &serverWrapper{
		impl:     impl,
		listener: options.Listener,
		address:  options.Address,
		timeout:  options.Timeout,
	}, nil
}

// IdentityClientFromFile creates a new identity client from a configuration file.
// This is a convenience function that loads configuration from a file.
func IdentityClientFromFile(ctx context.Context, path string, opts ...ClientOption) (Client, error) {
	provider := config.NewFileProvider()
	cfg, err := provider.LoadConfiguration(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from %s: %w", path, err)
	}

	// Prepend the file-based config option
	allOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return IdentityClient(ctx, allOpts...)
}

// IdentityServerFromFile creates a new identity server from a configuration file.
// This is a convenience function that loads configuration from a file.
func IdentityServerFromFile(ctx context.Context, path string, opts ...ServerOption) (Server, error) {
	provider := config.NewFileProvider()
	cfg, err := provider.LoadConfiguration(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from %s: %w", path, err)
	}

	// Prepend the file-based config option
	allOpts := append([]ServerOption{WithServerConfig(cfg)}, opts...)
	return IdentityServer(ctx, allOpts...)
}

// clientWrapper adapts a Dialer to the public Client interface
type clientWrapper struct {
	dialer  ports.Dialer
	timeout time.Duration
	mu      sync.RWMutex
	closed  bool
}

func (c *clientWrapper) Connect(ctx context.Context, target string, opts ...DialOption) (ClientConnection, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrServerClosed
	}

	// Apply dial options
	dialOpts := &dialOpts{
		Timeout: c.timeout,
	}
	for _, opt := range opts {
		opt(dialOpts)
	}

	// Apply timeout to context
	dialCtx := ctx
	if dialOpts.Timeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, dialOpts.Timeout)
		defer cancel()
	}

	// Establish connection using the dialer
	// For now, use a default service name - this could be enhanced with service discovery
	conn, err := c.dialer.Connect(dialCtx, "default", target)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return &clientConnectionImpl{conn: conn}, nil
}

func (c *clientWrapper) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil // idempotent
	}

	c.closed = true
	return c.dialer.Close()
}

// serverWrapper adapts an AuthenticatedServer to the public Server interface
type serverWrapper struct {
	impl     ports.AuthenticatedServer
	listener net.Listener
	address  string
	timeout  time.Duration
	mu       sync.RWMutex
	closed   bool
}

func (s *serverWrapper) ListenAndServe(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrServerClosed
	}

	// Create listener if not provided
	listener := s.listener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", s.address)
		if err != nil {
			return fmt.Errorf("%w: failed to listen on %s: %v", ErrInvalidAddress, s.address, err)
		}
		defer listener.Close()
	}

	// Apply timeout to context if configured
	serverCtx := ctx
	if s.timeout > 0 {
		var cancel context.CancelFunc
		serverCtx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	return s.impl.Serve(serverCtx, listener)
}

func (s *serverWrapper) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil // idempotent
	}

	s.closed = true
	return s.impl.Close()
}

func (s *serverWrapper) Addr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed || s.impl == nil {
		return nil
	}

	return s.impl.Addr()
}

// loadClientConfig loads configuration from client options
func loadClientConfig(opts *clientOpts) (*ports.Configuration, error) {
	if opts.Config != nil {
		return opts.Config, nil
	}

	if opts.Loader != nil {
		// Custom loader would need a source - this is a placeholder
		return nil, fmt.Errorf("custom loader specified but no source provided")
	}

	return nil, fmt.Errorf("no configuration provided")
}

// loadServerConfig loads configuration from server options
func loadServerConfig(opts *serverOpts) (*ports.Configuration, error) {
	if opts.Config != nil {
		return opts.Config, nil
	}

	if opts.Loader != nil {
		// Custom loader would need a source - this is a placeholder
		return nil, fmt.Errorf("custom loader specified but no source provided")
	}

	return nil, fmt.Errorf("no configuration provided")
}
