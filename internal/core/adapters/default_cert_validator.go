// Package adapters contains concrete implementations of ports interfaces.
// These adapters implement the port contracts defined in the ports package,
// completing the hexagonal architecture by providing concrete behavior
// for external integrations while keeping the domain layer pure.
package adapters

import (
	"fmt"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// DefaultCertValidator provides the standard certificate validation implementation.
// It implements the ports.CertValidatorPort interface by delegating to the domain
// Certificate's built-in validation logic, ensuring consistent validation behavior
// across the application.
//
// This adapter follows the hexagonal architecture pattern by implementing a port
// interface while delegating the actual business logic to domain entities.
type DefaultCertValidator struct{}

// NewDefaultCertValidator creates a new instance of the default certificate validator.
// This constructor provides a clear factory method for creating the validator adapter.
func NewDefaultCertValidator() ports.CertValidatorPort {
	return &DefaultCertValidator{}
}

// Validate delegates to the Certificate's Validate method.
// This implementation provides the standard certificate validation behavior
// by leveraging the domain entity's validation logic, ensuring that business
// rules remain in the domain layer while providing a port-compliant interface.
//
// Parameters:
//   cert: The certificate to validate (must not be nil)
//   opts: Validation options for controlling validation behavior
//
// Returns an error if validation fails, nil if the certificate is valid.
func (v *DefaultCertValidator) Validate(cert *domain.Certificate, opts domain.CertValidationOptions) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	return cert.Validate(opts)
}