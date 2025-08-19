// Package domain contains core business logic and domain models.
package domain

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"time"
)

// IdentityDocument represents a verifiable identity credential with format-agnostic naming.
// This entity encapsulates certificate-based identity documents (like X.509 SVIDs) while
// providing a clean domain interface that abstracts away specific credential formats.
type IdentityDocument struct {
	// Core credential components
	certificate *Certificate

	// Metadata
	issuedAt   time.Time
	validUntil time.Time

	// Optional fields for rich identity information
	subject string
	issuer  string

	// Internal state
	validated   bool
	lastChecked time.Time
}

// IdentityDocumentConfig provides configuration for creating a new identity document.
type IdentityDocumentConfig struct {
	CertChain  []*x509.Certificate
	PrivateKey crypto.Signer
	CACert     *x509.Certificate

	// Optional metadata
	Subject string
	Issuer  string
}

// NewIdentityDocument creates a new IdentityDocument with certificate chain validation.
// This is the primary constructor that validates the certificate chain using Go's crypto/x509.
func NewIdentityDocument(certChain []*x509.Certificate, privateKey crypto.Signer, ca *x509.Certificate) (*IdentityDocument, error) {
	if len(certChain) == 0 {
		return nil, fmt.Errorf("certificate chain cannot be empty")
	}

	if privateKey == nil {
		return nil, fmt.Errorf("private key cannot be nil")
	}

	leafCert := certChain[0]
	var intermediateCerts []*x509.Certificate
	if len(certChain) > 1 {
		intermediateCerts = certChain[1:]
	}

	// Create and validate the certificate using our domain Certificate entity
	certificate, err := NewCertificate(leafCert, privateKey, intermediateCerts)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Validate the certificate chain if CA is provided
	if ca != nil {
		// Create a trust bundle with the CA for validation
		trustBundle, err := NewTrustBundle([]*x509.Certificate{ca})
		if err != nil {
			return nil, fmt.Errorf("failed to create trust bundle for validation: %w", err)
		}

		// Validate against the trust bundle
		opts := CertValidationOptions{
			TrustBundle: trustBundle,
		}

		if err := certificate.Validate(opts); err != nil {
			return nil, fmt.Errorf("certificate chain validation failed: %w", err)
		}
	}

	// Extract metadata from the leaf certificate
	subject := leafCert.Subject.String()
	issuer := leafCert.Issuer.String()

	now := time.Now()

	return &IdentityDocument{
		certificate: certificate,
		issuedAt:    leafCert.NotBefore,
		validUntil:  leafCert.NotAfter,
		subject:     subject,
		issuer:      issuer,
		validated:   true,
		lastChecked: now,
	}, nil
}

// NewIdentityDocumentFromConfig creates a new IdentityDocument from configuration.
func NewIdentityDocumentFromConfig(config IdentityDocumentConfig) (*IdentityDocument, error) {
	return NewIdentityDocument(config.CertChain, config.PrivateKey, config.CACert)
}

// NewIdentityDocumentFromCertificate creates a new IdentityDocument from an existing Certificate entity.
func NewIdentityDocumentFromCertificate(cert *Certificate) (*IdentityDocument, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate cannot be nil")
	}

	// Validate the certificate first
	if err := cert.Validate(CertValidationOptions{}); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	subject := cert.Cert.Subject.String()
	issuer := cert.Cert.Issuer.String()

	now := time.Now()

	return &IdentityDocument{
		certificate: cert,
		issuedAt:    cert.Cert.NotBefore,
		validUntil:  cert.Cert.NotAfter,
		subject:     subject,
		issuer:      issuer,
		validated:   true,
		lastChecked: now,
	}, nil
}

// GetCertificate returns the underlying Certificate entity.
func (doc *IdentityDocument) GetCertificate() *Certificate {
	return doc.certificate
}

// GetPrivateKey returns the private key associated with this identity document.
func (doc *IdentityDocument) GetPrivateKey() crypto.Signer {
	if doc.certificate == nil {
		return nil
	}
	return doc.certificate.PrivateKey
}

// GetCertificateChain returns the full certificate chain.
func (doc *IdentityDocument) GetCertificateChain() []*x509.Certificate {
	if doc.certificate == nil {
		return nil
	}

	chain := []*x509.Certificate{doc.certificate.Cert}
	chain = append(chain, doc.certificate.Chain...)
	return chain
}

// GetLeafCertificate returns the leaf certificate.
func (doc *IdentityDocument) GetLeafCertificate() *x509.Certificate {
	if doc.certificate == nil {
		return nil
	}
	return doc.certificate.Cert
}

// GetIdentityNamespace extracts the identity namespace from the certificate's SPIFFE ID.
func (doc *IdentityDocument) GetIdentityNamespace() (IdentityNamespace, error) {
	if doc.certificate == nil {
		return IdentityNamespace{}, fmt.Errorf("certificate is nil")
	}

	spiffeID, err := doc.certificate.ToSPIFFEID()
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("failed to extract SPIFFE ID: %w", err)
	}

	return NewIdentityNamespaceFromString(spiffeID.String())
}

