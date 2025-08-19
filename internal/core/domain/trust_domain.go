// Package domain contains core business logic and domain models.
package domain

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// TrustDomain represents a trust boundary in the system, typically a domain name like "example.org".
// This is now a thin wrapper around go-spiffe's TrustDomain to leverage SDK validation.
type TrustDomain struct {
	td spiffeid.TrustDomain
}

// NewTrustDomain creates a validated TrustDomain from a string.
// Uses go-spiffe SDK for SPIFFE-compliant validation.
func NewTrustDomain(name string) (TrustDomain, error) {
	if name == "" {
		return TrustDomain{}, fmt.Errorf("trust domain cannot be empty")
	}

	// Use go-spiffe's built-in validation
	td, err := spiffeid.TrustDomainFromString(name)
	if err != nil {
		return TrustDomain{}, fmt.Errorf("invalid trust domain: %w", err)
	}

	return TrustDomain{td: td}, nil
}

// MustNewTrustDomain creates a TrustDomain or panics if validation fails.
// Use only in tests or when you're certain the input is valid.
func MustNewTrustDomain(name string) TrustDomain {
	td, err := NewTrustDomain(name)
	if err != nil {
		panic(fmt.Sprintf("invalid trust domain %q: %v", name, err))
	}
	return td
}

// String returns the string representation of the trust domain.
func (td TrustDomain) String() string {
	if td.IsZero() {
		return ""
	}
	return td.td.String()
}

// IsZero checks if the TrustDomain is empty/unset.
func (td TrustDomain) IsZero() bool {
	return td.td.IsZero()
}

// Equals checks equality with another TrustDomain.
// Uses go-spiffe's built-in comparison.
func (td TrustDomain) Equals(other TrustDomain) bool {
	// go-spiffe TrustDomain comparison is already case-insensitive
	return td.td.String() == other.td.String()
}

// Validate checks if the trust domain is valid.
// Returns nil if valid, error otherwise.
func (td TrustDomain) Validate() error {
	if td.IsZero() {
		return fmt.Errorf("trust domain is empty")
	}
	// Trust domain is already validated if it was created properly
	// Re-validate by trying to parse the string representation
	_, err := spiffeid.TrustDomainFromString(td.String())
	return err
}

// Compare returns an integer comparing two trust domains lexicographically.
// The result will be 0 if td==other, -1 if td < other, and +1 if td > other.
func (td TrustDomain) Compare(other TrustDomain) int {
	return strings.Compare(td.String(), other.String())
}

// MarshalJSON implements json.Marshaler interface.
func (td TrustDomain) MarshalJSON() ([]byte, error) {
	return json.Marshal(td.String())
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (td *TrustDomain) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Allow empty for optional fields
	if s == "" {
		*td = TrustDomain{}
		return nil
	}

	validated, err := NewTrustDomain(s)
	if err != nil {
		return fmt.Errorf("invalid trust domain in JSON: %w", err)
	}
	*td = validated
	return nil
}

// ToSPIFFEURI converts the trust domain to a SPIFFE URI format.
// This is useful for creating SPIFFE IDs.
func (td TrustDomain) ToSPIFFEURI() string {
	if td.IsZero() {
		return ""
	}
	return fmt.Sprintf("spiffe://%s", td)
}

// ParseFromSPIFFEID extracts the trust domain from a SPIFFE ID string.
// For example: "spiffe://example.org/service" returns "example.org".
// Uses go-spiffe SDK for parsing.
func ParseFromSPIFFEID(spiffeID string) (TrustDomain, error) {
	// Use go-spiffe to parse the SPIFFE ID
	id, err := spiffeid.FromString(spiffeID)
	if err != nil {
		return TrustDomain{}, fmt.Errorf("invalid SPIFFE ID: %w", err)
	}

	return TrustDomain{td: id.TrustDomain()}, nil
}

// ToSpiffeTrustDomain returns the underlying go-spiffe TrustDomain.
// This is for adapter layer use when interfacing with go-spiffe directly.
func (td TrustDomain) ToSpiffeTrustDomain() spiffeid.TrustDomain {
	return td.td
}

// FromSpiffeTrustDomain creates a TrustDomain from go-spiffe's TrustDomain.
// This is for adapter layer use when receiving values from go-spiffe.
func FromSpiffeTrustDomain(std spiffeid.TrustDomain) TrustDomain {
	return TrustDomain{td: std}
}