// Package identityprovider provides contract test suites for IdentityProvider implementations.
// These tests ensure that all IdentityProvider adapters behave consistently.
package identityprovider

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Factory creates a new IdentityProvider implementation for testing.
type Factory func(t *testing.T) ports.IdentityProvider

// Run executes the complete contract test suite against any IdentityProvider implementation.
func Run(t *testing.T, newImpl Factory) {
	t.Helper()
	t.Run("get service identity", func(t *testing.T) {
		testGetServiceIdentity(t, newImpl)
	})

	t.Run("get certificate", func(t *testing.T) {
		testGetCertificate(t, newImpl)
	})

	t.Run("get trust bundle", func(t *testing.T) {
		testGetTrustBundle(t, newImpl)
	})

	t.Run("get SVID", func(t *testing.T) {
		testGetSVID(t, newImpl)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		testCloseIdempotent(t, newImpl)
	})

	t.Run("consistency across calls", func(t *testing.T) {
		testConsistencyAcrossCalls(t, newImpl)
	})
}

// testGetServiceIdentity tests getting service identity.
func testGetServiceIdentity(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)
	defer closeProvider(t, provider)

	identity, err := provider.GetServiceIdentity()
	// Contract: Either returns valid identity or expected error
	if err != nil {
		t.Logf("GetServiceIdentity returned error (may be expected): %v", err)
		return
	}

	assertValidIdentity(t, identity)
}

// testGetCertificate tests getting certificate.
func testGetCertificate(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)
	defer closeProvider(t, provider)

	cert, err := provider.GetCertificate()
	// Contract: Either returns valid certificate or expected error
	if err != nil {
		t.Logf("GetCertificate returned error (may be expected): %v", err)
		return
	}

	assertValidCertificate(t, cert)
}

// testGetTrustBundle tests getting trust bundle.
func testGetTrustBundle(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)
	defer closeProvider(t, provider)

	bundle, err := provider.GetTrustBundle()
	// Contract: Either returns valid bundle or expected error
	if err != nil {
		t.Logf("GetTrustBundle returned error (may be expected): %v", err)
		return
	}

	assertValidTrustBundle(t, bundle)
}

// testGetSVID tests getting SVID.
func testGetSVID(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)
	defer closeProvider(t, provider)

	svid, err := provider.GetSVID()
	// Contract: Either returns valid SVID or expected error
	if err != nil {
		t.Logf("GetSVID returned error (may be expected): %v", err)
		return
	}

	assertValidSVID(t, svid)
}

// testCloseIdempotent tests that Close is idempotent.
func testCloseIdempotent(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)

	// First close should succeed
	if err := provider.Close(); err != nil {
		t.Errorf("First Close() failed: %v", err)
	}

	// Second close should be safe (idempotent)
	if err := provider.Close(); err != nil {
		t.Errorf("Second Close() failed (not idempotent): %v", err)
	}
}

// testConsistencyAcrossCalls tests consistency across multiple calls.
func testConsistencyAcrossCalls(t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)
	defer closeProvider(t, provider)

	identity1, err1 := provider.GetServiceIdentity()
	identity2, err2 := provider.GetServiceIdentity()

	// Both calls should have same error status
	if (err1 == nil) != (err2 == nil) {
		t.Error("GetServiceIdentity calls returned inconsistent error status")
		return
	}

	// If both succeeded, results should be consistent
	if err1 == nil && err2 == nil {
		assertIdentitiesConsistent(t, identity1, identity2)
	}
}

// closeProvider closes the provider with error logging.
func closeProvider(t *testing.T, provider ports.IdentityProvider) {
	t.Helper()
	if err := provider.Close(); err != nil {
		t.Logf("Failed to close provider: %v", err)
	}
}

// assertValidIdentity asserts that a service identity is valid.
func assertValidIdentity(t *testing.T, identity spiffeid.ID) {
	t.Helper()
	if identity.IsZero() {
		t.Fatal("GetServiceIdentity returned zero identity without error")
	}

	if identity.Path() == "" || identity.Path() == "/" {
		t.Error("SPIFFE ID path should not be empty or just root")
	}

	if identity.TrustDomain().IsZero() {
		t.Error("SPIFFE ID trust domain should not be empty")
	}

	if identity.String() == "" {
		t.Error("SPIFFE ID string representation should not be empty")
	}
}

// assertValidCertificate asserts that a certificate is valid.
func assertValidCertificate(t *testing.T, cert *domain.Certificate) {
	t.Helper()
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
}

// assertValidTrustBundle asserts that an x509bundle.Bundle is valid.
func assertValidTrustBundle(t *testing.T, bundle *x509bundle.Bundle) {
	t.Helper()
	if bundle == nil {
		t.Fatal("GetTrustBundle returned nil bundle without error")
	}

	authorities := bundle.X509Authorities()
	if len(authorities) == 0 {
		t.Error("Bundle.X509Authorities should not be empty")
	}

	for i, cert := range authorities {
		if cert == nil {
			t.Errorf("Bundle.X509Authorities[%d] should not be nil", i)
		}
	}

	if bundle.TrustDomain().String() == "" {
		t.Error("Bundle.TrustDomain should not be empty")
	}
}

// assertValidSVID asserts that an SVID is valid.
func assertValidSVID(t *testing.T, svid *x509svid.SVID) {
	t.Helper()
	if svid == nil {
		t.Fatal("GetSVID returned nil SVID without error")
	}

	if svid.ID.String() == "" {
		t.Error("SVID.ID should not be empty")
	}

	if len(svid.Certificates) == 0 {
		t.Error("SVID.Certificates should not be empty")
	}

	if svid.PrivateKey == nil {
		t.Error("SVID.PrivateKey should not be nil")
	}

	if len(svid.Certificates) > 0 {
		cert := svid.Certificates[0]
		if cert.NotBefore.IsZero() {
			t.Error("SVID certificate NotBefore should not be zero")
		}

		if cert.NotAfter.IsZero() {
			t.Error("SVID certificate NotAfter should not be zero")
		}

		if cert.NotAfter.Before(cert.NotBefore) {
			t.Error("SVID certificate NotAfter should be after NotBefore")
		}
	}
}

// assertIdentitiesConsistent asserts that two identities are consistent.
func assertIdentitiesConsistent(t *testing.T, identity1, identity2 spiffeid.ID) {
	t.Helper()
	if identity1.Path() != identity2.Path() {
		t.Error("GetServiceIdentity returned inconsistent Path")
	}
	if identity1.TrustDomain() != identity2.TrustDomain() {
		t.Error("GetServiceIdentity returned inconsistent TrustDomain")
	}
	if identity1.String() != identity2.String() {
		t.Error("GetServiceIdentity returned inconsistent URI")
	}
}
