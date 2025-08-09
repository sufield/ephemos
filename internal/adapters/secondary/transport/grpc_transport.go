// Package transport provides gRPC transport implementations for secure communication.
package transport

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/domain"
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

// CreateServerTransport creates server transport.
func (p *GRPCProvider) CreateServerTransport(
	_ context.Context,
	_ *domain.Certificate,
	_ *domain.TrustBundle,
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

// CreateClientTransport creates client transport.
func (p *GRPCProvider) CreateClientTransport(
	_ context.Context,
	_ *domain.Certificate,
	_ *domain.TrustBundle,
	_ *domain.AuthenticationPolicy,
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

func (p *GRPCProvider) createAuthInterceptor(_ *domain.AuthenticationPolicy) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}
