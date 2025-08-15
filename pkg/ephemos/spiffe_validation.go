// Package ephemos provides SPIFFE SVID validation using the official go-spiffe/v2 SDK.
package ephemos

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
)

// SPIFFEValidator provides SPIFFE-compliant validation using the official SDK.
type SPIFFEValidator struct {
	// bundleSource provides X.509 trust bundles for verification
	bundleSource x509bundle.Source
}

// NewSPIFFEValidator creates a new validator with the given bundle source.
func NewSPIFFEValidator(bundleSource x509bundle.Source) *SPIFFEValidator {
	return &SPIFFEValidator{
		bundleSource: bundleSource,
	}
}

// ValidateSPIFFEID validates a SPIFFE ID string using the official SDK.
// This replaces the basic string validation with proper SPIFFE specification compliance.
func (v *SPIFFEValidator) ValidateSPIFFEID(spiffeIDStr string) error {
	if spiffeIDStr == "" {
		return fmt.Errorf("SPIFFE ID cannot be empty")
	}

	// Use official go-spiffe SDK for proper parsing and validation
	spiffeID, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE ID format: %w", err)
	}

	// Additional validation can be added here, such as:
	// - Trust domain validation against allowed domains
	// - Path validation against allowed patterns
	// - Service name extraction and validation

	_ = spiffeID // Parsed successfully
	return nil
}

// ValidateX509SVID validates an X.509 SVID certificate chain against trust bundles.
// This provides cryptographic verification that goes beyond format validation.
func (v *SPIFFEValidator) ValidateX509SVID(certChain [][]byte) (*spiffeid.ID, error) {
	if v.bundleSource == nil {
		return nil, fmt.Errorf("bundle source not configured")
	}

	// Use official go-spiffe SDK for full SVID verification
	spiffeID, verifiedChains, err := x509svid.ParseAndVerify(
		certChain,
		v.bundleSource,
		// Optional: Add custom verification options
		// x509svid.WithTime(customTime), // For testing or custom time validation
	)
	if err != nil {
		return nil, fmt.Errorf("SVID verification failed: %w", err)
	}

	// Additional validation can be performed on verified chains
	_ = verifiedChains // Successfully verified certificate chains

	return &spiffeID, nil
}

// ValidateX509Certificates validates parsed X.509 certificates against trust bundles.
// This is useful when you already have parsed certificates from another source.
func (v *SPIFFEValidator) ValidateX509Certificates(certs []*x509.Certificate) (*spiffeid.ID, error) {
	if v.bundleSource == nil {
		return nil, fmt.Errorf("bundle source not configured")
	}

	// Use official go-spiffe SDK for verification of parsed certificates
	spiffeID, verifiedChains, err := x509svid.Verify(
		certs,
		v.bundleSource,
		// Optional: Add custom verification options
		x509svid.WithTime(time.Now()), // Explicit time validation
	)
	if err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	_ = verifiedChains // Successfully verified
	return &spiffeID, nil
}

// Examples of how to use this with the existing validation engine:

// ValidateSPIFFEIDWithSDK is an enhanced version of validateSPIFFEID that uses the official SDK.
// This can replace the current basic string validation in validation.go.
func ValidateSPIFFEIDWithSDK(spiffeIDStr string) error {
	if spiffeIDStr == "" {
		return nil // Match current behavior - empty allowed unless required
	}
	validator := &SPIFFEValidator{} // In production, pass actual bundle source
	return validator.ValidateSPIFFEID(spiffeIDStr)
}

// Example integration with existing workload API:
// 
// func (p *Provider) ValidateRemoteSVID(certChain [][]byte) error {
//     // Get bundle source from existing X509Source
//     bundleSource := p.x509Source // from internal/adapters/secondary/spiffe/provider.go
//     
//     validator := NewSPIFFEValidator(bundleSource)
//     spiffeID, err := validator.ValidateX509SVID(certChain)
//     if err != nil {
//         return fmt.Errorf("remote SVID validation failed: %w", err)
//     }
//     
//     // Additional authorization checks can be performed here
//     // For example, checking if spiffeID is in allowed list
//     log.Printf("Validated remote SPIFFE ID: %s", spiffeID.String())
//     return nil
// }