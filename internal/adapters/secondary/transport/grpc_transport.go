// Package transport provides gRPC transport implementations for secure communication with
// advanced connection management, backoff strategies, and retry policies.
package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// GRPCProvider provides gRPC transport with advanced connection management.
type GRPCProvider struct {
	spiffeProvider   *spiffe.Provider
	connectionConfig *ConnectionConfig
}

// NewGRPCProvider creates a new provider with default connection configuration.
func NewGRPCProvider(spiffeProvider *spiffe.Provider) *GRPCProvider {
	return &GRPCProvider{
		spiffeProvider:   spiffeProvider,
		connectionConfig: DefaultConnectionConfig(),
	}
}

// NewGRPCProviderWithConfig creates a new provider with custom connection configuration.
func NewGRPCProviderWithConfig(spiffeProvider *spiffe.Provider, config *ConnectionConfig) *GRPCProvider {
	return &GRPCProvider{
		spiffeProvider:   spiffeProvider,
		connectionConfig: config,
	}
}

// CreateServer creates server transport implementing domain interface.
func (p *GRPCProvider) CreateServer(
	_ *domain.Certificate,
	_ *domain.TrustBundle,
	policy *domain.AuthenticationPolicy,
) (ports.Server, error) {
	source := p.spiffeProvider.GetX509Source()
	if source == nil {
		return nil, fmt.Errorf("X509 source not initialized")
	}

	tlsConfig := tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(p.createAuthInterceptor(policy)),
	}

	grpcServer := grpc.NewServer(opts...)

	return &GRPCServer{
		server: grpcServer,
	}, nil
}

// CreateClient creates client transport implementing domain interface with advanced connection management.
func (p *GRPCProvider) CreateClient(
	_ *domain.Certificate,
	_ *domain.TrustBundle,
	_ *domain.AuthenticationPolicy,
) (ports.Client, error) {
	source := p.spiffeProvider.GetX509Source()
	if source == nil {
		return nil, fmt.Errorf("X509 source not initialized")
	}

	tlsConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsConfig.ServerName = ""

	creds := credentials.NewTLS(tlsConfig)
	credentialOption := grpc.WithTransportCredentials(creds)

	client := &GRPCClient{
		credentialOption: credentialOption,
		connectionConfig: p.connectionConfig,
		connectionPool:   make(map[string]*pooledConnection),
	}

	// Start background cleanup goroutine for connection pool if pooling is enabled
	if p.connectionConfig.EnablePooling {
		go client.poolCleanupRoutine()
	}

	return client, nil
}

// Pool cleanup constants.
const (
	poolCleanupInterval = 30 * time.Second
	poolIdleTimeout     = 10 * time.Minute
)

// GRPCClient implements ports.Client for gRPC with advanced connection management.
type GRPCClient struct {
	credentialOption grpc.DialOption
	connectionConfig *ConnectionConfig
	connectionPool   map[string]*pooledConnection
	poolMutex        sync.RWMutex
}

// poolCleanupRoutine periodically cleans up unused pooled connections.
func (c *GRPCClient) poolCleanupRoutine() {
	ticker := time.NewTicker(poolCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupIdleConnections()
	}
}

func (c *GRPCClient) cleanupIdleConnections() {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	now := time.Now()
	idleTimeout := poolIdleTimeout

	for key, pooled := range c.connectionPool {
		pooled.mutex.RLock()
		isIdle := pooled.inUse == 0 && now.Sub(pooled.lastUsed) > idleTimeout
		state := pooled.conn.GetState()
		pooled.mutex.RUnlock()

		// Remove connections that are idle too long or in bad state
		if isIdle || state == connectivity.Shutdown || state == connectivity.TransientFailure {
			if err := pooled.conn.Close(); err != nil {
				log.Printf("cleanup close error: %v", err)
			}
			delete(c.connectionPool, key)
		}
	}
}

// GRPCServer implements ports.Server for gRPC.
type GRPCServer struct {
	server *grpc.Server
}

// RegisterService registers a service with the gRPC server.
func (s *GRPCServer) RegisterService(serviceRegistrar ports.ServiceRegistrar) error {
	if serviceRegistrar == nil {
		return fmt.Errorf("service registrar cannot be nil")
	}
	serviceRegistrar.Register(s.server)
	return nil
}

// Start begins listening on the provided listener.
func (s *GRPCServer) Start(listener ports.Listener) error {
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}

	// Convert our domain listener to a net.Listener
	// We know NetListener wraps net.Listener, so extract it
	if netListener, ok := listener.(*NetListener); ok {
		if err := s.server.Serve(netListener.listener); err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil
	}

	return fmt.Errorf("listener must be a NetListener wrapping net.Listener")
}

// Stop gracefully shuts down the server.
func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	return nil
}

type pooledConnection struct {
	conn        *grpc.ClientConn
	serviceName string
	address     string
	lastUsed    time.Time
	inUse       int32
	mutex       sync.RWMutex
}

