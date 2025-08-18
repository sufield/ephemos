package domain_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

func TestNewIdentityDocument(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) ([]*x509.Certificate, *ecdsa.PrivateKey, *x509.Certificate)
		wantErr     bool
		errContains string
	}{
		{
			name: "valid identity document creation without CA validation",
			setupFunc: func(t *testing.T) ([]*x509.Certificate, *ecdsa.PrivateKey, *x509.Certificate) {
				cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
				return []*x509.Certificate{cert}, key, nil // No CA validation
			},
			wantErr: false,
		},
		{
			name: "empty certificate chain",
			setupFunc: func(t *testing.T) ([]*x509.Certificate, *ecdsa.PrivateKey, *x509.Certificate) {
				_, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
				caCert, _ := createTestCACertificate(t)
				return []*x509.Certificate{}, key, caCert
			},
			wantErr:     true,
			errContains: "certificate chain cannot be empty",
		},
		{
			name: "nil private key",
			setupFunc: func(t *testing.T) ([]*x509.Certificate, *ecdsa.PrivateKey, *x509.Certificate) {
				cert, _ := createValidTestCertificate(t, "spiffe://example.org/test-service")
				caCert, _ := createTestCACertificate(t)
				return []*x509.Certificate{cert}, nil, caCert
			},
			wantErr:     true,
			errContains: "private key cannot be nil",
		},
		{
			name: "certificate chain with intermediates",
			setupFunc: func(t *testing.T) ([]*x509.Certificate, *ecdsa.PrivateKey, *x509.Certificate) {
				cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
				intermediate := createTestIntermediateCertificate(t)
				caCert, _ := createTestCACertificate(t)
				return []*x509.Certificate{cert, intermediate}, key, caCert
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certChain, key, ca := tt.setupFunc(t)

			doc, err := domain.NewIdentityDocument(certChain, key, ca)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, doc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, doc)
				
				// Verify the document has expected properties
				assert.NotNil(t, doc.GetCertificate())
				assert.NotNil(t, doc.GetPrivateKey())
				assert.Equal(t, len(certChain), len(doc.GetCertificateChain()))
			}
		})
	}
}

func TestIdentityDocument_GetIdentityNamespace(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/payment-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	namespace, err := doc.GetIdentityNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "spiffe://example.org/payment-service", namespace.String())
}

func TestIdentityDocument_GetTrustDomain(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/payment-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	trustDomain, err := doc.GetTrustDomain()
	assert.NoError(t, err)
	assert.Equal(t, "example.org", trustDomain.String())
}

func TestIdentityDocument_IsExpired(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	// Should not be expired now
	assert.False(t, doc.IsExpired(time.Now()))
	
	// Should be expired in the future
	futureTime := time.Now().Add(48 * time.Hour) // Certificate is valid for 24 hours
	assert.True(t, doc.IsExpired(futureTime))
}

func TestIdentityDocument_IsExpiringSoon(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	// Should not be expiring soon (threshold of 1 hour)
	assert.False(t, doc.IsExpiringSoon(1*time.Hour))
	
	// Should be expiring soon (threshold of 25 hours, cert valid for 24 hours)
	assert.True(t, doc.IsExpiringSoon(25*time.Hour))
}

func TestIdentityDocument_IsValid(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	// Should be valid now
	assert.True(t, doc.IsValid(time.Now()))
	
	// Should not be valid in the future
	futureTime := time.Now().Add(48 * time.Hour)
	assert.False(t, doc.IsValid(futureTime))
	
	// Should not be valid in the past (before NotBefore)
	pastTime := time.Now().Add(-1 * time.Hour)
	assert.False(t, doc.IsValid(pastTime))
}

func TestIdentityDocument_ValidateAgainstBundle(t *testing.T) {
	// Create a properly signed certificate chain for this test
	caCert, caKey := createTestCACertificate(t)
	cert, key := createValidTestCertificateSignedBy(t, "spiffe://example.org/test-service", caCert, caKey)
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	// Create a trust bundle with the CA
	bundle, err := domain.NewTrustBundle([]*x509.Certificate{caCert})
	require.NoError(t, err)
	
	t.Run("valid bundle validation", func(t *testing.T) {
		err := doc.ValidateAgainstBundle(bundle)
		assert.NoError(t, err)
	})
	
	t.Run("nil bundle validation", func(t *testing.T) {
		err := doc.ValidateAgainstBundle(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle cannot be nil")
	})
}

func TestIdentityDocument_Validate(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	err = doc.Validate()
	assert.NoError(t, err)
}

func TestIdentityDocument_GetServiceIdentity(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/payment-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	serviceIdentity, err := doc.GetServiceIdentity()
	assert.NoError(t, err)
	assert.NotNil(t, serviceIdentity)
	assert.Equal(t, "payment-service", serviceIdentity.Name())
	assert.Equal(t, "example.org", serviceIdentity.Domain())
}

func TestIdentityDocument_TimeUntilExpiry(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	timeUntil := doc.TimeUntilExpiry()
	assert.True(t, timeUntil > 0)
	assert.True(t, timeUntil < 24*time.Hour+time.Minute) // Should be less than 24 hours + 1 minute
}

func TestIdentityDocument_SupportsKeyType(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	// Should support ECDSA keys
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	assert.True(t, doc.SupportsKeyType(ecdsaKey))
	
	// Should return requirements
	assert.True(t, doc.RequiresPrivateKey())
}

func TestNewIdentityDocumentFromCertificate(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	// Create a domain Certificate first
	domainCert, err := domain.NewCertificate(cert, key, nil)
	require.NoError(t, err)
	
	doc, err := domain.NewIdentityDocumentFromCertificate(domainCert)
	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, domainCert, doc.GetCertificate())
}

func TestNewIdentityDocumentFromConfig(t *testing.T) {
	cert, key := createValidTestCertificate(t, "spiffe://example.org/test-service")
	
	config := domain.IdentityDocumentConfig{
		CertChain:  []*x509.Certificate{cert},
		PrivateKey: key,
		CACert:     nil, // No CA validation for this test
		Subject:    "Test Subject",
		Issuer:     "Test Issuer",
	}
	
	doc, err := domain.NewIdentityDocumentFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, doc)
}

// Helper functions

func createValidTestCertificate(t *testing.T, spiffeID string) (*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	
	// Add SPIFFE URI SAN
	if spiffeID != "" {
		spiffeURI, err := url.Parse(spiffeID)
		require.NoError(t, err)
		template.URIs = []*url.URL{spiffeURI}
	}
	
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	return cert, key
}

func createValidTestCertificateSignedBy(t *testing.T, spiffeID string, caCert *x509.Certificate, caKey interface{}) (*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2), // Different serial from CA
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	
	// Add SPIFFE URI SAN
	if spiffeID != "" {
		spiffeURI, err := url.Parse(spiffeID)
		require.NoError(t, err)
		template.URIs = []*url.URL{spiffeURI}
	}
	
	// Sign with CA certificate and key
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	return cert, key
}

func createTestIntermediateCertificate(t *testing.T) *x509.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Intermediate CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	return cert
}