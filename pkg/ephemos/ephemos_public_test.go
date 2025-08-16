// ephemos_public_test.go
package ephemos_test

import (
	"errors"
	"testing"

	ephemos "github.com/sufield/ephemos/pkg/ephemos"
)

func Test_PublicSurface_Compiles(t *testing.T) {
	t.Parallel()

	// Interface presence (compile-time)
	var _ ephemos.Client
	var _ ephemos.Server

	// Type presence (compile-time)
	var _ *ephemos.ClientConnection
}

func Test_ClientConnection_HTTPClient_requiresSPIFFE(t *testing.T) {
	t.Parallel()

	// Zero-value connection must not produce an HTTP client.
	conn := &ephemos.ClientConnection{}
	client, err := conn.HTTPClient()

	if err == nil {
		t.Fatalf("expected error when SPIFFE auth is unavailable; got nil")
	}
	if client != nil {
		t.Fatalf("expected nil http client when SPIFFE auth is unavailable; got non-nil")
	}

	// Prefer a typed error; fall back to substring until you expose one.
	// Recommended: expose ephemos.ErrNoSPIFFEAuth and use errors.Is below.
	var wants error = ephemos.ErrNoSPIFFEAuth

	if wants != nil && !errors.Is(err, wants) {
		t.Fatalf("expected error %v; got %v", wants, err)
	}
}

func Test_ClientConnection_Close_isIdempotent(t *testing.T) {
	t.Parallel()

	conn := &ephemos.ClientConnection{}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() should be idempotent; got error: %v", err)
	}
}
