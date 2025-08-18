// Package memidentity provides an in-memory fake IdentityProvider for testing.
package memidentity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"sync"
	"time"

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

// fakePrivateKey implements crypto.Signer for testing purposes.
type fakePrivateKey struct{}

func (f *fakePrivateKey) Public() crypto.PublicKey {
	// Return a simple placeholder public key
	key, _ := rsa.GenerateKey(rand.Reader, 512) // Small key for testing
	return &key.PublicKey
}

func (f *fakePrivateKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return []byte("fake-signature"), nil
}

// New creates a new in-memory IdentityProvider with default test data.
func New() *Provider {
	// Create fake X.509 certificate for testing
	fakeCert := &x509.Certificate{
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		IsCA:      false,
	}

	// Create fake CA certificate for trust bundle
	fakeCACert := &x509.Certificate{
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		IsCA:      true,
	}

	return &Provider{
		identity: domain.NewServiceIdentity("test-service", "example.com"),
		cert: &domain.Certificate{
			Cert:       fakeCert,
			PrivateKey: &fakePrivateKey{}, // Now properly implements crypto.Signer
			Chain:      []*x509.Certificate{},
		},
		bundle: mustCreateTrustBundle([]*x509.Certificate{fakeCACert}),
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

// mustCreateTrustBundle creates a trust bundle or panics. For testing only.
func mustCreateTrustBundle(certs []*x509.Certificate) *domain.TrustBundle {
	bundle, err := domain.NewTrustBundle(certs)
	if err != nil {
		panic("failed to create test trust bundle: " + err.Error())
	}
	return bundle
}
