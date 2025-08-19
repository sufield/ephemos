package domain

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"time"
)

// RootCACertificate is a wrapper for a root CA certificate in the trust bundle.
// This allows abstraction from x509 in core domain while maintaining compatibility.
type RootCACertificate struct {
	Cert *x509.Certificate
}

// NewRootCACertificate creates a validated RootCACertificate.
func NewRootCACertificate(cert *x509.Certificate) (*RootCACertificate, error) {
	if cert == nil {
		return nil, fmt.Errorf("root CA certificate cannot be nil")
	}
	if !cert.IsCA {
		return nil, fmt.Errorf("certificate is not a CA")
	}
	return &RootCACertificate{Cert: cert}, nil
}

// TrustBundle holds SPIFFE trust anchor certificates for a trust domain.
// This is a domain value object that abstracts trust bundle management.
type TrustBundle struct {
	Certificates []*RootCACertificate
}

// NewTrustBundle creates a new TrustBundle with validation.
func NewTrustBundle(certificates []*x509.Certificate) (*TrustBundle, error) {
	return NewTrustBundleWithValidation(certificates, true)
}

// NewTrustBundleWithValidation creates a new TrustBundle with optional validation.
// Set validate to false only in trusted contexts where performance is critical
// and you're certain the trust bundle data is valid (e.g., internal caching).
func NewTrustBundleWithValidation(rawCerts []*x509.Certificate, validate bool) (*TrustBundle, error) {
	certs := make([]*RootCACertificate, 0, len(rawCerts))
	for _, rawCert := range rawCerts {
		if validate {
			ca, err := NewRootCACertificate(rawCert)
			if err != nil {
				return nil, fmt.Errorf("invalid CA certificate: %w", err)
			}
			certs = append(certs, ca)
		} else {
			// Skip validation for performance-critical paths
			certs = append(certs, &RootCACertificate{Cert: rawCert})
		}
	}

	tb := &TrustBundle{
		Certificates: certs,
	}

	if validate {
		if err := tb.Validate(); err != nil {
			return nil, fmt.Errorf("trust bundle validation failed: %w", err)
		}
	}

	return tb, nil
}

// Validate checks that the trust bundle is valid and contains valid certificates.
func (tb *TrustBundle) Validate() error {
	// Use domain predicate instead of primitive length check
	status := NewTrustBundleStatus(tb.getX509Certificates())
	if status.IsEmpty() {
		return fmt.Errorf("trust bundle cannot be empty")
	}

	// Track certificate uniqueness by public key
	seen := make(map[string]struct{})

	for i, ca := range tb.Certificates {
		if ca == nil || ca.Cert == nil {
			return fmt.Errorf("certificate at index %d is nil", i)
		}

		cert := ca.Cert

		// Check for duplicate certificates
		key := string(cert.RawSubjectPublicKeyInfo)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate certificate found in trust bundle at index %d", i)
		}
		seen[key] = struct{}{}

		// Check if certificate is a valid CA (should have CA:TRUE basic constraint)
		if !cert.IsCA {
			return fmt.Errorf("certificate at index %d is not a CA certificate", i)
		}

		// Check certificate validity period
		now := time.Now()
		if now.Before(cert.NotBefore) {
			return fmt.Errorf("CA certificate at index %d is not yet valid (NotBefore: %v)", i, cert.NotBefore)
		}
		if now.After(cert.NotAfter) {
			return fmt.Errorf("CA certificate at index %d has expired (NotAfter: %v)", i, cert.NotAfter)
		}

		// Warn if CA certificate expires soon (within 24 hours)
		if now.Add(24 * time.Hour).After(cert.NotAfter) {
			slog.Warn("CA certificate expires soon",
				"ca_subject", cert.Subject.String(),
				"expires_at", cert.NotAfter,
				"expires_in", time.Until(cert.NotAfter).String(),
				"serial_number", cert.SerialNumber.String(),
				"is_ca", cert.IsCA,
			)
		}
	}

	return nil
}

// IsEmpty returns true if the trust bundle contains no certificates.
// This expresses domain intent instead of asking for primitive length data.
func (tb *TrustBundle) IsEmpty() bool {
	status := NewTrustBundleStatus(tb.getX509Certificates())
	return status.IsEmpty()
}

// ContainsCertificate checks if the trust bundle contains a specific certificate.
func (tb *TrustBundle) ContainsCertificate(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}

	for _, ca := range tb.Certificates {
		if ca.Cert != nil && ca.Cert.Equal(cert) {
			return true
		}
	}

	return false
}

// CreateCertPool creates a new x509.CertPool from the trust bundle certificates.
// This creates a fresh pool each time to support dynamic reloading scenarios
// where trust bundles change.
// Note: This is a boundary method for adapter use.
func (tb *TrustBundle) CreateCertPool() *x509.CertPool {
	pool := x509.NewCertPool()

	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			pool.AddCert(ca.Cert)
		}
	}

	return pool
}

// RawCertificates returns the underlying x509 certificates for adapter use.
// This maintains compatibility with external systems that need direct x509 access.
func (tb *TrustBundle) RawCertificates() []*x509.Certificate {
	raw := make([]*x509.Certificate, 0, len(tb.Certificates))
	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			raw = append(raw, ca.Cert)
		}
	}
	return raw
}

// Count returns the number of certificates in the trust bundle.
func (tb *TrustBundle) Count() int {
	return len(tb.Certificates)
}

