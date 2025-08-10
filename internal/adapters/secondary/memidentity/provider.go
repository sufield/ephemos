// Package memidentity provides an in-memory fake IdentityProvider for testing.
package memidentity

import (
	"crypto/x509"
	"sync"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Provider is a fake in-memory IdentityProvider for testing.
type Provider struct {
	mu       sync.RWMutex
	identity *domain.ServiceIdentity
	cert     *domain.Certificate
	bundle   *domain.TrustBundle
	closed   bool
}

// New creates a new in-memory IdentityProvider with default test data.
func New() *Provider {
	// Create fake X.509 certificate for testing
	fakeCert := &x509.Certificate{}

	return &Provider{
		identity: &domain.ServiceIdentity{
			Name:   "test-service",
			Domain: "example.com",
			URI:    "spiffe://example.com/test-service",
		},
		cert: &domain.Certificate{
			Cert:       fakeCert,
			PrivateKey: "fake-private-key",
			Chain:      []*x509.Certificate{fakeCert},
		},
		bundle: &domain.TrustBundle{
			Certificates: []*x509.Certificate{fakeCert},
		},
	}
}

// WithIdentity sets custom identity for testing.
func (p *Provider) WithIdentity(identity *domain.ServiceIdentity) *Provider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.identity = identity
	return p
}

// WithCertificate sets custom certificate for testing.
func (p *Provider) WithCertificate(cert *domain.Certificate) *Provider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cert = cert
	return p
}

// GetServiceIdentity returns the configured service identity.
func (p *Provider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ports.ErrIdentityNotFound
	}

	return p.identity, nil
}

// GetCertificate returns the configured certificate.
func (p *Provider) GetCertificate() (*domain.Certificate, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ports.ErrIdentityNotFound
	}

	return p.cert, nil
}

// GetTrustBundle returns the configured trust bundle.
func (p *Provider) GetTrustBundle() (*domain.TrustBundle, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ports.ErrIdentityNotFound
	}

	return p.bundle, nil
}

// Close marks the provider as closed.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}
