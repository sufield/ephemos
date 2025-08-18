// Package common provides shared utilities for adapters.
package common

import (
	"fmt"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/sufield/ephemos/internal/core/domain"
)

// ToCoreTrustDomain converts a spiffeid.TrustDomain to domain.TrustDomain.
// This conversion is safe as spiffeid.TrustDomain is already validated.
func ToCoreTrustDomain(td spiffeid.TrustDomain) domain.TrustDomain {
	// spiffeid.TrustDomain.String() returns the domain name
	// We can safely create without validation since it's already valid
	return domain.TrustDomain(td.String())
}

// ToSpiffeTrustDomain converts a domain.TrustDomain to spiffeid.TrustDomain.
// Returns an error if the conversion fails (should be rare as both have similar validation).
func ToSpiffeTrustDomain(td domain.TrustDomain) (spiffeid.TrustDomain, error) {
	if td.IsZero() {
		return spiffeid.TrustDomain{}, fmt.Errorf("cannot convert empty trust domain")
	}
	return spiffeid.TrustDomainFromString(td.String())
}

// MustToSpiffeTrustDomain converts a domain.TrustDomain to spiffeid.TrustDomain.
// Panics if conversion fails. Use only when you're certain the domain is valid.
func MustToSpiffeTrustDomain(td domain.TrustDomain) spiffeid.TrustDomain {
	std, err := ToSpiffeTrustDomain(td)
	if err != nil {
		panic(fmt.Sprintf("failed to convert trust domain %q: %v", td, err))
	}
	return std
}

// ToCoreTrustDomains converts a slice of spiffeid.TrustDomain to []domain.TrustDomain.
func ToCoreTrustDomains(tds []spiffeid.TrustDomain) []domain.TrustDomain {
	result := make([]domain.TrustDomain, len(tds))
	for i, td := range tds {
		result[i] = ToCoreTrustDomain(td)
	}
	return result
}

// ToSpiffeTrustDomains converts a slice of domain.TrustDomain to []spiffeid.TrustDomain.
// Returns an error if any conversion fails.
func ToSpiffeTrustDomains(tds []domain.TrustDomain) ([]spiffeid.TrustDomain, error) {
	result := make([]spiffeid.TrustDomain, len(tds))
	for i, td := range tds {
		std, err := ToSpiffeTrustDomain(td)
		if err != nil {
			return nil, fmt.Errorf("failed to convert trust domain at index %d: %w", i, err)
		}
		result[i] = std
	}
	return result, nil
}

// ExtractTrustDomainFromSPIFFEID extracts the trust domain from a spiffeid.ID.
func ExtractTrustDomainFromSPIFFEID(id spiffeid.ID) domain.TrustDomain {
	return ToCoreTrustDomain(id.TrustDomain())
}

// ExtractTrustDomainFromString parses a SPIFFE ID string and extracts the trust domain.
// For example: "spiffe://example.org/service" returns domain.TrustDomain("example.org").
func ExtractTrustDomainFromString(spiffeIDStr string) (domain.TrustDomain, error) {
	id, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid SPIFFE ID: %w", err)
	}
	return ToCoreTrustDomain(id.TrustDomain()), nil
}

// ToCoreTrustBundle converts an x509bundle.Bundle to domain.TrustBundle.
// This enables interaction with SPIFFE bundles while maintaining domain abstraction.
func ToCoreTrustBundle(external *x509bundle.Bundle) (*domain.TrustBundle, error) {
	if external == nil {
		return nil, fmt.Errorf("external bundle is nil")
	}
	return domain.NewTrustBundle(external.X509Authorities())
}

// ToX509Bundle converts a domain.TrustBundle to x509bundle.Bundle.
// Requires a trust domain to create the SPIFFE bundle.
func ToX509Bundle(tb *domain.TrustBundle, td domain.TrustDomain) (*x509bundle.Bundle, error) {
	if tb == nil {
		return nil, fmt.Errorf("trust bundle is nil")
	}
	spiffeTD, err := ToSpiffeTrustDomain(td)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain: %w", err)
	}
	return x509bundle.FromX509Authorities(spiffeTD, tb.RawCertificates()), nil
}

// MustToX509Bundle converts a domain.TrustBundle to x509bundle.Bundle.
// Panics if conversion fails. Use only when you're certain the inputs are valid.
func MustToX509Bundle(tb *domain.TrustBundle, td domain.TrustDomain) *x509bundle.Bundle {
	bundle, err := ToX509Bundle(tb, td)
	if err != nil {
		panic(fmt.Sprintf("failed to convert trust bundle for domain %q: %v", td, err))
	}
	return bundle
}