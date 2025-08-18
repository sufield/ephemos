//go:build integration

// Package spiffe provides integration tests for SPIFFE adapters.
// These tests require a SPIFFE environment (SPIRE agent) to be running.
package spiffe

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// TestEnvironment represents the test environment setup
type TestEnvironment struct {
	SocketPath string
	Available  bool
}

// setupTestEnvironment checks if SPIFFE environment is available
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()
	
	// Check for SPIFFE socket environment variable
	socketPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if socketPath == "" {
		// Try default addresses
		defaultAddr, found := workloadapi.GetDefaultAddress()
		if found {
			socketPath = defaultAddr
		} else {
			socketPath = "unix:///tmp/spire-agent/public/api.sock"
		}
	}
	
	// Test if we can connect to the workload API
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	source, err := workloadapi.NewX509Source(
		ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(socketPath),
		),
	)
	
	available := err == nil
	if available {
		source.Close()
	}
	
	return &TestEnvironment{
		SocketPath: socketPath,
		Available:  available,
	}
}

// skipIfNoSPIFFE skips the test if SPIFFE environment is not available
func skipIfNoSPIFFE(t *testing.T, env *TestEnvironment) {
	t.Helper()
	if !env.Available {
		t.Skip("SPIFFE environment not available - skipping integration test")
	}
}

func TestIdentityDocumentAdapter_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	skipIfNoSPIFFE(t, env)
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := IdentityDocumentAdapterConfig{
		SocketPath: env.SocketPath,
		Logger:     logger,
	}
	
	adapter, err := NewIdentityDocumentAdapter(config)
	require.NoError(t, err)
	defer adapter.Close()
	
	ctx := context.Background()
	
	t.Run("GetServiceIdentity", func(t *testing.T) {
		identity, err := adapter.GetServiceIdentity(ctx)
		require.NoError(t, err)
		assert.NotNil(t, identity)
		assert.NotEmpty(t, identity.Name())
		assert.NotEmpty(t, identity.Domain())
		
		t.Logf("Service Identity: %s@%s", identity.Name(), identity.Domain())
	})
	
	t.Run("GetCertificate", func(t *testing.T) {
		cert, err := adapter.GetCertificate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, cert.Cert)
		assert.NotNil(t, cert.PrivateKey)
		
		t.Logf("Certificate Subject: %s", cert.Cert.Subject.String())
		t.Logf("Certificate Valid Until: %s", cert.Cert.NotAfter.String())
	})
	
	t.Run("GetIdentityDocument", func(t *testing.T) {
		doc, err := adapter.GetIdentityDocument(ctx)
		require.NoError(t, err)
		assert.NotNil(t, doc)
		assert.NotEmpty(t, doc.Subject())
		assert.True(t, doc.ValidUntil().After(time.Now()))
		
		t.Logf("Identity Document Subject: %s", doc.Subject())
		t.Logf("Identity Document Valid Until: %s", doc.ValidUntil().String())
	})
	
	t.Run("RefreshIdentity", func(t *testing.T) {
		err := adapter.RefreshIdentity(ctx)
		require.NoError(t, err)
		
		// Verify we can still get identity after refresh
		identity, err := adapter.GetServiceIdentity(ctx)
		require.NoError(t, err)
		assert.NotNil(t, identity)
	})
	
	t.Run("WatchIdentityChanges", func(t *testing.T) {
		watchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		
		updateChan, err := adapter.WatchIdentityChanges(watchCtx)
		require.NoError(t, err)
		
		// We should be able to get the channel even if no updates come
		assert.NotNil(t, updateChan)
		
		t.Log("Identity change watcher started successfully")
	})
}

func TestSpiffeBundleAdapter_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	skipIfNoSPIFFE(t, env)
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := SpiffeBundleAdapterConfig{
		SocketPath: env.SocketPath,
		Logger:     logger,
	}
	
	adapter, err := NewSpiffeBundleAdapter(config)
	require.NoError(t, err)
	defer adapter.Close()
	
	ctx := context.Background()
	
	t.Run("GetTrustBundle", func(t *testing.T) {
		bundle, err := adapter.GetTrustBundle(ctx)
		require.NoError(t, err)
		assert.NotNil(t, bundle)
		assert.Greater(t, bundle.Count(), 0)
		
		t.Logf("Trust Bundle contains %d CA certificates", bundle.Count())
	})
	
	t.Run("RefreshTrustBundle", func(t *testing.T) {
		err := adapter.RefreshTrustBundle(ctx)
		require.NoError(t, err)
		
		// Verify we can still get bundle after refresh
		bundle, err := adapter.GetTrustBundle(ctx)
		require.NoError(t, err)
		assert.NotNil(t, bundle)
	})
	
	t.Run("WatchTrustBundleChanges", func(t *testing.T) {
		watchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		
		updateChan, err := adapter.WatchTrustBundleChanges(watchCtx)
		require.NoError(t, err)
		
		// We should be able to get the channel even if no updates come
		assert.NotNil(t, updateChan)
		
		t.Log("Trust bundle change watcher started successfully")
	})
	
	t.Run("ValidateCertificateAgainstBundle", func(t *testing.T) {
		// First get a certificate from identity adapter
		identityConfig := IdentityDocumentAdapterConfig{
			SocketPath: env.SocketPath,
			Logger:     logger,
		}
		identityAdapter, err := NewIdentityDocumentAdapter(identityConfig)
		require.NoError(t, err)
		defer identityAdapter.Close()
		
		cert, err := identityAdapter.GetCertificate(ctx)
		require.NoError(t, err)
		
		// Now validate it against the trust bundle
		err = adapter.ValidateCertificateAgainstBundle(ctx, cert)
		require.NoError(t, err)
		
		t.Log("Certificate validated successfully against trust bundle")
	})
}

