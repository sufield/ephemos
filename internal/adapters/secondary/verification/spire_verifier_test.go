package verification

import (
	"context"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewSpireIdentityVerifier(t *testing.T) {
	tests := []struct {
		name        string
		config      *ports.VerificationConfig
		expectError bool
	}{
		{
			name:        "nil config should error",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid config should succeed",
			config: &ports.VerificationConfig{
				WorkloadAPISocket: "unix:///tmp/test.sock",
				Timeout:           30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "config with defaults",
			config: &ports.VerificationConfig{
				// Empty config should get defaults
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := NewSpireIdentityVerifier(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, verifier)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, verifier)
				assert.NotNil(t, verifier.config)

				// Check defaults are applied
				if tt.config != nil && tt.config.WorkloadAPISocket == "" {
					assert.Equal(t, "unix:///tmp/spire-agent/public/api.sock", verifier.config.WorkloadAPISocket)
				}
				if tt.config != nil && tt.config.Timeout == 0 {
					assert.Equal(t, 30*time.Second, verifier.config.Timeout)
				}
			}
		})
	}
}

func TestSpireIdentityVerifier_Close(t *testing.T) {
	config := &ports.VerificationConfig{
		WorkloadAPISocket: "unix:///tmp/test.sock",
		Timeout:           10 * time.Second,
	}

	verifier, err := NewSpireIdentityVerifier(config)
	require.NoError(t, err)
	require.NotNil(t, verifier)

	// Close should not error even if source is nil
	err = verifier.Close()
	assert.NoError(t, err)

	// Multiple closes should be safe
	err = verifier.Close()
	assert.NoError(t, err)
}

// Note: Key usage extraction tests removed as we now rely on SPIRE's built-in tools
// Use 'ephemos inspect svid --use-cli' for detailed certificate inspection

// Note: TLS version and cipher suite parsing tests removed as we now rely on 
// standard Go crypto/tls package and SPIRE's built-in tools for detailed TLS inspection

func TestValidateConnectionTimeout(t *testing.T) {
	t.Skip("Skipping timeout test - requires integration testing environment")
	
	// Test that connection timeout is properly handled
	config := &ports.VerificationConfig{
		WorkloadAPISocket: "unix:///tmp/test.sock",
		Timeout:           1 * time.Nanosecond, // Very short timeout
	}

	verifier, err := NewSpireIdentityVerifier(config)
	require.NoError(t, err)
	defer verifier.Close()

	ctx := context.Background()
	targetID := spiffeid.RequireFromString("spiffe://example.org/test")

	// This should timeout quickly when trying to connect to a non-existent address
	result, err := verifier.ValidateConnection(ctx, targetID, "localhost:99999")
	
	// We expect this to fail due to connection issues, not timeout issues in this test
	// The actual timeout behavior would require integration testing with real network calls
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
}

// Mock certificate for testing key usage extraction
type mockCertificate struct {
	keyUsage int
}

func (m *mockCertificate) KeyUsage() int {
	return m.keyUsage
}

// Note: Mock certificate functions removed as we now use SPIRE's built-in capabilities directly

// Note: Key usage conversion tests removed - use SPIRE's built-in tools for certificate details