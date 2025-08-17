// Package ephemos provides identity-based authentication for backend services.
package ephemos

import (
	"crypto/x509"
)

// Certificate represents an X.509 certificate with its chain and private key.
type Certificate struct {
	// Cert is the leaf certificate.
	Cert *x509.Certificate
	// Chain contains the intermediate certificates.
	Chain []*x509.Certificate
	// PrivateKey is the private key for the certificate.
	PrivateKey interface{}
}

// TrustBundle represents a collection of trusted root certificates.
type TrustBundle struct {
	// Certificates contains the trusted root certificates.
	Certificates []*x509.Certificate
}

// IdentityService provides access to service identity certificates and trust bundles.
// This interface is used by contrib middleware to access core identity primitives.
type IdentityService interface {
	// GetCertificate returns the current service certificate with its chain and private key.
	// This certificate is used for mTLS authentication.
	GetCertificate() (*Certificate, error)
	
	// GetTrustBundle returns the current trust bundle containing root certificates.
	// This bundle is used to verify peer certificates during mTLS.
	GetTrustBundle() (*TrustBundle, error)
}

