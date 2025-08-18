// Package domain provides domain value objects and entities.
package domain

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

// ServiceName is a value object for validated service names.
// It encapsulates validation logic centrally and enforces invariants at construction.
type ServiceName struct {
	value string // Private to enforce encapsulation
}

// Valid service name pattern: alphanumeric, hyphens, underscores, dots
// Must start and end with alphanumeric characters
var serviceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)

// NewServiceName creates a ServiceName, applying validation.
// Returns an error if the service name is invalid.
func NewServiceName(name string) (ServiceName, error) {
	trimmed := strings.TrimSpace(name)
	
	if trimmed == "" {
		return ServiceName{}, fmt.Errorf("service name cannot be empty or whitespace-only")
	}
	
	// Length limits based on common practices and SPIFFE ID constraints
	if len(trimmed) > 100 {
		return ServiceName{}, fmt.Errorf("service name too long: maximum 100 characters, got %d", len(trimmed))
	}
	
	if len(trimmed) < 1 {
		return ServiceName{}, fmt.Errorf("service name too short: minimum 1 character")
	}
	
	// Check for valid characters and pattern
	if !serviceNamePattern.MatchString(trimmed) {
		return ServiceName{}, fmt.Errorf("service name contains invalid characters: must contain only alphanumeric characters, hyphens, underscores, and dots, and must start/end with alphanumeric characters")
	}
	
	// Additional domain-specific rules
	if strings.Contains(strings.ToLower(trimmed), "example") {
		return ServiceName{}, fmt.Errorf("service name cannot contain 'example': use a real service name")
	}
	
	if strings.Contains(strings.ToLower(trimmed), "test") && !strings.HasSuffix(strings.ToLower(trimmed), "-test") {
		return ServiceName{}, fmt.Errorf("service name should not contain 'test' unless it's a proper test service ending with '-test'")
	}
	
	return ServiceName{value: trimmed}, nil
}

// NewServiceNameUnsafe creates a ServiceName without validation.
// This should only be used for testing or when validation has already been performed.
// Use NewServiceName in production code.
func NewServiceNameUnsafe(name string) ServiceName {
	return ServiceName{value: strings.TrimSpace(name)}
}

// String returns the service name as a string.
// This is a temporary compatibility method during refactoring.
func (sn ServiceName) String() string {
	return sn.value
}

// Value returns the service name value.
// This is the preferred method for accessing the service name.
func (sn ServiceName) Value() string {
	return sn.value
}

// Equals compares two ServiceNames for equality.
func (sn ServiceName) Equals(other ServiceName) bool {
	return sn.value == other.value
}

// IsEmpty returns true if the service name is empty.
func (sn ServiceName) IsEmpty() bool {
	return sn.value == ""
}

// Length returns the length of the service name.
func (sn ServiceName) Length() int {
	return len(sn.value)
}

// ToLower returns a new ServiceName with the value converted to lowercase.
func (sn ServiceName) ToLower() ServiceName {
	return ServiceName{value: strings.ToLower(sn.value)}
}

// Contains checks if the service name contains the given substring (case-insensitive).
func (sn ServiceName) Contains(substring string) bool {
	return strings.Contains(strings.ToLower(sn.value), strings.ToLower(substring))
}

// HasPrefix checks if the service name starts with the given prefix.
func (sn ServiceName) HasPrefix(prefix string) bool {
	return strings.HasPrefix(sn.value, prefix)
}

// HasSuffix checks if the service name ends with the given suffix.
func (sn ServiceName) HasSuffix(suffix string) bool {
	return strings.HasSuffix(sn.value, suffix)
}

// IsValidForProduction checks if the service name is suitable for production use.
// This applies additional rules beyond basic validation.
func (sn ServiceName) IsValidForProduction() error {
	if sn.Contains("demo") {
		return fmt.Errorf("service name contains 'demo': not suitable for production")
	}
	
	if sn.Contains("example") {
		return fmt.Errorf("service name contains 'example': not suitable for production")
	}
	
	if sn.Contains("localhost") {
		return fmt.Errorf("service name contains 'localhost': not suitable for production")
	}
	
	if strings.HasSuffix(strings.ToLower(sn.value), "-test") {
		return fmt.Errorf("service name ends with '-test': not suitable for production")
	}
	
	return nil
}

// ServiceNameDecodeHook provides a mapstructure decode hook for ServiceName.
// This allows automatic conversion from string to ServiceName during configuration unmarshalling.
func ServiceNameDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		// Only convert from string to ServiceName
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(ServiceName{}) {
			return data, nil
		}
		
		// Convert string to ServiceName using the validated constructor
		str, ok := data.(string)
		if !ok {
			return data, nil
		}
		
		serviceName, err := NewServiceName(str)
		if err != nil {
			// For configuration loading, we want to fail fast on invalid service names
			return nil, fmt.Errorf("invalid service name %q: %w", str, err)
		}
		
		return serviceName, nil
	}
}