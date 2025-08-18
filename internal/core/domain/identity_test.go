package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/core/domain"
)

func TestNewServiceIdentity(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
	}{
		{
			name:        "valid service and domain",
			serviceName: "test-service",
			domain:      "example.com",
		},
		{
			name:        "valid subdomain",
			serviceName: "api-service",
			domain:      "api.example.com",
		},
		{
			name:        "service with hyphen",
			serviceName: "my-service",
			domain:      "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)

			if identity == nil {
				t.Error("domain.NewServiceIdentity() returned nil identity")
				return
			}

			if identity.Name() != tt.serviceName {
				t.Errorf("Name = %v, want %v", identity.Name(), tt.serviceName)
			}

			if identity.Domain() != tt.domain {
				t.Errorf("Domain = %v, want %v", identity.Domain(), tt.domain)
			}

			expectedURI := fmt.Sprintf("spiffe://%s/%s", tt.domain, tt.serviceName)
			if identity.URI() != expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI(), expectedURI)
			}
		})
	}
}

func TestServiceIdentity_Validate(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
		wantErr     bool
		errorMsg    string
	}{
		{
			name:        "valid identity",
			serviceName: "test-service",
			domain:      "example.com",
			wantErr:     false,
		},
		{
			name:        "empty service name",
			serviceName: "",
			domain:      "example.com",
			wantErr:     true,
			errorMsg:    "service name cannot be empty",
		},
		{
			name:        "empty domain",
			serviceName: "test-service",
			domain:      "",
			wantErr:     true,
			errorMsg:    "domain cannot be empty",
		},
		{
			name:        "both empty",
			serviceName: "",
			domain:      "",
			wantErr:     true,
			errorMsg:    "service name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)
			err := identity.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestServiceIdentity_URIGeneration(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
		expectedURI string
	}{
		{
			name:        "simple service",
			serviceName: "echo",
			domain:      "example.com",
			expectedURI: "spiffe://example.com/echo",
		},
		{
			name:        "hyphenated service",
			serviceName: "my-service",
			domain:      "example.com",
			expectedURI: "spiffe://example.com/my-service",
		},
		{
			name:        "subdomain",
			serviceName: "api",
			domain:      "prod.example.com",
			expectedURI: "spiffe://prod.example.com/api",
		},
		{
			name:        "complex names",
			serviceName: "user-auth-service",
			domain:      "auth.prod.example.com",
			expectedURI: "spiffe://auth.prod.example.com/user-auth-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)

			if identity.URI() != tt.expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI(), tt.expectedURI)
			}
		})
	}
}

const (
	testServiceName = "test-service"
	testDomain      = "example.com"
)

func TestServiceIdentity_Fields(t *testing.T) {
	// Test that all fields are properly set
	serviceName := testServiceName
	domainName := testDomain

	identity := domain.NewServiceIdentity(serviceName, domainName)

	if identity.Name() != serviceName {
		t.Errorf("Name = %v, want %v", identity.Name(), serviceName)
	}

	if identity.Domain() != domainName {
		t.Errorf("Domain = %v, want %v", identity.Domain(), domainName)
	}

	expectedURI := fmt.Sprintf("spiffe://%s/%s", domainName, serviceName)
	if identity.URI() != expectedURI {
		t.Errorf("URI = %v, want %v", identity.URI(), expectedURI)
	}
}

func TestServiceIdentity_Immutability(t *testing.T) {
	// Test that the identity remains immutable after creation
	identity := domain.NewServiceIdentity("test-service", "example.com")

	originalName := identity.Name()
	originalDomain := identity.Domain()
	originalURI := identity.URI()

	// Values should remain the same on multiple accesses
	if identity.Name() != originalName {
		t.Error("Name changed after creation")
	}

	if identity.Domain() != originalDomain {
		t.Error("Domain changed after creation")
	}

	if identity.URI() != originalURI {
		t.Error("URI changed after creation")
	}
}

func TestServiceIdentity_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
		description string
	}{
		{
			name:        "single character service",
			serviceName: "a",
			domain:      "example.com",
			description: "single character service name",
		},
		{
			name:        "single character domain",
			serviceName: "service",
			domain:      "a",
			description: "single character domain",
		},
		{
			name:        "numeric service name",
			serviceName: "123",
			domain:      "example.com",
			description: "numeric service name",
		},
		{
			name:        "service with numbers",
			serviceName: "service-v2",
			domain:      "example.com",
			description: "service name with version number",
		},
		{
			name:        "long names",
			serviceName: strings.Repeat("a", 100),
			domain:      strings.Repeat("b", 100) + ".com",
			description: "very long names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)

			if identity == nil {
				t.Error("domain.NewServiceIdentity() returned nil")
				return
			}

			if identity.Name() != tt.serviceName {
				t.Errorf("Name = %v, want %v", identity.Name(), tt.serviceName)
			}

			if identity.Domain() != tt.domain {
				t.Errorf("Domain = %v, want %v", identity.Domain(), tt.domain)
			}

			// URI should be properly formatted
			expectedURI := fmt.Sprintf("spiffe://%s/%s", tt.domain, tt.serviceName)
			if identity.URI() != expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI(), expectedURI)
			}
		})
	}
}

