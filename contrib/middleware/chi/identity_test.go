package chi

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

	t.Run("untrusted domain", func(t *testing.T) {
		spiffeURI, err := url.Parse("spiffe://untrusted.com/workload")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		_, err = validateClientCertificate(cert, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trust domain")
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("no trust domain restriction", func(t *testing.T) {
		configNoRestriction := &IdentityConfig{
			TrustDomains: []string{}, // No restrictions
		}

		spiffeURI, err := url.Parse("spiffe://any-domain.com/service")
		require.NoError(t, err)

		cert := &x509.Certificate{
			URIs: []*url.URL{spiffeURI},
		}

		identity, err := validateClientCertificate(cert, configNoRestriction)
		require.NoError(t, err)
		assert.Equal(t, "any-domain.com", identity.Domain)
		assert.Equal(t, "service", identity.Name)
	})
}

func TestIdentityFromContext(t *testing.T) {
	t.Run("context with identity", func(t *testing.T) {
		expectedIdentity := &ServiceIdentity{
			ID:     "spiffe://example.org/test-service",
			Name:   "test-service",
			Domain: "example.org",
		}

		ctx := context.WithValue(context.Background(), IdentityContextKey{}, expectedIdentity)
		
		identity := IdentityFromContext(ctx)
		assert.Equal(t, expectedIdentity, identity)
	})

	t.Run("context without identity", func(t *testing.T) {
		ctx := context.Background()
		
		identity := IdentityFromContext(ctx)
		assert.Nil(t, identity)
	})

	t.Run("context with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), IdentityContextKey{}, "not-an-identity")
		
		identity := IdentityFromContext(ctx)
		assert.Nil(t, identity)
	})
}

func TestServiceIdentity(t *testing.T) {
	identity := &ServiceIdentity{
		ID:     "spiffe://example.org/ns/production/sa/payment-service",
		Name:   "payment-service",
		Domain: "example.org",
	}

	assert.Equal(t, "spiffe://example.org/ns/production/sa/payment-service", identity.ID)
	assert.Equal(t, "payment-service", identity.Name)
	assert.Equal(t, "example.org", identity.Domain)
}

func TestIdentityConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		config := &IdentityConfig{
			ConfigPath: "/etc/ephemos/config.yaml",
		}

		// Test that we can work with partial configuration
		assert.Equal(t, "/etc/ephemos/config.yaml", config.ConfigPath)
		assert.False(t, config.RequireClientCert) // Default false
		assert.Empty(t, config.TrustDomains)      // Default empty (allow all)
		assert.Nil(t, config.Logger)             // Default nil (will be set by middleware)
	})

	t.Run("full configuration", func(t *testing.T) {
		config := &IdentityConfig{
			ConfigPath:        "/etc/ephemos/config.yaml",
			RequireClientCert: true,
			TrustDomains:      []string{"example.org", "trusted.com"},
		}

		assert.True(t, config.RequireClientCert)
		assert.Len(t, config.TrustDomains, 2)
		assert.Contains(t, config.TrustDomains, "example.org")
		assert.Contains(t, config.TrustDomains, "trusted.com")
	})
}