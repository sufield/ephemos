// Package domain handles service identity and authentication policies.
package domain

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// Certificate holds SPIFFE X.509 SVID certificate data with proper type safety.
type Certificate struct {
	Cert       *x509.Certificate   // Leaf certificate
	PrivateKey crypto.Signer       // Private key (must implement crypto.Signer for SPIFFE)
	Chain      []*x509.Certificate // Intermediate certificates (leaf-to-root order)
}

// CertValidationOptions configures certificate validation behavior.
// Supports RSA, ECDSA, and Ed25519 key types per go-spiffe v2.5.0 compatibility.
type CertValidationOptions struct {
	ExpectedIdentity *ServiceIdentity // Optional: Expected SPIFFE identity for matching
	WarningThreshold time.Duration    // Optional: Warning threshold for near-expiry (e.g., 1h)
	TrustBundle      *TrustBundle     // Optional: Trust bundle for chain verification
	SkipExpiry       bool             // Optional: Skip expiry checks (testing only)
	SkipChainVerify  bool             // Optional: Skip chain cryptographic verification
	Logger           *slog.Logger     // Optional: Logger for warnings and info (uses default if nil)
}

// NewCertificate creates a new Certificate with validation.
func NewCertificate(cert *x509.Certificate, key crypto.Signer, chain []*x509.Certificate) (*Certificate, error) {
	return NewCertificateWithValidation(cert, key, chain, true)
}

// NewCertificateWithValidation creates a new Certificate with optional validation.
// Set skipValidation to true only in trusted contexts where performance is critical
// and you're certain the certificate data is valid (e.g., internal caching).
func NewCertificateWithValidation(cert *x509.Certificate, key crypto.Signer, chain []*x509.Certificate, validate bool) (*Certificate, error) {
	c := &Certificate{
		Cert:       cert,
		PrivateKey: key,
		Chain:      chain,
	}

	if validate {
		if err := c.Validate(CertValidationOptions{}); err != nil {
			return nil, fmt.Errorf("certificate validation failed: %w", err)
		}
	}

	return c, nil
}

// Validate performs comprehensive certificate validation with configurable options.
// This is the primary validation method that should be used in production code.
func (c *Certificate) Validate(opts CertValidationOptions) error {
	// Basic structure validation
	if c == nil || c.Cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}
	if c.PrivateKey == nil || reflect.ValueOf(c.PrivateKey).IsNil() {
		return fmt.Errorf("private key cannot be nil")
	}

	// Verify the private key matches the certificate's public key
	if err := c.verifyKeyMatch(); err != nil {
		return fmt.Errorf("private key validation failed: %w", err)
	}

	// Expiry checks (can be skipped for testing)
	if !opts.SkipExpiry {
		now := time.Now()
		if now.Before(c.Cert.NotBefore) {
			return fmt.Errorf("certificate is not yet valid (NotBefore: %v)", c.Cert.NotBefore)
		}
		if now.After(c.Cert.NotAfter) {
			return fmt.Errorf("certificate has expired (NotAfter: %v)", c.Cert.NotAfter)
		}

		// Near-expiry warning with configurable threshold
		warningThreshold := opts.WarningThreshold
		if warningThreshold == 0 {
			warningThreshold = 30 * time.Minute // Default warning threshold
		}
		if now.Add(warningThreshold).After(c.Cert.NotAfter) {
			logger := opts.Logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.Warn("Certificate expires soon",
				"cert_subject", c.Cert.Subject.String(),
				"expires_at", c.Cert.NotAfter,
				"expires_in", time.Until(c.Cert.NotAfter).String(),
				"serial_number", c.Cert.SerialNumber.String(),
			)
		}
	}

	// Chain validation
	if len(c.Chain) > 0 {
		if err := c.validateChainOrder(); err != nil {
			return fmt.Errorf("certificate chain validation failed: %w", err)
		}
	}

	// Trust bundle verification (if provided)
	if opts.TrustBundle != nil && !opts.SkipChainVerify {
		if err := c.verifyWithTrustBundle(opts.TrustBundle); err != nil {
			return fmt.Errorf("trust bundle verification failed: %w", err)
		}
	}

	// SPIFFE identity matching (if expected identity provided)
	if opts.ExpectedIdentity != nil {
		actualID, err := c.ToSPIFFEID()
		if err != nil {
			return fmt.Errorf("failed to extract SPIFFE ID: %w", err)
		}

		expectedID, err := opts.ExpectedIdentity.ToSPIFFEID()
		if err != nil {
			return fmt.Errorf("failed to get expected SPIFFE ID: %w", err)
		}

		// Compare SPIFFE IDs using String() for compatibility
		// Note: go-spiffe v2.5.0 may add Equal method in future releases
		if actualID.String() != expectedID.String() {
			return fmt.Errorf("SPIFFE ID mismatch: expected %s, got %s", expectedID, actualID)
		}
	}

	return nil
}