func BenchmarkNewServiceIdentity(b *testing.B) {
	serviceName := "test-service"
	domainName := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identity := domain.NewServiceIdentity(serviceName, domainName)
		if identity == nil {
			b.Error("domain.NewServiceIdentity returned nil")
		}
	}
}

func BenchmarkServiceIdentity_Validate(b *testing.B) {
	identity := domain.NewServiceIdentity("test-service", "example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := identity.Validate()
		if err != nil {
			b.Errorf("Validate() failed: %v", err)
		}
	}
}

func BenchmarkServiceIdentity_URIAccess(b *testing.B) {
	identity := domain.NewServiceIdentity("test-service", "example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := identity.URI()
		if uri == "" {
			b.Error("URI is empty")
		}
	}
}

func TestServiceIdentity_String(t *testing.T) {
	// Test string representation through URI field
	identity := domain.NewServiceIdentity("test-service", "example.com")

	if identity.URI() == "" {
		t.Error("URI should not be empty")
	}

	if !strings.Contains(identity.URI(), "spiffe://") {
		t.Error("URI should contain SPIFFE scheme")
	}

	if !strings.Contains(identity.URI(), "example.com") {
		t.Error("URI should contain domain")
	}

	if !strings.Contains(identity.URI(), "test-service") {
		t.Error("URI should contain service name")
	}
}

func TestServiceIdentity_Concurrent(t *testing.T) {
	// Test concurrent access to identity fields
	identity := domain.NewServiceIdentity("test-service", "example.com")

	done := make(chan bool, 10)

	// Run multiple goroutines accessing the identity
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				name := identity.Name()
				domain := identity.Domain()
				uri := identity.URI()

				if name != "test-service" {
					t.Errorf("Name = %v, want test-service", name)
					return
				}

				if domain != "example.com" {
					t.Errorf("Domain = %v, want example.com", domain)
					return
				}

				if uri != "spiffe://example.com/test-service" {
					t.Errorf("URI = %v, want spiffe://example.com/test-service", uri)
					return
				}

				err := identity.Validate()
				if err != nil {
					t.Errorf("Validate() failed: %v", err)
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test edge cases for enhanced SPIFFE constraint validation
func TestServiceIdentity_ValidateSPIFFEConstraints(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid multi-segment path",
			serviceName: "api/v1/service",
			domain:      "example.com",
			wantErr:     false,
		},
		{
			name:        "valid deep path",
			serviceName: "namespace/service/component/instance",
			domain:      "example.com",
			wantErr:     false,
		},
		{
			name:        "invalid double slash",
			serviceName: "api//service",
			domain:      "example.com",
			wantErr:     true,
			errContains: "double slashes",
		},
		{
			name:        "invalid leading slash in name",
			serviceName: "/api/service",
			domain:      "example.com",
			wantErr:     true,
			errContains: "cannot start or end with slash",
		},
		{
			name:        "invalid trailing slash in name",
			serviceName: "api/service/",
			domain:      "example.com",
			wantErr:     true,
			errContains: "cannot start or end with slash",
		},
		{
			name:        "invalid dot segment",
			serviceName: "api/./service",
			domain:      "example.com",
			wantErr:     true,
			errContains: "cannot contain '.' or '..' path segments",
		},
		{
			name:        "invalid dot-dot segment",
			serviceName: "api/../service",
			domain:      "example.com",
			wantErr:     true,
			errContains: "cannot contain '.' or '..' path segments",
		},
		{
			name:        "invalid characters in service name",
			serviceName: "api service",
			domain:      "example.com",
			wantErr:     true,
			errContains: "invalid characters",
		},
		{
			name:        "invalid uppercase in domain",
			serviceName: "service",
			domain:      "Example.COM",
			wantErr:     true,
			errContains: "trust domain must be lowercase",
		},
		{
			name:        "very long path - within limit",
			serviceName: strings.Repeat("a", 200) + "/" + strings.Repeat("b", 200),
			domain:      "example.com",
			wantErr:     false,
		},
		{
			name:        "extremely long path - exceeds limit",
			serviceName: strings.Repeat("a", 1000) + "/" + strings.Repeat("b", 1000) + "/" + strings.Repeat("c", 500),
			domain:      "example.com",
			wantErr:     true,
			errContains: "exceeds maximum length",
		},
		{
			name:        "very long trust domain - exceeds limit",
			serviceName: "service",
			domain:      strings.Repeat("a", 300) + ".com",
			wantErr:     true,
			errContains: "exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := domain.NewServiceIdentity(tt.serviceName, tt.domain)
			err := identity.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// Test edge cases for ServiceIdentity creation from SPIFFE ID
func TestNewServiceIdentityFromSPIFFEID_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		spiffeID   string
		wantName   string
		wantDomain string
		wantURI    string
	}{
		{
			name:       "multi-segment path",
			spiffeID:   "spiffe://example.com/api/v1/service",
			wantName:   "api/v1/service",
			wantDomain: "example.com",
			wantURI:    "spiffe://example.com/api/v1/service",
		},
		{
			name:       "deep nested path",
			spiffeID:   "spiffe://example.com/namespace/team/service/instance",
			wantName:   "namespace/team/service/instance",
			wantDomain: "example.com",
			wantURI:    "spiffe://example.com/namespace/team/service/instance",
		},
		{
			name:       "single segment path",
			spiffeID:   "spiffe://example.com/service",
			wantName:   "service",
			wantDomain: "example.com",
			wantURI:    "spiffe://example.com/service",
		},
		{
			name:       "empty path",
			spiffeID:   "spiffe://example.com",
			wantName:   "",
			wantDomain: "example.com",
			wantURI:    "spiffe://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spiffeID, err := spiffeid.FromString(tt.spiffeID)
			if err != nil {
				t.Fatalf("Failed to parse SPIFFE ID %q: %v", tt.spiffeID, err)
			}

			identity := domain.NewServiceIdentityFromSPIFFEID(spiffeID)

			if identity.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", identity.Name(), tt.wantName)
			}

			if identity.Domain() != tt.wantDomain {
				t.Errorf("Domain() = %q, want %q", identity.Domain(), tt.wantDomain)
			}

			if identity.URI() != tt.wantURI {
				t.Errorf("URI() = %q, want %q", identity.URI(), tt.wantURI)
			}
		})
	}
}

