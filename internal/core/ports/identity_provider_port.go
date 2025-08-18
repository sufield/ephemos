// Package ports defines interfaces that represent the application's ports in hexagonal architecture.
package ports

import (
	"context"
	
	"github.com/sufield/ephemos/internal/core/domain"
)

// IdentityProviderPort defines the contract for identity provisioning operations.
// This port abstracts the identity acquisition and management behavior, supporting
// various identity sources (SPIRE, file-based, cloud providers, etc.).
//
// This interface enables the application layer to obtain and manage identities
// without being coupled to specific identity infrastructure.
//
// Implementations must be thread-safe as they may be called concurrently.
type IdentityProviderPort interface {
	// GetServiceIdentity retrieves the current service identity.
	// The identity includes the service name and trust domain.
	//
	// Returns:
	//   - A ServiceIdentity containing the service's SPIFFE identity information
	//   - An error if the identity cannot be retrieved
	GetServiceIdentity(ctx context.Context) (*domain.ServiceIdentity, error)

	// GetCertificate retrieves the current service certificate.
	// The certificate includes the private key and certificate chain.
	//
	// Returns:
	//   - A Certificate containing the X.509 certificate and private key
	//   - An error if the certificate cannot be retrieved
	GetCertificate(ctx context.Context) (*domain.Certificate, error)

	// GetIdentityDocument retrieves the complete identity document.
	// The document encapsulates certificate-based identity with metadata.
	//
	// Returns:
	//   - An IdentityDocument containing the complete identity information
	//   - An error if the identity document cannot be retrieved
	GetIdentityDocument(ctx context.Context) (*domain.IdentityDocument, error)

	// RefreshIdentity triggers a refresh of the identity credentials.
	// This may involve fetching new certificates from the identity provider.
	//
	// Returns:
	//   - An error if the refresh operation fails
	RefreshIdentity(ctx context.Context) error

	// WatchIdentityChanges returns a channel that receives notifications
	// when the identity is updated (e.g., certificate rotation).
	//
	// Returns:
	//   - A channel that receives identity update events
	//   - An error if watching cannot be established
	WatchIdentityChanges(ctx context.Context) (<-chan *domain.IdentityDocument, error)

	// Close releases any resources held by the provider.
	Close() error
}

// BundleProviderPort defines the contract for trust bundle provisioning operations.
// This port abstracts trust bundle acquisition and management, supporting various
// trust bundle sources (SPIRE, file-based, network-fetched, etc.).
//
// This interface enables the application layer to obtain and validate trust bundles
// without being coupled to specific trust infrastructure.
//
// Implementations must be thread-safe as they may be called concurrently.
type BundleProviderPort interface {
	// GetTrustBundle retrieves the current trust bundle.
	// The bundle contains the trust anchors for certificate validation.
	//
	// Returns:
	//   - A TrustBundle containing the current set of trust anchors
	//   - An error if the trust bundle cannot be retrieved
	GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error)

	// GetTrustBundleForDomain retrieves a trust bundle for a specific trust domain.
	// This allows for multi-domain trust scenarios.
	//
	// Returns:
	//   - A TrustBundle for the specified domain
	//   - An error if the trust bundle cannot be retrieved
	GetTrustBundleForDomain(ctx context.Context, trustDomain domain.TrustDomain) (*domain.TrustBundle, error)

	// RefreshTrustBundle triggers a refresh of the trust bundle.
	// This may involve fetching updated trust anchors from the provider.
	//
	// Returns:
	//   - An error if the refresh operation fails
	RefreshTrustBundle(ctx context.Context) error

	// WatchTrustBundleChanges returns a channel that receives notifications
	// when the trust bundle is updated.
	//
	// Returns:
	//   - A channel that receives trust bundle update events
	//   - An error if watching cannot be established
	WatchTrustBundleChanges(ctx context.Context) (<-chan *domain.TrustBundle, error)

	// ValidateCertificateAgainstBundle validates a certificate against the trust bundle.
	// This performs cryptographic validation using the trust anchors.
	//
	// Returns:
	//   - An error if the certificate is not valid according to the trust bundle
	ValidateCertificateAgainstBundle(ctx context.Context, cert *domain.Certificate) error

	// Close releases any resources held by the provider.
	Close() error
}