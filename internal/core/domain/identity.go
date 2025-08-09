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

// AuthenticationPolicy defines auth rules.
type AuthenticationPolicy struct {
	ServiceIdentity   *ServiceIdentity
	AuthorizedClients []string
	TrustedServers    []string
}

// NewAuthenticationPolicy creates a policy.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		ServiceIdentity:   identity,
		AuthorizedClients: []string{},
		TrustedServers:    []string{},
	}
}

// AddAuthorizedClient adds a client.
func (p *AuthenticationPolicy) AddAuthorizedClient(clientName string) {
	p.AuthorizedClients = append(p.AuthorizedClients, clientName)
}

// AddTrustedServer adds a server.
func (p *AuthenticationPolicy) AddTrustedServer(serverName string) {
	p.TrustedServers = append(p.TrustedServers, serverName)
}

// IsClientAuthorized checks client.
func (p *AuthenticationPolicy) IsClientAuthorized(clientName string) bool {
	if len(p.AuthorizedClients) == 0 {
		return true
	}
	for _, authorized := range p.AuthorizedClients {
		if authorized == clientName {
			return true
		}
	}
	return false
}

// IsServerTrusted checks server.
func (p *AuthenticationPolicy) IsServerTrusted(serverName string) bool {
	if len(p.TrustedServers) == 0 {
		return true
	}
	for _, trusted := range p.TrustedServers {
		if trusted == serverName {
			return true
		}
	}
	return false
}
