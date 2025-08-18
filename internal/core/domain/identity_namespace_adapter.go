// Package domain provides adapters for integrating with external SPIFFE libraries
// while maintaining clean domain boundaries.
package domain

import (
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// SPIFFELibraryAdapter provides a bridge between our domain value objects
// and the go-spiffe library. This allows gradual migration while maintaining
// compatibility with external SPIFFE dependencies.
type SPIFFELibraryAdapter struct{}

// NewSPIFFELibraryAdapter creates a new adapter for go-spiffe integration.
func NewSPIFFELibraryAdapter() *SPIFFELibraryAdapter {
	return &SPIFFELibraryAdapter{}
}

// ToGoSPIFFEID converts our IdentityNamespace to a go-spiffe ID.
// This enables compatibility with existing go-spiffe based code.
func (a *SPIFFELibraryAdapter) ToGoSPIFFEID(namespace IdentityNamespace) (spiffeid.ID, error) {
	if namespace.IsZero() {
		return spiffeid.ID{}, fmt.Errorf("cannot convert zero identity namespace to SPIFFE ID")
	}

	// Use go-spiffe parsing to ensure compatibility
	return spiffeid.FromString(namespace.String())
}

// FromGoSPIFFEID converts a go-spiffe ID to our IdentityNamespace.
// This enables migration from go-spiffe to our domain value objects.
func (a *SPIFFELibraryAdapter) FromGoSPIFFEID(id spiffeid.ID) (IdentityNamespace, error) {
	return NewIdentityNamespaceFromString(id.String())
}

// ToGoSPIFFETrustDomain converts our TrustDomain to a go-spiffe trust domain.
func (a *SPIFFELibraryAdapter) ToGoSPIFFETrustDomain(trustDomain TrustDomain) (spiffeid.TrustDomain, error) {
	if trustDomain.IsZero() {
		return spiffeid.TrustDomain{}, fmt.Errorf("cannot convert zero trust domain to go-spiffe trust domain")
	}

	return spiffeid.TrustDomainFromString(trustDomain.String())
}

// FromGoSPIFFETrustDomain converts a go-spiffe trust domain to our TrustDomain.
func (a *SPIFFELibraryAdapter) FromGoSPIFFETrustDomain(td spiffeid.TrustDomain) (TrustDomain, error) {
	return NewTrustDomain(td.String())
}

// ValidateWithGoSPIFFE validates our IdentityNamespace using go-spiffe validation.
// This provides additional compatibility checking when migrating.
func (a *SPIFFELibraryAdapter) ValidateWithGoSPIFFE(namespace IdentityNamespace) error {
	if namespace.IsZero() {
		return fmt.Errorf("identity namespace is zero")
	}

	// Parse with go-spiffe to ensure compatibility
	_, err := spiffeid.FromString(namespace.String())
	if err != nil {
		return fmt.Errorf("go-spiffe validation failed: %w", err)
	}

	// Validate our domain rules
	return namespace.Validate()
}

// CreateIdentityNamespaceFromComponents creates an IdentityNamespace from separate components
// with go-spiffe validation. This is useful when migrating existing code that constructs
// SPIFFE IDs from separate trust domain and path components.
func (a *SPIFFELibraryAdapter) CreateIdentityNamespaceFromComponents(trustDomainStr, path string) (IdentityNamespace, error) {
	// First validate with go-spiffe
	goTrustDomain, err := spiffeid.TrustDomainFromString(trustDomainStr)
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid trust domain for go-spiffe: %w", err)
	}

	goID, err := spiffeid.FromPath(goTrustDomain, path)
	if err != nil {
		return IdentityNamespace{}, fmt.Errorf("invalid SPIFFE path for go-spiffe: %w", err)
	}

	// Now create our domain value object
	return NewIdentityNamespaceFromString(goID.String())
}

// MigrateServiceIdentity helps migrate from the existing ServiceIdentity to IdentityNamespace.
// This is a helper function for gradual migration.
func (a *SPIFFELibraryAdapter) MigrateServiceIdentity(serviceIdentity *ServiceIdentity) (IdentityNamespace, error) {
	if serviceIdentity == nil {
		return IdentityNamespace{}, fmt.Errorf("service identity cannot be nil")
	}

	// Use the existing URI from ServiceIdentity
	return NewIdentityNamespaceFromString(serviceIdentity.URI())
}

// CreateServiceIdentityFromNamespace creates a ServiceIdentity from IdentityNamespace.
// This enables backward compatibility during migration.
func (a *SPIFFELibraryAdapter) CreateServiceIdentityFromNamespace(namespace IdentityNamespace) (*ServiceIdentity, error) {
	if namespace.IsZero() {
		return nil, fmt.Errorf("identity namespace cannot be zero")
	}

	// Extract service name from path
	serviceName := namespace.GetServiceName()
	if serviceName == "" {
		// For root paths or complex paths, use the full path as service name
		serviceName = namespace.GetPath()
		if serviceName == "/" {
			return nil, fmt.Errorf("cannot create service identity from root path")
		}
		// Remove leading slash for service name
		serviceName = strings.TrimPrefix(serviceName, "/")
	}

	trustDomain := namespace.GetTrustDomain()
	
	// Create ServiceIdentity using existing constructor
	return NewServiceIdentityWithValidation(serviceName, trustDomain.String(), true)
}