// GetTrustDomain extracts the trust domain from the certificate's SPIFFE ID.
func (doc *IdentityDocument) GetTrustDomain() (TrustDomain, error) {
	namespace, err := doc.GetIdentityNamespace()
	if err != nil {
		return TrustDomain(""), fmt.Errorf("failed to get identity namespace: %w", err)
	}

	return namespace.GetTrustDomain(), nil
}

// GetServiceIdentity extracts a ServiceIdentity from the certificate.
func (doc *IdentityDocument) GetServiceIdentity() (*ServiceIdentity, error) {
	if doc.certificate == nil {
		return nil, fmt.Errorf("certificate is nil")
	}

	return doc.certificate.ToServiceIdentity()
}

// IsExpired checks if the identity document has expired at the given time.
func (doc *IdentityDocument) IsExpired(now time.Time) bool {
	return now.After(doc.validUntil)
}

// IsExpiringSoon returns true if the identity document expires within the given threshold.
func (doc *IdentityDocument) IsExpiringSoon(threshold time.Duration) bool {
	if doc.certificate == nil {
		return true
	}
	return doc.certificate.IsExpiringSoon(threshold)
}

// IsValid checks if the identity document is currently valid (not expired and properly formed).
func (doc *IdentityDocument) IsValid(now time.Time) bool {
	if doc.certificate == nil {
		return false
	}

	// Check expiration
	if doc.IsExpired(now) {
		return false
	}

	// Check if it's not yet valid
	if now.Before(doc.issuedAt) {
		return false
	}

	return doc.validated
}

// ValidateAgainstBundle validates the identity document against a trust bundle.
func (doc *IdentityDocument) ValidateAgainstBundle(bundle *TrustBundle) error {
	if doc.certificate == nil {
		return fmt.Errorf("certificate is nil")
	}

	if bundle == nil {
		return fmt.Errorf("trust bundle cannot be nil")
	}

	// Validate using certificate validation with trust bundle
	opts := CertValidationOptions{
		TrustBundle: bundle,
	}

	if err := doc.certificate.Validate(opts); err != nil {
		return fmt.Errorf("trust bundle validation failed: %w", err)
	}

	doc.validated = true
	doc.lastChecked = time.Now()
	return nil
}

// Validate performs comprehensive validation of the identity document.
func (doc *IdentityDocument) Validate() error {
	if doc.certificate == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	// Validate the underlying certificate
	if err := doc.certificate.Validate(CertValidationOptions{}); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	// Check that the certificate has a SPIFFE ID
	_, err := doc.certificate.ToSPIFFEID()
	if err != nil {
		return fmt.Errorf("certificate does not contain a valid SPIFFE ID: %w", err)
	}

	// Validate time bounds
	now := time.Now()
	if now.Before(doc.issuedAt) {
		return fmt.Errorf("identity document is not yet valid (issued at: %v)", doc.issuedAt)
	}

	if now.After(doc.validUntil) {
		return fmt.Errorf("identity document has expired (valid until: %v)", doc.validUntil)
	}

	doc.validated = true
	doc.lastChecked = now
	return nil
}

// IssuedAt returns when the identity document was issued.
func (doc *IdentityDocument) IssuedAt() time.Time {
	return doc.issuedAt
}

// ValidUntil returns when the identity document expires.
func (doc *IdentityDocument) ValidUntil() time.Time {
	return doc.validUntil
}

// Subject returns the subject of the identity document.
func (doc *IdentityDocument) Subject() string {
	return doc.subject
}

// Issuer returns the issuer of the identity document.
func (doc *IdentityDocument) Issuer() string {
	return doc.issuer
}

// TimeUntilExpiry returns the duration until the identity document expires.
func (doc *IdentityDocument) TimeUntilExpiry() time.Duration {
	return time.Until(doc.validUntil)
}

// LastValidated returns when the identity document was last validated.
func (doc *IdentityDocument) LastValidated() time.Time {
	return doc.lastChecked
}

// RequiresPrivateKey returns true if this identity document type requires a private key.
// For certificate-based documents, this is always true.
func (doc *IdentityDocument) RequiresPrivateKey() bool {
	return true
}

// SupportsKeyType returns true if the given private key type is supported.
// This method expresses domain intent rather than mechanical type assertions.
func (doc *IdentityDocument) SupportsKeyType(key crypto.Signer) bool {
	// For MVP, we only support ECDSA keys as they are the standard for SPIFFE
	return SupportsKeyType(key)
}

// String returns a string representation of the identity document for debugging.
func (doc *IdentityDocument) String() string {
	if doc.certificate == nil {
		return "IdentityDocument{empty}"
	}

	spiffeID, err := doc.certificate.ToSPIFFEID()
	if err != nil {
		return fmt.Sprintf("IdentityDocument{invalid: %v}", err)
	}

	return fmt.Sprintf("IdentityDocument{ID:%s, ValidUntil:%s, Subject:%s}",
		spiffeID.String(), doc.validUntil.Format(time.RFC3339), doc.subject)
}
