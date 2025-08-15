package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// MockIdentityService implements CertificateProvider for testing

// createTestCertificate creates a test X.509 certificate for testing
func createTestCertificate() (*x509.Certificate, interface{}, error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create certificate template
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
		URIs:         nil,
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

func TestClientConnection_ExtractSPIFFEConfig(t *testing.T) {
	t.Run("successful SPIFFE certificate extraction", func(t *testing.T) {
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

		// Test SPIFFE config extraction
		spiffeConfig, err := clientConn.extractSPIFFEConfig()
		
		// Verify successful extraction
		assert.NoError(t, err)
		assert.NotNil(t, spiffeConfig)
		assert.NotEmpty(t, spiffeConfig.ClientCertificates)
		assert.NotNil(t, spiffeConfig.RootCAs)
		assert.NotNil(t, spiffeConfig.VerifyPeerCertificate)
		assert.NotNil(t, spiffeConfig.VerifyConnection)

		// Verify TLS certificate conversion
		assert.Len(t, spiffeConfig.ClientCertificates, 1)
		tlsCert := spiffeConfig.ClientCertificates[0]
		assert.NotEmpty(t, tlsCert.Certificate)
		assert.NotNil(t, tlsCert.PrivateKey)
		assert.Equal(t, testCert, tlsCert.Leaf)
	})

	t.Run("handles missing identity service", func(t *testing.T) {
		clientConn := &ClientConnection{
			identityService: nil,
		}

		spiffeConfig, err := clientConn.extractSPIFFEConfig()
		
		assert.Error(t, err)
		assert.Nil(t, spiffeConfig)
		assert.Contains(t, err.Error(), "no identity service available")
	})

	t.Run("handles certificate retrieval error", func(t *testing.T) {
		mockIdentityService := &MockIdentityService{
			shouldError: true,
		}

		clientConn := &ClientConnection{
			identityService: mockIdentityService,
		}

		spiffeConfig, err := clientConn.extractSPIFFEConfig()
		
		assert.Error(t, err)
		assert.Nil(t, spiffeConfig)
		assert.Contains(t, err.Error(), "failed to get SPIFFE certificate")
	})
}

func TestClientConnection_ConvertToTLSCertificates(t *testing.T) {
	t.Run("successful conversion", func(t *testing.T) {
		testCert, privateKey, err := createTestCertificate()
		require.NoError(t, err)

		domainCert := &domain.Certificate{
			Cert:       testCert,
			PrivateKey: privateKey,
			Chain:      []*x509.Certificate{},
		}

		clientConn := &ClientConnection{}
		tlsCerts, err := clientConn.convertToTLSCertificates(domainCert)
		
		assert.NoError(t, err)
		assert.Len(t, tlsCerts, 1)
		assert.Equal(t, testCert, tlsCerts[0].Leaf)
		assert.Equal(t, privateKey, tlsCerts[0].PrivateKey)
	})

	t.Run("handles nil certificate", func(t *testing.T) {
		clientConn := &ClientConnection{}
		tlsCerts, err := clientConn.convertToTLSCertificates(nil)
		
		assert.Error(t, err)
		assert.Nil(t, tlsCerts)
		assert.Contains(t, err.Error(), "certificate is nil")
	})
}

func TestClientConnection_ConvertToX509CertPool(t *testing.T) {
	t.Run("successful conversion", func(t *testing.T) {
		testCert, _, err := createTestCertificate()
		require.NoError(t, err)

		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{testCert},
		}

		clientConn := &ClientConnection{}
		certPool, err := clientConn.convertToX509CertPool(trustBundle)
		
		assert.NoError(t, err)
		assert.NotNil(t, certPool)
		
		// Verify the certificate is in the pool
		subjects := certPool.Subjects()
		assert.Len(t, subjects, 1)
	})

	t.Run("handles empty trust bundle", func(t *testing.T) {
		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{},
		}

		clientConn := &ClientConnection{}
		certPool, err := clientConn.convertToX509CertPool(trustBundle)
		
		assert.Error(t, err)
		assert.Nil(t, certPool)
		assert.Contains(t, err.Error(), "trust bundle is empty")
	})
}

func TestClientConnection_ConnectionVerifiers(t *testing.T) {
	t.Run("peer certificate verifier", func(t *testing.T) {
		testCert, _, err := createTestCertificate()
		require.NoError(t, err)

		trustBundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{testCert},
		}

		clientConn := &ClientConnection{}
		verifier := clientConn.createPeerCertificateVerifier(trustBundle)
		
		// Test with valid certificate
		err = verifier([][]byte{testCert.Raw}, [][]*x509.Certificate{})
		assert.NoError(t, err)

		// Test with no certificates
		err = verifier([][]byte{}, [][]*x509.Certificate{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no peer certificates provided")
	})

	t.Run("connection verifier", func(t *testing.T) {
		testCert, _, err := createTestCertificate()
		require.NoError(t, err)

		clientConn := &ClientConnection{}
		verifier := clientConn.createConnectionVerifier()

		// Test with valid connection state
		state := tls.ConnectionState{
			Version:          tls.VersionTLS12,
			PeerCertificates: []*x509.Certificate{testCert},
		}
		err = verifier(state)
		assert.NoError(t, err)

		// Test with no peer certificates
		state = tls.ConnectionState{
			Version:          tls.VersionTLS12,
			PeerCertificates: []*x509.Certificate{},
		}
		err = verifier(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no peer certificates in connection")

		// Test with insecure TLS version
		state = tls.ConnectionState{
			Version:          tls.VersionTLS10,
			PeerCertificates: []*x509.Certificate{testCert},
		}
		err = verifier(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insecure TLS version")
	})
}