func TestTLSAdapter_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	skipIfNoSPIFFE(t, env)
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := TLSAdapterConfig{
		SocketPath: env.SocketPath,
		Logger:     logger,
	}
	
	adapter, err := NewTLSAdapter(config)
	require.NoError(t, err)
	defer adapter.Close()
	
	ctx := context.Background()
	
	t.Run("CreateClientTLSConfig", func(t *testing.T) {
		policy := &domain.AuthenticationPolicy{
			// Use default policy (AuthorizeAny)
		}
		
		tlsConfig, err := adapter.CreateClientTLSConfig(ctx, policy)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)
		assert.NotNil(t, tlsConfig.VerifyPeerCertificate)
		
		t.Log("Client TLS config created successfully")
	})
	
	t.Run("CreateServerTLSConfig", func(t *testing.T) {
		policy := &domain.AuthenticationPolicy{
			// Use default policy (AuthorizeAny)
		}
		
		tlsConfig, err := adapter.CreateServerTLSConfig(ctx, policy)
		require.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)
		assert.NotNil(t, tlsConfig.VerifyPeerCertificate)
		
		t.Log("Server TLS config created successfully")
	})
	
	t.Run("GetTLSAuthorizer", func(t *testing.T) {
		policy := &domain.AuthenticationPolicy{
			// Use default policy (AuthorizeAny)
		}
		
		authorizer, err := adapter.GetTLSAuthorizer(policy)
		require.NoError(t, err)
		assert.NotNil(t, authorizer)
		
		t.Log("TLS authorizer created successfully")
	})
}

func TestProvider_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	skipIfNoSPIFFE(t, env)
	
	config := &ports.AgentConfig{
		SocketPath: env.SocketPath,
	}
	
	provider, err := NewProvider(config)
	require.NoError(t, err)
	defer provider.Close()
	
	t.Run("GetServiceIdentity", func(t *testing.T) {
		identity, err := provider.GetServiceIdentity()
		require.NoError(t, err)
		assert.NotNil(t, identity)
		
		t.Logf("Provider Service Identity: %s@%s", identity.Name(), identity.Domain())
	})
	
	t.Run("GetCertificate", func(t *testing.T) {
		cert, err := provider.GetCertificate()
		require.NoError(t, err)
		assert.NotNil(t, cert)
		
		t.Logf("Provider Certificate Subject: %s", cert.Cert.Subject.String())
	})
	
	t.Run("GetTrustBundle", func(t *testing.T) {
		bundle, err := provider.GetTrustBundle()
		require.NoError(t, err)
		assert.NotNil(t, bundle)
		
		t.Logf("Provider Trust Bundle contains %d CA certificates", bundle.Count())
	})
	
	t.Run("GetIdentityDocument", func(t *testing.T) {
		doc, err := provider.GetIdentityDocument()
		require.NoError(t, err)
		assert.NotNil(t, doc)
		
		t.Logf("Provider Identity Document Subject: %s", doc.Subject())
	})
	
	t.Run("GetTLSConfig", func(t *testing.T) {
		ctx := context.Background()
		authorizer, err := provider.GetTLSConfig(ctx)
		require.NoError(t, err)
		assert.NotNil(t, authorizer)
		
		t.Log("Provider TLS config created successfully")
	})
	
	t.Run("GetSocketPath", func(t *testing.T) {
		socketPath := provider.GetSocketPath()
		assert.NotEmpty(t, socketPath)
		assert.Equal(t, env.SocketPath, socketPath)
		
		t.Logf("Provider Socket Path: %s", socketPath)
	})
}

// TestAdaptersImplementInterfaces verifies that adapters implement the required interfaces
func TestAdaptersImplementInterfaces(t *testing.T) {
	// These tests don't need SPIFFE environment - just compile-time interface checks
	
	t.Run("IdentityDocumentAdapter implements IdentityProviderPort", func(t *testing.T) {
		var _ ports.IdentityProviderPort = (*IdentityDocumentAdapter)(nil)
		t.Log("IdentityDocumentAdapter correctly implements IdentityProviderPort")
	})
	
	t.Run("SpiffeBundleAdapter implements BundleProviderPort", func(t *testing.T) {
		var _ ports.BundleProviderPort = (*SpiffeBundleAdapter)(nil)
		t.Log("SpiffeBundleAdapter correctly implements BundleProviderPort")
	})
}