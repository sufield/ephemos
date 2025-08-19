// Package domain contains core business logic and domain models.
package domain

import (
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// IdentityNamespace represents a complete SPIFFE identity namespace, consisting of
// a trust domain and a path. It enforces SPIFFE specification constraints and
// provides type safety for identity operations.
//
// Example: "spiffe://example.org/service/payment-processor"
// - TrustDomain: "example.org"
// - Path: "/service/payment-processor"
type IdentityNamespace struct {
	trustDomain TrustDomain
	path        string
}

// Maximum lengths according to SPIFFE specification
const (
	MaxSPIFFEIDLength = 2048 // Maximum total SPIFFE ID length
	MaxPathLength     = 1792 // Maximum path length (total - scheme - trust domain)
)

// NewIdentityNamespace creates a validated IdentityNamespace from trust domain and path.
// The path should start with '/' and follow SPIFFE path constraints.
func NewIdentityNamespace(trustDomain TrustDomain, path string) (IdentityNamespace, error) {
	if trustDomain.IsZero() {
		return IdentityNamespace{}, fmt.Errorf("trust domain cannot be empty")
	}

	// Validate trust domain
	if err := trustDomain.Validate(); err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid trust domain: %w", err)
	}

	// Use go-spiffe's built-in path validation
	// Note: go-spiffe expects empty string for root path, not "/"
	pathForValidation := path
	if path == "/" {
		pathForValidation = ""
	}
	if err := spiffeid.ValidatePath(pathForValidation); err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid path: %w", err)
	}

	namespace := IdentityNamespace{
		trustDomain: trustDomain,
		path:        path,
	}

	// Validate total length
	if err := namespace.validateTotalLength(); err != nil {
		return IdentityNamespace{}, err
	}

	return namespace, nil
}

// NewIdentityNamespaceFromString creates an IdentityNamespace by parsing a SPIFFE ID string.
// The input must be a valid SPIFFE URI like "spiffe://example.org/service/name".
// This uses go-spiffe's built-in validation instead of custom parsing.
func NewIdentityNamespaceFromString(spiffeID string) (IdentityNamespace, error) {
	if spiffeID == "" {
		return IdentityNamespace{}, fmt.Errorf("SPIFFE ID cannot be empty")
	}

	// Handle trailing slash case - go-spiffe doesn't accept "spiffe://domain/"
	// but we want to support it as equivalent to "spiffe://domain" (root path)
	normalizedID := spiffeID
	if strings.HasSuffix(spiffeID, "/") && strings.Count(spiffeID, "/") == 3 {
		// Remove trailing slash for root path case: "spiffe://domain/" -> "spiffe://domain"
		normalizedID = strings.TrimSuffix(spiffeID, "/")
	}

	// Use go-spiffe's built-in parsing and validation
	parsedID, err := spiffeid.FromString(normalizedID)
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid SPIFFE ID: %w", err)
	}

	// Create trust domain from validated SPIFFE ID
	trustDomain, err := NewTrustDomain(parsedID.TrustDomain().String())
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid trust domain in SPIFFE ID: %w", err)
	}

	// Extract path (defaults to "/" if empty)
	path := parsedID.Path()
	if path == "" {
		path = "/"
	}

	return NewIdentityNamespace(trustDomain, path)
}

// MustNewIdentityNamespace creates an IdentityNamespace or panics if validation fails.
// Use only in tests or when you're certain the input is valid.
func MustNewIdentityNamespace(trustDomain TrustDomain, path string) IdentityNamespace {
	namespace, err := NewIdentityNamespace(trustDomain, path)
	if err != nil {
		panic(fmt.Sprintf("invalid identity namespace (trust domain: %q, path: %q): %v", trustDomain, path, err))
	}
	return namespace
}

