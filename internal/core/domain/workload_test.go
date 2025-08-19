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

func TestNewWorkload(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	tests := []struct {
		name        string
		config      domain.WorkloadConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid workload creation",
			config: domain.WorkloadConfig{
				ID:          "workload-1",
				Identity:    identity,
				TrustDomain: trustDomain,
				Status:      domain.WorkloadStatusActive,
			},
			wantErr: false,
		},
		{
			name: "workload with identity document",
			config: domain.WorkloadConfig{
				ID:          "workload-2",
				Identity:    identity,
				TrustDomain: trustDomain,
				IdentityDoc: createTestIdentityDocument(t, "spiffe://example.org/payment-service"),
				Status:      domain.WorkloadStatusActive,
			},
			wantErr: false,
		},
		{
			name: "empty workload ID",
			config: domain.WorkloadConfig{
				Identity:    identity,
				TrustDomain: trustDomain,
			},
			wantErr:     true,
			errContains: "workload ID cannot be empty",
		},
		{
			name: "empty identity namespace",
			config: domain.WorkloadConfig{
				ID:          "workload-1",
				TrustDomain: trustDomain,
			},
			wantErr:     true,
			errContains: "workload identity namespace cannot be empty",
		},
		{
			name: "empty trust domain",
			config: domain.WorkloadConfig{
				ID:       "workload-1",
				Identity: identity,
			},
			wantErr:     true,
			errContains: "workload trust domain cannot be empty",
		},
		{
			name: "mismatched trust domains",
			config: domain.WorkloadConfig{
				ID:          "workload-1",
				Identity:    identity,
				TrustDomain: domain.MustNewTrustDomain("different.org"),
			},
			wantErr:     true,
			errContains: "does not match identity namespace trust domain",
		},
		{
			name: "mismatched identity document",
			config: domain.WorkloadConfig{
				ID:          "workload-1",
				Identity:    identity,
				TrustDomain: trustDomain,
				IdentityDoc: createTestIdentityDocument(t, "spiffe://different.org/different-service"),
			},
			wantErr:     true,
			errContains: "identity document SPIFFE ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workload, err := domain.NewWorkload(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, workload)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workload)
				assert.Equal(t, tt.config.ID, workload.ID())
				assert.True(t, workload.Identity().Equals(tt.config.Identity))
				assert.True(t, workload.TrustDomain().Equals(tt.config.TrustDomain))
			}
		})
	}
}

func TestWorkload_UpdateIdentityDocument(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
		Status:      domain.WorkloadStatusActive,
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	t.Run("valid identity document update", func(t *testing.T) {
		doc := createTestIdentityDocument(t, "spiffe://example.org/payment-service")

		err := workload.UpdateIdentityDocument(doc)
		assert.NoError(t, err)
		assert.Equal(t, doc, workload.IdentityDocument())
	})

	t.Run("mismatched identity document", func(t *testing.T) {
		doc := createTestIdentityDocument(t, "spiffe://example.org/different-service")

		err := workload.UpdateIdentityDocument(doc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "identity document SPIFFE ID")
	})

	t.Run("nil identity document", func(t *testing.T) {
		err := workload.UpdateIdentityDocument(nil)
		assert.NoError(t, err)
		assert.Nil(t, workload.IdentityDocument())
	})
}

func TestWorkload_UpdateTrustBundle(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
		Status:      domain.WorkloadStatusActive,
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	t.Run("valid trust bundle update", func(t *testing.T) {
		caCert := createWorkloadTestCACertificate(t)
		bundle, err := domain.NewTrustBundle([]*x509.Certificate{caCert})
		require.NoError(t, err)

		err = workload.UpdateTrustBundle(bundle)
		assert.NoError(t, err)
		assert.Equal(t, bundle, workload.TrustBundle())
	})

	t.Run("nil trust bundle", func(t *testing.T) {
		err := workload.UpdateTrustBundle(nil)
		assert.NoError(t, err)
		assert.Nil(t, workload.TrustBundle())
	})
}

func TestWorkload_StatusManagement(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
		Status:      domain.WorkloadStatusPending,
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	assert.Equal(t, domain.WorkloadStatusPending, workload.Status())
	assert.False(t, workload.IsActive())

	workload.UpdateStatus(domain.WorkloadStatusActive)
	assert.Equal(t, domain.WorkloadStatusActive, workload.Status())
	assert.True(t, workload.IsActive())
}

func TestWorkload_Labels(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
		Labels:      map[string]string{"env": "prod", "team": "payments"},
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	labels := workload.Labels()
	assert.Equal(t, "prod", labels["env"])
	assert.Equal(t, "payments", labels["team"])

	// Test adding label
	workload.AddLabel("version", "v1.2.3")
	labels = workload.Labels()
	assert.Equal(t, "v1.2.3", labels["version"])

	// Test removing label
	workload.RemoveLabel("env")
	labels = workload.Labels()
	_, exists := labels["env"]
	assert.False(t, exists)
}

func TestWorkload_HasValidIdentity(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	// No identity document
	assert.False(t, workload.HasValidIdentity())

	// Valid identity document
	doc := createTestIdentityDocument(t, "spiffe://example.org/payment-service")
	err = workload.UpdateIdentityDocument(doc)
	require.NoError(t, err)
	assert.True(t, workload.HasValidIdentity())
}

func TestWorkload_Validate(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	t.Run("valid workload", func(t *testing.T) {
		config := domain.WorkloadConfig{
			ID:          "workload-1",
			Identity:    identity,
			TrustDomain: trustDomain,
			IdentityDoc: createTestIdentityDocument(t, "spiffe://example.org/payment-service"),
			Status:      domain.WorkloadStatusActive,
		}

		workload, err := domain.NewWorkload(config)
		require.NoError(t, err)

		err = workload.Validate()
		assert.NoError(t, err)
	})
}

func TestWorkload_GetServiceName(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	identity := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

	config := domain.WorkloadConfig{
		ID:          "workload-1",
		Identity:    identity,
		TrustDomain: trustDomain,
	}

	workload, err := domain.NewWorkload(config)
	require.NoError(t, err)

	assert.Equal(t, "payment-service", workload.GetServiceName())
}

// Helper functions

func createTestIdentityDocument(t *testing.T, spiffeID string) *domain.IdentityDocument {
	cert, key := createTestCertificateForSPIFFEID(t, spiffeID)

	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	return doc
}

func createTestCertificateForSPIFFEID(t *testing.T, spiffeID string) (*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Add SPIFFE URI SAN
	if spiffeID != "" {
		template.URIs = []*url.URL{{Scheme: "spiffe", Host: "example.org", Path: "/payment-service"}}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert, key
}

func createWorkloadTestCACertificate(t *testing.T) *x509.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
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

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}
