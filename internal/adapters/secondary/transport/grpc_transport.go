package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCTransportProvider struct {
	spiffeProvider *spiffe.SPIFFEProvider
}

func NewGRPCTransportProvider(spiffeProvider *spiffe.SPIFFEProvider) *GRPCTransportProvider {
	return &GRPCTransportProvider{
		spiffeProvider: spiffeProvider,
	}
}

func (p *GRPCTransportProvider) CreateServerTransport(
	ctx context.Context,
	cert *domain.Certificate,
	bundle *domain.TrustBundle,
	policy *domain.AuthenticationPolicy,
) (*grpc.Server, error) {
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
	
	return grpc.NewServer(opts...), nil
}

func (p *GRPCTransportProvider) CreateClientTransport(
	ctx context.Context,
	cert *domain.Certificate,
	bundle *domain.TrustBundle,
	policy *domain.AuthenticationPolicy,
) (grpc.DialOption, error) {
	source := p.spiffeProvider.GetX509Source()
	if source == nil {
		return nil, fmt.Errorf("X509 source not initialized")
	}
	
	tlsConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsConfig.ServerName = ""
	
	creds := credentials.NewTLS(tlsConfig)
	
	return grpc.WithTransportCredentials(creds), nil
}

func (p *GRPCTransportProvider) createAuthInterceptor(policy *domain.AuthenticationPolicy) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}

func createTLSConfig(cert *domain.Certificate, bundle *domain.TrustBundle) *tls.Config {
	certPool := x509.NewCertPool()
	for _, ca := range bundle.Certificates {
		certPool.AddCert(ca)
	}
	
	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Cert.Raw},
		PrivateKey:  cert.PrivateKey,
	}
	
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		RootCAs:      certPool,
	}
}