// Package errors provides structured error types for validation and configuration.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for production validation
var (
	// Trust domain errors
	ErrExampleTrustDomain   = errors.New("trust domain contains example.org")
	ErrLocalhostTrustDomain = errors.New("trust domain contains localhost")
	ErrDemoTrustDomain      = errors.New("trust domain contains demo values")

	// Service name errors
	ErrExampleServiceName = errors.New("service name contains example")
	ErrDemoServiceName    = errors.New("service name contains demo")

	// Configuration security errors
	ErrDebugEnabled        = errors.New("debug mode enabled")
	ErrInsecureSkipVerify  = errors.New("certificate validation disabled")
	ErrWildcardClients     = errors.New("wildcard authorized clients")
	ErrInsecureSocketPath  = errors.New("socket path not in secure directory")

	// Environment errors
	ErrVerboseLogging = errors.New("verbose logging enabled")
)

// ProductionValidationError wraps multiple validation errors
type ProductionValidationError struct {
	Errors []error
}

func (e *ProductionValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "production validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("production validation failed: %v", e.Errors[0])
	}
	return fmt.Sprintf("production validation failed with %d errors", len(e.Errors))
}

func (e *ProductionValidationError) Unwrap() []error {
	return e.Errors
}

// NewProductionValidationError creates a new production validation error
func NewProductionValidationError(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	return &ProductionValidationError{Errors: errs}
}