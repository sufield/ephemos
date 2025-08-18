// Package domain contains core business logic and domain models.
package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// TrustDomain represents a trust boundary in the system, typically a domain name like "example.org".
// It is a value object with built-in validation that enforces SPIFFE trust domain constraints.
// This type is independent of external libraries to maintain clean architecture principles.
type TrustDomain string

// trustDomainRegex validates trust domain format according to SPIFFE specification.
// Trust domains must be valid DNS names without protocol, port, or path.
var trustDomainRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// NewTrustDomain creates a validated TrustDomain from a string.
// It ensures the domain follows SPIFFE trust domain requirements:
// - Must not be empty
// - Must be lowercase
// - Must be a valid DNS name format
// - Must not contain protocol, port, or path components
func NewTrustDomain(name string) (TrustDomain, error) {
	if name == "" {
		return "", fmt.Errorf("trust domain cannot be empty")
	}

	// Trim whitespace and convert to lowercase for normalization
	name = strings.TrimSpace(strings.ToLower(name))

	// Check for invalid characters or patterns
	if strings.Contains(name, "://") {
		return "", fmt.Errorf("trust domain must not contain protocol: %q", name)
	}
	if strings.Contains(name, "/") {
		return "", fmt.Errorf("trust domain must not contain path: %q", name)
	}
	if strings.Contains(name, ":") {
		return "", fmt.Errorf("trust domain must not contain port: %q", name)
	}
	if strings.Contains(name, " ") {
		return "", fmt.Errorf("trust domain must not contain spaces: %q", name)
	}

	// Validate DNS name format
	if !trustDomainRegex.MatchString(name) {
		return "", fmt.Errorf("invalid trust domain format (must be valid DNS name): %q", name)
	}

	// Additional SPIFFE constraints
	if len(name) > 255 {
		return "", fmt.Errorf("trust domain exceeds maximum length of 255 characters: %q", name)
	}

	return TrustDomain(name), nil
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
	return string(td)
}

// IsZero checks if the TrustDomain is empty/unset.
func (td TrustDomain) IsZero() bool {
	return td == ""
}

// Equals checks equality with another TrustDomain.
// Trust domain comparison is case-insensitive per SPIFFE spec.
func (td TrustDomain) Equals(other TrustDomain) bool {
	return strings.EqualFold(td.String(), other.String())
}

// Validate checks if the trust domain is valid.
// Returns nil if valid, error otherwise.
// This is useful when working with TrustDomain values that may have been
// created without validation (e.g., from JSON unmarshaling).
func (td TrustDomain) Validate() error {
	if td.IsZero() {
		return fmt.Errorf("trust domain is empty")
	}
	// Re-validate using the same rules as NewTrustDomain
	_, err := NewTrustDomain(td.String())
	return err
}

// Compare returns an integer comparing two trust domains lexicographically.
// The result will be 0 if td==other, -1 if td < other, and +1 if td > other.
func (td TrustDomain) Compare(other TrustDomain) int {
	return strings.Compare(td.String(), other.String())
}

// MarshalJSON implements json.Marshaler interface.
func (td TrustDomain) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", td.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (td *TrustDomain) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	
	// Allow empty for optional fields
	if s == "" {
		*td = TrustDomain("")
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
func ParseFromSPIFFEID(spiffeID string) (TrustDomain, error) {
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return "", fmt.Errorf("not a valid SPIFFE ID: %q", spiffeID)
	}

	// Remove spiffe:// prefix
	remainder := strings.TrimPrefix(spiffeID, "spiffe://")
	
	// Find the trust domain (everything before the first '/')
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("missing trust domain in SPIFFE ID: %q", spiffeID)
	}

	return NewTrustDomain(parts[0])
}