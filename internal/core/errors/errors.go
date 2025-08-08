// Package errors defines custom error types for the Ephemos library
package errors

import "fmt"

// DomainError represents errors in the domain logic
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Common domain errors
var (
	ErrInvalidServiceName = &DomainError{
		Code:    "INVALID_SERVICE_NAME",
		Message: "service name is invalid",
	}
	
	ErrInvalidDomain = &DomainError{
		Code:    "INVALID_DOMAIN",
		Message: "domain is invalid",
	}
	
	ErrMissingConfiguration = &DomainError{
		Code:    "MISSING_CONFIGURATION",
		Message: "required configuration is missing",
	}
	
	ErrSPIFFERegistration = &DomainError{
		Code:    "SPIFFE_REGISTRATION_FAILED",
		Message: "failed to register service with SPIFFE",
	}
	
	ErrInvalidSocketPath = &DomainError{
		Code:    "INVALID_SOCKET_PATH",
		Message: "SPIFFE socket path is invalid",
	}
	
	ErrCertificateUnavailable = &DomainError{
		Code:    "CERTIFICATE_UNAVAILABLE",
		Message: "certificate is not available",
	}
	
	ErrTrustBundleUnavailable = &DomainError{
		Code:    "TRUST_BUNDLE_UNAVAILABLE",
		Message: "trust bundle is not available",
	}
	
	ErrConnectionFailed = &DomainError{
		Code:    "CONNECTION_FAILED",
		Message: "failed to establish connection",
	}
)

// NewDomainError creates a new domain error with context
func NewDomainError(base *DomainError, err error) error {
	return &DomainError{
		Code:    base.Code,
		Message: base.Message,
		Err:     err,
	}
}

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}