package domain_test

import (
	"strings"
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/sufield/ephemos/internal/core/domain"
)

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