// Test edge cases for the new constructor with validation flag
func TestNewServiceIdentityWithValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		domain      string
		validate    bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid identity with validation",
			serviceName: "service",
			domain:      "example.com",
			validate:    true,
			wantErr:     false,
		},
		{
			name:        "valid identity without validation",
			serviceName: "service",
			domain:      "example.com",
			validate:    false,
			wantErr:     false,
		},
		{
			name:        "invalid identity with validation",
			serviceName: "", // empty name is invalid
			domain:      "example.com",
			validate:    true,
			wantErr:     true,
			errContains: "trailing slash",
		},
		{
			name:        "invalid identity without validation - allowed",
			serviceName: "", // empty name is invalid but validation is skipped
			domain:      "example.com",
			validate:    false,
			wantErr:     false,
		},
		{
			name:        "multi-segment path with validation",
			serviceName: "api/v1/service",
			domain:      "example.com",
			validate:    true,
			wantErr:     false,
		},
		{
			name:        "invalid multi-segment path with validation",
			serviceName: "api//service", // double slash
			domain:      "example.com",
			validate:    true,
			wantErr:     true,
			errContains: "empty segments",
		},
		{
			name:        "invalid multi-segment path without validation - allowed",
			serviceName: "api//service", // double slash but validation is skipped
			domain:      "example.com",
			validate:    false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := domain.NewServiceIdentityWithValidation(tt.serviceName, tt.domain, tt.validate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewServiceIdentityWithValidation() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewServiceIdentityWithValidation() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewServiceIdentityWithValidation() unexpected error = %v", err)
				}
				if identity == nil {
					t.Error("NewServiceIdentityWithValidation() returned nil identity")
				}
			}
		})
	}
}

// Test edge cases for Equal method with nil handling
func TestServiceIdentity_Equal_EdgeCases(t *testing.T) {
	validIdentity := domain.NewServiceIdentity("service", "example.com")
	otherIdentity := domain.NewServiceIdentity("other", "example.com")
	sameIdentity := domain.NewServiceIdentity("service", "example.com")

	tests := []struct {
		name     string
		identity *domain.ServiceIdentity
		other    *domain.ServiceIdentity
		want     bool
	}{
		{
			name:     "both nil",
			identity: nil,
			other:    nil,
			want:     true,
		},
		{
			name:     "first nil, second not nil",
			identity: nil,
			other:    validIdentity,
			want:     false,
		},
		{
			name:     "first not nil, second nil",
			identity: validIdentity,
			other:    nil,
			want:     false,
		},
		{
			name:     "both not nil, same values",
			identity: validIdentity,
			other:    sameIdentity,
			want:     true,
		},
		{
			name:     "both not nil, different values",
			identity: validIdentity,
			other:    otherIdentity,
			want:     false,
		},
		{
			name:     "same instance",
			identity: validIdentity,
			other:    validIdentity,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.identity.Equal(tt.other)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
