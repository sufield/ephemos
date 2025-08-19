// Package config provides configuration-based implementations of core ports.
package config

import (
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"

	"github.com/sufield/ephemos/internal/core/ports"
)

// TrustDomainAdapter provides trust domain capabilities by adapting from configuration.
// This adapter encapsulates configuration access and provides a clean interface.
type TrustDomainAdapter struct {
	config *ports.Configuration
}

// NewTrustDomainAdapter creates a new trust domain adapter from configuration.
func NewTrustDomainAdapter(config *ports.Configuration) *TrustDomainAdapter {
	return &TrustDomainAdapter{
		config: config,
	}
}

// GetTrustDomain returns the configured trust domain as a string.
func (t *TrustDomainAdapter) GetTrustDomain() (string, error) {
	if t.config == nil {
		return "", fmt.Errorf("configuration is nil")
	}

	if t.config.Service.Domain == "" {
		return "", fmt.Errorf("trust domain not configured")
	}

	return t.config.Service.Domain, nil
}

// CreateDefaultAuthorizer creates a secure default authorizer for the trust domain.
func (t *TrustDomainAdapter) CreateDefaultAuthorizer() (tlsconfig.Authorizer, error) {
	trustDomainStr, err := t.GetTrustDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to get trust domain: %w", err)
	}

	// Parse trust domain using go-spiffe
	trustDomain, err := spiffeid.TrustDomainFromString(trustDomainStr)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain %q: %w", trustDomainStr, err)
	}

	// Create secure authorizer that only allows members of this trust domain
	authorizer := tlsconfig.AuthorizeMemberOf(trustDomain)
	return authorizer, nil
}

// IsConfigured returns true if a trust domain has been properly configured.
func (t *TrustDomainAdapter) IsConfigured() bool {
	if t.config == nil {
		return false
	}

	// Check if trust domain is set and valid
	if t.config.Service.Domain == "" {
		return false
	}

	// Validate trust domain format
	_, err := spiffeid.TrustDomainFromString(t.config.Service.Domain)
	return err == nil
}

// ShouldSkipCertificateValidation returns true if certificate validation should be skipped (development only).
func (t *TrustDomainAdapter) ShouldSkipCertificateValidation() bool {
	if t.config == nil {
		return false
	}
	return t.config.ShouldSkipCertificateValidation()
}
