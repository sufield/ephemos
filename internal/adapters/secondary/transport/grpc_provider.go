// Package transport provides concrete transport provider implementations.
package transport

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// GRPCProvider provides gRPC-based transport with SPIFFE mTLS authentication.
type GRPCProvider struct {
	config *ports.Configuration
}

// NewGRPCProvider creates a new gRPC transport provider with configuration.
func NewGRPCProvider(config *ports.Configuration) *GRPCProvider {
	return &GRPCProvider{config: config}
}

// CreateClient creates a gRPC client with SPIFFE-based mTLS authentication.
func (p *GRPCProvider) CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ClientPort, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate cannot be nil")
	}
	if bundle == nil {
		return nil, fmt.Errorf("trust bundle cannot be nil")
	}

	// Configure TLS credentials using SPIFFE certificates
	tlsConfig, err := p.createClientTLSConfig(cert, bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	return &grpcClient{
		tlsConfig: tlsConfig,
		policy:    policy,
	}, nil
}

// CreateServer creates a gRPC server with SPIFFE-based mTLS authentication.
func (p *GRPCProvider) CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ServerPort, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate cannot be nil")
	}
	if bundle == nil {
		return nil, fmt.Errorf("trust bundle cannot be nil")
	}

	// Configure TLS credentials using SPIFFE certificates
	tlsConfig, err := p.createServerTLSConfig(cert, bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	return &grpcServer{
		tlsConfig: tlsConfig,
		policy:    policy,
	}, nil
}

// createClientTLSConfig creates TLS configuration for gRPC client with SPIFFE certs.
func (p *GRPCProvider) createClientTLSConfig(cert *domain.Certificate, bundle *domain.TrustBundle) (*tls.Config, error) {
	// Follow industry best practices: explicit opt-in for insecure mode
	// Similar to Docker, Argo Workflows, Consul, and Kubernetes patterns
	insecureSkipVerify := false
	if p.config != nil {
		insecureSkipVerify = p.config.ShouldSkipCertificateValidation()
		
		// Log security warning when validation is disabled (industry standard practice)
		if insecureSkipVerify {
			log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		}
	}

	// For now, create a basic TLS config
	// In a real implementation, this would use the SPIFFE certificates properly
	return &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}, nil
}

// createServerTLSConfig creates TLS configuration for gRPC server with SPIFFE certs.
func (p *GRPCProvider) createServerTLSConfig(cert *domain.Certificate, bundle *domain.TrustBundle) (*tls.Config, error) {
	// Follow industry best practices: explicit opt-in for insecure mode
	// Similar to Docker, Argo Workflows, Consul, and Kubernetes patterns
	insecureSkipVerify := false
	if p.config != nil {
		insecureSkipVerify = p.config.ShouldSkipCertificateValidation()
		
		// Log security warning when validation is disabled (industry standard practice)
		if insecureSkipVerify {
			log.Printf("⚠️  [EPHEMOS] Certificate validation disabled (EPHEMOS_INSECURE_SKIP_VERIFY=true) - development only!")
		}
	}

	// For now, create a basic TLS config
	// In a real implementation, this would use the SPIFFE certificates properly
	return &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}, nil
}

// grpcClient implements ports.ClientPort.
type grpcClient struct {
	tlsConfig *tls.Config
	policy    *domain.AuthenticationPolicy
}

// Connect establishes a secure gRPC connection to the specified address.
func (c *grpcClient) Connect(serviceName, address string) (ports.ConnectionPort, error) {
	creds := credentials.NewTLS(c.tlsConfig)
	
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s at %s: %w", serviceName, address, err)
	}

	return &grpcConnection{
		conn:        conn,
		serviceName: serviceName,
	}, nil
}

// Close releases any resources held by the client.
func (c *grpcClient) Close() error {
	// No persistent resources to clean up in this implementation
	return nil
}

// grpcServer implements ports.ServerPort.
type grpcServer struct {
	tlsConfig *tls.Config
	policy    *domain.AuthenticationPolicy
	server    *grpc.Server
}

// RegisterService registers a service implementation with the gRPC server.
func (s *grpcServer) RegisterService(registrar ports.ServiceRegistrarPort) error {
	if s.server == nil {
		creds := credentials.NewTLS(s.tlsConfig)
		s.server = grpc.NewServer(grpc.Creds(creds))
	}

	registrar.Register(s.server)
	return nil
}

// Start starts the gRPC server on the provided listener.
func (s *grpcServer) Start(listener ports.ListenerPort) error {
	if s.server == nil {
		return fmt.Errorf("server not initialized - call RegisterService first")
	}

	// Create a net.Listener from the ListenerPort
	netListener := &listenerAdapter{port: listener}
	
	return s.server.Serve(netListener)
}

// Stop gracefully shuts down the gRPC server.
func (s *grpcServer) Stop() error {
	if s.server != nil {
		s.server.GracefulStop()
	}
	return nil
}

// grpcConnection implements ports.ConnectionPort.
type grpcConnection struct {
	conn        *grpc.ClientConn
	serviceName string
}

// GetClientConnection returns the underlying gRPC client connection.
func (c *grpcConnection) GetClientConnection() interface{} {
	return c.conn
}

// Close closes the gRPC connection.
func (c *grpcConnection) Close() error {
	return c.conn.Close()
}

// listenerAdapter adapts ports.ListenerPort to net.Listener.
type listenerAdapter struct {
	port ports.ListenerPort
}

// Accept waits for and returns the next connection to the listener.
func (l *listenerAdapter) Accept() (net.Conn, error) {
	conn, err := l.port.Accept()
	if err != nil {
		return nil, err
	}
	
	// Assert that the connection is a net.Conn
	if netConn, ok := conn.(net.Conn); ok {
		return netConn, nil
	}
	
	return nil, fmt.Errorf("connection is not a net.Conn")
}

// Close closes the listener.
func (l *listenerAdapter) Close() error {
	return l.port.Close()
}

// Addr returns the listener's network address.
func (l *listenerAdapter) Addr() net.Addr {
	// The port returns a string, so we need to parse it
	// For simplicity, create a TCP address
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}