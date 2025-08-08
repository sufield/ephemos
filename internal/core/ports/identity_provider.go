package ports

import (
	"context"
	"errors"
	"github.com/sufield/ephemos/internal/core/domain"
)

// ErrIdentityNotFound is returned when an identity cannot be found
var ErrIdentityNotFound = errors.New("identity not found")

type ServiceIdentity interface {
	GetDomain() string
	GetName() string
	Validate() error
	Close() error
}

type IdentityProvider interface {
	GetServiceIdentity(ctx context.Context) (*domain.ServiceIdentity, error)
	GetCertificate(ctx context.Context) (*domain.Certificate, error)
	GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error)
	Close() error
}