// ValidateAgainstBundle validates this trust bundle against another trust bundle.
// This is useful for verifying trust relationships between bundles.
func (tb *TrustBundle) ValidateAgainstBundle(other *TrustBundle) error {
	if other == nil {
		return fmt.Errorf("comparison trust bundle cannot be nil")
	}

	if tb.IsEmpty() {
		return fmt.Errorf("cannot validate empty trust bundle")
	}

	if other.IsEmpty() {
		return fmt.Errorf("comparison trust bundle is empty")
	}

	// Check if any certificates in this bundle are present in the other bundle
	hasCommonCert := false
	for _, cert := range tb.Certificates {
		if cert != nil && cert.Cert != nil {
			if other.ContainsCertificate(cert.Cert) {
				hasCommonCert = true
				break
			}
		}
	}

	if !hasCommonCert {
		return fmt.Errorf("no common certificates found between trust bundles")
	}

	return nil
}

// ValidateCertificateChain validates a certificate chain against this trust bundle.
// This verifies that the chain can be trusted by the certificates in this bundle.
func (tb *TrustBundle) ValidateCertificateChain(chain []*x509.Certificate) error {
	if len(chain) == 0 {
		return fmt.Errorf("certificate chain cannot be empty")
	}

	if tb.IsEmpty() {
		return fmt.Errorf("cannot validate against empty trust bundle")
	}

	leafCert := chain[0]

	// Create cert pool from trust bundle
	roots := tb.CreateCertPool()

	// Setup verification options
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// Add intermediate certificates to the pool if present
	if len(chain) > 1 {
		for _, intermediate := range chain[1:] {
			opts.Intermediates.AddCert(intermediate)
		}
	}

	// Perform cryptographic verification
	_, err := leafCert.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate chain validation failed: %w", err)
	}

	return nil
}

// ValidateIdentityDocument validates an identity document against this trust bundle.
func (tb *TrustBundle) ValidateIdentityDocument(doc *IdentityDocument) error {
	if doc == nil {
		return fmt.Errorf("identity document cannot be nil")
	}

	// Use the identity document's built-in validation method
	return doc.ValidateAgainstBundle(tb)
}

// ContainsTrustDomain checks if this trust bundle contains certificates for a specific trust domain.
// This examines the certificate subjects to determine trust domain coverage.
func (tb *TrustBundle) ContainsTrustDomain(trustDomain TrustDomain) bool {
	if trustDomain.IsZero() {
		return false
	}

	trustDomainStr := trustDomain.String()

	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			// Check if the certificate subject contains the trust domain
			// This is a heuristic as CA certificates may not have SPIFFE URIs
			subject := ca.Cert.Subject.String()
			if contains(subject, trustDomainStr) {
				return true
			}

			// Also check URI SANs for SPIFFE URIs
			for _, uri := range ca.Cert.URIs {
				if uri.Scheme == "spiffe" && uri.Host == trustDomainStr {
					return true
				}
			}
		}
	}

	return false
}

// GetTrustDomains extracts all trust domains that this bundle can validate.
// This analyzes the certificates to determine which trust domains they cover.
func (tb *TrustBundle) GetTrustDomains() []TrustDomain {
	var trustDomains []TrustDomain
	seen := make(map[string]bool)

	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			// Check URI SANs for SPIFFE URIs
			for _, uri := range ca.Cert.URIs {
				if uri.Scheme == "spiffe" && uri.Host != "" {
					if !seen[uri.Host] {
						if trustDomain, err := NewTrustDomain(uri.Host); err == nil {
							trustDomains = append(trustDomains, trustDomain)
							seen[uri.Host] = true
						}
					}
				}
			}
		}
	}

	return trustDomains
}

// MergeBundles creates a new trust bundle by merging this bundle with another.
// Duplicate certificates are automatically removed.
func (tb *TrustBundle) MergeBundles(other *TrustBundle) (*TrustBundle, error) {
	if other == nil {
		return nil, fmt.Errorf("cannot merge with nil trust bundle")
	}

	// Collect all unique certificates
	seen := make(map[string]*x509.Certificate)
	var allCerts []*x509.Certificate

	// Add certificates from this bundle
	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			key := string(ca.Cert.RawSubjectPublicKeyInfo)
			if _, exists := seen[key]; !exists {
				seen[key] = ca.Cert
				allCerts = append(allCerts, ca.Cert)
			}
		}
	}

	// Add certificates from other bundle
	for _, ca := range other.Certificates {
		if ca != nil && ca.Cert != nil {
			key := string(ca.Cert.RawSubjectPublicKeyInfo)
			if _, exists := seen[key]; !exists {
				seen[key] = ca.Cert
				allCerts = append(allCerts, ca.Cert)
			}
		}
	}

	// Create new bundle with merged certificates
	return NewTrustBundle(allCerts)
}

// getX509Certificates extracts x509.Certificate pointers from the trust bundle.
// This helper method supports domain predicate validation.
func (tb *TrustBundle) getX509Certificates() []*x509.Certificate {
	var certs []*x509.Certificate
	for _, ca := range tb.Certificates {
		if ca != nil && ca.Cert != nil {
			certs = append(certs, ca.Cert)
		}
	}
	return certs
}

// contains is a helper function to check if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexContains(s, substr) >= 0))
}

// indexContains is a helper to find substring in string.
func indexContains(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
