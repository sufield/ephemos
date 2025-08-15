package ephemos

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func TestSPIFFEValidatorInterface(t *testing.T) {
	// Test that the public interface correctly delegates to the domain layer
	validator := NewSPIFFEValidator(nil)
	if validator == nil {
		t.Fatal("NewSPIFFEValidator() returned nil")
	}

	// Test SPIFFE ID validation
	err := validator.ValidateSPIFFEID("spiffe://example.org/service")
	if err != nil {
		t.Errorf("ValidateSPIFFEID() failed with valid ID: %v", err)
	}

	err = validator.ValidateSPIFFEID("invalid-id")
	if err == nil {
		t.Error("ValidateSPIFFEID() should have failed with invalid ID")
	}
}

func TestValidateSPIFFEIDFunction(t *testing.T) {
	// Test the convenience function
	err := ValidateSPIFFEID("spiffe://example.org/service")
	if err != nil {
		t.Errorf("ValidateSPIFFEID() failed with valid ID: %v", err)
	}

	err = ValidateSPIFFEID("invalid-id")
	if err == nil {
		t.Error("ValidateSPIFFEID() should have failed with invalid ID")
	}

	err = ValidateSPIFFEID("")
	if err == nil {
		t.Error("ValidateSPIFFEID() should have failed with empty ID")
	}
}

func TestValidateX509SVIDFunction(t *testing.T) {
	// Test with nil bundle source (should fail)
	_, err := ValidateX509SVID(nil, [][]byte{})
	if err == nil {
		t.Error("ValidateX509SVID() should have failed with nil bundle source")
	}

	// Test with mock bundle source
	mockBundle := &mockBundleSource{}
	_, err = ValidateX509SVID(mockBundle, [][]byte{[]byte("invalid cert data")})
	if err == nil {
		t.Error("ValidateX509SVID() should have failed with invalid cert data")
	}
}

func TestValidateX509CertificatesFunction(t *testing.T) {
	// Test with nil bundle source (should fail)
	_, err := ValidateX509Certificates(nil, nil)
	if err == nil {
		t.Error("ValidateX509Certificates() should have failed with nil bundle source")
	}

	// Test with mock bundle source  
	mockBundle := &mockBundleSource{}
	_, err = ValidateX509Certificates(mockBundle, nil)
	if err == nil {
		t.Error("ValidateX509Certificates() should have failed with nil certs")
	}
}

// Mock implementation for testing
type mockBundleSource struct{}

func (m *mockBundleSource) GetX509BundleForTrustDomain(trustDomain spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	// Return a mock bundle - in real implementation this would contain actual trust anchors
	return x509bundle.New(trustDomain), nil
}