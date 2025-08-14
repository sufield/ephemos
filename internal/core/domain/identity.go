// Package domain handles service identity and authentication policies.
package domain

import (
	"crypto/x509"
	"fmt"
)

// ServiceIdentity represents a SPIFFE service identity with name, domain, and URI.
type ServiceIdentity struct {
	Name   string
	Domain string
	URI    string
}

// NewServiceIdentity creates a new ServiceIdentity with the given name and domain.
func NewServiceIdentity(name, domain string) *ServiceIdentity {
	return &ServiceIdentity{
		Name:   name,
		Domain: domain,
		URI:    fmt.Sprintf("spiffe://%s/%s", domain, name),
	}
}

// Validate checks the identity.
func (s *ServiceIdentity) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if s.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	return nil
}

// Certificate holds cert data.
type Certificate struct {
	Cert       *x509.Certificate
	PrivateKey interface{}
	Chain      []*x509.Certificate
}

// TrustBundle holds trust data.
type TrustBundle struct {
	Certificates []*x509.Certificate
}

// AuthenticationPolicy defines authentication context.
// This is now purely for identity-based authentication without authorization.
type AuthenticationPolicy struct {
	ServiceIdentity *ServiceIdentity
}

// NewAuthenticationPolicy creates a policy for authentication.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		ServiceIdentity: identity,
	}
}
