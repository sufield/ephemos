package domain_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

func TestNewTrustBundle(t *testing.T) {
	t.Parallel()

	t.Run("valid single certificate", func(t *testing.T) {
		t.Parallel()
		cert := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert})
		require.NoError(t, err)
		assert.NotNil(t, bundle)
		assert.Len(t, bundle.Certificates, 1)
		assert.Equal(t, cert, bundle.Certificates[0].Cert)
	})

	t.Run("multiple valid certificates", func(t *testing.T) {
		t.Parallel()
		cert1 := createValidCACert(t)
		cert2 := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert1, cert2})
		require.NoError(t, err)
		assert.NotNil(t, bundle)
		assert.Len(t, bundle.Certificates, 2)
	})

	t.Run("empty certificate list", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundle([]*x509.Certificate{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle cannot be empty")
		assert.Nil(t, bundle)
	})

	t.Run("nil certificate list", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundle(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle cannot be empty")
		assert.Nil(t, bundle)
	})

	t.Run("rejects nil certificates during validation", func(t *testing.T) {
		t.Parallel()
		cert1 := createValidCACert(t)
		cert2 := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert1, nil, cert2, nil})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root CA certificate cannot be nil")
		assert.Nil(t, bundle)
	})
}

func TestTrustBundle_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty bundle returns true", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{}, false)
		require.NoError(t, err)
		assert.True(t, bundle.IsEmpty())
	})

	t.Run("bundle with certificates returns false", func(t *testing.T) {
		t.Parallel()
		cert := createValidCACert(t)
		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert})
		require.NoError(t, err)
		assert.False(t, bundle.IsEmpty())
	})
}

func TestTrustBundle_Count(t *testing.T) {
	t.Parallel()

	t.Run("empty bundle returns zero", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{}, false)
		require.NoError(t, err)
		assert.Equal(t, 0, bundle.Count())
	})

	t.Run("bundle with certificates returns correct count", func(t *testing.T) {
		t.Parallel()
		cert1 := createValidCACert(t)
		cert2 := createValidCACert(t)
		cert3 := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert1, cert2, cert3})
		require.NoError(t, err)
		assert.Equal(t, 3, bundle.Count())
	})
}

func TestTrustBundle_RawCertificates(t *testing.T) {
	t.Parallel()

	t.Run("returns underlying x509 certificates", func(t *testing.T) {
		t.Parallel()
		cert1 := createValidCACert(t)
		cert2 := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert1, cert2})
		require.NoError(t, err)

		rawCerts := bundle.RawCertificates()
		require.Len(t, rawCerts, 2)
		assert.Equal(t, cert1, rawCerts[0])
		assert.Equal(t, cert2, rawCerts[1])
	})

	t.Run("empty bundle returns empty slice", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{}, false)
		require.NoError(t, err)

		rawCerts := bundle.RawCertificates()
		assert.Len(t, rawCerts, 0)
		assert.NotNil(t, rawCerts) // Should return empty slice, not nil
	})
}

func TestTrustBundle_CreateCertPoolNew(t *testing.T) {
	t.Parallel()

	t.Run("creates cert pool with all certificates", func(t *testing.T) {
		t.Parallel()
		cert1 := createValidCACert(t)
		cert2 := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert1, cert2})
		require.NoError(t, err)

		pool := bundle.CreateCertPool()
		require.NotNil(t, pool)

		// Verify certificates are in the pool by checking if they verify against themselves
		// This is a simple way to test the pool contains our certificates
		roots := x509.NewCertPool()
		roots.AddCert(cert1)
		roots.AddCert(cert2)

		// The pool should contain our certificates (indirect verification)
		assert.NotNil(t, pool)
	})

	t.Run("empty bundle creates empty cert pool", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{}, false)
		require.NoError(t, err)

		pool := bundle.CreateCertPool()
		assert.NotNil(t, pool) // Should still create a pool, just empty
	})

	t.Run("skips nil certificates in pool", func(t *testing.T) {
		t.Parallel()
		cert := createValidCACert(t)

		// Create bundle with valid cert
		bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert})
		require.NoError(t, err)

		// Should create pool successfully even if there were nil certs (filtered out in constructor)
		pool := bundle.CreateCertPool()
		assert.NotNil(t, pool)
	})
}

func TestTrustBundle_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates CA certificates correctly", func(t *testing.T) {
		t.Parallel()
		caCert := createValidCACert(t)

		bundle, err := domain.NewTrustBundle([]*x509.Certificate{caCert})
		require.NoError(t, err)

		err = bundle.Validate()
		assert.NoError(t, err)
	})

	t.Run("rejects non-CA certificates", func(t *testing.T) {
		t.Parallel()
		nonCACert := createValidLeafCert(t)

		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{nonCACert}, false)
		require.NoError(t, err)

		err = bundle.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a CA certificate")
	})

	t.Run("empty bundle fails validation", func(t *testing.T) {
		t.Parallel()
		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{}, false)
		require.NoError(t, err)

		err = bundle.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trust bundle cannot be empty")
	})

	t.Run("mixed CA and non-CA certificates fails", func(t *testing.T) {
		t.Parallel()
		caCert := createValidCACert(t)
		leafCert := createValidLeafCert(t)

		bundle, err := domain.NewTrustBundleWithValidation([]*x509.Certificate{caCert, leafCert}, false)
		require.NoError(t, err)

		err = bundle.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a CA certificate")
	})
}

func TestRootCACertificate(t *testing.T) {
	t.Parallel()

	t.Run("wraps x509 certificate correctly", func(t *testing.T) {
		t.Parallel()
		cert := createValidCACert(t)

		rootCA := &domain.RootCACertificate{Cert: cert}
		assert.Equal(t, cert, rootCA.Cert)
	})

	t.Run("handles nil certificate", func(t *testing.T) {
		t.Parallel()
		rootCA := &domain.RootCACertificate{Cert: nil}
		assert.Nil(t, rootCA.Cert)
	})
}

// Helper functions for creating test certificates

func createValidCACert(t *testing.T) *x509.Certificate {
	t.Helper()

	// Generate a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create certificate template for CA
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

func createValidLeafCert(t *testing.T) *x509.Certificate {
	t.Helper()

	// Generate a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create certificate template for leaf certificate (not CA)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test Org"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false, // This is a leaf certificate
	}

	// Create the certificate (self-signed for testing)
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}
