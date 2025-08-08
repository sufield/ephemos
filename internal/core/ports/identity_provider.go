package ports

import (
	"context"
	"github.com/sufield/ephemos/internal/core/domain"
)

type IdentityProvider interface {
	GetServiceIdentity(ctx context.Context) (*domain.ServiceIdentity, error)
	GetCertificate(ctx context.Context) (*domain.Certificate, error)
	GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error)
}