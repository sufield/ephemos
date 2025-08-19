package ports

import (
	"errors"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
)

// ErrIdentityNotFound is returned when an identity cannot be found.
var ErrIdentityNotFound = errors.New("identity not found")

// ServiceIdentityProvider defines service ID.
type ServiceIdentityProvider interface {
	GetDomain() string
	GetName() string
	Validate() error
	Close() error
}

// IdentityProvider provides identities without context dependencies in core.
// Context management is handled at the adapter layer.
type IdentityProvider interface {
	GetServiceIdentity() (*domain.ServiceIdentity, error)
	GetCertificate() (*domain.Certificate, error)
	GetTrustBundle() (*domain.TrustBundle, error)
	GetSVID() (*x509svid.SVID, error)
	Close() error
}
