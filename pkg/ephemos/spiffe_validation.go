// Package ephemos provides simple SPIFFE validation interfaces that delegate to the core domain layer.
package ephemos

import (
	"crypto/x509"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/core/domain"
)

// SPIFFEValidator is a public interface that delegates to the domain layer.
type SPIFFEValidator = domain.SPIFFEValidator

// NewSPIFFEValidator creates a new SPIFFE validator.
func NewSPIFFEValidator(bundleSource x509bundle.Source) *SPIFFEValidator {
	return domain.NewSPIFFEValidator(bundleSource)
}

// ValidateSPIFFEID validates a SPIFFE ID string.
func ValidateSPIFFEID(spiffeIDStr string) error {
	validator := domain.NewSPIFFEValidator(nil)
	return validator.ValidateSPIFFEID(spiffeIDStr)
}

// ValidateX509SVID validates an X.509 SVID certificate chain.
func ValidateX509SVID(bundleSource x509bundle.Source, certChain [][]byte) (*spiffeid.ID, error) {
	validator := domain.NewSPIFFEValidator(bundleSource)
	return validator.ValidateX509SVID(certChain)
}

// ValidateX509Certificates validates parsed X.509 certificates.
func ValidateX509Certificates(bundleSource x509bundle.Source, certs []*x509.Certificate) (*spiffeid.ID, error) {
	validator := domain.NewSPIFFEValidator(bundleSource)
	return validator.ValidateX509Certificates(certs)
}