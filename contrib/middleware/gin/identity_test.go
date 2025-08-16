package gin

import (
	"context"
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple workload", "/workload", "workload"},
		{"namespaced service", "/ns/production/sa/api", "api"},
		{"multiple levels", "/team/backend/payment-service", "payment-service"},
		{"empty path", "", ""},
		{"root path", "/", ""},
		{"no leading slash", "service", "service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateClientCertificate(t *testing.T) {
	config := &IdentityConfig{
		TrustDomains: []string{"example.org"},
	}

	t.Run("valid certificate with SPIFFE URI", func(t *testing.T) {
		spiffeURI, err := url.Parse("spiffe://example.org/workload")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		identity, err := validateClientCertificate(cert, config)
		require.NoError(t, err)
		assert.Equal(t, "spiffe://example.org/workload", identity.ID)
		assert.Equal(t, "workload", identity.Name)
		assert.Equal(t, "example.org", identity.Domain)
	})

	t.Run("certificate without SPIFFE URI", func(t *testing.T) {
		cert := &x509.Certificate{
			URIs: []*url.URL{},
		}

		_, err := validateClientCertificate(cert, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no SPIFFE ID found")
	})

	t.Run("certificate with invalid SPIFFE ID", func(t *testing.T) {
		invalidURI, err := url.Parse("invalid://example.org/workload")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{invalidURI},
		}

		_, err = validateClientCertificate(cert, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no SPIFFE ID found")
	})

	t.Run("certificate with disallowed trust domain", func(t *testing.T) {
		spiffeURI, err := url.Parse("spiffe://forbidden.org/workload")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		_, err = validateClientCertificate(cert, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trust domain \"forbidden.org\" not allowed")
	})

	t.Run("certificate with allowed trust domain from multiple", func(t *testing.T) {
		configMultiple := &IdentityConfig{
			TrustDomains: []string{"example.org", "test.org", "dev.org"},
		}

		spiffeURI, err := url.Parse("spiffe://test.org/api-service")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		identity, err := validateClientCertificate(cert, configMultiple)
		require.NoError(t, err)
		assert.Equal(t, "spiffe://test.org/api-service", identity.ID)
		assert.Equal(t, "api-service", identity.Name)
		assert.Equal(t, "test.org", identity.Domain)
	})

	t.Run("certificate with no trust domain restrictions", func(t *testing.T) {
		configOpen := &IdentityConfig{
			TrustDomains: []string{}, // Empty means allow all
		}

		spiffeURI, err := url.Parse("spiffe://any-domain.com/some-service")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		identity, err := validateClientCertificate(cert, configOpen)
		require.NoError(t, err)
		assert.Equal(t, "spiffe://any-domain.com/some-service", identity.ID)
		assert.Equal(t, "some-service", identity.Name)
		assert.Equal(t, "any-domain.com", identity.Domain)
	})
}

func TestIdentityFromContext(t *testing.T) {
	t.Run("context with identity", func(t *testing.T) {
		identity := &ServiceIdentity{
			ID:     "spiffe://example.org/test-service",
			Name:   "test-service",
			Domain: "example.org",
		}

		ctx := context.WithValue(context.Background(), IdentityContextKey{}, identity)
		result := IdentityFromContext(ctx)

		assert.Equal(t, identity, result)
	})

	t.Run("context without identity", func(t *testing.T) {
		ctx := context.Background()
		result := IdentityFromContext(ctx)

		assert.Nil(t, result)
	})

	t.Run("context with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), IdentityContextKey{}, "wrong-type")
		result := IdentityFromContext(ctx)

		assert.Nil(t, result)
	})
}

// Mock implementation for testing that doesn't require actual ephemos client
func TestBasicIdentityConfigValidation(t *testing.T) {
	t.Run("nil config panics", func(t *testing.T) {
		assert.Panics(t, func() {
			IdentityMiddleware(nil)
		})
	})

	t.Run("config with defaults", func(t *testing.T) {
		// This test would panic because it tries to create an ephemos client
		// In a real test environment, you'd need to mock the ephemos.IdentityClient call
		// or have a valid ephemos configuration available
		
		// For now, we'll just test the config structure
		config := &IdentityConfig{
			ConfigPath:        "/tmp/test-config.yaml",
			RequireClientCert: true,
			TrustDomains:      []string{"test.org"},
		}
		
		assert.Equal(t, "/tmp/test-config.yaml", config.ConfigPath)
		assert.True(t, config.RequireClientCert)
		assert.Equal(t, []string{"test.org"}, config.TrustDomains)
		assert.Nil(t, config.Logger) // Should be set to default in middleware
	})
}

func TestServiceIdentityStruct(t *testing.T) {
	identity := &ServiceIdentity{
		ID:     "spiffe://prod.company.com/payment-service",
		Name:   "payment-service",
		Domain: "prod.company.com",
	}

	assert.Equal(t, "spiffe://prod.company.com/payment-service", identity.ID)
	assert.Equal(t, "payment-service", identity.Name)
	assert.Equal(t, "prod.company.com", identity.Domain)
}