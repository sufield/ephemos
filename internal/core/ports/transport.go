package ports

import (
	"context"
	"errors"
	"github.com/sufield/ephemos/internal/core/domain"
	"google.golang.org/grpc"
)

// ErrTransportCreationFailed is returned when transport creation fails
var ErrTransportCreationFailed = errors.New("failed to create transport")

// ServerTransport represents a server-side transport
type ServerTransport interface {
	Start() error
	Stop() error
	GetListener() interface{}
}

// ClientTransport represents a client-side transport  
type ClientTransport interface {
	Connect() error
	Close() error
	GetConnection() interface{}
}

type TransportProvider interface {
	CreateServerTransport(ctx context.Context, cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (*grpc.Server, error)
	CreateClientTransport(ctx context.Context, cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (grpc.DialOption, error)
}