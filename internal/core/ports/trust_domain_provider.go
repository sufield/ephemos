// Package ports defines trust domain provider capabilities for dependency injection.
package ports

import "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"

// TrustDomainProvider provides trust domain capabilities without exposing configuration internals.
// This interface enables capability injection instead of configuration deep access.
type TrustDomainProvider interface {
	// GetTrustDomain returns the configured trust domain as a string.
	GetTrustDomain() (string, error)

	// CreateDefaultAuthorizer creates a secure default authorizer for the trust domain.
	// Returns an authorizer that verifies certificates belong to the configured trust domain.
	CreateDefaultAuthorizer() (tlsconfig.Authorizer, error)

	// IsConfigured returns true if a trust domain has been properly configured.
	IsConfigured() bool

	// ShouldSkipCertificateValidation returns true if certificate validation should be skipped (development only).
	// This is a security-sensitive operation and should only be used in development environments.
	ShouldSkipCertificateValidation() bool
}
