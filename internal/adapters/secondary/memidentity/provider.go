// Package memidentity provides an in-memory fake IdentityProvider for testing.
package memidentity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Provider is a fake in-memory IdentityProvider for testing.
type Provider struct {
	mu       sync.RWMutex
	identity *domain.ServiceIdentity
	cert     *domain.Certificate
	bundle   *x509bundle.Bundle
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
		bundle: mustCreateX509Bundle([]*x509.Certificate{fakeCACert}),
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

// GetServiceIdentity returns the configured service identity as spiffeid.ID.
func (p *Provider) GetServiceIdentity() (spiffeid.ID, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return spiffeid.ID{}, ports.ErrIdentityNotFound
	}

	// Create SPIFFE ID from stored identity
	spiffeID, err := spiffeid.FromURI(&url.URL{
		Scheme: "spiffe",
		Host:   p.identity.Domain(),
		Path:   "/" + p.identity.Name(),
	})
	if err != nil {
		return spiffeid.ID{}, err
	}

	return spiffeID, nil
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
func (p *Provider) GetTrustBundle() (*x509bundle.Bundle, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ports.ErrIdentityNotFound
	}

	return p.bundle, nil
}

// GetSVID returns a fake SVID for testing.
func (p *Provider) GetSVID() (*x509svid.SVID, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ports.ErrIdentityNotFound
	}

	// Create identity document from certificate and trust bundle
	if p.cert == nil {
		return nil, ports.ErrIdentityNotFound
	}

	// Create a fake SPIFFE ID for testing
	spiffeID, err := spiffeid.FromURI(&url.URL{
		Scheme: "spiffe",
		Host:   p.identity.Domain(),
		Path:   "/" + p.identity.Name(),
	})
	if err != nil {
		return nil, err
	}

	// Create fake x509svid.SVID for testing
	return &x509svid.SVID{
		ID:           spiffeID,
		Certificates: []*x509.Certificate{p.cert.Cert},
		PrivateKey:   p.cert.PrivateKey,
	}, nil
}

// Close marks the provider as closed.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

// mustCreateX509Bundle creates an x509bundle.Bundle or panics. For testing only.
func mustCreateX509Bundle(certs []*x509.Certificate) *x509bundle.Bundle {
	// Create a fake trust domain for testing
	td, err := spiffeid.TrustDomainFromString("example.com")
	if err != nil {
		panic("failed to create test trust domain: " + err.Error())
	}
	
	bundle := x509bundle.New(td)
	for _, cert := range certs {
		bundle.AddX509Authority(cert)
	}
	return bundle
}
