// Package transport provides concrete transport provider implementations.
package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
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
			// Return insecure config for development
			return &tls.Config{
				InsecureSkipVerify: true,
			}, nil
		}
	}

	// Create root CA pool from trust bundle
	rootCAs := x509.NewCertPool()
	for _, caCert := range bundle.Certificates {
		rootCAs.AddCert(caCert)
	}

	// Configure client certificate for mutual TLS
	clientCerts := []tls.Certificate{}
	if cert.Cert != nil && cert.PrivateKey != nil {
		// Build certificate chain
		certChain := []*x509.Certificate{cert.Cert}
		certChain = append(certChain, cert.Chain...)
		
		// Convert to PEM format for tls.Certificate
		tlsCert, err := buildTLSCertificate(certChain, cert.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to build client certificate: %w", err)
		}
		clientCerts = append(clientCerts, *tlsCert)
	}

	// Create TLS config with SPIFFE certificate validation
	return &tls.Config{
		RootCAs:                  rootCAs,
		Certificates:             clientCerts,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:       false,
		VerifyPeerCertificate:   p.createSPIFFEVerifier(),
		VerifyConnection:        p.createConnectionVerifier(),
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
			// Return insecure config for development
			return &tls.Config{
				InsecureSkipVerify: true,
			}, nil
		}
	}

	// Create client CA pool from trust bundle for client certificate validation
	clientCAs := x509.NewCertPool()
	for _, caCert := range bundle.Certificates {
		clientCAs.AddCert(caCert)
	}

	// Configure server certificate for TLS
	serverCerts := []tls.Certificate{}
	if cert.Cert != nil && cert.PrivateKey != nil {
		// Build certificate chain
		certChain := []*x509.Certificate{cert.Cert}
		certChain = append(certChain, cert.Chain...)
		
		// Convert to PEM format for tls.Certificate
		tlsCert, err := buildTLSCertificate(certChain, cert.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to build server certificate: %w", err)
		}
		serverCerts = append(serverCerts, *tlsCert)
	}

	// Create TLS config with SPIFFE certificate validation
	return &tls.Config{
		ClientCAs:                clientCAs,
		Certificates:             serverCerts,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:       false,
		VerifyPeerCertificate:   p.createSPIFFEVerifier(),
		VerifyConnection:        p.createConnectionVerifier(),
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

// buildTLSCertificate converts X.509 certificates and private key to tls.Certificate
func buildTLSCertificate(certs []*x509.Certificate, privateKey interface{}) (*tls.Certificate, error) {
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates provided")
	}
	if privateKey == nil {
		return nil, fmt.Errorf("private key is required")
	}

	// Convert certificates to raw DER format
	var certDER [][]byte
	for _, cert := range certs {
		certDER = append(certDER, cert.Raw)
	}

	return &tls.Certificate{
		Certificate: certDER,
		PrivateKey:  privateKey,
		Leaf:        certs[0], // First cert is the leaf certificate
	}, nil
}

// createSPIFFEVerifier creates a certificate verification function for SPIFFE IDs
func (p *GRPCProvider) createSPIFFEVerifier() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("no certificates provided")
		}

		// Parse the leaf certificate
		leafCert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("failed to parse leaf certificate: %w", err)
		}

		// Extract and validate SPIFFE IDs from Subject Alternative Names
		spiffeIDs, err := extractSPIFFEIDs(leafCert)
		if err != nil {
			return fmt.Errorf("failed to extract SPIFFE IDs: %w", err)
		}

		if len(spiffeIDs) == 0 {
			return fmt.Errorf("no SPIFFE IDs found in certificate")
		}

		// Validate SPIFFE ID format
		for _, spiffeID := range spiffeIDs {
			if _, err := spiffeid.FromString(spiffeID); err != nil {
				return fmt.Errorf("invalid SPIFFE ID format %q: %w", spiffeID, err)
			}
		}

		return nil
	}
}

// createConnectionVerifier creates a connection verification function
func (p *GRPCProvider) createConnectionVerifier() func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return fmt.Errorf("no peer certificates provided")
		}

		// Additional connection-level verification can be added here
		// For example, validating specific SPIFFE IDs based on service policies
		
		return nil
	}
}

// extractSPIFFEIDs extracts SPIFFE IDs from certificate Subject Alternative Names
func extractSPIFFEIDs(cert *x509.Certificate) ([]string, error) {
	var spiffeIDs []string

	// Check URI SANs for SPIFFE IDs
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			spiffeIDs = append(spiffeIDs, uri.String())
		}
	}

	return spiffeIDs, nil
}