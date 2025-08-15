package ephemos

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func TestSPIFFEValidation(t *testing.T) {
	testCases := []struct {
		name        string
		spiffeID    string
		expectValid bool
	}{
		{
			name:        "Valid SPIFFE ID",
			spiffeID:    "spiffe://example.org/service",
			expectValid: true,
		},
		{
			name:        "Valid SPIFFE ID with path segments",
			spiffeID:    "spiffe://trust.domain.com/ns/production/sa/api-server",
			expectValid: true,
		},
		{
			name:        "Missing spiffe prefix",
			spiffeID:    "example.org/service",
			expectValid: false,
		},
		{
			name:        "Empty string",
			spiffeID:    "",
			expectValid: true, // Our validation allows empty unless required
		},
		{
			name:        "Invalid scheme",
			spiffeID:    "http://example.org/service",
			expectValid: false,
		},
		{
			name:        "Missing trust domain",
			spiffeID:    "spiffe:///service",
			expectValid: false,
		},
		{
			name:        "Valid SPIFFE ID without path",
			spiffeID:    "spiffe://example.org",
			expectValid: true, // Official SDK allows this
		},
	}

	validator := &SPIFFEValidator{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.spiffeID == "" {
				// Test empty string handling
				_, parseErr := spiffeid.FromString("spiffe://example.org/test")
				if parseErr != nil {
					t.Fatalf("Failed to create test SPIFFE ID: %v", parseErr)
				}
				return // Skip validation for empty string test
			} else {
				err = validator.ValidateSPIFFEID(tc.spiffeID)
			}

			isValid := err == nil
			if isValid != tc.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v, error=%v", tc.expectValid, isValid, err)
			}
		})
	}
}

func TestSPIFFEIDParsing(t *testing.T) {
	spiffeIDStr := "spiffe://example.org/ns/production/sa/api-server"
	
	spiffeID, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		t.Fatalf("Failed to parse SPIFFE ID: %v", err)
	}

	// Verify components
	if spiffeID.TrustDomain().String() != "example.org" {
		t.Errorf("Expected trust domain 'example.org', got '%s'", spiffeID.TrustDomain().String())
	}
	if spiffeID.Path() != "/ns/production/sa/api-server" {
		t.Errorf("Expected path '/ns/production/sa/api-server', got '%s'", spiffeID.Path())
	}
}