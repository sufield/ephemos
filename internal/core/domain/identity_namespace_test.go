package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

func TestNewIdentityNamespace(t *testing.T) {
	tests := []struct {
		name        string
		trustDomain string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid simple identity",
			trustDomain: "example.org",
			path:        "/service",
			wantErr:     false,
		},
		{
			name:        "valid complex identity",
			trustDomain: "prod.company.com",
			path:        "/service/payment-processor",
			wantErr:     false,
		},
		{
			name:        "valid root path",
			trustDomain: "example.org",
			path:        "/",
			wantErr:     false,
		},
		{
			name:        "empty trust domain",
			trustDomain: "",
			path:        "/service",
			wantErr:     true,
			errContains: "trust domain cannot be empty",
		},
		{
			name:        "invalid trust domain",
			trustDomain: "invalid..domain",
			path:        "/service",
			wantErr:     true,
			errContains: "invalid trust domain",
		},
		{
			name:        "empty path",
			trustDomain: "example.org",
			path:        "",
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name:        "path without leading slash",
			trustDomain: "example.org",
			path:        "service",
			wantErr:     true,
			errContains: "path must start with '/'",
		},
		{
			name:        "path with double slashes",
			trustDomain: "example.org",
			path:        "/service//payment",
			wantErr:     true,
			errContains: "path cannot contain double slashes",
		},
		{
			name:        "path ending with slash",
			trustDomain: "example.org",
			path:        "/service/",
			wantErr:     true,
			errContains: "path cannot end with '/'",
		},
		{
			name:        "path with dot segment",
			trustDomain: "example.org",
			path:        "/service/./payment",
			wantErr:     true,
			errContains: "path cannot contain dot segments",
		},
		{
			name:        "path with invalid characters",
			trustDomain: "example.org",
			path:        "/service/@payment",
			wantErr:     true,
			errContains: "path contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustDomain, err := domain.NewTrustDomain(tt.trustDomain)
			if tt.trustDomain == "" || strings.Contains(tt.trustDomain, "..") {
				// Skip trust domain creation if we expect it to fail
				trustDomain = domain.TrustDomain(tt.trustDomain)
			} else {
				require.NoError(t, err, "Trust domain creation should not fail for test setup")
			}

			namespace, err := domain.NewIdentityNamespace(trustDomain, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.False(t, namespace.IsZero())
				assert.Equal(t, trustDomain, namespace.GetTrustDomain())
				assert.Equal(t, tt.path, namespace.GetPath())
			}
		})
	}
}

func TestNewIdentityNamespaceFromString(t *testing.T) {
	tests := []struct {
		name         string
		spiffeID     string
		wantErr      bool
		errContains  string
		expectedTD   string
		expectedPath string
	}{
		{
			name:         "valid simple SPIFFE ID",
			spiffeID:     "spiffe://example.org/service",
			wantErr:      false,
			expectedTD:   "example.org",
			expectedPath: "/service",
		},
		{
			name:         "valid complex SPIFFE ID",
			spiffeID:     "spiffe://prod.company.com/service/payment-processor",
			wantErr:      false,
			expectedTD:   "prod.company.com",
			expectedPath: "/service/payment-processor",
		},
		{
			name:         "valid root path",
			spiffeID:     "spiffe://example.org/",
			wantErr:      false,
			expectedTD:   "example.org",
			expectedPath: "/",
		},
		{
			name:         "valid without path",
			spiffeID:     "spiffe://example.org",
			wantErr:      false,
			expectedTD:   "example.org",
			expectedPath: "/",
		},
		{
			name:        "empty string",
			spiffeID:    "",
			wantErr:     true,
			errContains: "SPIFFE ID cannot be empty",
		},
		{
			name:        "not spiffe scheme",
			spiffeID:    "https://example.org/service",
			wantErr:     true,
			errContains: "SPIFFE ID must start with 'spiffe://'",
		},
		{
			name:        "missing trust domain",
			spiffeID:    "spiffe:///service",
			wantErr:     true,
			errContains: "SPIFFE ID must contain trust domain",
		},
		{
			name:        "invalid trust domain",
			spiffeID:    "spiffe://invalid..domain/service",
			wantErr:     true,
			errContains: "invalid trust domain",
		},
		{
			name:        "invalid URL format",
			spiffeID:    "spiffe://example.org:80:80/service",
			wantErr:     true,
			errContains: "invalid trust domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, err := domain.NewIdentityNamespaceFromString(tt.spiffeID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.False(t, namespace.IsZero())
				assert.Equal(t, tt.expectedTD, namespace.GetTrustDomain().String())
				assert.Equal(t, tt.expectedPath, namespace.GetPath())
				// For URIs without path, we normalize to include trailing slash
				expectedString := tt.spiffeID
				if tt.expectedPath == "/" && !strings.HasSuffix(tt.spiffeID, "/") {
					expectedString = tt.spiffeID + "/"
				}
				assert.Equal(t, expectedString, namespace.String())
			}
		})
	}
}

