package ephemos

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

type mockAuthenticatedServer struct {
	started chan struct{}
	addr    net.Addr
}

func newMockAuthenticatedServer() *mockAuthenticatedServer {
	return &mockAuthenticatedServer{started: make(chan struct{})}
}

func (m *mockAuthenticatedServer) HTTPHandler() http.Handler {
	return http.NewServeMux()
}

func (m *mockAuthenticatedServer) Serve(ctx context.Context, l net.Listener) error {
	m.addr = l.Addr()
	close(m.started)
	<-ctx.Done()
	return ctx.Err()
}

func (m *mockAuthenticatedServer) Close() error { return nil }

func (m *mockAuthenticatedServer) Addr() net.Addr { return m.addr }

func TestServerWrapperListenAndServeReleasesLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := newMockAuthenticatedServer()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv, err := IdentityServer(ctx, WithServerImpl(mock), WithListener(ln))
	if err != nil {
		t.Fatalf("IdentityServer: %v", err)
	}

	serveDone := make(chan error)
	go func() { serveDone <- srv.ListenAndServe(ctx) }()

	// Wait for Serve to start
	select {
	case <-mock.started:
	case <-time.After(time.Second):
		t.Fatalf("Serve did not start")
	}

	closeDone := make(chan error)
	go func() { closeDone <- srv.Close() }()

	// Close should return without deadlock
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Close did not return")
	}

	cancel()

	// ListenAndServe should return after context cancel
	select {
	case err := <-serveDone:
		if err != context.Canceled && err != nil {
			t.Fatalf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("ListenAndServe did not return")
	}
}
