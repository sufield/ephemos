// Package config provides configuration-based implementations of core ports.
package config

import (
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"

	"github.com/sufield/ephemos/internal/core/ports"
)

// TrustDomainCapability provides access to trust domain configuration.
type TrustDomainCapability interface {
	GetServiceDomain() string
	ShouldSkipCertificateValidation() bool
}

// TrustDomainAdapter provides trust domain capabilities by adapting from configuration.
// This adapter encapsulates configuration access and provides a clean interface.
type TrustDomainAdapter struct {
	capability TrustDomainCapability
}

// configCapability implements TrustDomainCapability for ports.Configuration
type configCapability struct {
	config *ports.Configuration
}

func (c *configCapability) GetServiceDomain() string {
	if c.config == nil {
		return ""
	}
	return c.config.Service.Domain
}

func (c *configCapability) ShouldSkipCertificateValidation() bool {
	if c.config == nil {
		return false
	}
	return c.config.ShouldSkipCertificateValidation()
}

// NewTrustDomainAdapter creates a new trust domain adapter from configuration.
func NewTrustDomainAdapter(config *ports.Configuration) *TrustDomainAdapter {
	return &TrustDomainAdapter{
		capability: &configCapability{config: config},
	}
}

// NewTrustDomainAdapterWithCapability creates a new trust domain adapter with injected capability.
func NewTrustDomainAdapterWithCapability(capability TrustDomainCapability) *TrustDomainAdapter {
	return &TrustDomainAdapter{
		capability: capability,
	}
}

// GetTrustDomain returns the configured trust domain as a string.
func (t *TrustDomainAdapter) GetTrustDomain() (string, error) {
	if t.capability == nil {
		return "", fmt.Errorf("capability is nil")
	}

	domain := t.capability.GetServiceDomain()
	if domain == "" {
		return "", fmt.Errorf("trust domain not configured")
	}

	return domain, nil
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
	if t.capability == nil {
		return false
	}

	// Check if trust domain is set and valid
	domain := t.capability.GetServiceDomain()
	if domain == "" {
		return false
	}

	// Validate trust domain format
	_, err := spiffeid.TrustDomainFromString(domain)
	return err == nil
}

// ShouldSkipCertificateValidation returns true if certificate validation should be skipped (development only).
func (t *TrustDomainAdapter) ShouldSkipCertificateValidation() bool {
	if t.capability == nil {
		return false
	}
	return t.capability.ShouldSkipCertificateValidation()
}
