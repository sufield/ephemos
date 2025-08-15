// Package ephemos provides simple validation interfaces that delegate to the core domain layer.
package ephemos

import (
	"github.com/sufield/ephemos/internal/core/domain"
)

// ValidationEngine is a public interface that delegates to the domain layer.
type ValidationEngine = domain.ValidationEngine

// ValidationCollectionError is a public interface that delegates to the domain layer.
type ValidationCollectionError = domain.ValidationCollectionError

// NewValidationEngine creates a new validation engine.
func NewValidationEngine() *ValidationEngine {
	return domain.NewValidationEngine()
}

// ValidateStruct is a convenience function that validates a struct using the domain validation engine.
func ValidateStruct(v any) error {
	return domain.ValidateStruct(v)
}

// ValidateStructWithEngine validates a struct with a custom validation engine.
func ValidateStructWithEngine(v any, engine *ValidationEngine) error {
	return domain.ValidateStructWithEngine(v, engine)
}

// GetValidationErrors extracts all validation errors from an error.
func GetValidationErrors(err error) []ValidationError {
	domainErrors := domain.GetValidationErrors(err)
	if domainErrors == nil {
		return nil
	}
	
	// Convert domain ValidationError to public ValidationError
	result := make([]ValidationError, len(domainErrors))
	for i, domainErr := range domainErrors {
		result[i] = ValidationError{
			Field:   domainErr.Field,
			Message: domainErr.Message,
			Value:   domainErr.Value,
		}
	}
	return result
}