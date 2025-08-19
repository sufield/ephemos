// Package domain provides domain predicates that leverage go-spiffe's built-in validation.
package domain

import (
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// SPIFFEPath represents a validated SPIFFE ID path component.
// This value object leverages go-spiffe's built-in validation instead of reimplementing it.
type SPIFFEPath struct {
	path string
	// Cache validation result
	validated bool
}

// NewSPIFFEPath creates a validated SPIFFE path from a raw path string.
// This uses go-spiffe's built-in validation to ensure SPIFFE compliance.
func NewSPIFFEPath(trustDomain spiffeid.TrustDomain, rawPath string) (SPIFFEPath, error) {
	// Clean the path - go-spiffe expects no leading slash for FromPath
	cleanPath := strings.TrimPrefix(rawPath, "/")
	
	// Use go-spiffe's built-in validation
	_, err := spiffeid.FromPath(trustDomain, cleanPath)
	if err != nil {
		return SPIFFEPath{}, fmt.Errorf("invalid SPIFFE path: %w", err)
	}
	
	return SPIFFEPath{
		path:      cleanPath,
		validated: true,
	}, nil
}

// NewSPIFFEPathFromID extracts and validates the path from an existing SPIFFE ID.
// This is guaranteed to be valid since it comes from a validated SPIFFE ID.
func NewSPIFFEPathFromID(id spiffeid.ID) SPIFFEPath {
	// Path from a valid SPIFFE ID is already validated
	path := strings.TrimPrefix(id.Path(), "/")
	return SPIFFEPath{
		path:      path,
		validated: true,
	}
}

// IsEmpty returns true if the SPIFFE path is empty.
// This expresses domain intent: "is this path empty?".
func (sp SPIFFEPath) IsEmpty() bool {
	return len(sp.path) == 0
}

// IsValid returns true if the path has been validated by go-spiffe.
func (sp SPIFFEPath) IsValid() bool {
	return sp.validated && !sp.IsEmpty()
}

// String returns the string representation of the SPIFFE path.
func (sp SPIFFEPath) String() string {
	return sp.path
}

// ToServiceName converts the SPIFFE path to a service name string.
// This expresses domain intent: "convert to service name".
func (sp SPIFFEPath) ToServiceName() string {
	return sp.path
}

// Segments returns the path segments split by '/'.
// This expresses domain intent: "give me the path segments".
func (sp SPIFFEPath) Segments() []string {
	if sp.IsEmpty() {
		return []string{}
	}
	return strings.Split(sp.path, "/")
}

// HasMultipleSegments returns true if the path has multiple segments.
// This expresses domain intent: "is this a complex path?".
func (sp SPIFFEPath) HasMultipleSegments() bool {
	return len(sp.Segments()) > 1
}