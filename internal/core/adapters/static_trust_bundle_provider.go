// Package adapters contains concrete implementations of ports interfaces.
// These adapters implement the port contracts defined in the ports package,
// completing the hexagonal architecture by providing concrete behavior
// for external integrations while keeping the domain layer pure.
package adapters

import (
	"crypto/x509"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/sufield/ephemos/internal/core/ports"
)

// StaticTrustBundleProvider provides a static trust bundle for testing and simple cases.
// It implements the ports.TrustBundleProvider interface by holding a fixed trust bundle
// that doesn't change over time, making it suitable for environments where trust bundles
// are static or for testing scenarios.
//
// This adapter follows the hexagonal architecture pattern by implementing a port
// interface while encapsulating the specific behavior of static trust bundle management.
type StaticTrustBundleProvider struct {
	bundle *x509bundle.Bundle
}

// NewStaticTrustBundleProvider creates a provider with a fixed trust bundle.
// This constructor ensures the adapter implements the correct port interface
// and provides a clear factory method for creating static trust bundle providers.
//
// Parameters:
//
//	bundle: The trust bundle to use for all requests (can be nil, but GetTrustBundle will error)
//
// Returns a TrustBundleProvider that always returns the same trust bundle.
func NewStaticTrustBundleProvider(bundle *x509bundle.Bundle) ports.TrustBundleProvider {
	return &StaticTrustBundleProvider{bundle: bundle}
}

// GetTrustBundle returns the static trust bundle.
// This implementation always returns the same trust bundle that was provided
// during construction, making it suitable for environments where trust bundles
// don't rotate or change dynamically.
//
// Returns:
//   - The configured trust bundle
//   - An error if no trust bundle was configured (nil bundle)
func (p *StaticTrustBundleProvider) GetTrustBundle() (*x509bundle.Bundle, error) {
	if p.bundle == nil {
		return nil, fmt.Errorf("no trust bundle configured")
	}
	return p.bundle, nil
}

// CreateCertPool creates a cert pool from the static trust bundle.
// This convenience method provides direct access to an x509.CertPool
// for use with standard Go TLS operations without exposing the internal
// TrustBundle structure to external components.
//
// Returns:
//   - An x509.CertPool populated with certificates from the trust bundle
//   - An error if the trust bundle cannot be retrieved
func (p *StaticTrustBundleProvider) CreateCertPool() (*x509.CertPool, error) {
	bundle, err := p.GetTrustBundle()
	if err != nil {
		return nil, err
	}
	
	pool := x509.NewCertPool()
	for _, cert := range bundle.X509Authorities() {
		pool.AddCert(cert)
	}
	return pool, nil
}
