// Package transportprovider provides contract test suites for TransportProvider implementations.
// These tests ensure that all TransportProvider adapters behave consistently.
package transportprovider

import (
	"testing"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Factory creates a new TransportProvider implementation for testing.
type Factory func(t *testing.T) ports.TransportProvider

// TestDeps provides test dependencies needed by transport providers.
type TestDeps struct {
	Certificate *domain.Certificate
	TrustBundle *domain.TrustBundle
	AuthPolicy  *domain.AuthenticationPolicy
}

// Run executes the complete contract test suite against any TransportProvider implementation.
func Run(t *testing.T, newImpl Factory, deps TestDeps) {
	t.Helper()
	t.Run("create server", func(t *testing.T) {
		provider := newImpl(t)

		server, err := provider.CreateServer(deps.Certificate, deps.TrustBundle, deps.AuthPolicy)
		// Contract: Either returns functional server or expected error
		if err != nil {
			t.Logf("CreateServer returned error (may be expected): %v", err)
			return
		}

		if server == nil {
			t.Fatal("CreateServer returned nil server without error")
		}

		// Server should handle Stop gracefully
		if err := server.Stop(); err != nil {
			t.Logf("Server Stop returned: %v", err)
		}
	})

	t.Run("create client", func(t *testing.T) {
		provider := newImpl(t)

		client, err := provider.CreateClient(deps.Certificate, deps.TrustBundle, deps.AuthPolicy)
		// Contract: Either returns functional client or expected error
		if err != nil {
			t.Logf("CreateClient returned error (may be expected): %v", err)
			return
		}

		if client == nil {
			t.Fatal("CreateClient returned nil client without error")
		}

		// Client should handle Close gracefully
		if err := client.Close(); err != nil {
			t.Logf("Client Close returned: %v", err)
		}
	})

	t.Run("nil parameters rejected", func(t *testing.T) {
		provider := newImpl(t)

		// CreateServer with nil parameters should error
		if _, err := provider.CreateServer(nil, nil, nil); err == nil {
			t.Error("CreateServer(nil, nil, nil) should return error")
		}

		// CreateClient with nil parameters should error
		if _, err := provider.CreateClient(nil, nil, nil); err == nil {
			t.Error("CreateClient(nil, nil, nil) should return error")
		}
	})
}

// ServerFactory creates a Server implementation for testing.
type ServerFactory func(t *testing.T) ports.ServerPort

// RunServerSuite tests Server interface compliance.
func RunServerSuite(t *testing.T, newServer ServerFactory) {
	t.Helper()
	t.Run("register service validation", func(t *testing.T) {
		server := newServer(t)
		defer func() {
			if err := server.Stop(); err != nil {
				t.Logf("Failed to stop server: %v", err)
			}
		}()

		// RegisterService with nil should error
		if err := server.RegisterService(nil); err == nil {
			t.Error("RegisterService(nil) should return error")
		}
	})

	t.Run("start validation", func(t *testing.T) {
		server := newServer(t)
		defer func() {
			if err := server.Stop(); err != nil {
				t.Logf("Failed to stop server: %v", err)
			}
		}()

		// Start with nil listener should error
		if err := server.Start(nil); err == nil {
			t.Error("Start(nil) should return error")
		}
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		server := newServer(t)

		// First stop
		if err := server.Stop(); err != nil {
			t.Logf("First Stop() returned: %v", err)
		}

		// Second stop should be safe
		if err := server.Stop(); err != nil {
			t.Logf("Second Stop() returned: %v", err)
		}
	})
}

// ClientFactory creates a Client implementation for testing.
type ClientFactory func(t *testing.T) ports.ClientPort

// RunClientSuite tests Client interface compliance.
func RunClientSuite(t *testing.T, newClient ClientFactory) {
	t.Helper()
	t.Run("connect validation", func(t *testing.T) {
		client := newClient(t)
		defer func() {
			if err := client.Close(); err != nil {
				t.Logf("Failed to close client: %v", err)
			}
		}()

		// Connect with empty parameters should error
		if _, err := client.Connect("", ""); err == nil {
			t.Error("Connect(\"\", \"\") should return error")
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		client := newClient(t)

		// First close
		if err := client.Close(); err != nil {
			t.Logf("First Close() returned: %v", err)
		}

		// Second close should be safe
		if err := client.Close(); err != nil {
			t.Logf("Second Close() returned: %v", err)
		}
	})
}
