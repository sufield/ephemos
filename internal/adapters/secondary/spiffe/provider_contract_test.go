package spiffe_test

import (
	"os"
	"testing"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/contract/identityprovider"
	"github.com/sufield/ephemos/internal/core/ports"
)

// TestSPIFFEProvider_Conformance runs the IdentityProvider contract suite against SPIFFE provider.
func TestSPIFFEProvider_Conformance(t *testing.T) {
	// Skip if SPIRE agent socket is not available
	if _, err := os.Stat("/tmp/spire-agent/public/api.sock"); os.IsNotExist(err) {
		t.Skip("Skipping SPIFFE integration test: SPIRE agent not available")
	}

	identityprovider.Run(t, func(t *testing.T) ports.IdentityProvider {
		t.Helper()
		// Create SPIFFE provider with default config
		provider, err := spiffe.NewProvider(nil)
		if err != nil {
			t.Fatalf("Failed to create SPIFFE provider: %v", err)
		}
		return provider
	})
}
