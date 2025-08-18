// Package domain contains core business logic and domain models.
package domain

import (
	"fmt"
	"net/url"
	"strings"
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

	// Validate path
	if err := validateSPIFFEPath(path); err != nil {
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
func NewIdentityNamespaceFromString(spiffeID string) (IdentityNamespace, error) {
	if spiffeID == "" {
		return IdentityNamespace{}, fmt.Errorf("SPIFFE ID cannot be empty")
	}

	// Must start with spiffe://
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return IdentityNamespace{}, fmt.Errorf("SPIFFE ID must start with 'spiffe://': %q", spiffeID)
	}

	// Parse as URL for validation
	parsedURL, err := url.Parse(spiffeID)
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid SPIFFE ID format: %w", err)
	}

	if parsedURL.Scheme != "spiffe" {
		return IdentityNamespace{}, fmt.Errorf("SPIFFE ID must use 'spiffe' scheme: %q", spiffeID)
	}

	if parsedURL.Host == "" {
		return IdentityNamespace{}, fmt.Errorf("SPIFFE ID must contain trust domain: %q", spiffeID)
	}

	// Validate and create trust domain
	trustDomain, err := NewTrustDomain(parsedURL.Host)
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid trust domain in SPIFFE ID: %w", err)
	}

	// Extract path (defaults to "/" if empty)
	path := parsedURL.Path
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

	if err := validateSPIFFEPath(ns.path); err != nil {
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

// validateSPIFFEPath validates a SPIFFE path according to specification constraints.
func validateSPIFFEPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with '/': %q", path)
	}

	// Root path is always valid
	if path == "/" {
		return nil
	}

	// Check for invalid patterns
	if strings.Contains(path, "//") {
		return fmt.Errorf("path cannot contain double slashes: %q", path)
	}

	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("path cannot end with '/' unless it's root: %q", path)
	}

	if strings.Contains(path, "/.") || strings.Contains(path, "./") {
		return fmt.Errorf("path cannot contain dot segments: %q", path)
	}

	// Validate each path segment
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, segment := range segments {
		if segment == "" {
			continue // Skip empty segments (shouldn't happen due to earlier checks)
		}
		
		// Check for invalid characters in segment
		for _, r := range segment {
			if !isValidSPIFFEPathChar(r) {
				return fmt.Errorf("path contains invalid characters (allowed: a-z, A-Z, 0-9, ., _, -): %q", path)
			}
		}
	}

	// Check length
	if len(path) > MaxPathLength {
		return fmt.Errorf("path exceeds maximum length of %d characters: %q", MaxPathLength, path)
	}

	return nil
}

// isValidSPIFFEPathChar checks if a character is valid in a SPIFFE path segment.
func isValidSPIFFEPathChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '.' || r == '_' || r == '-'
}

// validateTotalLength checks if the complete SPIFFE ID length is within limits.
func (ns IdentityNamespace) validateTotalLength() error {
	totalLength := len(ns.String())
	if totalLength > MaxSPIFFEIDLength {
		return fmt.Errorf("SPIFFE ID exceeds maximum length of %d characters (current: %d): %q", 
			MaxSPIFFEIDLength, totalLength, ns.String())
	}
	return nil
}