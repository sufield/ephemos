package domain

import (
	"crypto/x509"
	"fmt"
)

type ServiceIdentity struct {
	Name   string
	Domain string
	URI    string
}

func NewServiceIdentity(name, domain string) *ServiceIdentity {
	return &ServiceIdentity{
		Name:   name,
		Domain: domain,
		URI:    fmt.Sprintf("spiffe://%s/%s", domain, name),
	}
}

func (s *ServiceIdentity) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if s.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	return nil
}

type Certificate struct {
	Cert       *x509.Certificate
	PrivateKey interface{}
	Chain      []*x509.Certificate
}

type TrustBundle struct {
	Certificates []*x509.Certificate
}

type AuthenticationPolicy struct {
	ServiceIdentity   *ServiceIdentity
	AuthorizedClients []string
	TrustedServers    []string
}

func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		ServiceIdentity:   identity,
		AuthorizedClients: []string{},
		TrustedServers:    []string{},
	}
}

func (p *AuthenticationPolicy) AddAuthorizedClient(clientName string) {
	p.AuthorizedClients = append(p.AuthorizedClients, clientName)
}

func (p *AuthenticationPolicy) AddTrustedServer(serverName string) {
	p.TrustedServers = append(p.TrustedServers, serverName)
}

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
