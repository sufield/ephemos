package memidentity_test

import (
	"testing"

	"github.com/sufield/ephemos/internal/adapters/secondary/memidentity"
	"github.com/sufield/ephemos/internal/contract/identityprovider"
	"github.com/sufield/ephemos/internal/core/ports"
)

// TestMemIdentityProvider_Conformance runs the IdentityProvider contract suite against the fake.
func TestMemIdentityProvider_Conformance(t *testing.T) {
	identityprovider.Run(t, func(t *testing.T) ports.IdentityProvider {
		return memidentity.New()
	})
}