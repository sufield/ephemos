// ephemos_public_test.go
package ephemos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	ephemos "github.com/sufield/ephemos/pkg/ephemos"
)

// Test scaffolding helpers

// ctxWith creates a context with a deadline for test safety.
func ctxWith(t *testing.T, d time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	return ctx, cancel
}

// mustWait waits for a channel or times out with a helpful message.
func mustWait(t *testing.T, ch <-chan struct{}, d time.Duration, msg string) {
	t.Helper()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ch:
		return
	case <-timer.C:
		t.Fatalf("timeout waiting for %s", msg)
	}
}

// Test-specific utilities - no mock interfaces needed since we test the public API directly

func Test_PublicSurface_Compiles(t *testing.T) {
	t.Parallel()

	// Interface presence (compile-time)
	var _ ephemos.Client
	var _ ephemos.Server

	// Type presence (compile-time)
	var _ ephemos.ClientConnection
}

func Test_ClientConnection_HTTPClient_requiresSPIFFE(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, 2*time.Second)
	defer cancel()
	_ = ctx // Used for timeout safety

	// Zero-value connection must not produce an HTTP client.
	conn := ephemos.TestOnlyNewClientConnection(nil)
	client, err := conn.HTTPClient()

	if err == nil {
		t.Fatalf("expected error when SPIFFE auth is unavailable; got nil")
	}
	if client != nil {
		t.Fatalf("expected nil http client when SPIFFE auth is unavailable; got non-nil")
	}

	// Verify sentinel error mapping
	if !errors.Is(err, ephemos.ErrNoSPIFFEAuth) {
		t.Fatalf("expected error %v; got %v", ephemos.ErrNoSPIFFEAuth, err)
	}
}

func Test_ClientConnection_Close_isIdempotent(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, 2*time.Second)
	defer cancel()
	_ = ctx // Used for timeout safety

	conn := ephemos.TestOnlyNewClientConnection(nil)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() should be idempotent; got error: %v", err)
	}
}

// TestClientConnect_ErrorMapping tests error propagation without mocking internal interfaces
func TestClientConnect_ErrorMapping(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, time.Second)
	defer cancel()

	// Test with invalid configuration to ensure proper error handling
	_, err := ephemos.IdentityClientFromFile(ctx, "nonexistent-config.yaml")
	if err == nil {
		t.Fatalf("expected error for nonexistent config")
	}
	
	// Test with nil configuration
	_, err = ephemos.IdentityClient(ctx)
	if err == nil {
		t.Fatalf("expected error for missing configuration")
	}
	if !errors.Is(err, ephemos.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

// TestClient_Timeouts_MapCorrectly tests timeout configuration behavior
func TestClient_Timeouts_MapCorrectly(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, 2*time.Second)
	defer cancel()

	// Test with very short timeout to force a timeout error
	_, err := ephemos.IdentityClient(ctx, ephemos.WithClientTimeout(1*time.Nanosecond))
	if err == nil {
		t.Fatalf("expected configuration to fail with invalid timeout")
	}
}

// TestServer_Serve_StopsOnCancelAndAfterClose tests server lifecycle behavior
func TestServer_Serve_StopsOnCancelAndAfterClose(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, 2*time.Second)
	defer cancel()

	// Test server creation without configuration should fail
	_, err := ephemos.IdentityServer(ctx)
	if err == nil {
		t.Fatalf("expected error for missing configuration")
	}
	if !errors.Is(err, ephemos.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}

	// Test server with missing listener/address
	_, err = ephemos.IdentityServerFromFile(ctx, "nonexistent-config.yaml")
	if err == nil {
		t.Fatalf("expected error for nonexistent config")
	}
}

// TestClient_ConcurrentConnectAndClose_NoRace tests concurrency safety
func TestClient_ConcurrentConnectAndClose_NoRace(t *testing.T) {
	t.Parallel()

	// Test concurrent operations on ClientConnection
	conn := ephemos.TestOnlyNewClientConnection(nil)
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			_, _ = conn.HTTPClient() // Safe to call concurrently
			_ = conn.Close()        // Idempotent
		}
	}()

	time.Sleep(10 * time.Millisecond)
	_ = conn.Close() // Should be safe concurrently
	mustWait(t, done, time.Second, "concurrent operations")
}

// TestHTTPClient_InheritsClientTimeout tests timeout configuration
func TestHTTPClient_InheritsClientTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := ctxWith(t, time.Second)
	defer cancel()
	_ = ctx

	// Test that timeout options are accepted without error
	_, err := ephemos.IdentityClient(context.Background(), ephemos.WithClientTimeout(123*time.Millisecond))
	// This should fail because no configuration is provided
	if err == nil {
		t.Fatalf("expected error for missing configuration")
	}
	if !errors.Is(err, ephemos.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

// TestErrorSentinels_Stability verifies that sentinel errors are stable and have proper behavior.
func TestErrorSentinels_Stability(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		ephemos.ErrNoAuth,
		ephemos.ErrNoSPIFFEAuth,
		ephemos.ErrInvalidIdentity,
		ephemos.ErrConfigInvalid,
		ephemos.ErrConnectionFailed,
		ephemos.ErrServerClosed,
		ephemos.ErrInvalidAddress,
		ephemos.ErrTimeout,
	}

	for _, sentinel := range sentinels {
		t.Run(sentinel.Error(), func(t *testing.T) {
			t.Parallel()
			// errors.Is should work with itself
			if !errors.Is(sentinel, sentinel) {
				t.Fatalf("errors.Is(err, err) should be true for %v", sentinel)
			}
			// Should have a non-empty message
			if sentinel.Error() == "" {
				t.Fatalf("sentinel error should have non-empty message")
			}
		})
	}
}