// Connect establishes a connection to a service with advanced connection management.
func (c *GRPCClient) Connect(serviceName, address string) (ports.Connection, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Check if pooling is enabled
	if c.connectionConfig.EnablePooling {
		return c.getPooledConnection(serviceName, address)
	}

	// Create a new connection with full configuration
	return c.createConnection(serviceName, address)
}

func (c *GRPCClient) createConnection(serviceName, address string) (ports.Connection, error) {
	// Build dial options with connection configuration
	dialOptions := c.connectionConfig.ToDialOptions()
	dialOptions = append(dialOptions, c.credentialOption)

	// Create connection (dialing is lazy and handled via connection params)
	conn, err := grpc.NewClient(address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for service %s at %s: %w", serviceName, address, err)
	}

	return &GRPCConnection{
		conn:        conn,
		serviceName: serviceName,
		address:     address,
	}, nil
}

func (c *GRPCClient) getPooledConnection(serviceName, address string) (ports.Connection, error) {
	key := fmt.Sprintf("%s:%s", serviceName, address)

	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	// Check if connection exists in pool
	if pooled, exists := c.connectionPool[key]; exists {
		pooled.mutex.RLock()
		state := pooled.conn.GetState()
		pooled.mutex.RUnlock()

		// If connection is healthy, reuse it
		if state == connectivity.Ready || state == connectivity.Idle {
			pooled.mutex.Lock()
			pooled.inUse++
			pooled.lastUsed = time.Now()
			pooled.mutex.Unlock()

			return &GRPCConnection{
				conn:        pooled.conn,
				serviceName: serviceName,
				address:     address,
				pooled:      pooled,
			}, nil
		}

		// Connection is unhealthy, remove from pool
		if err := pooled.conn.Close(); err != nil {
			log.Printf("cleanup close error: %v", err)
		}
		delete(c.connectionPool, key)
	}

	// Create new connection
	conn, err := c.createConnection(serviceName, address)
	if err != nil {
		return nil, err
	}

	// Add to pool
	grpcConn, ok := conn.(*GRPCConnection)
	if !ok {
		return nil, fmt.Errorf("expected *GRPCConnection, got %T", conn)
	}
	pooled := &pooledConnection{
		conn:        grpcConn.conn,
		serviceName: serviceName,
		address:     address,
		lastUsed:    time.Now(),
		inUse:       1,
	}

	c.connectionPool[key] = pooled
	grpcConn.pooled = pooled

	return conn, nil
}

// Close releases client resources and closes all pooled connections.
func (c *GRPCClient) Close() error {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	for key, pooled := range c.connectionPool {
		if err := pooled.conn.Close(); err != nil {
			log.Printf("cleanup close error: %v", err)
		}
		delete(c.connectionPool, key)
	}

	return nil
}

// GRPCConnection implements ports.Connection for gRPC with connection health monitoring.
type GRPCConnection struct {
	conn        *grpc.ClientConn
	serviceName string
	address     string
	pooled      *pooledConnection // nil if not pooled
}

// GetClientConnection returns the underlying gRPC connection.
func (c *GRPCConnection) GetClientConnection() interface{} {
	return c.conn
}

// GetState returns the current connectivity state of the connection.
func (c *GRPCConnection) GetState() connectivity.State {
	return c.conn.GetState()
}

// WaitForStateChange blocks until the connection state changes or context is canceled.
func (c *GRPCConnection) WaitForStateChange(ctx context.Context, sourceState connectivity.State) bool {
	return c.conn.WaitForStateChange(ctx, sourceState)
}

// IsHealthy checks if the connection is in a healthy state.
func (c *GRPCConnection) IsHealthy() bool {
	state := c.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// Close closes the connection. For pooled connections, decrements usage counter.
func (c *GRPCConnection) Close() error {
	if c.pooled != nil {
		// This is a pooled connection, just decrement usage
		c.pooled.mutex.Lock()
		c.pooled.inUse--
		c.pooled.mutex.Unlock()
		return nil
	}

	// This is a direct connection, close it
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection to %s at %s: %w", c.serviceName, c.address, err)
	}
	return nil
}

// NetListener adapts net.Listener to ports.Listener.
type NetListener struct {
	listener net.Listener
}

// NewNetListener creates a new NetListener.
func NewNetListener(listener net.Listener) ports.Listener {
	return &NetListener{listener: listener}
}

// Accept waits for and returns the next connection.
func (l *NetListener) Accept() (interface{}, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("failed to accept connection: %w", err)
	}
	return conn, nil
}

// Close closes the listener.
func (l *NetListener) Close() error {
	if err := l.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}
	return nil
}

// Addr returns the listener's network address.
func (l *NetListener) Addr() string {
	return l.listener.Addr().String()
}

func (p *GRPCProvider) createAuthInterceptor(_ *domain.AuthenticationPolicy) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}
