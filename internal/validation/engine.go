// Package validation provides internal validation engine implementation.
package validation

import (
	"strings"

	"github.com/sufield/ephemos/internal/core/domain"
)

// Engine is the internal validation engine (V2 validator).
type Engine = domain.V2Validator

// Error represents a single validation error.
type Error struct {
	Field   string
	Message string
	Value   interface{}
}

// NewEngine creates a new V2 validation engine.
func NewEngine() *Engine {
	return domain.NewV2Validator()
}

// ValidateStruct validates a struct using the V2 validation engine.
func ValidateStruct(v any) error {
	return domain.ValidateStructV2(v)
}

// ValidateStructWithEngine validates a struct with a custom V2 validation engine.
func ValidateStructWithEngine(v any, engine *Engine) error {
	return engine.Validate(v)
}

// GetErrors extracts all validation errors from an error.
func GetErrors(err error) []Error {
	validationErrors := domain.ConvertValidationErrors(err)
	if validationErrors == nil {
		return nil
	}

	// Convert domain ValidationErrorV2 to internal Error
	result := make([]Error, len(validationErrors))
	for i, validationErr := range validationErrors {
		result[i] = Error{
			Field:   validationErr.Field,
			Message: validationErr.Message,
			Value:   validationErr.Value,
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
