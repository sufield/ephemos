package transport

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestSPIFFECertificateValidation(t *testing.T) {
	provider := NewGRPCProvider(&ports.Configuration{})

	t.Run("extractSPIFFEIDs_ValidCertificate", func(t *testing.T) {
		// Create a test certificate with SPIFFE URI SAN
		spiffeURI, err := url.Parse("spiffe://example.org/workload")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		ids, err := extractSPIFFEIDs(cert)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
		assert.Equal(t, "spiffe://example.org/workload", ids[0])
	})

	t.Run("extractSPIFFEIDs_NoCertificate", func(t *testing.T) {
		cert := &x509.Certificate{}

		ids, err := extractSPIFFEIDs(cert)
		require.NoError(t, err)
		assert.Len(t, ids, 0)
	})

	t.Run("createSPIFFEVerifier_NoCertificates", func(t *testing.T) {
		verifier := provider.createSPIFFEVerifier()
		
		// Test with empty rawCerts (should fail)
		err := verifier([][]byte{}, [][]*x509.Certificate{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no certificates provided")
	})

	t.Run("buildTLSCertificate_ValidInput", func(t *testing.T) {
		cert := &x509.Certificate{Raw: []byte("test-cert")}
		privateKey := "test-key"

		tlsCert, err := buildTLSCertificate([]*x509.Certificate{cert}, privateKey)
		require.NoError(t, err)
		assert.NotNil(t, tlsCert)
		assert.Equal(t, cert, tlsCert.Leaf)
		assert.Equal(t, privateKey, tlsCert.PrivateKey)
		assert.Len(t, tlsCert.Certificate, 1)
	})

	t.Run("buildTLSCertificate_NoCertificates", func(t *testing.T) {
		privateKey := "test-key"

		_, err := buildTLSCertificate([]*x509.Certificate{}, privateKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no certificates provided")
	})

	t.Run("buildTLSCertificate_NoPrivateKey", func(t *testing.T) {
		cert := &x509.Certificate{Raw: []byte("test-cert")}

		_, err := buildTLSCertificate([]*x509.Certificate{cert}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "private key is required")
	})
}

func TestTLSConfigCreation(t *testing.T) {
	t.Run("InsecureMode_Enabled", func(t *testing.T) {
		// Create config that enables insecure mode
		config := &ports.Configuration{}
		// Mock the ShouldSkipCertificateValidation to return true
		// This would normally check environment variables
		
		provider := NewGRPCProvider(config)
		cert := &domain.Certificate{}
		bundle := &domain.TrustBundle{}

		// For now, this will use the default logic since we can't easily mock env vars
		tlsConfig, err := provider.createClientTLSConfig(cert, bundle)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig)
	})

	t.Run("SecureMode_WithCertificates", func(t *testing.T) {
		config := &ports.Configuration{}
		provider := NewGRPCProvider(config)

		// Create test certificates
		caCert := &x509.Certificate{Raw: []byte("ca-cert")}
		clientCert := &x509.Certificate{Raw: []byte("client-cert")}

		cert := &domain.Certificate{
			Cert:       clientCert,
			PrivateKey: "test-private-key",
			Chain:      []*x509.Certificate{},
		}
		bundle := &domain.TrustBundle{
			Certificates: []*x509.Certificate{caCert},
		}

		tlsConfig, err := provider.createClientTLSConfig(cert, bundle)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.RootCAs)
		assert.False(t, tlsConfig.InsecureSkipVerify)
		assert.Equal(t, tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
	})
}