// MustNewIdentityNamespaceFromString creates an IdentityNamespace from string or panics.
// Use only in tests or when you're certain the input is valid.
func MustNewIdentityNamespaceFromString(spiffeID string) IdentityNamespace {
	namespace, err := NewIdentityNamespaceFromString(spiffeID)
	if err != nil {
		panic(fmt.Sprintf("invalid SPIFFE ID %q: %v", spiffeID, err))
	}
	return namespace
}

// GetTrustDomain returns the trust domain component.
func (ns IdentityNamespace) GetTrustDomain() TrustDomain {
	return ns.trustDomain
}

// GetPath returns the path component.
func (ns IdentityNamespace) GetPath() string {
	return ns.path
}

// String returns the complete SPIFFE ID as a string.
func (ns IdentityNamespace) String() string {
	if ns.IsZero() {
		return ""
	}
	return fmt.Sprintf("spiffe://%s%s", ns.trustDomain, ns.path)
}

// IsZero checks if the IdentityNamespace is empty/unset.
func (ns IdentityNamespace) IsZero() bool {
	return ns.trustDomain.IsZero() && ns.path == ""
}

// Equals checks equality with another IdentityNamespace.
func (ns IdentityNamespace) Equals(other IdentityNamespace) bool {
	return ns.trustDomain.Equals(other.trustDomain) && ns.path == other.path
}

// Validate checks if the identity namespace is valid.
func (ns IdentityNamespace) Validate() error {
	if ns.IsZero() {
		return fmt.Errorf("identity namespace is empty")
	}

	if err := ns.trustDomain.Validate(); err != nil {
		return fmt.Errorf("invalid trust domain: %w", err)
	}

	// Use go-spiffe's built-in path validation
	// Note: go-spiffe expects empty string for root path, not "/"
	pathForValidation := ns.path
	if ns.path == "/" {
		pathForValidation = ""
	}
	if err := spiffeid.ValidatePath(pathForValidation); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	return ns.validateTotalLength()
}

// IsChildOf checks if this identity namespace is a child of the given parent namespace.
// A namespace is considered a child if it has the same trust domain and its path
// starts with the parent's path followed by '/'.
func (ns IdentityNamespace) IsChildOf(parent IdentityNamespace) bool {
	if !ns.trustDomain.Equals(parent.trustDomain) {
		return false
	}

	if parent.path == "/" {
		return true // Everything is a child of root
	}

	parentPath := parent.path
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}

	return strings.HasPrefix(ns.path, parentPath)
}

// GetServiceName extracts the service name from the path, assuming it's the last segment.
// For example, "/service/payment-processor" returns "payment-processor".
// Returns empty string if the path doesn't contain a service name.
func (ns IdentityNamespace) GetServiceName() string {
	if ns.path == "" || ns.path == "/" {
		return ""
	}

	// Remove leading slash and split by '/'
	cleanPath := strings.TrimPrefix(ns.path, "/")
	segments := strings.Split(cleanPath, "/")

	if len(segments) == 0 {
		return ""
	}

	return segments[len(segments)-1]
}

// WithPath creates a new IdentityNamespace with the same trust domain but different path.
func (ns IdentityNamespace) WithPath(newPath string) (IdentityNamespace, error) {
	return NewIdentityNamespace(ns.trustDomain, newPath)
}

// WithTrustDomain creates a new IdentityNamespace with the same path but different trust domain.
func (ns IdentityNamespace) WithTrustDomain(newTrustDomain TrustDomain) (IdentityNamespace, error) {
	return NewIdentityNamespace(newTrustDomain, ns.path)
}

// Note: Custom SPIFFE path validation removed - using go-spiffe's built-in validation instead.

// validateTotalLength checks if the complete SPIFFE ID length is within limits.
func (ns IdentityNamespace) validateTotalLength() error {
	totalLength := len(ns.String())
	if totalLength > MaxSPIFFEIDLength {
		return fmt.Errorf("SPIFFE ID exceeds maximum length of %d characters (current: %d): %q",
			MaxSPIFFEIDLength, totalLength, ns.String())
	}
	return nil
}
