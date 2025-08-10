package identityprovider

import (
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Factory creates a new IdentityProvider implementation for testing.
type Factory func(t *testing.T) ports.IdentityProvider

// Run executes the complete contract test suite against any IdentityProvider implementation.
func Run(t *testing.T, newImpl Factory) {
	t.Run("get service identity", func(t *testing.T) {
		provider := newImpl(t)
		defer provider.Close()

		identity, err := provider.GetServiceIdentity()
		
		// Contract: Either returns valid identity or expected error
		if err != nil {
			t.Logf("GetServiceIdentity returned error (may be expected): %v", err)
			return
		}
		
		if identity == nil {
			t.Fatal("GetServiceIdentity returned nil identity without error")
		}
		
		if identity.Name == "" {
			t.Error("ServiceIdentity.Name should not be empty")
		}
		
		if identity.Domain == "" {
			t.Error("ServiceIdentity.Domain should not be empty")
		}
		
		if identity.URI == "" {
			t.Error("ServiceIdentity.URI should not be empty")
		}
		
		if err := identity.Validate(); err != nil {
			t.Errorf("ServiceIdentity should be valid: %v", err)
		}
	})

	t.Run("get certificate", func(t *testing.T) {
		provider := newImpl(t)
		defer provider.Close()

		cert, err := provider.GetCertificate()
		
		// Contract: Either returns valid certificate or expected error
		if err != nil {
			t.Logf("GetCertificate returned error (may be expected): %v", err)
			return
		}
		
		if cert == nil {
			t.Fatal("GetCertificate returned nil certificate without error")
		}
		
		if cert.Cert == nil {
			t.Error("Certificate.Cert should not be nil")
		}
		
		if cert.PrivateKey == nil {
			t.Error("Certificate.PrivateKey should not be nil")
		}
		
		if len(cert.Chain) == 0 {
			t.Error("Certificate.Chain should not be empty")
		}
		
		// Certificate chain consistency
		if cert.Cert != nil && len(cert.Chain) > 0 && cert.Cert != cert.Chain[0] {
			t.Error("Certificate.Cert should be first in chain")
		}
	})

	t.Run("get trust bundle", func(t *testing.T) {
		provider := newImpl(t)
		defer provider.Close()

		bundle, err := provider.GetTrustBundle()
		
		// Contract: Either returns valid bundle or expected error
		if err != nil {
			t.Logf("GetTrustBundle returned error (may be expected): %v", err)
			return
		}
		
		if bundle == nil {
			t.Fatal("GetTrustBundle returned nil bundle without error")
		}
		
		if len(bundle.Certificates) == 0 {
			t.Error("TrustBundle.Certificates should not be empty")
		}
		
		for i, cert := range bundle.Certificates {
			if cert == nil {
				t.Errorf("TrustBundle.Certificates[%d] should not be nil", i)
			}
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		provider := newImpl(t)
		
		// First close should succeed
		if err := provider.Close(); err != nil {
			t.Errorf("First Close() failed: %v", err)
		}
		
		// Second close should be safe (idempotent)
		if err := provider.Close(); err != nil {
			t.Errorf("Second Close() failed (not idempotent): %v", err)
		}
	})

	t.Run("consistency across calls", func(t *testing.T) {
		provider := newImpl(t)
		defer provider.Close()

		identity1, err1 := provider.GetServiceIdentity()
		identity2, err2 := provider.GetServiceIdentity()
		
		// Both calls should have same error status
		if (err1 == nil) != (err2 == nil) {
			t.Error("GetServiceIdentity calls returned inconsistent error status")
			return
		}
		
		// If both succeeded, results should be consistent
		if err1 == nil && err2 == nil {
			if identity1.Name != identity2.Name {
				t.Error("GetServiceIdentity returned inconsistent Name")
			}
			if identity1.Domain != identity2.Domain {
				t.Error("GetServiceIdentity returned inconsistent Domain")
			}
			if identity1.URI != identity2.URI {
				t.Error("GetServiceIdentity returned inconsistent URI")
			}
		}
	})
}