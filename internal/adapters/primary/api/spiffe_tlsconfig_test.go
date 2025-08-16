package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/core/domain"
)

var errMock = errors.New("mock")

// MockIdentityService is a mock implementation for testing (decoupled from testify)
type MockIdentityService struct {
	cert        *domain.Certificate
	trustBundle *domain.TrustBundle
	shouldError bool
}

func (m *MockIdentityService) GetCertificate() (*domain.Certificate, error) {
	if m.shouldError {
		return nil, errMock
	}
	return m.cert, nil
}

func (m *MockIdentityService) GetTrustBundle() (*domain.TrustBundle, error) {
	if m.shouldError {
		return nil, errMock
	}
	return m.trustBundle, nil
}

// createTestCertificate builds a self-signed leaf with ed25519 for fast, deterministic tests.
// If spiffeID == "", the cert has no SPIFFE URI SAN.
func createTestCertificate(t *testing.T, spiffeID string) (*x509.Certificate, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	var uris []*url.URL
	if spiffeID != "" {
		u, err := url.Parse(spiffeID)
		if err != nil {
			t.Fatalf("parse spiffe: %v", err)
		}
		uris = []*url.URL{u}
	}

	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test"}},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		URIs:                  uris,
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, pub, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	return cert, priv
}

func TestClientConnection_CreateSVIDSource(t *testing.T) {
	t.Parallel()

	t.Run("successful SVID source creation", func(t *testing.T) {
		t.Parallel()
		cert, key := createTestCertificate(t, "spiffe://example.org/test-service")
		domainCert := &domain.Certificate{Cert: cert, PrivateKey: key}

		mockID := &MockIdentityService{cert: domainCert}
		cc := &ClientConnection{identityService: mockID}

		svidSrc, err := cc.createSVIDSource()
		require.NoError(t, err)
		require.NotNil(t, svidSrc)

		svid, err := svidSrc.GetX509SVID()
		require.NoError(t, err)
		assert.Equal(t, "spiffe://example.org/test-service", svid.ID.String())
		assert.Same(t, cert, svid.Certificates[0])
		assert.Equal(t, key, svid.PrivateKey)
	})

	t.Run("no identity service", func(t *testing.T) {
		t.Parallel()
		cc := &ClientConnection{identityService: nil}
		svidSrc, err := cc.createSVIDSource()
		require.Error(t, err)
		require.Nil(t, svidSrc)
		assert.Contains(t, err.Error(), "no identity service")
	})

	t.Run("certificate retrieval error", func(t *testing.T) {
		t.Parallel()
		cc := &ClientConnection{identityService: &MockIdentityService{shouldError: true}}
		svidSrc, err := cc.createSVIDSource()
		require.NoError(t, err)

		_, err = svidSrc.GetX509SVID()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get certificate")
	})

	t.Run("missing SPIFFE URI SAN", func(t *testing.T) {
		t.Parallel()
		cert, key := createTestCertificate(t, "") // no spiffe SAN
		cc := &ClientConnection{
			identityService: &MockIdentityService{cert: &domain.Certificate{Cert: cert, PrivateKey: key}},
		}
		svidSrc, err := cc.createSVIDSource()
		require.NoError(t, err)

		_, err = svidSrc.GetX509SVID()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid SPIFFE ID")
	})

	t.Run("private key not a crypto.Signer", func(t *testing.T) {
		t.Parallel()
		// This test case is not practically testable due to Go's type system.
		// The domain.Certificate.PrivateKey field is typed as crypto.Signer,
		// so it's impossible to put a non-crypto.Signer value there without unsafe operations.
		// The code path exists for defensive programming but can't be reached in normal operation.
		t.Skip("Cannot create invalid domain.Certificate due to type constraints - code path exists for defensive programming")
	})
}

func TestClientConnection_CreateBundleSource(t *testing.T) {
	t.Parallel()

	t.Run("successful bundle source", func(t *testing.T) {
		t.Parallel()
		cert, _ := createTestCertificate(t, "spiffe://example.org/test-service")
		tb := &domain.TrustBundle{Certificates: []*x509.Certificate{cert}}
		cc := &ClientConnection{identityService: &MockIdentityService{trustBundle: tb}}

		src, err := cc.createBundleSource()
		require.NoError(t, err)
		require.NotNil(t, src)

		td := mustParseTrustDomain("example.org")
		b, err := src.GetX509BundleForTrustDomain(td)
		require.NoError(t, err)
		require.NotNil(t, b)
		assert.Equal(t, td, b.TrustDomain())
	})

	t.Run("empty trust bundle", func(t *testing.T) {
		t.Parallel()
		cc := &ClientConnection{
			identityService: &MockIdentityService{trustBundle: &domain.TrustBundle{}},
		}
		src, err := cc.createBundleSource()
		require.NoError(t, err)

		_, err = src.GetX509BundleForTrustDomain(mustParseTrustDomain("example.org"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle is empty")
	})

	t.Run("trust bundle retrieval error", func(t *testing.T) {
		t.Parallel()
		cc := &ClientConnection{identityService: &MockIdentityService{shouldError: true}}
		src, err := cc.createBundleSource()
		require.NoError(t, err)

		_, err = src.GetX509BundleForTrustDomain(mustParseTrustDomain("example.org"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get trust bundle")
	})
}

func TestClientConnection_SPIFFEAdapters_Together(t *testing.T) {
	t.Parallel()
	cert, key := createTestCertificate(t, "spiffe://example.org/test-service")
	tb := &domain.TrustBundle{Certificates: []*x509.Certificate{cert}}
	cc := &ClientConnection{
		identityService: &MockIdentityService{
			cert:        &domain.Certificate{Cert: cert, PrivateKey: key},
			trustBundle: tb,
		},
	}
	svidSrc, err := cc.createSVIDSource()
	require.NoError(t, err)
	bundleSrc, err := cc.createBundleSource()
	require.NoError(t, err)

	svid, err := svidSrc.GetX509SVID()
	require.NoError(t, err)
	assert.Equal(t, "spiffe://example.org/test-service", svid.ID.String())

	b, err := bundleSrc.GetX509BundleForTrustDomain(mustParseTrustDomain("example.org"))
	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestClientConnection_CreateBundleSourceForTrustDomain(t *testing.T) {
	t.Parallel()

	t.Run("enforces trust domain isolation", func(t *testing.T) {
		t.Parallel()
		cert, _ := createTestCertificate(t, "spiffe://example.org/test-service")
		tb := &domain.TrustBundle{Certificates: []*x509.Certificate{cert}}
		cc := &ClientConnection{identityService: &MockIdentityService{trustBundle: tb}}

		restrictedTD := mustParseTrustDomain("example.org")
		src, err := cc.createBundleSourceForTrustDomain(restrictedTD)
		require.NoError(t, err)

		// Should work for the allowed trust domain
		b, err := src.GetX509BundleForTrustDomain(restrictedTD)
		require.NoError(t, err)
		assert.Equal(t, restrictedTD, b.TrustDomain())

		// Should reject a different trust domain
		differentTD := mustParseTrustDomain("other.org")
		_, err = src.GetX509BundleForTrustDomain(differentTD)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trust domain other.org not allowed")
		assert.Contains(t, err.Error(), "restricted to example.org")
	})
}

// Helper function to parse trust domain
func mustParseTrustDomain(td string) spiffeid.TrustDomain {
	trustDomain, err := spiffeid.TrustDomainFromString(td)
	if err != nil {
		panic(err)
	}
	return trustDomain
}