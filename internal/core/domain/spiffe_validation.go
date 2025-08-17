// Package domain provides SPIFFE SVID validation using the official go-spiffe/v2 SDK.
package domain

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

// SPIFFEValidator provides SPIFFE-compliant validation using the official SDK.
type SPIFFEValidator struct {
	bundleSource x509bundle.Source
}

// NewSPIFFEValidator creates a new validator with the given bundle source.
func NewSPIFFEValidator(bundleSource x509bundle.Source) *SPIFFEValidator {
	return &SPIFFEValidator{
		bundleSource: bundleSource,
	}
}

// ValidateSPIFFEID validates a SPIFFE ID string using the official SDK.
// This is an internal method - external users should use spiffeid.FromString() directly.
func (v *SPIFFEValidator) ValidateSPIFFEID(spiffeIDStr string) error {
	_, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE ID format: %w", err)
	}
	return nil
}

// ValidateX509SVID validates an X.509 SVID certificate chain against trust bundles.
func (v *SPIFFEValidator) ValidateX509SVID(certChain [][]byte) (*spiffeid.ID, error) {
	if v.bundleSource == nil {
		return nil, fmt.Errorf("bundle source not configured")
	}

	spiffeID, _, err := x509svid.ParseAndVerify(certChain, v.bundleSource)
	if err != nil {
		return nil, fmt.Errorf("SVID verification failed: %w", err)
	}

	return &spiffeID, nil
}

// ValidateX509Certificates validates parsed X.509 certificates against trust bundles.
func (v *SPIFFEValidator) ValidateX509Certificates(certs []*x509.Certificate) (*spiffeid.ID, error) {
	if v.bundleSource == nil {
		return nil, fmt.Errorf("bundle source not configured")
	}

	spiffeID, _, err := x509svid.Verify(certs, v.bundleSource, x509svid.WithTime(time.Now()))
	if err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	return &spiffeID, nil
}
