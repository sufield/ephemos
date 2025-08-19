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
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
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


// Validate performs comprehensive certificate validation using go-spiffe SDK.
// This replaces custom validation with battle-tested SDK implementations.
func (c *Certificate) Validate(opts CertValidationOptions) error {
	// Basic structure validation
	if c == nil || c.Cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}
	if c.PrivateKey == nil || reflect.ValueOf(c.PrivateKey).IsNil() {
		return fmt.Errorf("private key cannot be nil")
	}

	// Use go-spiffe SDK validation when trust bundle is provided
	if opts.TrustBundle != nil && !opts.SkipChainVerify {
		// Convert TrustBundle to x509bundle.Source
		bundleSource, err := c.createBundleSource(opts.TrustBundle)
		if err != nil {
			return fmt.Errorf("failed to create bundle source: %w", err)
		}

		// Build certificate chain for SDK validation (as DER bytes)
		var certChainDER [][]byte
		certChainDER = append(certChainDER, c.Cert.Raw)
		for _, cert := range c.Chain {
			certChainDER = append(certChainDER, cert.Raw)
		}
		
		// Use go-spiffe SDK for comprehensive validation
		// This handles: chain order, signature verification, trust verification, expiry checks
		spiffeID, _, err := x509svid.ParseAndVerify(certChainDER, bundleSource)
		if err != nil {
			return fmt.Errorf("SDK certificate validation failed: %w", err)
		}
		
		// Verify private key matches (SDK doesn't validate private key matching)
		if err := c.verifyPrivateKeyWithID(spiffeID); err != nil {
			return fmt.Errorf("private key validation failed: %w", err)
		}

		// Identity matching (if specified)
		if opts.ExpectedIdentity != nil {
			expectedID, err := opts.ExpectedIdentity.ToSPIFFEID()
			if err != nil {
				return fmt.Errorf("failed to get expected SPIFFE ID: %w", err)
			}
			if spiffeID.String() != expectedID.String() {
				return fmt.Errorf("certificate identity mismatch: got %q, expected %q",
					spiffeID.String(), expectedID.String())
			}
		}

		return nil
	}

	// Fallback: basic validation without trust bundle
	return c.validateBasicWithoutTrust(opts)
}

// createBundleSource converts our TrustBundle to x509bundle.Source for SDK use
func (c *Certificate) createBundleSource(trustBundle *TrustBundle) (x509bundle.Source, error) {
	if trustBundle == nil {
		return nil, fmt.Errorf("trust bundle is nil")
	}
	
	// Extract the trust domain from the certificate
	spiffeID, err := c.ToSPIFFEID()
	if err != nil {
		return nil, fmt.Errorf("failed to extract SPIFFE ID from certificate: %w", err)
	}
	
	// Create bundle from our trust bundle's certificates
	bundle := x509bundle.FromX509Authorities(spiffeID.TrustDomain(), trustBundle.RawCertificates())
	
	// Return a static bundle source
	return x509bundle.NewSet(bundle), nil
}

// verifyPrivateKeyWithID verifies private key matches certificate using SDK-validated ID
func (c *Certificate) verifyPrivateKeyWithID(spiffeID spiffeid.ID) error {
	// Extract public key from private key
	privateKeyPublic, err := ExtractPublicKeyFromSigner(c.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to extract public key from private key: %w", err)
	}

	// Compare with the public key from the certificate (already validated by SDK)
	if err := ValidateKeyPairMatching(c.Cert.PublicKey, privateKeyPublic); err != nil {
		return fmt.Errorf("certificate key validation failed: %w", err)
	}

	return nil
}

// validateBasicWithoutTrust performs basic validation when no trust bundle is available
func (c *Certificate) validateBasicWithoutTrust(opts CertValidationOptions) error {
	// Basic private key matching
	privateKeyPublic, err := ExtractPublicKeyFromSigner(c.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to extract public key from private key: %w", err)
	}

	if err := ValidateKeyPairMatching(c.Cert.PublicKey, privateKeyPublic); err != nil {
		return fmt.Errorf("certificate key validation failed: %w", err)
	}

	// Basic expiry checks (if not skipped)
	if !opts.SkipExpiry {
		now := time.Now()
		if now.Before(c.Cert.NotBefore) {
			return fmt.Errorf("certificate is not yet valid (NotBefore: %v)", c.Cert.NotBefore)
		}
		if now.After(c.Cert.NotAfter) {
			return fmt.Errorf("certificate has expired (NotAfter: %v)", c.Cert.NotAfter)
		}
	}

	// Basic identity matching (if specified)
	if opts.ExpectedIdentity != nil {
		identity, err := c.ToServiceIdentity()
		if err != nil {
			return fmt.Errorf("failed to extract identity from certificate: %w", err)
		}
		if !identity.Equal(opts.ExpectedIdentity) {
			return fmt.Errorf("certificate identity mismatch: got %q, expected %q",
				identity.String(), opts.ExpectedIdentity.String())
		}
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



