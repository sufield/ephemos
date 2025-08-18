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
	if len(tb.Certificates) == 0 {
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
func (tb *TrustBundle) IsEmpty() bool {
	return len(tb.Certificates) == 0
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

