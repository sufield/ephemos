// Package domain contains trust bundle status validation with domain intent rather than primitive checks.
package domain

import "crypto/x509"

// TrustBundleStatus represents the validation status of a trust bundle with domain predicates.
// This value object expresses domain intent instead of asking for raw slice/count data.
type TrustBundleStatus struct {
	certificates []*x509.Certificate
}

// NewTrustBundleStatus creates a trust bundle status from certificates.
func NewTrustBundleStatus(certificates []*x509.Certificate) TrustBundleStatus {
	return TrustBundleStatus{certificates: certificates}
}

// IsEmpty returns true if the trust bundle has no certificates.
// This expresses domain intent: "is trust bundle empty?" instead of "is length == 0?".
func (tbs TrustBundleStatus) IsEmpty() bool {
	return len(tbs.certificates) == 0
}

// HasCertificates returns true if the trust bundle contains certificates.
// This expresses domain intent: "does it have certs?" instead of "is length > 0?".
func (tbs TrustBundleStatus) HasCertificates() bool {
	return len(tbs.certificates) > 0
}

// HasMultipleCertificates returns true if the trust bundle has more than one certificate.
// This expresses domain intent: "has multiple certs?" instead of "length > 1?".
func (tbs TrustBundleStatus) HasMultipleCertificates() bool {
	return len(tbs.certificates) > 1
}

// CertificateCount returns the number of certificates.
// This expresses domain intent: "how many certificates?" instead of accessing slice length.
func (tbs TrustBundleStatus) CertificateCount() int {
	return len(tbs.certificates)
}

// IsValidForTrust returns true if the trust bundle is suitable for trust operations.
// This expresses domain intent: "can we trust this bundle?" instead of length checks.
func (tbs TrustBundleStatus) IsValidForTrust() bool {
	return tbs.HasCertificates() // At least one certificate required for trust
}

// IsSufficientForValidation returns true if the bundle can validate certificate chains.
// This expresses domain intent: "can validate chains?" instead of checking count.
func (tbs TrustBundleStatus) IsSufficientForValidation() bool {
	return tbs.HasCertificates() // Need at least one CA cert for validation
}

// Certificates returns the underlying certificates (for interop with existing code).
func (tbs TrustBundleStatus) Certificates() []*x509.Certificate {
	return tbs.certificates
}
