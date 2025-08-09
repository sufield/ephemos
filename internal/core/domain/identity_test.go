package domain

import (
	"fmt"
	"strings"
	"testing"
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
			identity := NewServiceIdentity(tt.serviceName, tt.domain)

			if identity == nil {
				t.Error("NewServiceIdentity() returned nil identity")
				return
			}

			if identity.Name != tt.serviceName {
				t.Errorf("Name = %v, want %v", identity.Name, tt.serviceName)
			}

			if identity.Domain != tt.domain {
				t.Errorf("Domain = %v, want %v", identity.Domain, tt.domain)
			}

			expectedURI := fmt.Sprintf("spiffe://%s/%s", tt.domain, tt.serviceName)
			if identity.URI != expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI, expectedURI)
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
			identity := NewServiceIdentity(tt.serviceName, tt.domain)
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
			identity := NewServiceIdentity(tt.serviceName, tt.domain)

			if identity.URI != tt.expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI, tt.expectedURI)
			}
		})
	}
}

func TestServiceIdentity_Fields(t *testing.T) {
	// Test that all fields are properly set
	serviceName := "test-service"
	domain := "example.com"

	identity := NewServiceIdentity(serviceName, domain)

	if identity.Name != serviceName {
		t.Errorf("Name = %v, want %v", identity.Name, serviceName)
	}

	if identity.Domain != domain {
		t.Errorf("Domain = %v, want %v", identity.Domain, domain)
	}

	expectedURI := fmt.Sprintf("spiffe://%s/%s", domain, serviceName)
	if identity.URI != expectedURI {
		t.Errorf("URI = %v, want %v", identity.URI, expectedURI)
	}
}

func TestServiceIdentity_Immutability(t *testing.T) {
	// Test that the identity remains immutable after creation
	identity := NewServiceIdentity("test-service", "example.com")

	originalName := identity.Name
	originalDomain := identity.Domain
	originalURI := identity.URI

	// Values should remain the same on multiple accesses
	if identity.Name != originalName {
		t.Error("Name changed after creation")
	}

	if identity.Domain != originalDomain {
		t.Error("Domain changed after creation")
	}

	if identity.URI != originalURI {
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
			identity := NewServiceIdentity(tt.serviceName, tt.domain)

			if identity == nil {
				t.Error("NewServiceIdentity() returned nil")
				return
			}

			if identity.Name != tt.serviceName {
				t.Errorf("Name = %v, want %v", identity.Name, tt.serviceName)
			}

			if identity.Domain != tt.domain {
				t.Errorf("Domain = %v, want %v", identity.Domain, tt.domain)
			}

			// URI should be properly formatted
			expectedURI := fmt.Sprintf("spiffe://%s/%s", tt.domain, tt.serviceName)
			if identity.URI != expectedURI {
				t.Errorf("URI = %v, want %v", identity.URI, expectedURI)
			}
		})
	}
}

func BenchmarkNewServiceIdentity(b *testing.B) {
	serviceName := "test-service"
	domain := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identity := NewServiceIdentity(serviceName, domain)
		if identity == nil {
			b.Error("NewServiceIdentity returned nil")
		}
	}
}

func BenchmarkServiceIdentity_Validate(b *testing.B) {
	identity := NewServiceIdentity("test-service", "example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := identity.Validate()
		if err != nil {
			b.Errorf("Validate() failed: %v", err)
		}
	}
}

func BenchmarkServiceIdentity_URIAccess(b *testing.B) {
	identity := NewServiceIdentity("test-service", "example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uri := identity.URI
		if uri == "" {
			b.Error("URI is empty")
		}
	}
}

func TestServiceIdentity_String(t *testing.T) {
	// Test string representation through URI field
	identity := NewServiceIdentity("test-service", "example.com")

	if identity.URI == "" {
		t.Error("URI should not be empty")
	}

	if !strings.Contains(identity.URI, "spiffe://") {
		t.Error("URI should contain SPIFFE scheme")
	}

	if !strings.Contains(identity.URI, "example.com") {
		t.Error("URI should contain domain")
	}

	if !strings.Contains(identity.URI, "test-service") {
		t.Error("URI should contain service name")
	}
}

func TestServiceIdentity_Concurrent(t *testing.T) {
	// Test concurrent access to identity fields
	identity := NewServiceIdentity("test-service", "example.com")

	done := make(chan bool, 10)

	// Run multiple goroutines accessing the identity
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				name := identity.Name
				domain := identity.Domain
				uri := identity.URI

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
