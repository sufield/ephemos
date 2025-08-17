// Package transport provides concrete transport provider implementations.
package transport

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// grpcClient implements ports.ClientPort.
type grpcClient struct {
	tlsConfig *tls.Config
	policy    *domain.AuthenticationPolicy
	closed    bool // Track if client has been closed
}

// Connect establishes a secure gRPC connection to the specified address.
func (c *grpcClient) Connect(serviceName, address string) (ports.ConnectionPort, error) {
	// Validate inputs
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Check if client is closed
	if c.closed {
		return nil, fmt.Errorf("client has been closed and cannot create new connections")
	}

	// Validate TLS configuration
	if c.tlsConfig == nil {
		return nil, fmt.Errorf("TLS configuration is required but not provided")
	}

	// Create credentials
	creds := credentials.NewTLS(c.tlsConfig)

	// Configure connection options with modern gRPC practices
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		// Enable keepalive for better connection health
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // Send keepalive pings every 10 seconds
			Timeout:             3 * time.Second,  // Wait 3 seconds for ping ack before considering connection dead
			PermitWithoutStream: true,             // Send pings even when no active RPCs
		}),
		// Set maximum message sizes (4MB default is usually fine, but being explicit)
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	// Establish connection with modern gRPC practices (grpc.NewClient is preferred over grpc.Dial)
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		// Provide more specific error information based on error type
		if isNetworkError(err) {
			return nil, fmt.Errorf("network error connecting to service %q at %s: %w", serviceName, address, err)
		} else if isTLSError(err) {
			return nil, fmt.Errorf("TLS/authentication error connecting to service %q at %s: %w", serviceName, address, err)
		}
		return nil, fmt.Errorf("failed to connect to service %q at %s: %w", serviceName, address, err)
	}

	// Note: grpc.NewClient establishes connections lazily, so we don't need to wait for readiness here
	// The connection will be established on the first RPC call

	return &grpcConnection{
		conn:        conn,
		serviceName: serviceName,
	}, nil
}

// isNetworkError checks if the error is network-related.
func isNetworkError(err error) bool {
	// Check for common network error patterns
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "network unreachable") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "no such host")
}

// isTLSError checks if the error is TLS/authentication-related.
func isTLSError(err error) bool {
	// Check for common TLS error patterns
	return strings.Contains(err.Error(), "certificate") ||
		strings.Contains(err.Error(), "tls") ||
		strings.Contains(err.Error(), "handshake") ||
		strings.Contains(err.Error(), "authentication")
}

// Close releases any resources held by the client.
func (c *grpcClient) Close() error {
	// Mark as closed to prevent new connections
	c.closed = true

	// No persistent resources to clean up in this implementation
	// Individual connections are responsible for their own cleanup
	return nil
}

// grpcServer implements ports.ServerPort.
type grpcServer struct {
	tlsConfig   *tls.Config
	policy      *domain.AuthenticationPolicy
	server      *grpc.Server
	initialized bool       // Track initialization state
	serving     bool       // Track serving state
	mu          sync.Mutex // Protect concurrent access to state
}

// RegisterService registers a service implementation with the gRPC server.
func (s *grpcServer) RegisterService(registrar ports.ServiceRegistrarPort) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate inputs
	if registrar == nil {
		return fmt.Errorf("service registrar cannot be nil")
	}

	// Check if server is already serving
	if s.serving {
		return fmt.Errorf("cannot register services on a server that is already serving")
	}

	// Initialize server if needed
	if s.server == nil {
		if err := s.initializeServer(); err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}

	// Register the service
	registrar.Register(s.server)
	s.initialized = true
	return nil
}

// initializeServer creates and configures the underlying gRPC server.
func (s *grpcServer) initializeServer() error {
	if s.tlsConfig == nil {
		return fmt.Errorf("TLS configuration is required but not provided")
	}

	// Create credentials
	creds := credentials.NewTLS(s.tlsConfig)

	// Configure server options with modern gRPC practices
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		// Enable keepalive enforcement for better connection health
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // Minimum time between keepalive pings
			PermitWithoutStream: true,            // Allow pings even when no active RPCs
		}),
		// Configure server keepalive parameters
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute, // Close idle connections after 15 minutes
			MaxConnectionAge:      30 * time.Minute, // Close connections after 30 minutes regardless of activity
			MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for active RPCs to complete before force-closing
			Time:                  5 * time.Second,  // Send keepalive pings every 5 seconds
			Timeout:               1 * time.Second,  // Wait 1 second for ping ack before considering connection dead
		}),
		// Set maximum message sizes
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
		// Set maximum concurrent streams per connection
		grpc.MaxConcurrentStreams(1000),
	}

	s.server = grpc.NewServer(opts...)
	return nil
}

