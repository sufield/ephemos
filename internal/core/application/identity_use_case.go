package application

import (
	"context"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// IdentityUseCaseImpl implements the IdentityUseCase interface by delegating
// to the IdentityService. This provides a clean application layer interface
// while preserving existing functionality.
type IdentityUseCaseImpl struct {
	identityService *services.IdentityService
}

// NewIdentityUseCase creates a new identity use case implementation.
func NewIdentityUseCase(identityService *services.IdentityService) IdentityUseCase {
	return &IdentityUseCaseImpl{
		identityService: identityService,
	}
}

// CreateServerIdentity creates a server with identity-based authentication.
func (u *IdentityUseCaseImpl) CreateServerIdentity(ctx context.Context) (ports.ServerPort, error) {
	return u.identityService.CreateServerIdentity()
}

// CreateClientIdentity creates a client connection with identity-based authentication.
func (u *IdentityUseCaseImpl) CreateClientIdentity(ctx context.Context) (ports.ClientPort, error) {
	return u.identityService.CreateClientIdentity()
}

// ValidateServiceIdentity validates that a certificate is valid, not expired, and matches expected identity.
func (u *IdentityUseCaseImpl) ValidateServiceIdentity(ctx context.Context, cert *domain.Certificate) error {
	return u.identityService.ValidateServiceIdentity(cert)
}

// GetCertificate retrieves the service certificate for client connections.
func (u *IdentityUseCaseImpl) GetCertificate(ctx context.Context) (*domain.Certificate, error) {
	return u.identityService.GetCertificate()
}

// GetTrustBundle retrieves the trust bundle for certificate validation.
func (u *IdentityUseCaseImpl) GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error) {
	return u.identityService.GetTrustBundle()
}
