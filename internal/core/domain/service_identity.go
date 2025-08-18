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
	name        string
	trustDomain TrustDomain
	uri         string
}

// Name returns the service name.
func (s *ServiceIdentity) Name() string {
	return s.name
}

// Domain returns the trust domain as a string.
func (s *ServiceIdentity) Domain() string {
	return s.trustDomain.String()
}

// TrustDomain returns the trust domain value object.
func (s *ServiceIdentity) TrustDomain() TrustDomain {
	return s.trustDomain
}

// URI returns the SPIFFE URI.
func (s *ServiceIdentity) URI() string {
	return s.uri
}

// NewServiceIdentity creates a new ServiceIdentity with backward compatibility.
// This maintains the old behavior of returning *ServiceIdentity (for compatibility).
// For new code requiring error handling, use NewServiceIdentityValidated.
func NewServiceIdentity(name, domainStr string) *ServiceIdentity {
	// Create and validate trust domain using our value object
	trustDomain, err := NewTrustDomain(domainStr)
	if err != nil {
		// Fallback to simple construction with invalid trust domain
		// This maintains backward compatibility for existing code
		return &ServiceIdentity{
			name:        name,
			trustDomain: TrustDomain(domainStr), // Raw assignment for backward compatibility
			uri:         fmt.Sprintf("spiffe://%s/%s", domainStr, name),
		}
	}

	// Try to validate with go-spiffe for additional compatibility
	goTrustDomain, err := spiffeid.TrustDomainFromString(trustDomain.String())
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:        name,
			trustDomain: trustDomain,
			uri:         fmt.Sprintf("spiffe://%s/%s", trustDomain.String(), name),
		}
	}

	// Use official SPIFFE ID construction
	spiffeID, err := spiffeid.FromPath(goTrustDomain, "/"+name)
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:        name,
			trustDomain: trustDomain,
			uri:         fmt.Sprintf("spiffe://%s/%s", trustDomain.String(), name),
		}
	}

	return &ServiceIdentity{
		name:        name,
		trustDomain: trustDomain,
		uri:         spiffeID.String(),
	}
}

// NewServiceIdentityValidated creates a validated ServiceIdentity.
// This constructor performs full validation and returns an error if validation fails.
func NewServiceIdentityValidated(name, domainStr string) (*ServiceIdentity, error) {
	// Check basic requirements first before calling go-spiffe
	if name == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	if domainStr == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	// Create and validate our trust domain value object
	trustDomain, err := NewTrustDomain(domainStr)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain %q: %w", domainStr, err)
	}

	// Also validate with go-spiffe for compatibility
	goTrustDomain, err := spiffeid.TrustDomainFromString(trustDomain.String())
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain for go-spiffe %q: %w", domainStr, err)
	}

	// Use official SPIFFE ID construction
	spiffeID, err := spiffeid.FromPath(goTrustDomain, "/"+name)
	if err != nil {
		return nil, fmt.Errorf("invalid SPIFFE path %q: %w", name, err)
	}

	identity := &ServiceIdentity{
		name:        name,
		trustDomain: trustDomain,
		uri:         spiffeID.String(),
	}

	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("service identity validation failed: %w", err)
	}

	return identity, nil
}

// NewServiceIdentityUnchecked creates an unchecked ServiceIdentity (for trusted inputs or tests).
// This constructor bypasses validation for performance in trusted contexts.
// WARNING: Only use this for trusted inputs where validation has already been performed
// or in test scenarios where invalid data is intentionally being tested.
func NewServiceIdentityUnchecked(name, domainStr string) *ServiceIdentity {
	// Simple construction without validation - create trust domain without validation
	trustDomain := TrustDomain(domainStr) // Raw assignment for performance
	return &ServiceIdentity{
		name:        name,
		trustDomain: trustDomain,
		uri:         fmt.Sprintf("spiffe://%s/%s", domainStr, name),
	}
}

// NewServiceIdentityWithTrustDomain creates a new ServiceIdentity with our TrustDomain value object.
// This is the preferred constructor for new code using our domain value objects.
func NewServiceIdentityWithTrustDomain(name string, trustDomain TrustDomain) *ServiceIdentity {
	// Validate trust domain
	if err := trustDomain.Validate(); err != nil {
		// Return with error - this shouldn't happen with proper usage
		return &ServiceIdentity{
			name:        name,
			trustDomain: trustDomain,
			uri:         fmt.Sprintf("spiffe://%s/%s", trustDomain.String(), name),
		}
	}

	// Try to create with go-spiffe for compatibility
	goTrustDomain, err := spiffeid.TrustDomainFromString(trustDomain.String())
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:        name,
			trustDomain: trustDomain,
			uri:         fmt.Sprintf("spiffe://%s/%s", trustDomain.String(), name),
		}
	}

	// Use official SPIFFE ID construction
	spiffeID, err := spiffeid.FromPath(goTrustDomain, "/"+name)
	if err != nil {
		// Fallback to simple construction
		return &ServiceIdentity{
			name:        name,
			trustDomain: trustDomain,
			uri:         fmt.Sprintf("spiffe://%s/%s", trustDomain.String(), name),
		}
	}

	return &ServiceIdentity{
		name:        name,
		trustDomain: trustDomain,
		uri:         spiffeID.String(),
	}
}

// NewServiceIdentityFromSPIFFEID creates a ServiceIdentity from a SPIFFE ID.
// Supports multi-segment paths (e.g., "/api/v1/service" becomes "api/v1/service").
func NewServiceIdentityFromSPIFFEID(id spiffeid.ID) *ServiceIdentity {
	// Create trust domain from go-spiffe trust domain
	trustDomainStr := id.TrustDomain().String()
	trustDomain, err := NewTrustDomain(trustDomainStr)
	if err != nil {
		// Fallback to raw assignment for backward compatibility
		trustDomain = TrustDomain(trustDomainStr)
	}

	// Support multi-segment paths by preserving the full path structure
	serviceName := strings.TrimPrefix(id.Path(), "/")

	return &ServiceIdentity{
		name:        serviceName,
		trustDomain: trustDomain,
		uri:         id.String(),
	}
}

// Validate checks the identity for SPIFFE compliance.
func (s *ServiceIdentity) Validate() error {
	if s.name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if s.trustDomain.IsZero() {
		return fmt.Errorf("trust domain cannot be empty")
	}

	// Validate trust domain using our value object validation
	if err := s.trustDomain.Validate(); err != nil {
		return fmt.Errorf("invalid trust domain %q: %w", s.trustDomain.String(), err)
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