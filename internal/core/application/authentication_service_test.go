package application_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/application"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports/mocks"
)

func TestNewAuthenticationService(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})
	
	t.Run("missing identity provider", func(t *testing.T) {
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		config := application.AuthenticationServiceConfig{
			BundleProvider: mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "identity provider is required")
		assert.Nil(t, service)
	})
	
	t.Run("missing bundle provider", func(t *testing.T) {
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bundle provider is required")
		assert.Nil(t, service)
	})
}

func TestAuthenticationService_GetValidatedIdentity(t *testing.T) {
	ctx := context.Background()
	
	t.Run("successful identity validation", func(t *testing.T) {
		// Create test identity document
		identityDoc := createTestIdentityDocument(t, "spiffe://example.org/test-service")
		trustBundle := createTestTrustBundle(t)
		
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockIdentityProvider.On("GetIdentityDocument", ctx).Return(identityDoc, nil)
		mockBundleProvider.On("GetTrustBundle", ctx).Return(trustBundle, nil)
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
			ExpiryThreshold:  5 * time.Minute,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		result, err := service.GetValidatedIdentity(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, identityDoc.Subject(), result.Subject())
		
		// Verify mock calls
		mockIdentityProvider.AssertExpectations(t)
		mockBundleProvider.AssertExpectations(t)
	})
	
	t.Run("identity provider error", func(t *testing.T) {
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockIdentityProvider.On("GetIdentityDocument", ctx).Return(nil, errors.New("provider error"))
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		result, err := service.GetValidatedIdentity(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get identity document")
		assert.Nil(t, result)
		
		// Verify mock calls
		mockIdentityProvider.AssertExpectations(t)
	})
	
	t.Run("identity expiring soon triggers refresh", func(t *testing.T) {
		// Create identity that expires in 3 minutes
		identityDoc := createTestIdentityDocumentWithExpiry(t, "spiffe://example.org/test-service", 3*time.Minute)
		refreshedDoc := createTestIdentityDocument(t, "spiffe://example.org/test-service")
		trustBundle := createTestTrustBundle(t)
		
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		// First call returns expiring identity
		mockIdentityProvider.On("GetIdentityDocument", ctx).Return(identityDoc, nil).Once()
		// Refresh is triggered
		mockIdentityProvider.On("RefreshIdentity", ctx).Return(nil).Once()
		// Second call returns refreshed identity
		mockIdentityProvider.On("GetIdentityDocument", ctx).Return(refreshedDoc, nil).Once()
		// Trust bundle for validation
		mockBundleProvider.On("GetTrustBundle", ctx).Return(trustBundle, nil)
		
		// Create service with 5 minute threshold
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
			ExpiryThreshold:  5 * time.Minute,
			MaxRetries:       1,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		result, err := service.GetValidatedIdentity(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should return the refreshed identity
		assert.Equal(t, refreshedDoc.Subject(), result.Subject())
		
		// Verify mock calls
		mockIdentityProvider.AssertExpectations(t)
		mockBundleProvider.AssertExpectations(t)
	})
}

func TestAuthenticationService_ValidatePeerIdentity(t *testing.T) {
	ctx := context.Background()
	
	t.Run("successful peer validation", func(t *testing.T) {
		// Create test certificate with specific SPIFFE ID
		peerCert := createTestCertificate(t, "spiffe://example.org/peer-service")
		trustBundle := createTestTrustBundle(t)
		
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockBundleProvider.On("GetTrustBundle", ctx).Return(trustBundle, nil)
		mockBundleProvider.On("ValidateCertificateAgainstBundle", ctx, peerCert).Return(nil)
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.ValidatePeerIdentity(ctx, peerCert, "spiffe://example.org/peer-service")
		assert.NoError(t, err)
		
		// Verify mock calls
		mockBundleProvider.AssertExpectations(t)
	})
	
	t.Run("nil peer certificate", func(t *testing.T) {
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.ValidatePeerIdentity(ctx, nil, "spiffe://example.org/peer-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "peer certificate is nil")
	})
	
	t.Run("identity mismatch", func(t *testing.T) {
		// Create test certificate with different SPIFFE ID
		peerCert := createTestCertificate(t, "spiffe://example.org/wrong-service")
		trustBundle := createTestTrustBundle(t)
		
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockBundleProvider.On("GetTrustBundle", ctx).Return(trustBundle, nil)
		mockBundleProvider.On("ValidateCertificateAgainstBundle", ctx, peerCert).Return(nil)
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.ValidatePeerIdentity(ctx, peerCert, "spiffe://example.org/peer-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "peer identity mismatch")
		
		// Verify mock calls
		mockBundleProvider.AssertExpectations(t)
	})
	
	t.Run("certificate not trusted", func(t *testing.T) {
		// Create test certificate
		peerCert := createTestCertificate(t, "spiffe://example.org/peer-service")
		trustBundle := createTestTrustBundle(t)
		
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockBundleProvider.On("GetTrustBundle", ctx).Return(trustBundle, nil)
		mockBundleProvider.On("ValidateCertificateAgainstBundle", ctx, peerCert).Return(errors.New("not trusted"))
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.ValidatePeerIdentity(ctx, peerCert, "spiffe://example.org/peer-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "peer certificate not trusted")
		
		// Verify mock calls
		mockBundleProvider.AssertExpectations(t)
	})
}

func TestAuthenticationService_RefreshCredentials(t *testing.T) {
	ctx := context.Background()
	
	t.Run("successful refresh", func(t *testing.T) {
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockIdentityProvider.On("RefreshIdentity", ctx).Return(nil)
		mockBundleProvider.On("RefreshTrustBundle", ctx).Return(nil)
		
		// Create service
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
			MaxRetries:       1,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.RefreshCredentials(ctx)
		assert.NoError(t, err)
		
		// Verify mock calls
		mockIdentityProvider.AssertExpectations(t)
		mockBundleProvider.AssertExpectations(t)
	})
	
	t.Run("identity refresh failure", func(t *testing.T) {
		// Setup mocks
		mockIdentityProvider := mocks.NewMockIdentityProviderPort()
		mockBundleProvider := mocks.NewMockBundleProviderPort()
		
		mockIdentityProvider.On("RefreshIdentity", ctx).Return(errors.New("refresh failed"))
		
		// Create service with single retry
		config := application.AuthenticationServiceConfig{
			IdentityProvider: mockIdentityProvider,
			BundleProvider:   mockBundleProvider,
			MaxRetries:       1,
		}
		
		service, err := application.NewAuthenticationService(config)
		require.NoError(t, err)
		
		// Test
		err = service.RefreshCredentials(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to refresh identity")
		
		// Verify mock calls
		mockIdentityProvider.AssertExpectations(t)
	})
}

// Helper functions

func createTestIdentityDocument(t *testing.T, spiffeID string) *domain.IdentityDocument {
	cert, key := createTestCertAndKey(t, spiffeID)
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	return doc
}

func createTestIdentityDocumentWithExpiry(t *testing.T, spiffeID string, timeUntilExpiry time.Duration) *domain.IdentityDocument {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(timeUntilExpiry),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	
	if spiffeID != "" {
		spiffeURI, err := url.Parse(spiffeID)
		require.NoError(t, err)
		template.URIs = []*url.URL{spiffeURI}
	}
	
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	doc, err := domain.NewIdentityDocument([]*x509.Certificate{cert}, key, nil)
	require.NoError(t, err)
	
	return doc
}

func createTestCertificate(t *testing.T, spiffeID string) *domain.Certificate {
	cert, key := createTestCertAndKey(t, spiffeID)
	
	domainCert, err := domain.NewCertificate(cert, key, nil)
	require.NoError(t, err)
	
	return domainCert
}

func createTestCertAndKey(t *testing.T, spiffeID string) (*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-service",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	
	if spiffeID != "" {
		spiffeURI, err := url.Parse(spiffeID)
		require.NoError(t, err)
		template.URIs = []*url.URL{spiffeURI}
	}
	
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	return cert, key
}

func createTestTrustBundle(t *testing.T) *domain.TrustBundle {
	// Create a test CA certificate
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	
	bundle, err := domain.NewTrustBundle([]*x509.Certificate{cert})
	require.NoError(t, err)
	
	return bundle
}