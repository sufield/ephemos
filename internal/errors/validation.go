// Package errors provides internal error handling utilities.
package errors

import "fmt"

// ValidationError represents a validation failure with context about what was invalid.
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