func TestIdentityNamespace_String(t *testing.T) {
	tests := []struct {
		name        string
		trustDomain string
		path        string
		expected    string
	}{
		{
			name:        "simple identity",
			trustDomain: "example.org",
			path:        "/service",
			expected:    "spiffe://example.org/service",
		},
		{
			name:        "complex identity",
			trustDomain: "prod.company.com",
			path:        "/service/payment-processor",
			expected:    "spiffe://prod.company.com/service/payment-processor",
		},
		{
			name:        "root path",
			trustDomain: "example.org",
			path:        "/",
			expected:    "spiffe://example.org/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustDomain := domain.MustNewTrustDomain(tt.trustDomain)
			namespace := domain.MustNewIdentityNamespace(trustDomain, tt.path)
			assert.Equal(t, tt.expected, namespace.String())
		})
	}
}

func TestIdentityNamespace_IsZero(t *testing.T) {
	var emptyNamespace domain.IdentityNamespace
	assert.True(t, emptyNamespace.IsZero())

	trustDomain := domain.MustNewTrustDomain("example.org")
	namespace := domain.MustNewIdentityNamespace(trustDomain, "/service")
	assert.False(t, namespace.IsZero())
}

func TestIdentityNamespace_Equals(t *testing.T) {
	trustDomain1 := domain.MustNewTrustDomain("example.org")
	trustDomain2 := domain.MustNewTrustDomain("other.org")

	namespace1 := domain.MustNewIdentityNamespace(trustDomain1, "/service")
	namespace2 := domain.MustNewIdentityNamespace(trustDomain1, "/service")
	namespace3 := domain.MustNewIdentityNamespace(trustDomain1, "/other")
	namespace4 := domain.MustNewIdentityNamespace(trustDomain2, "/service")

	assert.True(t, namespace1.Equals(namespace2))
	assert.False(t, namespace1.Equals(namespace3))
	assert.False(t, namespace1.Equals(namespace4))
}

func TestIdentityNamespace_IsChildOf(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	otherTrustDomain := domain.MustNewTrustDomain("other.org")

	root := domain.MustNewIdentityNamespace(trustDomain, "/")
	service := domain.MustNewIdentityNamespace(trustDomain, "/service")
	servicePayment := domain.MustNewIdentityNamespace(trustDomain, "/service/payment")
	otherService := domain.MustNewIdentityNamespace(trustDomain, "/other")
	differentDomain := domain.MustNewIdentityNamespace(otherTrustDomain, "/service")

	// Everything is a child of root
	assert.True(t, service.IsChildOf(root))
	assert.True(t, servicePayment.IsChildOf(root))
	assert.True(t, otherService.IsChildOf(root))

	// Service payment is a child of service
	assert.True(t, servicePayment.IsChildOf(service))

	// Service is not a child of service payment
	assert.False(t, service.IsChildOf(servicePayment))

	// Other service is not a child of service
	assert.False(t, otherService.IsChildOf(service))

	// Different trust domain cannot be a child
	assert.False(t, differentDomain.IsChildOf(service))
}

