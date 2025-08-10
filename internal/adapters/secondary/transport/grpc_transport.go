// Package transport provides gRPC transport implementations for secure communication.
package transport

import (
	"context"
	"fmt"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// GRPCProvider provides gRPC transport.
type GRPCProvider struct {
	spiffeProvider *spiffe.Provider
}

// NewGRPCProvider creates a new provider.
func NewGRPCProvider(spiffeProvider *spiffe.Provider) *GRPCProvider {
	return &GRPCProvider{
		spiffeProvider: spiffeProvider,
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

// CreateClient creates client transport implementing domain interface.
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
	dialOption := grpc.WithTransportCredentials(creds)
	
	return &GRPCClient{
		dialOption: dialOption,
	}, nil
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
		return s.server.Serve(netListener.listener)
	}
	
	return fmt.Errorf("listener must be a NetListener wrapping net.Listener")
}

// Stop gracefully shuts down the server.
func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	return nil
}

// GRPCClient implements ports.Client for gRPC.
type GRPCClient struct {
	dialOption grpc.DialOption
}

// Connect establishes a connection to a service.
func (c *GRPCClient) Connect(serviceName, address string) (ports.Connection, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	conn, err := grpc.NewClient(address, c.dialOption)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}

	return &GRPCConnection{
		conn: conn,
	}, nil
}

// Close releases client resources.
func (c *GRPCClient) Close() error {
	return nil // No resources to clean up for dial options
}

// GRPCConnection implements ports.Connection for gRPC.
type GRPCConnection struct {
	conn *grpc.ClientConn
}

// GetClientConnection returns the underlying gRPC connection.
func (c *GRPCConnection) GetClientConnection() interface{} {
	return c.conn
}

// Close closes the connection.
func (c *GRPCConnection) Close() error {
	return c.conn.Close()
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
	return l.listener.Accept()
}

// Close closes the listener.
func (l *NetListener) Close() error {
	return l.listener.Close()
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
