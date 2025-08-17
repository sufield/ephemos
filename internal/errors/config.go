// Package errors provides internal error handling utilities.
package errors

import (
	"errors"
	"fmt"
)

// Domain-specific errors for configuration validation failures.
var (
	// ErrInvalidConfig is returned when configuration validation fails.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConfigFileNotFound is returned when the specified config file doesn't exist.
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrConfigFileUnreadable is returned when config file exists but cannot be read.
	ErrConfigFileUnreadable = errors.New("configuration file unreadable")

	// ErrConfigMalformed is returned when config file has invalid YAML syntax.
	ErrConfigMalformed = errors.New("configuration file malformed")
)

// ConfigValidationError provides detailed information about configuration validation failures.
type ConfigValidationError struct {
	File    string // Configuration file path
	Field   string // Field that failed validation
	Value   any    // Invalid value
	Message string // Human-readable error message
	Cause   error  // Underlying error
}

func (e *ConfigValidationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("config validation failed in %s: %s", e.File, e.Message)
	}
	return fmt.Sprintf("config validation failed: %s", e.Message)
}

func (e *ConfigValidationError) Unwrap() error {
	return e.Cause
}

// IsConfigurationError checks if an error is a configuration-related error.
func IsConfigurationError(err error) bool {
	var configErr *ConfigValidationError
	if errors.As(err, &configErr) {
		return true
	}

	// Check for known config errors
	return errors.Is(err, ErrInvalidConfig) ||
		errors.Is(err, ErrConfigFileNotFound) ||
		errors.Is(err, ErrConfigFileUnreadable) ||
		errors.Is(err, ErrConfigMalformed)
}

// GetConfigValidationError attempts to extract a ConfigValidationError from an error.
func GetConfigValidationError(err error) *ConfigValidationError {
	var configErr *ConfigValidationError
	if errors.As(err, &configErr) {
		return configErr
	}
	return nil
}
