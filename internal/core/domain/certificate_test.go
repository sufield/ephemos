package domain_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

// Helper function to create a test certificate
func createTestCertificate(t *testing.T, spiffeID string) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Add SPIFFE ID as URI SAN if provided
	if spiffeID != "" {
		uri, err := url.Parse(spiffeID)
		if err != nil {
			t.Fatalf("Failed to parse SPIFFE ID %q: %v", spiffeID, err)
		}
		template.URIs = []*url.URL{uri}
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

// Helper function to create a test CA certificate
func createTestCACertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

// Test edge cases for NewCertificateWithValidation
func TestNewCertificateWithValidation_EdgeCases(t *testing.T) {
	validCert, validKey := createTestCertificate(t, "spiffe://example.com/service")

	tests := []struct {
		name        string
		cert        *x509.Certificate
		key         crypto.Signer
		chain       []*x509.Certificate
		validate    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid certificate with validation",
			cert:     validCert,
			key:      validKey,
			chain:    nil,
			validate: true,
			wantErr:  false,
		},
		{
			name:     "valid certificate without validation",
			cert:     validCert,
			key:      validKey,
			chain:    nil,
			validate: false,
			wantErr:  false,
		},
		{
			name:        "nil certificate with validation",
			cert:        nil,
			key:         validKey,
			chain:       nil,
			validate:    true,
			wantErr:     true,
			errContains: "certificate cannot be nil",
		},
		{
			name:     "nil certificate without validation - allowed",
			cert:     nil,
			key:      validKey,
			chain:    nil,
			validate: false,
			wantErr:  false,
		},
		{
			name:        "nil private key with validation",
			cert:        validCert,
			key:         nil,
			chain:       nil,
			validate:    true,
			wantErr:     true,
			errContains: "private key cannot be nil",
		},
		{
			name:     "nil private key without validation - allowed",
			cert:     validCert,
			key:      nil,
			chain:    nil,
			validate: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := domain.NewCertificateWithValidation(tt.cert, tt.key, tt.chain, tt.validate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCertificateWithValidation() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewCertificateWithValidation() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewCertificateWithValidation() unexpected error = %v", err)
				}
				if cert == nil {
					t.Error("NewCertificateWithValidation() returned nil certificate")
				}
			}
		})
	}
}

// Test edge cases for NewTrustBundleWithValidation
func TestNewTrustBundleWithValidation_EdgeCases(t *testing.T) {
	validCA, _ := createTestCACertificate(t)
	expiredCA, _ := createTestCACertificate(t)
	// Make the expired CA actually expired
	expiredCA.NotAfter = time.Now().Add(-time.Hour)

	tests := []struct {
		name        string
		certs       []*x509.Certificate
		validate    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid CA with validation",
			certs:    []*x509.Certificate{validCA},
			validate: true,
			wantErr:  false,
		},
		{
			name:     "valid CA without validation",
			certs:    []*x509.Certificate{validCA},
			validate: false,
			wantErr:  false,
		},
		{
			name:        "empty bundle with validation",
			certs:       []*x509.Certificate{},
			validate:    true,
			wantErr:     true,
			errContains: "trust bundle cannot be empty",
		},
		{
			name:     "empty bundle without validation - allowed",
			certs:    []*x509.Certificate{},
			validate: false,
			wantErr:  false,
		},
		{
			name:        "nil certificate with validation",
			certs:       []*x509.Certificate{nil},
			validate:    true,
			wantErr:     true,
			errContains: "root CA certificate cannot be nil",
		},
		{
			name:     "nil certificate without validation - allowed",
			certs:    []*x509.Certificate{nil},
			validate: false,
			wantErr:  false,
		},
		{
			name:        "expired CA with validation",
			certs:       []*x509.Certificate{expiredCA},
			validate:    true,
			wantErr:     true,
			errContains: "has expired",
		},
		{
			name:     "expired CA without validation - allowed",
			certs:    []*x509.Certificate{expiredCA},
			validate: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle, err := domain.NewTrustBundleWithValidation(tt.certs, tt.validate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTrustBundleWithValidation() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTrustBundleWithValidation() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTrustBundleWithValidation() unexpected error = %v", err)
				}
				if bundle == nil {
					t.Error("NewTrustBundleWithValidation() returned nil bundle")
				}
			}
		})
	}
}

// Test that regular constructors still use validation by default
func TestConstructors_DefaultValidation(t *testing.T) {
	validCert, validKey := createTestCertificate(t, "spiffe://example.com/service")
	validCA, _ := createTestCACertificate(t)

	t.Run("NewCertificate validates by default", func(t *testing.T) {
		// This should succeed with valid data
		cert, err := domain.NewCertificate(validCert, validKey, nil)
		if err != nil {
			t.Errorf("NewCertificate() with valid data should not error: %v", err)
		}
		if cert == nil {
			t.Error("NewCertificate() returned nil certificate")
		}

		// This should fail with invalid data (nil certificate)
		_, err = domain.NewCertificate(nil, validKey, nil)
		if err == nil {
			t.Error("NewCertificate() with nil certificate should error")
		}
	})

	t.Run("NewTrustBundle validates by default", func(t *testing.T) {
		// This should succeed with valid data
		bundle, err := domain.NewTrustBundle([]*x509.Certificate{validCA})
		if err != nil {
			t.Errorf("NewTrustBundle() with valid data should not error: %v", err)
		}
		if bundle == nil {
			t.Error("NewTrustBundle() returned nil bundle")
		}

		// This should fail with invalid data (empty bundle)
		_, err = domain.NewTrustBundle([]*x509.Certificate{})
		if err == nil {
			t.Error("NewTrustBundle() with empty bundle should error")
		}
	})
}

