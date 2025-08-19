// Package domain contains core business logic and domain models.
//
// This package implements the domain layer of the hexagonal architecture pattern
// and contains the following key components:
//
// - ServiceIdentity: SPIFFE service identity representation
// - Certificate: X.509 SVID certificate management
// - TrustBundle: Trust anchor certificate bundles
// - AuthenticationPolicy: Authentication and authorization policies
// - TrustDomain: Trust domain value object
//
// The domain layer is independent of external frameworks and infrastructure
// concerns, ensuring clean separation of business logic from technical implementation details.
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

// NewServiceIdentity creates a new ServiceIdentity with strict validation.
// No fallback patterns - panics if parameters are invalid to ensure proper configuration.
// This enforces forward-only development with no backward compatibility compromises.
func NewServiceIdentity(name, domainStr string) *ServiceIdentity {
	identity, err := NewServiceIdentityValidated(name, domainStr)
	if err != nil {
		// Panic on invalid input to prevent silent failures and enforce proper configuration
		panic(fmt.Sprintf("NewServiceIdentity failed with invalid parameters (name=%q, domain=%q): %v", name, domainStr, err))
	}
	return identity
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
// No fallback patterns - panics if the trust domain or name is invalid.
func NewServiceIdentityWithTrustDomain(name string, trustDomain TrustDomain) *ServiceIdentity {
	// Validate trust domain - fail fast
	if err := trustDomain.Validate(); err != nil {
		panic(fmt.Sprintf("NewServiceIdentityWithTrustDomain failed: invalid trust domain %q: %v", trustDomain.String(), err))
	}

	// Validate with go-spiffe - fail fast
	goTrustDomain, err := spiffeid.TrustDomainFromString(trustDomain.String())
	if err != nil {
		panic(fmt.Sprintf("NewServiceIdentityWithTrustDomain failed: trust domain %q not compatible with go-spiffe: %v", trustDomain.String(), err))
	}

	// Use official SPIFFE ID construction - fail fast
	spiffeID, err := spiffeid.FromPath(goTrustDomain, "/"+name)
	if err != nil {
		panic(fmt.Sprintf("NewServiceIdentityWithTrustDomain failed: invalid service name %q for SPIFFE path: %v", name, err))
	}

	return &ServiceIdentity{
		name:        name,
		trustDomain: trustDomain,
		uri:         spiffeID.String(),
	}
}

// NewServiceIdentityFromSPIFFEID creates a ServiceIdentity from a SPIFFE ID.
// No fallback patterns - panics if the SPIFFE ID contains invalid trust domain.
func NewServiceIdentityFromSPIFFEID(id spiffeid.ID) *ServiceIdentity {
	// Create trust domain from go-spiffe trust domain - fail fast
	trustDomainStr := id.TrustDomain().String()
	trustDomain, err := NewTrustDomain(trustDomainStr)
	if err != nil {
		panic(fmt.Sprintf("NewServiceIdentityFromSPIFFEID failed: invalid trust domain in SPIFFE ID %q: %v", id.String(), err))
	}

	// Support multi-segment paths by preserving the full path structure
	serviceName := strings.TrimPrefix(id.Path(), "/")
	if serviceName == "" {
		panic(fmt.Sprintf("NewServiceIdentityFromSPIFFEID failed: empty service name in SPIFFE ID %q", id.String()))
	}

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

// GetTrustDomain returns the trust domain value object (facade method to hide SPIFFE internals).
func (s *ServiceIdentity) GetTrustDomain() TrustDomain {
	return s.trustDomain
}

// GetTrustDomainString returns the trust domain as a string (facade method to hide SPIFFE internals).
func (s *ServiceIdentity) GetTrustDomainString() string {
	return s.trustDomain.String()
}

// IsMemberOf checks if this identity belongs to the specified trust domain (facade method).
func (s *ServiceIdentity) IsMemberOf(trustDomain string) bool {
	return s.trustDomain.String() == trustDomain
}