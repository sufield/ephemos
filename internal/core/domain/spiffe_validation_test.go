package domain

import (
	"crypto/x509"
	"testing"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func TestSPIFFEValidator_ValidateSPIFFEID(t *testing.T) {
	validator := NewSPIFFEValidator(nil)

	tests := []struct {
		name      string
		spiffeID  string
		wantError bool
	}{
		{
			name:      "valid SPIFFE ID",
			spiffeID:  "spiffe://example.org/service",
			wantError: false,
		},
		{
			name:      "valid SPIFFE ID with path",
			spiffeID:  "spiffe://trust.domain/path/to/service",
			wantError: false,
		},
		{
			name:      "invalid format - no spiffe prefix",
			spiffeID:  "http://example.org/service",
			wantError: true,
		},
		{
			name:      "invalid format - no trust domain",
			spiffeID:  "spiffe:///service",
			wantError: true,
		},
		{
			name:      "empty SPIFFE ID",
			spiffeID:  "",
			wantError: true, // Empty SPIFFE IDs are invalid
		},
		{
			name:      "invalid format - spaces",
			spiffeID:  "spiffe://example.org/service with spaces",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSPIFFEID(tt.spiffeID)

			if tt.wantError && err == nil {
				t.Errorf("ValidateSPIFFEID() expected error for %q, got nil", tt.spiffeID)
			}

			if !tt.wantError && err != nil {
				t.Errorf("ValidateSPIFFEID() unexpected error for %q: %v", tt.spiffeID, err)
			}
		})
	}
}

func TestSPIFFEValidator_ValidateX509SVID_NoBundleSource(t *testing.T) {
	validator := NewSPIFFEValidator(nil)

	// Test with nil bundle source
	certChain := [][]byte{
		[]byte("dummy cert data"),
	}

	_, err := validator.ValidateX509SVID(certChain)
	if err == nil {
		t.Error("ValidateX509SVID() expected error with nil bundle source, got nil")
	}

	if err.Error() != "bundle source not configured" {
		t.Errorf("ValidateX509SVID() expected 'bundle source not configured', got: %v", err)
	}
}

func TestSPIFFEValidator_ValidateX509Certificates_NoBundleSource(t *testing.T) {
	validator := NewSPIFFEValidator(nil)

	// Test with nil bundle source
	certs := []*x509.Certificate{
		{}, // dummy certificate
	}

	_, err := validator.ValidateX509Certificates(certs)
	if err == nil {
		t.Error("ValidateX509Certificates() expected error with nil bundle source, got nil")
	}

	if err.Error() != "bundle source not configured" {
		t.Errorf("ValidateX509Certificates() expected 'bundle source not configured', got: %v", err)
	}
}

func TestNewSPIFFEValidator(t *testing.T) {
	// Test with nil bundle source
	validator := NewSPIFFEValidator(nil)
	if validator == nil {
		t.Error("NewSPIFFEValidator() returned nil")
		return
	}

	if validator.bundleSource != nil {
		t.Error("NewSPIFFEValidator() expected nil bundle source")
	}

	// Test with mock bundle source
	mockBundle := &mockBundleSource{}
	validator = NewSPIFFEValidator(mockBundle)
	if validator == nil {
		t.Error("NewSPIFFEValidator() returned nil with bundle source")
		return
	}

	if validator.bundleSource != mockBundle {
		t.Error("NewSPIFFEValidator() bundle source not set correctly")
	}
}

// Mock implementation for testing
type mockBundleSource struct{}

func (m *mockBundleSource) GetX509BundleForTrustDomain(trustDomain spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Return a mock bundle - in real implementation this would contain actual trust anchors
	return x509bundle.New(trustDomain), nil
}

func TestSPIFFEValidator_ValidateSPIFFEID_EdgeCases(t *testing.T) {
	validator := NewSPIFFEValidator(nil)

	tests := []struct {
		name      string
		spiffeID  string
		wantError bool
	}{
		{
			name:      "SPIFFE ID with port",
			spiffeID:  "spiffe://example.org:8080/service",
			wantError: true, // Ports are not allowed in trust domain
		},
		{
			name:      "SPIFFE ID with query params",
			spiffeID:  "spiffe://example.org/service?param=value",
			wantError: true,
		},
		{
			name:      "SPIFFE ID with fragment",
			spiffeID:  "spiffe://example.org/service#fragment",
			wantError: true,
		},
		{
			name:      "SPIFFE ID with very long path",
			spiffeID:  "spiffe://example.org/" + string(make([]byte, 2048)),
			wantError: true, // Should be invalid due to length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSPIFFEID(tt.spiffeID)

			if tt.wantError && err == nil {
				t.Errorf("ValidateSPIFFEID() expected error for %q, got nil", tt.spiffeID)
			}

			if !tt.wantError && err != nil {
				t.Errorf("ValidateSPIFFEID() unexpected error for %q: %v", tt.spiffeID, err)
			}
		})
	}
}

func TestSPIFFEValidationIntegration(t *testing.T) {
	// Test SPIFFE ID validation using the official SDK
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid basic", "spiffe://example.org/service", true},
		{"valid with path segments", "spiffe://trust.domain/path/to/service", true},
		{"valid with dashes", "spiffe://trust-domain.org/my-service", true},
		{"invalid scheme", "https://example.org/service", false},
		{"invalid empty trust domain", "spiffe:///service", false},
		{"invalid characters", "spiffe://example.org/service with spaces", false},
		{"empty string", "", false}, // Empty is invalid
	}

	validator := NewSPIFFEValidator(nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateSPIFFEID(tc.input)
			isValid := err == nil

			if isValid != tc.expected {
				t.Errorf("SPIFFE ID %q: expected valid=%v, got valid=%v, error=%v",
					tc.input, tc.expected, isValid, err)
			}
		})
	}
}