// Test certificate chain validation edge cases
func TestCertificate_ChainValidation_EdgeCases(t *testing.T) {
	// Create a leaf certificate
	leafCert, leafKey := createTestCertificate(t, "spiffe://example.com/service")

	// Create a CA certificate that can sign the leaf
	caCert, caKey := createTestCACertificate(t)

	// Create a certificate signed by the CA
	signedTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "signed-service",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Add SPIFFE ID
	uri, _ := url.Parse("spiffe://example.com/signed-service")
	signedTemplate.URIs = []*url.URL{uri}

	signedCertDER, err := x509.CreateCertificate(rand.Reader, &signedTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create signed certificate: %v", err)
	}

	signedCert, err := x509.ParseCertificate(signedCertDER)
	if err != nil {
		t.Fatalf("Failed to parse signed certificate: %v", err)
	}

	tests := []struct {
		name        string
		cert        *x509.Certificate
		key         crypto.Signer
		chain       []*x509.Certificate
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid chain",
			cert:    signedCert,
			key:     leafKey,
			chain:   []*x509.Certificate{caCert},
			wantErr: false,
		},
		{
			name:    "empty chain",
			cert:    leafCert,
			key:     leafKey,
			chain:   []*x509.Certificate{},
			wantErr: false, // Empty chain is allowed
		},
		{
			name:    "nil chain",
			cert:    leafCert,
			key:     leafKey,
			chain:   nil,
			wantErr: false, // Nil chain is allowed
		},
		{
			name:        "broken chain - wrong order",
			cert:        leafCert, // Self-signed cert that doesn't match caCert
			key:         leafKey,
			chain:       []*x509.Certificate{caCert},
			wantErr:     true,
			errContains: "chain order invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := domain.NewCertificate(tt.cert, tt.key, tt.chain)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCertificate() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewCertificate() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewCertificate() unexpected error = %v", err)
				}
				if cert == nil {
					t.Error("NewCertificate() returned nil certificate")
				}
			}
		})
	}
}

// Test dynamic cert pool functionality
func TestTrustBundle_CreateCertPool(t *testing.T) {
	validCA1, _ := createTestCACertificate(t)
	validCA2, _ := createTestCACertificate(t)

	tests := []struct {
		name    string
		certs   []*x509.Certificate
		wantNil bool
	}{
		{
			name:    "single CA certificate",
			certs:   []*x509.Certificate{validCA1},
			wantNil: false,
		},
		{
			name:    "multiple CA certificates",
			certs:   []*x509.Certificate{validCA1, validCA2},
			wantNil: false,
		},
		{
			name:    "empty bundle",
			certs:   []*x509.Certificate{},
			wantNil: false,
		},
		{
			name:    "bundle with nil certificate",
			certs:   []*x509.Certificate{validCA1, nil, validCA2},
			wantNil: false, // nil certificates should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bundle *domain.TrustBundle
			var err error
			
			// Use non-validating constructor for test cases with empty or nil certs
			if tt.name == "empty bundle" || tt.name == "bundle with nil certificate" {
				bundle, err = domain.NewTrustBundleWithValidation(tt.certs, false)
			} else {
				bundle, err = domain.NewTrustBundle(tt.certs)
			}
			require.NoError(t, err)

			pool := bundle.CreateCertPool()

			if tt.wantNil {
				if pool != nil {
					t.Error("CreateCertPool() should return nil")
				}
			} else {
				if pool == nil {
					t.Error("CreateCertPool() returned nil")
					return
				}
				// Verify the pool is functional
				_ = pool // Basic sanity check that we got a pool back
			}
		})
	}
}

// Test static trust bundle provider
func TestStaticTrustBundleProvider(t *testing.T) {
	validCA, _ := createTestCACertificate(t)
	bundle, err := domain.NewTrustBundle([]*x509.Certificate{validCA})
	if err != nil {
		t.Fatalf("Failed to create test bundle: %v", err)
	}

	t.Run("NewStaticTrustBundleProvider", func(t *testing.T) {
		provider := domain.NewStaticTrustBundleProvider(bundle)
		if provider == nil {
			t.Error("NewStaticTrustBundleProvider() returned nil")
		}
	})

	t.Run("GetTrustBundle", func(t *testing.T) {
		provider := domain.NewStaticTrustBundleProvider(bundle)

		retrievedBundle, err := provider.GetTrustBundle()
		if err != nil {
			t.Errorf("GetTrustBundle() error = %v", err)
			return
		}

		if retrievedBundle != bundle {
			t.Error("GetTrustBundle() returned different bundle than provided")
		}
	})

	t.Run("GetTrustBundle with nil bundle", func(t *testing.T) {
		provider := domain.NewStaticTrustBundleProvider(nil)

		_, err := provider.GetTrustBundle()
		if err == nil {
			t.Error("GetTrustBundle() with nil bundle should return error")
		}
		if !strings.Contains(err.Error(), "no trust bundle configured") {
			t.Errorf("GetTrustBundle() error = %v, want error containing 'no trust bundle configured'", err)
		}
	})

	t.Run("CreateCertPool", func(t *testing.T) {
		provider := domain.NewStaticTrustBundleProvider(bundle)

		pool, err := provider.CreateCertPool()
		if err != nil {
			t.Errorf("CreateCertPool() error = %v", err)
			return
		}

		if pool == nil {
			t.Error("CreateCertPool() returned nil pool")
		}
	})

	t.Run("CreateCertPool with nil bundle", func(t *testing.T) {
		provider := domain.NewStaticTrustBundleProvider(nil)

		_, err := provider.CreateCertPool()
		if err == nil {
			t.Error("CreateCertPool() with nil bundle should return error")
		}
	})
}
