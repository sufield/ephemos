package ports

import (
	"context"
	"github.com/sufield/ephemos/internal/core/domain"
	"google.golang.org/grpc"
)

type TransportProvider interface {
	CreateServerTransport(ctx context.Context, cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (*grpc.Server, error)
	CreateClientTransport(ctx context.Context, cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (grpc.DialOption, error)
}