// Start starts the gRPC server on the provided listener.
func (s *grpcServer) Start(listener ports.ListenerPort) error {
	// Validate inputs first (without lock)
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}

	// Acquire lock for state checks and updates
	s.mu.Lock()

	// Check initialization state
	if s.server == nil {
		s.mu.Unlock()
		return fmt.Errorf("server not initialized - call RegisterService first")
	}

	if !s.initialized {
		s.mu.Unlock()
		return fmt.Errorf("no services registered - call RegisterService first")
	}

	// Check if already serving
	if s.serving {
		s.mu.Unlock()
		return fmt.Errorf("server is already serving")
	}

	// Mark as serving before starting (in case Serve blocks)
	s.serving = true
	s.mu.Unlock()

	// Create a net.Listener from the ListenerPort
	netListener := &listenerAdapter{port: listener}

	// Start serving (this call blocks until server stops)
	err := s.server.Serve(netListener)

	// Mark as not serving when we return
	s.mu.Lock()
	s.serving = false
	s.mu.Unlock()

	return err
}

// Stop gracefully shuts down the gRPC server.
func (s *grpcServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		// Already stopped or never initialized
		return nil
	}

	if !s.serving {
		// Not currently serving, but server exists - this is fine
		return nil
	}

	// Gracefully stop the server
	// Note: This call may block briefly while stopping active connections
	s.server.GracefulStop()
	s.serving = false

	return nil
}

// grpcConnection implements ports.ConnectionPort.
type grpcConnection struct {
	conn        *grpc.ClientConn
	serviceName string
	closed      bool       // Track if connection has been closed
	mu          sync.Mutex // Protect concurrent access to closed state
}

// GetClientConnection returns the underlying gRPC client connection.
func (c *grpcConnection) GetClientConnection() interface{} {
	return c.conn
}

// AsNetConn safely converts the connection to a net.Conn if possible.
// gRPC connections are not net.Conn instances, so this returns nil.
func (c *grpcConnection) AsNetConn() net.Conn {
	// gRPC ClientConn is not a net.Conn - it manages multiple HTTP/2 streams
	// Return nil to indicate this connection type doesn't support net.Conn interface
	return nil
}

// Close closes the gRPC connection.
func (c *grpcConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already closed to prevent double-close
	if c.closed {
		return nil // Safe to call multiple times
	}

	// Validate that connection exists
	if c.conn == nil {
		c.closed = true
		return nil // Nothing to close
	}

	// Close the underlying connection
	err := c.conn.Close()
	c.closed = true // Mark as closed regardless of error

	if err != nil {
		return fmt.Errorf("failed to close gRPC connection to service %q: %w", c.serviceName, err)
	}

	return nil
}

// listenerAdapter adapts ports.ListenerPort to net.Listener.
type listenerAdapter struct {
	port   ports.ListenerPort
	closed bool       // Track if listener has been closed
	mu     sync.Mutex // Protect concurrent access to closed state
}

// Accept waits for and returns the next connection to the listener.
func (l *listenerAdapter) Accept() (net.Conn, error) {
	// Check if listener is closed
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, fmt.Errorf("listener is closed")
	}
	l.mu.Unlock()

	conn, err := l.port.Accept()
	if err != nil {
		return nil, err
	}

	// Try to safely convert to ConnectionPort first
	if connPort, ok := conn.(ports.ConnectionPort); ok {
		// Use the safe AsNetConn method to get net.Conn
		if netConn := connPort.AsNetConn(); netConn != nil {
			return netConn, nil
		}
		return nil, fmt.Errorf("connection does not support net.Conn interface")
	}

	// Fallback: direct net.Conn type assertion for raw connections
	if netConn, ok := conn.(net.Conn); ok {
		return netConn, nil
	}

	return nil, fmt.Errorf("connection is neither ConnectionPort nor net.Conn (got %T)", conn)
}

// Close closes the listener.
func (l *listenerAdapter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already closed to prevent double-close
	if l.closed {
		return nil // Safe to call multiple times
	}

	// Validate that port exists
	if l.port == nil {
		l.closed = true
		return nil // Nothing to close
	}

	// Close the underlying port
	err := l.port.Close()
	l.closed = true // Mark as closed regardless of error

	if err != nil {
		return fmt.Errorf("failed to close listener adapter: %w", err)
	}

	return nil
}

// Addr returns the listener's network address.
func (l *listenerAdapter) Addr() net.Addr {
	// Get the actual address from the underlying port
	addrStr := l.port.Addr()

	// Parse the address string into a proper net.Addr
	if tcpAddr, err := net.ResolveTCPAddr("tcp", addrStr); err == nil {
		return tcpAddr
	}

	// Fallback for other address types or parsing errors
	if addr, err := net.ResolveUnixAddr("unix", addrStr); err == nil {
		return addr
	}

	// Last resort: return a custom address implementation
	return &stringAddr{addr: addrStr}
}

// stringAddr implements net.Addr for addresses that can't be parsed as TCP/Unix
type stringAddr struct {
	addr string
}

func (s *stringAddr) Network() string {
	return "unknown"
}

func (s *stringAddr) String() string {
	return s.addr
}
