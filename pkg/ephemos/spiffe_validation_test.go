package ephemos

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func TestSPIFFEValidation_CompareApproaches(t *testing.T) {
	testCases := []struct {
		name           string
		spiffeID       string
		expectValid    bool
		expectSDKError bool // Whether the official SDK should catch additional issues
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
			expectValid: true, // Our validation allows empty (unless required)
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
			name:        "Missing path",
			spiffeID:    "spiffe://example.org",
			expectValid: false,
		},
		{
			name:           "Invalid characters in trust domain",
			spiffeID:       "spiffe://INVALID_DOMAIN/service",
			expectValid:    true,  // Current validation would pass this
			expectSDKError: true,  // But official SDK should catch it
		},
		{
			name:           "Malformed trust domain",
			spiffeID:       "spiffe://example..org/service", 
			expectValid:    true,  // Current validation would pass this
			expectSDKError: true,  // But official SDK should catch it
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test current basic validation approach
			currentErr := validateSPIFFEIDBasic(tc.spiffeID)
			currentValid := currentErr == nil

			// Test official SDK approach
			sdkErr := ValidateSPIFFEIDWithSDK(tc.spiffeID)
			sdkValid := sdkErr == nil

			// Verify expectations for current approach
			if currentValid != tc.expectValid {
				t.Errorf("Current validation: expected valid=%v, got valid=%v, error=%v",
					tc.expectValid, currentValid, currentErr)
			}

			// If we expect the SDK to catch additional errors
			if tc.expectSDKError {
				if sdkValid {
					t.Errorf("Official SDK validation should have failed but passed for: %s", tc.spiffeID)
				}
			} else {
				// SDK should match the expected validity
				if sdkValid != tc.expectValid {
					t.Errorf("Official SDK validation: expected valid=%v, got valid=%v, error=%v",
						tc.expectValid, sdkValid, sdkErr)
				}
			}

			// Log the difference for analysis
			if currentValid != sdkValid {
				t.Logf("VALIDATION DIFFERENCE for '%s':", tc.spiffeID)
				t.Logf("  Current approach: %v (error: %v)", currentValid, currentErr)
				t.Logf("  Official SDK:     %v (error: %v)", sdkValid, sdkErr)
			}
		})
	}
}

// validateSPIFFEIDBasic replicates the current basic validation approach for comparison
func validateSPIFFEIDBasic(spiffeIDStr string) error {
	if spiffeIDStr == "" {
		return nil // Empty SPIFFE IDs are allowed unless required
	}

	// Current basic validation (from validation.go before our change)
	if len(spiffeIDStr) < 9 || spiffeIDStr[:9] != "spiffe://" {
		return fmt.Errorf("SPIFFE ID must start with 'spiffe://'")
	}

	// Basic structure validation - must have trust domain and path
	remainder := spiffeIDStr[9:] // Remove "spiffe://" prefix
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("SPIFFE ID must have format 'spiffe://trust-domain/path'")
	}

	return nil
}

func TestOfficialSDKFeatures(t *testing.T) {
	t.Run("Proper SPIFFE ID Parsing", func(t *testing.T) {
		spiffeIDStr := "spiffe://example.org/ns/production/sa/api-server"
		
		// Parse using official SDK
		spiffeID, err := spiffeid.FromString(spiffeIDStr)
		if err != nil {
			t.Fatalf("Failed to parse SPIFFE ID: %v", err)
		}

		// Demonstrate structured access to components
		t.Logf("Trust Domain: %s", spiffeID.TrustDomain().String())
		t.Logf("Path: %s", spiffeID.Path())
		t.Logf("Full ID: %s", spiffeID.String())

		// Verify components
		if spiffeID.TrustDomain().String() != "example.org" {
			t.Errorf("Expected trust domain 'example.org', got '%s'", spiffeID.TrustDomain().String())
		}
		if spiffeID.Path() != "/ns/production/sa/api-server" {
			t.Errorf("Expected path '/ns/production/sa/api-server', got '%s'", spiffeID.Path())
		}
	})

	t.Run("SPIFFE ID Comparison", func(t *testing.T) {
		id1, _ := spiffeid.FromString("spiffe://example.org/service")
		id2, _ := spiffeid.FromString("spiffe://example.org/service")
		id3, _ := spiffeid.FromString("spiffe://example.org/other-service")

		// Demonstrate proper equality comparison
		if id1 != id2 {
			t.Error("Identical SPIFFE IDs should be equal")
		}
		if id1 == id3 {
			t.Error("Different SPIFFE IDs should not be equal")
		}
	})

	t.Run("Trust Domain Operations", func(t *testing.T) {
		id, _ := spiffeid.FromString("spiffe://example.org/service")
		
		// Demonstrate trust domain operations
		trustDomain := id.TrustDomain()
		t.Logf("Trust domain: %s", trustDomain.String())

		// Check if same trust domain
		otherID, _ := spiffeid.FromString("spiffe://example.org/other-service")
		if id.TrustDomain() != otherID.TrustDomain() {
			t.Error("Services in same trust domain should have equal trust domains")
		}
	})
}

// Benchmark comparison between current and SDK approaches
func BenchmarkSPIFFEValidation(b *testing.B) {
	spiffeID := "spiffe://example.org/ns/production/sa/api-server"

	b.Run("Current Basic Validation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = validateSPIFFEIDBasic(spiffeID)
		}
	})

	b.Run("Official SDK Validation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateSPIFFEIDWithSDK(spiffeID)
		}
	})

	b.Run("Official SDK Parsing Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = spiffeid.FromString(spiffeID)
		}
	})
}