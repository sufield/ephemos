// Package domain handles service identity and authentication policies.
package domain

import (
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// ServiceIdentity represents a SPIFFE service identity with name, domain, and URI.
// Use the constructors (NewServiceIdentity, NewServiceIdentityFromSPIFFEID) to ensure proper validation.
type ServiceIdentity struct {
	name   string
	domain string
	uri    string
}

// Name returns the service name.
func (s *ServiceIdentity) Name() string {
	return s.name
}

// Domain returns the trust domain.
func (s *ServiceIdentity) Domain() string {
	return s.domain
}

// URI returns the SPIFFE URI.
func (s *ServiceIdentity) URI() string {
	return s.uri
}

// NewServiceIdentity creates a new ServiceIdentity with the given name and domain.
func NewServiceIdentity(name, domain string) *ServiceIdentity {
	// Parse trust domain to ensure it's valid
	trustDomain, err := spiffeid.TrustDomainFromString(domain)
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}

	// Use official SPIFFE ID construction
	spiffeID, err := spiffeid.FromPath(trustDomain, "/"+name)
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}

	return &ServiceIdentity{
		name:   name,
		domain: domain,
		uri:    spiffeID.String(),
	}
}

// NewServiceIdentityWithValidation creates a new ServiceIdentity with optional validation.
// Set validate to false only in trusted contexts where performance is critical
// and you're certain the identity data is valid (e.g., internal caching).
func NewServiceIdentityWithValidation(name, domain string, validate bool) (*ServiceIdentity, error) {
	// Parse trust domain to ensure it's valid when validating
	var identity *ServiceIdentity
	if validate {
		// Check basic requirements first before calling go-spiffe
		if name == "" {
			return nil, fmt.Errorf("service name cannot be empty")
		}
		if domain == "" {
			return nil, fmt.Errorf("domain cannot be empty")
		}

		trustDomain, err := spiffeid.TrustDomainFromString(domain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain %q: %w", domain, err)
		}

		// Use official SPIFFE ID construction
		spiffeID, err := spiffeid.FromPath(trustDomain, "/"+name)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE path %q: %w", name, err)
		}

		identity = &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    spiffeID.String(),
		}

		if err := identity.Validate(); err != nil {
			return nil, fmt.Errorf("service identity validation failed: %w", err)
		}
	} else {
		// Simple construction without validation
		identity = &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}

	return identity, nil
}

// NewServiceIdentityFromSPIFFEID creates a ServiceIdentity from a SPIFFE ID.
// Supports multi-segment paths (e.g., "/api/v1/service" becomes "api/v1/service").
func NewServiceIdentityFromSPIFFEID(id spiffeid.ID) *ServiceIdentity {
	trustDomain := id.TrustDomain().String()
	// Support multi-segment paths by preserving the full path structure
	serviceName := strings.TrimPrefix(id.Path(), "/")

	return &ServiceIdentity{
		name:   serviceName,
		domain: trustDomain,
		uri:    id.String(),
	}
}

// Validate checks the identity for SPIFFE compliance.
func (s *ServiceIdentity) Validate() error {
	if s.name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if s.domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Validate trust domain format (must be valid DNS-like string)
	if err := s.validateTrustDomain(s.domain); err != nil {
		return fmt.Errorf("invalid trust domain %q: %w", s.domain, err)
	}

	// Validate service name format (SPIFFE path component)
	if err := s.validateServiceName(s.name); err != nil {
		return fmt.Errorf("invalid service name %q: %w", s.name, err)
	}

	// Validate the constructed SPIFFE URI using go-spiffe validation
	spiffeID, err := spiffeid.FromString(s.uri)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE URI %q: %w", s.uri, err)
	}

	// Additional SPIFFE spec constraints
	if err := s.validateSPIFFEConstraints(spiffeID); err != nil {
		return fmt.Errorf("SPIFFE spec violation in URI %q: %w", s.uri, err)
	}

	return nil
}

// validateTrustDomain checks if a trust domain follows SPIFFE specifications.
func (s *ServiceIdentity) validateTrustDomain(domain string) error {
	// Trust domain must be lowercase and DNS-compliant
	if domain != strings.ToLower(domain) {
		return fmt.Errorf("trust domain must be lowercase")
	}

	// Check for invalid characters in trust domain
	invalidChars := " !@#$%^&*()+=[]{}|\\:;\"'<>?/`~"
	if strings.ContainsAny(domain, invalidChars) {
		return fmt.Errorf("trust domain contains invalid characters")
	}

	// Must not start or end with hyphen or period
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") ||
		strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("trust domain cannot start or end with hyphen or period")
	}

	return nil
}

// validateServiceName checks if a service name is valid for SPIFFE path component.
// Now supports multi-segment paths (e.g., "api/v1/service") using official SPIFFE validation.
func (s *ServiceIdentity) validateServiceName(name string) error {
	// Check for empty name (handled earlier but be explicit)
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for leading/trailing slashes
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("service name cannot start or end with slash")
	}

	// Check for double slashes
	if strings.Contains(name, "//") {
		return fmt.Errorf("service name cannot contain double slashes")
	}

	// Check for dot segments
	segments := strings.Split(name, "/")
	for _, segment := range segments {
		if segment == "." || segment == ".." {
			return fmt.Errorf("service name cannot contain '.' or '..' path segments")
		}
	}

	// Check for invalid characters (space is a common one)
	if strings.Contains(name, " ") {
		return fmt.Errorf("service name contains invalid characters")
	}

	// Use official SPIFFE path validation for additional checks
	path := "/" + name // SPIFFE paths must start with "/"
	if err := spiffeid.ValidatePath(path); err != nil {
		return fmt.Errorf("service name validation failed: %w", err)
	}

	return nil
}

// validateSPIFFEConstraints validates additional SPIFFE specification constraints
// beyond what the go-spiffe library checks.
func (s *ServiceIdentity) validateSPIFFEConstraints(spiffeID spiffeid.ID) error {
	path := spiffeID.Path()

	// Use official SPIFFE path validation if there's a path
	if path != "" {
		if err := spiffeid.ValidatePath(path); err != nil {
			return fmt.Errorf("SPIFFE path validation failed: %w", err)
		}
	}

	// Additional practical constraints
	// SPIFFE spec: path length should be reasonable (practical limit)
	if len(path) > 2048 {
		return fmt.Errorf("SPIFFE path exceeds maximum length of 2048 characters")
	}

	// Trust domain validation (additional checks beyond go-spiffe)
	trustDomain := spiffeID.TrustDomain().String()

	// SPIFFE spec: trust domain length limit (practical limit)
	if len(trustDomain) > 255 {
		return fmt.Errorf("trust domain exceeds maximum length of 255 characters")
	}

	// SPIFFE spec: trust domain must not contain uppercase (should be normalized)
	if trustDomain != strings.ToLower(trustDomain) {
		return fmt.Errorf("trust domain must be lowercase")
	}

	return nil
}

// ToSPIFFEID converts the ServiceIdentity to a SPIFFE ID.
func (s *ServiceIdentity) ToSPIFFEID() (spiffeid.ID, error) {
	return spiffeid.FromString(s.uri)
}

// Equal checks if two ServiceIdentity instances are equivalent.
func (s *ServiceIdentity) Equal(other *ServiceIdentity) bool {
	if s == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return s.uri == other.uri
}

// String returns the SPIFFE URI string representation.
func (s *ServiceIdentity) String() string {
	return s.uri
}