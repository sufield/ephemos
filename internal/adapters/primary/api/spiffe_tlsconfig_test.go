package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/core/domain"
)

// MockIdentityService is a mock implementation for testing
type MockIdentityService struct {
	cert        *domain.Certificate
	trustBundle *domain.TrustBundle
	shouldError bool
}

func (m *MockIdentityService) GetCertificate() (*domain.Certificate, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	return m.cert, nil
}

func (m *MockIdentityService) GetTrustBundle() (*domain.TrustBundle, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	return m.trustBundle, nil
}

// createTestCertificate creates a test X.509 certificate with SPIFFE URI SAN for testing
func createTestCertificate() (*x509.Certificate, interface{}, error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create SPIFFE URI for the certificate
	spiffeURI, err := url.Parse("spiffe://example.org/test-service")
	if err != nil {
		return nil, nil, err
	}

	// Create certificate template with SPIFFE URI SAN
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Org"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  nil,
		URIs:         []*url.URL{spiffeURI}, // Add SPIFFE URI SAN
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	return cert, privateKey, nil
}

func TestClientConnection_CreateSVIDSource(t *testing.T) {
	t.Run("successful SVID source creation", func(t *testing.T) {
		// Create test certificate and private key
		testCert, privateKey, err := createTestCertificate()
		require.NoError(t, err)

		// Create test domain certificate
		domainCert := &domain.Certificate{
			Cert:       testCert,
			PrivateKey: privateKey,
			Chain:      []*x509.Certificate{},
		}

		// Create mock identity service
		mockIdentityService := &MockIdentityService{
			cert:        domainCert,
			shouldError: false,
		}

		// Create client connection with mock identity service
		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		// Test SVID source creation
		svidSource, err := clientConn.createSVIDSource()
		
		// Verify successful creation
		assert.NoError(t, err)
		assert.NotNil(t, svidSource)

		// Test getting SVID from source
		svid, err := svidSource.GetX509SVID()
		assert.NoError(t, err)
		assert.NotNil(t, svid)
		assert.Equal(t, "spiffe://example.org/test-service", svid.ID.String())
		assert.Equal(t, testCert, svid.Certificates[0])
		assert.Equal(t, privateKey, svid.PrivateKey)
	})

	t.Run("handles missing identity service", func(t *testing.T) {
		clientConn := &ClientConnection{
			identityService: nil,
		}

		svidSource, err := clientConn.createSVIDSource()
		
		assert.Error(t, err)
		assert.Nil(t, svidSource)
		assert.Contains(t, err.Error(), "no identity service available")
	})

	t.Run("handles certificate retrieval error", func(t *testing.T) {
		mockIdentityService := &MockIdentityService{
			shouldError: true,
		}

		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		svidSource, err := clientConn.createSVIDSource()
		require.NoError(t, err)

		_, err = svidSource.GetX509SVID()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get certificate")
	})
}

func TestClientConnection_CreateBundleSource(t *testing.T) {
	t.Run("successful bundle source creation", func(t *testing.T) {
		testCert, _, err := createTestCertificate()
		require.NoError(t, err)

		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{testCert},
		}

		mockIdentityService := &MockIdentityService{
			trustBundle: trustBundle,
			shouldError: false,
		}

		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		bundleSource, err := clientConn.createBundleSource()
		
		assert.NoError(t, err)
		assert.NotNil(t, bundleSource)

		// Test getting bundle from source
		trustDomain := mustParseTrustDomain("example.org")
		bundle, err := bundleSource.GetX509BundleForTrustDomain(trustDomain)
		assert.NoError(t, err)
		assert.NotNil(t, bundle)
		assert.Equal(t, trustDomain, bundle.TrustDomain())
	})

	t.Run("handles empty trust bundle", func(t *testing.T) {
		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{},
		}

		mockIdentityService := &MockIdentityService{
			trustBundle: trustBundle,
			shouldError: false,
		}

		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		bundleSource, err := clientConn.createBundleSource()
		require.NoError(t, err)

		trustDomain := mustParseTrustDomain("example.org")
		_, err = bundleSource.GetX509BundleForTrustDomain(trustDomain)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle is empty")
	})
}

func TestClientConnection_SPIFFEAdapters(t *testing.T) {
	t.Run("SVID and bundle sources work together", func(t *testing.T) {
		// Create test certificate and private key
		testCert, privateKey, err := createTestCertificate()
		require.NoError(t, err)

		// Create test domain certificate
		domainCert := &domain.Certificate{
			Cert:       testCert,
			PrivateKey: privateKey,
			Chain:      []*x509.Certificate{},
		}

		// Create test trust bundle
		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{testCert},
		}

		// Create mock identity service
		mockIdentityService := &MockIdentityService{
			cert:        domainCert,
			trustBundle: trustBundle,
			shouldError: false,
		}

		// Create client connection with mock identity service
		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		// Test that both adapters can be created successfully
		svidSource, err := clientConn.createSVIDSource()
		assert.NoError(t, err)
		assert.NotNil(t, svidSource)

		bundleSource, err := clientConn.createBundleSource()
		assert.NoError(t, err)
		assert.NotNil(t, bundleSource)

		// Verify they work together - get SVID and validate against bundle
		svid, err := svidSource.GetX509SVID()
		assert.NoError(t, err)
		assert.Equal(t, "spiffe://example.org/test-service", svid.ID.String())

		trustDomain := mustParseTrustDomain("example.org")
		bundle, err := bundleSource.GetX509BundleForTrustDomain(trustDomain)
		assert.NoError(t, err)
		assert.NotNil(t, bundle)
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