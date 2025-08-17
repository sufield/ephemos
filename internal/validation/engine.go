// Package validation provides internal validation engine implementation.
package validation

import (
	"strings"

	"github.com/sufield/ephemos/internal/core/domain"
)

// Engine is the internal validation engine.
type Engine = domain.ValidationEngine

// CollectionError represents multiple validation errors.
type CollectionError = domain.ValidationCollectionError

// Error represents a single validation error.
type Error struct {
	Field   string
	Message string
	Value   interface{}
}

// NewEngine creates a new validation engine.
func NewEngine() *Engine {
	return domain.NewValidationEngine()
}

// ValidateStruct validates a struct using the domain validation engine.
func ValidateStruct(v any) error {
	return domain.ValidateStruct(v)
}

// ValidateStructWithEngine validates a struct with a custom validation engine.
func ValidateStructWithEngine(v any, engine *Engine) error {
	return domain.ValidateStructWithEngine(v, engine)
}

// GetErrors extracts all validation errors from an error.
func GetErrors(err error) []Error {
	domainErrors := domain.GetValidationErrors(err)
	if domainErrors == nil {
		return nil
	}

	// Convert domain ValidationError to internal Error
	result := make([]Error, len(domainErrors))
	for i, domainErr := range domainErrors {
		result[i] = Error{
			Field:   domainErr.Field,
			Message: domainErr.Message,
			Value:   domainErr.Value,
		}
	}
	return result
}

// IsValidationError checks if an error is a validation-related error.
func IsValidationError(err error) bool {
	// Check if the error is a validation error by checking the error message or type
	if err == nil {
		return false
	}
	// For now, just check if it contains validation-related keywords
	errStr := err.Error()
	return strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid")
}
