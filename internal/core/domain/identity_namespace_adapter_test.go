package domain_test

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

func TestSPIFFELibraryAdapter_ToGoSPIFFEID(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	tests := []struct {
		name       string
		namespace  domain.IdentityNamespace
		wantErr    bool
		expectedID string
	}{
		{
			name:       "valid simple identity",
			namespace:  domain.MustNewIdentityNamespaceFromString("spiffe://example.org/service"),
			wantErr:    false,
			expectedID: "spiffe://example.org/service",
		},
		{
			name:       "valid complex identity",
			namespace:  domain.MustNewIdentityNamespaceFromString("spiffe://prod.company.com/api/v1/payment"),
			wantErr:    false,
			expectedID: "spiffe://prod.company.com/api/v1/payment",
		},
		{
			name:      "zero namespace",
			namespace: domain.IdentityNamespace{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := adapter.ToGoSPIFFEID(tt.namespace)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id.String())
			}
		})
	}
}

func TestSPIFFELibraryAdapter_FromGoSPIFFEID(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	tests := []struct {
		name        string
		spiffeIDStr string
		wantErr     bool
		expectedNS  string
	}{
		{
			name:        "valid simple SPIFFE ID",
			spiffeIDStr: "spiffe://example.org/service",
			wantErr:     false,
			expectedNS:  "spiffe://example.org/service",
		},
		{
			name:        "valid complex SPIFFE ID",
			spiffeIDStr: "spiffe://prod.company.com/api/v1/payment",
			wantErr:     false,
			expectedNS:  "spiffe://prod.company.com/api/v1/payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go-spiffe ID
			goID, err := spiffeid.FromString(tt.spiffeIDStr)
			require.NoError(t, err)

			// Convert to our domain object
			namespace, err := adapter.FromGoSPIFFEID(goID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedNS, namespace.String())
			}
		})
	}
}

func TestSPIFFELibraryAdapter_TrustDomainConversion(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	t.Run("ToGoSPIFFETrustDomain", func(t *testing.T) {
		trustDomain := domain.MustNewTrustDomain("example.org")

		goTD, err := adapter.ToGoSPIFFETrustDomain(trustDomain)
		assert.NoError(t, err)
		assert.Equal(t, "example.org", goTD.String())

		// Test zero trust domain
		var zeroTD domain.TrustDomain
		_, err = adapter.ToGoSPIFFETrustDomain(zeroTD)
		assert.Error(t, err)
	})

	t.Run("FromGoSPIFFETrustDomain", func(t *testing.T) {
		goTD, err := spiffeid.TrustDomainFromString("example.org")
		require.NoError(t, err)

		trustDomain, err := adapter.FromGoSPIFFETrustDomain(goTD)
		assert.NoError(t, err)
		assert.Equal(t, "example.org", trustDomain.String())
	})
}

func TestSPIFFELibraryAdapter_ValidateWithGoSPIFFE(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	tests := []struct {
		name        string
		namespace   domain.IdentityNamespace
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid namespace",
			namespace: domain.MustNewIdentityNamespaceFromString("spiffe://example.org/service"),
			wantErr:   false,
		},
		{
			name:        "zero namespace",
			namespace:   domain.IdentityNamespace{},
			wantErr:     true,
			errContains: "identity namespace is zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.ValidateWithGoSPIFFE(tt.namespace)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSPIFFELibraryAdapter_CreateIdentityNamespaceFromComponents(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	tests := []struct {
		name        string
		trustDomain string
		path        string
		wantErr     bool
		expectedNS  string
	}{
		{
			name:        "valid components",
			trustDomain: "example.org",
			path:        "/service",
			wantErr:     false,
			expectedNS:  "spiffe://example.org/service",
		},
		{
			name:        "complex path",
			trustDomain: "prod.company.com",
			path:        "/api/v1/payment",
			wantErr:     false,
			expectedNS:  "spiffe://prod.company.com/api/v1/payment",
		},
		{
			name:        "invalid trust domain",
			trustDomain: "invalid..domain",
			path:        "/service",
			wantErr:     true,
		},
		{
			name:        "invalid path",
			trustDomain: "example.org",
			path:        "invalid-path",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, err := adapter.CreateIdentityNamespaceFromComponents(tt.trustDomain, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedNS, namespace.String())
			}
		})
	}
}

func TestSPIFFELibraryAdapter_ServiceIdentityMigration(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	t.Run("MigrateServiceIdentity", func(t *testing.T) {
		// Create a ServiceIdentity
		serviceIdentity := domain.NewServiceIdentity("payment-service", "example.org")

		// Migrate to IdentityNamespace
		namespace, err := adapter.MigrateServiceIdentity(serviceIdentity)
		assert.NoError(t, err)
		assert.Equal(t, serviceIdentity.URI(), namespace.String())

		// Test nil service identity
		_, err = adapter.MigrateServiceIdentity(nil)
		assert.Error(t, err)
	})

	t.Run("CreateServiceIdentityFromNamespace", func(t *testing.T) {
		namespace := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/payment-service")

		serviceIdentity, err := adapter.CreateServiceIdentityFromNamespace(namespace)
		assert.NoError(t, err)
		assert.Equal(t, "payment-service", serviceIdentity.Name())
		assert.Equal(t, "example.org", serviceIdentity.Domain())

		// Test zero namespace
		var zeroNS domain.IdentityNamespace
		_, err = adapter.CreateServiceIdentityFromNamespace(zeroNS)
		assert.Error(t, err)

		// Test root path namespace
		rootNS := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/")
		_, err = adapter.CreateServiceIdentityFromNamespace(rootNS)
		assert.Error(t, err)
	})
}

func TestSPIFFELibraryAdapter_RoundTripConversion(t *testing.T) {
	adapter := domain.NewSPIFFELibraryAdapter()

	originalIDs := []string{
		"spiffe://example.org/service",
		"spiffe://prod.company.com/api/v1/payment",
		"spiffe://localhost/test-service",
	}

	for _, originalID := range originalIDs {
		t.Run(originalID, func(t *testing.T) {
			// Convert to our domain object
			namespace, err := domain.NewIdentityNamespaceFromString(originalID)
			require.NoError(t, err)

			// Convert to go-spiffe ID
			goID, err := adapter.ToGoSPIFFEID(namespace)
			require.NoError(t, err)

			// Convert back to our domain object
			backToNamespace, err := adapter.FromGoSPIFFEID(goID)
			require.NoError(t, err)

			// Should be identical
			assert.True(t, namespace.Equals(backToNamespace))
			assert.Equal(t, originalID, backToNamespace.String())
		})
	}
}
