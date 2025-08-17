// Package ports defines interfaces for the application's ports (hexagonal architecture).
package ports

import (
	"github.com/sufield/ephemos/internal/core/domain"
)

// CertValidatorPort defines the interface for certificate validation.
// This allows for custom validation strategies beyond the default implementation.
type CertValidatorPort interface {
	// Validate performs certificate validation with the provided options.
	// Returns an error if validation fails, nil if successful.
	Validate(cert *domain.Certificate, opts domain.CertValidationOptions) error
}