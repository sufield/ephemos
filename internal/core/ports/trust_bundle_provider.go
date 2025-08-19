// Package ports defines interfaces that represent the application's ports in hexagonal architecture.
// These interfaces define contracts for external behaviors and infrastructure dependencies,
// enabling clean separation between the core domain logic and external adapters.
package ports

import (
	"crypto/x509"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
)

// TrustBundleProvider defines an interface for dynamic trust bundle access.
// This port abstracts trust bundle provisioning behavior, supporting SVID rotation
// scenarios where trust bundles may change over time.
//
// This interface enables the domain layer to access trust bundles without being
// coupled to specific trust bundle sources (e.g., SPIRE server, file-based bundles,
// network-fetched bundles, etc.).
//
// Implementations should ensure thread-safety as this interface may be called
// concurrently from multiple goroutines during TLS handshakes and certificate validation.
type TrustBundleProvider interface {
	// GetTrustBundle returns the current trust bundle using go-spiffe SDK.
	// Implementations should return the most up-to-date bundle available.
	//
	// This method may be called frequently during TLS operations, so implementations
	// should consider caching strategies while ensuring they provide fresh data
	// when trust bundles are rotated.
	//
	// Returns:
	//   - An x509bundle.Bundle containing the current set of trust anchors
	//   - An error if the trust bundle cannot be retrieved
	GetTrustBundle() (*x509bundle.Bundle, error)

	// CreateCertPool creates a cert pool from the current trust bundle.
	// This is a convenience method that gets the bundle and creates a cert pool.
	//
	// This method provides a direct way to get an x509.CertPool for use with
	// standard Go TLS operations without exposing the internal bundle structure
	// to adapters that only need the cert pool.
	//
	// Returns:
	//   - An x509.CertPool populated with certificates from the current trust bundle
	//   - An error if the trust bundle cannot be retrieved or cert pool cannot be created
	CreateCertPool() (*x509.CertPool, error)
}