// validateChainOrder checks that the certificate chain is properly ordered
// and cryptographically valid with full signature verification.
func (c *Certificate) validateChainOrder() error {
	if len(c.Chain) == 0 {
		return nil // No chain to validate
	}

	// Start with the leaf certificate and verify each link in the chain
	current := c.Cert
	for i, next := range c.Chain {
		// Check issuer-subject name matching first (fast check)
		if current.Issuer.String() != next.Subject.String() {
			return fmt.Errorf("chain order invalid at position %d: current issuer %q != next subject %q",
				i, current.Issuer.String(), next.Subject.String())
		}

		// Perform cryptographic signature verification
		// Create a certificate pool with just the issuer certificate
		issuerPool := x509.NewCertPool()
		issuerPool.AddCert(next)

		// Verify the current certificate was signed by the next certificate
		verifyOpts := x509.VerifyOptions{
			Roots:         issuerPool,
			Intermediates: x509.NewCertPool(),   // Empty intermediate pool for single-step verification
			KeyUsages:     []x509.ExtKeyUsage{}, // Don't enforce key usage for chain validation
		}

		// Verify the signature (this checks the cryptographic validity)
		_, err := current.Verify(verifyOpts)
		if err != nil {
			return fmt.Errorf("signature verification failed at chain position %d: certificate %q was not properly signed by %q: %w",
				i, current.Subject.String(), next.Subject.String(), err)
		}

		// Check that the signing certificate is authorized to sign other certificates
		if !next.IsCA {
			slog.Warn("Certificate in chain is not marked as CA but is signing other certificates",
				"position", i,
				"subject", next.Subject.String(),
				"serial_number", next.SerialNumber.String(),
			)
		}

		// Move to the next link in the chain
		current = next
	}

	return nil
}

// IsExpired returns true if the certificate has expired.
func (c *Certificate) IsExpired() bool {
	if c.Cert == nil {
		return true // Treat nil certificate as expired
	}
	return time.Now().After(c.Cert.NotAfter)
}

// ExpiresAt returns the certificate's expiration time.
func (c *Certificate) ExpiresAt() time.Time {
	if c.Cert == nil {
		return time.Time{} // Return zero time for nil certificate
	}
	return c.Cert.NotAfter
}

// TimeToExpiry returns the duration until the certificate expires.
func (c *Certificate) TimeToExpiry() time.Duration {
	if c.Cert == nil {
		return 0 // Return zero duration for nil certificate
	}
	return time.Until(c.Cert.NotAfter)
}

// IsExpiringWithin returns true if the certificate expires within the given threshold.
func (c *Certificate) IsExpiringWithin(threshold time.Duration) bool {
	if c.Cert == nil {
		return true // Treat nil certificate as expired
	}
	return time.Until(c.Cert.NotAfter) <= threshold
}

// IsExpiringSoon returns true if the certificate expires within the given duration.
func (c *Certificate) IsExpiringSoon(threshold time.Duration) bool {
	if c.Cert == nil {
		return true // Treat nil certificate as expired
	}
	return time.Now().Add(threshold).After(c.Cert.NotAfter)
}

// ToSPIFFEID extracts the SPIFFE ID from the certificate's URI SAN.
func (c *Certificate) ToSPIFFEID() (spiffeid.ID, error) {
	if c.Cert == nil {
		return spiffeid.ID{}, fmt.Errorf("certificate is nil")
	}

	for _, uri := range c.Cert.URIs {
		if uri.Scheme == "spiffe" {
			return spiffeid.FromURI(uri)
		}
	}

	return spiffeid.ID{}, fmt.Errorf("no SPIFFE ID found in certificate URI SANs")
}

// ToServiceIdentity extracts a ServiceIdentity from the certificate's SPIFFE ID.
func (c *Certificate) ToServiceIdentity() (*ServiceIdentity, error) {
	spiffeID, err := c.ToSPIFFEID()
	if err != nil {
		return nil, fmt.Errorf("failed to extract SPIFFE ID: %w", err)
	}

	// Parse trust domain and service name from SPIFFE ID
	trustDomain := spiffeID.TrustDomain().String()

	// Extract service name directly from SPIFFE ID path (already validated)
	serviceName := extractServiceNameFromPath(spiffeID.Path())

	identity, err := NewServiceIdentityValidated(serviceName, trustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to create service identity: %w", err)
	}
	return identity, nil
}

// verifyKeyMatch verifies that the private key matches the certificate's public key.
// This method expresses domain intent rather than mechanical key comparison operations.
// It assumes that c.PrivateKey and c.Cert have already been validated to be non-nil.
func (c *Certificate) verifyKeyMatch() error {
	// Extract public key from private key with domain validation
	privateKeyPublic, err := ExtractPublicKeyFromSigner(c.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to extract public key from private key: %w", err)
	}

	// Validate that the key pair matches using domain intent
	if err := ValidateKeyPairMatching(c.Cert.PublicKey, privateKeyPublic); err != nil {
		return fmt.Errorf("certificate key validation failed: %w", err)
	}

	return nil
}

// verifyWithTrustBundle verifies the certificate chain against a trust bundle.
func (c *Certificate) verifyWithTrustBundle(trustBundle *TrustBundle) error {
	if trustBundle == nil {
		return fmt.Errorf("trust bundle is nil")
	}

	// Create cert pool from trust bundle
	roots := trustBundle.CreateCertPool()
	if roots == nil {
		return fmt.Errorf("failed to create cert pool from trust bundle")
	}

	// Setup verification options
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// Add intermediate certificates to the pool if present
	if len(c.Chain) > 0 {
		for _, intermediate := range c.Chain {
			opts.Intermediates.AddCert(intermediate)
		}
	}

	// Perform cryptographic verification
	_, err := c.Cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate chain cryptographic verification failed: %w", err)
	}

	return nil
}

