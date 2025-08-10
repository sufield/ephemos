package spiffe_test

import (
	"testing"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/contract/identityprovider"
	"github.com/sufield/ephemos/internal/core/ports"
)

// TestSPIFFEProvider_Conformance runs the IdentityProvider contract suite against SPIFFE provider.
func TestSPIFFEProvider_Conformance(t *testing.T) {
	identityprovider.Run(t, func(t *testing.T) ports.IdentityProvider {
		// Create SPIFFE provider with default config
		provider, err := spiffe.NewProvider(nil)
		if err != nil {
			t.Fatalf("Failed to create SPIFFE provider: %v", err)
		}
		return provider
	})
}