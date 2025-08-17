package domain_test

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
	"github.com/sufield/ephemos/internal/core/domain"
)

// TestCertificateValidateWithOptions tests the new Validate method with options
func TestCertificateValidateWithOptions(t *testing.T) {
	// Create a test certificate with SPIFFE ID
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	spiffeURI, err := url.Parse("spiffe://example.com/service")
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		URIs:      []*url.URL{spiffeURI},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	validCert := &domain.Certificate{
		Cert:       cert,
		PrivateKey: key,
	}

	t.Run("basic validation passes", func(t *testing.T) {
		opts := domain.CertValidationOptions{}
		err := validCert.Validate(opts)
		assert.NoError(t, err)
	})

	t.Run("expired certificate fails", func(t *testing.T) {
		expiredTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject: pkix.Name{
				Organization: []string{"Test"},
			},
			NotBefore: time.Now().Add(-2 * time.Hour),
			NotAfter:  time.Now().Add(-time.Hour), // Expired
			URIs:      []*url.URL{spiffeURI},
		}

		expiredCertDER, err := x509.CreateCertificate(rand.Reader, expiredTemplate, expiredTemplate, &key.PublicKey, key)
		require.NoError(t, err)

		expiredCert, err := x509.ParseCertificate(expiredCertDER)
		require.NoError(t, err)

		cert := &domain.Certificate{
			Cert:       expiredCert,
			PrivateKey: key,
		}

		opts := domain.CertValidationOptions{
			SkipExpiry: false,
		}
		err = cert.Validate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("skip expiry check", func(t *testing.T) {
		expiredTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(3),
			Subject: pkix.Name{
				Organization: []string{"Test"},
			},
			NotBefore: time.Now().Add(-2 * time.Hour),
			NotAfter:  time.Now().Add(-time.Hour), // Expired
			URIs:      []*url.URL{spiffeURI},
		}

		expiredCertDER, err := x509.CreateCertificate(rand.Reader, expiredTemplate, expiredTemplate, &key.PublicKey, key)
		require.NoError(t, err)

		expiredCert, err := x509.ParseCertificate(expiredCertDER)
		require.NoError(t, err)

		cert := &domain.Certificate{
			Cert:       expiredCert,
			PrivateKey: key,
		}

		opts := domain.CertValidationOptions{
			SkipExpiry: true, // Skip expiry check
		}
		err = cert.Validate(opts)
		assert.NoError(t, err)
	})

	t.Run("SPIFFE ID matching", func(t *testing.T) {
		expectedIdentity := domain.NewServiceIdentity("service", "example.com")
		
		opts := domain.CertValidationOptions{
			ExpectedIdentity: expectedIdentity,
		}
		err := validCert.Validate(opts)
		assert.NoError(t, err)
	})

	t.Run("SPIFFE ID mismatch", func(t *testing.T) {
		wrongIdentity := domain.NewServiceIdentity("wrong-service", "example.com")
		
		opts := domain.CertValidationOptions{
			ExpectedIdentity: wrongIdentity,
		}
		err := validCert.Validate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SPIFFE ID mismatch")
	})

	t.Run("nil certificate", func(t *testing.T) {
		var nilCert *domain.Certificate
		opts := domain.CertValidationOptions{}
		err := nilCert.Validate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("missing private key", func(t *testing.T) {
		cert := &domain.Certificate{
			Cert:       validCert.Cert,
			PrivateKey: nil,
		}
		opts := domain.CertValidationOptions{}
		err := cert.Validate(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "private key")
	})
}

// TestDefaultCertValidator tests the DefaultCertValidator implementation
func TestDefaultCertValidator(t *testing.T) {
	validator := &domain.DefaultCertValidator{}

	// Create a test certificate
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	spiffeURI, err := url.Parse("spiffe://example.com/service")
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		URIs:      []*url.URL{spiffeURI},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	validCert := &domain.Certificate{
		Cert:       cert,
		PrivateKey: key,
	}

	t.Run("validates valid certificate", func(t *testing.T) {
		opts := domain.CertValidationOptions{}
		err := validator.Validate(validCert, opts)
		assert.NoError(t, err)
	})

	t.Run("rejects nil certificate", func(t *testing.T) {
		opts := domain.CertValidationOptions{}
		err := validator.Validate(nil, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("validates with identity matching", func(t *testing.T) {
		expectedIdentity := domain.NewServiceIdentity("service", "example.com")
		opts := domain.CertValidationOptions{
			ExpectedIdentity: expectedIdentity,
		}
		err := validator.Validate(validCert, opts)
		assert.NoError(t, err)
	})
}

// TestValidateWithEmptyOptions tests that Validate works with empty options
func TestValidateWithEmptyOptions(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	validCert := &domain.Certificate{
		Cert:       cert,
		PrivateKey: key,
	}

	// Test that basic validation with empty options works
	validationErr := validCert.Validate(domain.CertValidationOptions{})
	assert.NoError(t, validationErr)
}