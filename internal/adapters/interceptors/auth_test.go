package interceptors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestNewAuthInterceptor(t *testing.T) {
	config := DefaultAuthConfig()
	interceptor := NewAuthInterceptor(config)

	if interceptor == nil {
		t.Fatal("NewAuthInterceptor returned nil")
	}
	if interceptor.config != config {
		t.Error("Config not properly set")
	}
	if interceptor.logger == nil {
		t.Error("Logger not set")
	}
}

func TestNewAuthInterceptor_WithNilLogger(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		Logger:                nil,
	}
	interceptor := NewAuthInterceptor(config)

	if interceptor.logger == nil {
		t.Error("Logger should be set to default when nil provided")
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_SkipMethod(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		SkipMethods:           []string{"/test.Service/SkipMethod"},
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/SkipMethod",
	}

	result, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != defaultResultCode {
		t.Errorf("Expected 'success', got: %v", result)
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_NoAuth(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: false,
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	result, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != defaultResultCode {
		t.Errorf("Expected 'success', got: %v", result)
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_RequireAuth_NoPeer(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)

	if err == nil {
		t.Error("Expected error for missing peer info")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error, got: %v", status.Code(err))
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_RequireAuth_NoTLS(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with non-TLS peer
	p := &peer.Peer{
		Addr:     &mockAddr{},
		AuthInfo: &mockAuthInfo{},
	}
	ctx := peer.NewContext(t.Context(), p)

	_, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)

	if err == nil {
		t.Error("Expected error for non-TLS auth")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error, got: %v", status.Code(err))
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_WithValidCert(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		// Check that identity is in context
		identity, ok := GetIdentityFromContext(ctx)
		if !ok {
			t.Error("Identity not found in context")
		}
		if identity.SPIFFEID != "spiffe://example.org/test-service" {
			t.Errorf("Unexpected SPIFFE ID: %s", identity.SPIFFEID)
		}
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with valid SPIFFE certificate
	cert := createTestSPIFFECert(t, "spiffe://example.org/test-service")
	p := &peer.Peer{
		Addr: &mockAddr{},
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{cert},
			},
		},
	}
	ctx := peer.NewContext(t.Context(), p)

	result, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != defaultResultCode {
		t.Errorf("Expected 'success', got: %v", result)
	}
}

func TestAuthInterceptor_UnaryServerInterceptor_DeniedService(t *testing.T) {
	config := &AuthConfig{
		RequireAuthentication: true,
		AllowedServices:       []string{"spiffe://example.org/denied-service"},
		DenyMode:              true, // Blacklist mode
		Logger:                slog.Default(),
	}
	interceptor := NewAuthInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with denied service certificate
	cert := createTestSPIFFECert(t, "spiffe://example.org/denied-service")
	p := &peer.Peer{
		Addr: &mockAddr{},
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{cert},
			},
		},
	}
	ctx := peer.NewContext(t.Context(), p)

	_, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)

	if err == nil {
		t.Error("Expected error for denied service")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Errorf("Expected PermissionDenied error, got: %v", status.Code(err))
	}
}

func TestParseSpiffeID(t *testing.T) {
	interceptor := NewAuthInterceptor(DefaultAuthConfig())

	tests := []struct {
		name        string
		spiffeID    string
		expectError bool
		expected    *AuthenticatedIdentity
	}{
		{
			name:     "valid_spiffe_id_with_path",
			spiffeID: "spiffe://example.org/workload/test-service",
			expected: &AuthenticatedIdentity{
				SPIFFEID:     "spiffe://example.org/workload/test-service",
				TrustDomain:  "example.org",
				ServiceName:  "test-service",
				WorkloadPath: "/workload/test-service",
				Claims:       make(map[string]string),
			},
		},
		{
			name:     "valid_spiffe_id_without_path",
			spiffeID: "spiffe://example.org",
			expected: &AuthenticatedIdentity{
				SPIFFEID:     "spiffe://example.org",
				TrustDomain:  "example.org",
				ServiceName:  "",
				WorkloadPath: "",
				Claims:       make(map[string]string),
			},
		},
		{
			name:        "invalid_scheme",
			spiffeID:    "https://example.org/test",
			expectError: true,
		},
		{
			name:        "empty_string",
			spiffeID:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interceptor.parseSpiffeID(tt.spiffeID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.SPIFFEID != tt.expected.SPIFFEID {
				t.Errorf("SPIFFE ID mismatch: expected %s, got %s",
					tt.expected.SPIFFEID, result.SPIFFEID)
			}
			if result.TrustDomain != tt.expected.TrustDomain {
				t.Errorf("Trust domain mismatch: expected %s, got %s",
					tt.expected.TrustDomain, result.TrustDomain)
			}
			if result.ServiceName != tt.expected.ServiceName {
				t.Errorf("Service name mismatch: expected %s, got %s",
					tt.expected.ServiceName, result.ServiceName)
			}
			if result.WorkloadPath != tt.expected.WorkloadPath {
				t.Errorf("Workload path mismatch: expected %s, got %s",
					tt.expected.WorkloadPath, result.WorkloadPath)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	interceptor := NewAuthInterceptor(DefaultAuthConfig())

	tests := []struct {
		name     string
		spiffeID string
		pattern  string
		expected bool
	}{
		{
			name:     "exact_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://example.org/test-service",
			expected: true,
		},
		{
			name:     "prefix_wildcard",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://example.org/*",
			expected: true,
		},
		{
			name:     "suffix_wildcard",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "*test-service",
			expected: true,
		},
		{
			name:     "no_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://other.org/test-service",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interceptor.matchesPattern(tt.spiffeID, tt.pattern)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetIdentityFromContext(t *testing.T) {
	identity := &AuthenticatedIdentity{
		SPIFFEID:    "spiffe://example.org/test",
		ServiceName: "test",
	}

	ctx := context.WithValue(t.Context(), IdentityContextKey{}, identity)

	retrieved, ok := GetIdentityFromContext(ctx)
	if !ok {
		t.Error("Identity not found in context")
	}
	if retrieved.SPIFFEID != identity.SPIFFEID {
		t.Errorf("SPIFFE ID mismatch: expected %s, got %s",
			identity.SPIFFEID, retrieved.SPIFFEID)
	}
}

func TestRequireIdentity(t *testing.T) {
	t.Run("with_identity", func(t *testing.T) {
		identity := &AuthenticatedIdentity{
			SPIFFEID: "spiffe://example.org/test",
		}
		ctx := context.WithValue(t.Context(), IdentityContextKey{}, identity)

		retrieved, err := RequireIdentity(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if retrieved.SPIFFEID != identity.SPIFFEID {
			t.Error("Identity mismatch")
		}
	})

	t.Run("without_identity", func(t *testing.T) {
		ctx := t.Context()

		_, err := RequireIdentity(ctx)
		if err == nil {
			t.Error("Expected error for missing identity")
		}
		if status.Code(err) != codes.Unauthenticated {
			t.Errorf("Expected Unauthenticated error, got: %v", status.Code(err))
		}
	})
}

// Helper functions and mocks

type mockAddr struct{}

func (m *mockAddr) Network() string { return "tcp" }
func (m *mockAddr) String() string  { return "127.0.0.1:12345" }

type mockAuthInfo struct{}

func (m *mockAuthInfo) AuthType() string { return "insecure" }

func createTestSPIFFECert(t *testing.T, spiffeID string) *x509.Certificate {
	t.Helper()

	spiffeURI, err := url.Parse(spiffeID)
	if err != nil {
		t.Fatalf("Failed to parse SPIFFE ID: %v", err)
	}

	return &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "test-cert",
		},
		URIs:      []*url.URL{spiffeURI},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}
}
