//go:build prod_arch_off

package arch

// Production stubs that disable all runtime validation when built with -tags=prod_arch_off

// Validator is a no-op stub for production builds.
type Validator struct{}

// NewValidator creates a no-op validator for production.
func NewValidator(enabled bool) *Validator {
	return &Validator{}
}

// SetEnabled is a no-op in production builds.
func (v *Validator) SetEnabled(b bool) {}

// IsEnabled always returns false in production builds.
func (v *Validator) IsEnabled() bool { return false }

// ValidateCall is a no-op in production builds.
func (v *Validator) ValidateCall(operation string) error { return nil }

// GetViolations returns empty slice in production builds.
func (v *Validator) GetViolations() []string { return nil }

// ClearViolations is a no-op in production builds.
func (v *Validator) ClearViolations() {}

// Allow is a no-op in production builds.
func (v *Validator) Allow(from, to string) {}

// Global API stubs for production
var globalValidator = NewValidator(false) //nolint:gochecknoglobals

func SetGlobalValidationEnabled(enabled bool) {}
func IsGlobalValidationEnabled() bool         { return false }
func ValidateBoundary(operation string) error { return nil }
func GetGlobalViolations() []string           { return nil }
func ClearGlobalViolations()                  {}
func AllowGlobalCrossing(from, to string)     {}