func TestIdentityNamespace_GetServiceName(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedName string
	}{
		{
			name:         "simple service",
			path:         "/service",
			expectedName: "service",
		},
		{
			name:         "nested service",
			path:         "/service/payment-processor",
			expectedName: "payment-processor",
		},
		{
			name:         "deeply nested",
			path:         "/org/team/service/payment",
			expectedName: "payment",
		},
		{
			name:         "root path",
			path:         "/",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustDomain := domain.MustNewTrustDomain("example.org")
			namespace := domain.MustNewIdentityNamespace(trustDomain, tt.path)
			assert.Equal(t, tt.expectedName, namespace.GetServiceName())
		})
	}
}

func TestIdentityNamespace_WithPath(t *testing.T) {
	trustDomain := domain.MustNewTrustDomain("example.org")
	original := domain.MustNewIdentityNamespace(trustDomain, "/service")

	newNamespace, err := original.WithPath("/other")
	require.NoError(t, err)

	assert.Equal(t, trustDomain, newNamespace.GetTrustDomain())
	assert.Equal(t, "/other", newNamespace.GetPath())
	assert.NotEqual(t, original.GetPath(), newNamespace.GetPath())
}

func TestIdentityNamespace_WithTrustDomain(t *testing.T) {
	trustDomain1 := domain.MustNewTrustDomain("example.org")
	trustDomain2 := domain.MustNewTrustDomain("other.org")
	original := domain.MustNewIdentityNamespace(trustDomain1, "/service")

	newNamespace, err := original.WithTrustDomain(trustDomain2)
	require.NoError(t, err)

	assert.Equal(t, trustDomain2, newNamespace.GetTrustDomain())
	assert.Equal(t, "/service", newNamespace.GetPath())
	assert.NotEqual(t, original.GetTrustDomain(), newNamespace.GetTrustDomain())
}

func TestIdentityNamespace_Validate(t *testing.T) {
	// Valid namespace
	trustDomain := domain.MustNewTrustDomain("example.org")
	namespace := domain.MustNewIdentityNamespace(trustDomain, "/service")
	assert.NoError(t, namespace.Validate())

	// Zero namespace
	var zeroNamespace domain.IdentityNamespace
	assert.Error(t, zeroNamespace.Validate())
}

func TestIdentityNamespace_MustConstructors(t *testing.T) {
	// Valid case should not panic
	trustDomain := domain.MustNewTrustDomain("example.org")
	namespace := domain.MustNewIdentityNamespace(trustDomain, "/service")
	assert.False(t, namespace.IsZero())

	// Valid string case should not panic
	namespace2 := domain.MustNewIdentityNamespaceFromString("spiffe://example.org/service")
	assert.False(t, namespace2.IsZero())

	// Invalid case should panic
	assert.Panics(t, func() {
		domain.MustNewIdentityNamespaceFromString("invalid")
	})
}

func TestIdentityNamespace_EdgeCases(t *testing.T) {
	t.Run("maximum length validation", func(t *testing.T) {
		trustDomain := domain.MustNewTrustDomain("example.org")

		// Create a very long path that would exceed SPIFFE ID length limit
		longPath := "/" + strings.Repeat("a", 2000)

		_, err := domain.NewIdentityNamespace(trustDomain, longPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("path validation edge cases", func(t *testing.T) {
		trustDomain := domain.MustNewTrustDomain("example.org")

		invalidPaths := []struct {
			path        string
			errContains string
		}{
			{"//", "path cannot contain double slashes"},
			{"/service/../other", "path cannot contain dot segments"},
			{"/service/./current", "path cannot contain dot segments"},
			{"/service/with space", "path contains invalid characters"},
			{"/service/with@symbol", "path contains invalid characters"},
		}

		for _, test := range invalidPaths {
			_, err := domain.NewIdentityNamespace(trustDomain, test.path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.errContains)
		}
	})
}
