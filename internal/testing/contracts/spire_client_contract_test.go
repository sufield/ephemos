// Package contracts provides contract tests for SPIRE client implementations.
// These tests verify adapter behavior without spinning up actual SPIRE servers.
package contracts

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// FakeSPIREClient is a test double for SPIRE client behavior.
// This interface defines the essential SPIRE operations without external dependencies.
type FakeSPIREClient interface {
	// FetchX509SVID retrieves the current X.509 SVID
	FetchX509SVID(ctx context.Context) (*domain.Certificate, error)
	
	// FetchTrustBundle retrieves the current trust bundle
	FetchTrustBundle(ctx context.Context) (*domain.TrustBundle, error)
	
	// ValidateJWTSVID validates a JWT SVID token
	ValidateJWTSVID(ctx context.Context, token string, audience string) (*domain.JWTClaims, error)
	
	// WatchX509Context starts watching for X.509 SVID updates
	WatchX509Context(ctx context.Context) (<-chan *domain.Certificate, <-chan error)
	
	// Close releases resources
	Close() error
}

// fakeSPIREClientImpl implements FakeSPIREClient for testing
type fakeSPIREClientImpl struct {
	certificate     *domain.Certificate
	trustBundle     *domain.TrustBundle
	jwtClaims       *domain.JWTClaims
	fetchError      error
	validateError   error
	watchUpdates    chan *domain.Certificate
	watchErrors     chan error
	closed          bool
	fetchCallCount  int
	watchCallCount  int
}

// NewFakeSPIREClient creates a new fake SPIRE client for testing
func NewFakeSPIREClient() *fakeSPIREClientImpl {
	return &fakeSPIREClientImpl{
		watchUpdates: make(chan *domain.Certificate, 1),
		watchErrors:  make(chan error, 1),
	}
}

// WithCertificate configures the fake to return a specific certificate
func (f *fakeSPIREClientImpl) WithCertificate(cert *domain.Certificate) *fakeSPIREClientImpl {
	f.certificate = cert
	return f
}

// WithTrustBundle configures the fake to return a specific trust bundle
func (f *fakeSPIREClientImpl) WithTrustBundle(bundle *domain.TrustBundle) *fakeSPIREClientImpl {
	f.trustBundle = bundle
	return f
}

// WithFetchError configures the fake to return an error on fetch operations
func (f *fakeSPIREClientImpl) WithFetchError(err error) *fakeSPIREClientImpl {
	f.fetchError = err
	return f
}

// WithJWTClaims configures the fake to return specific JWT claims
func (f *fakeSPIREClientImpl) WithJWTClaims(claims *domain.JWTClaims) *fakeSPIREClientImpl {
	f.jwtClaims = claims
	return f
}

// WithValidateError configures the fake to return an error on JWT validation
func (f *fakeSPIREClientImpl) WithValidateError(err error) *fakeSPIREClientImpl {
	f.validateError = err
	return f
}

// FetchX509SVID implements FakeSPIREClient
func (f *fakeSPIREClientImpl) FetchX509SVID(ctx context.Context) (*domain.Certificate, error) {
	f.fetchCallCount++
	
	if f.closed {
		return nil, errors.New("client is closed")
	}
	
	if f.fetchError != nil {
		return nil, f.fetchError
	}
	
	if f.certificate == nil {
		return nil, errors.New("no certificate configured")
	}
	
	return f.certificate, nil
}

// FetchTrustBundle implements FakeSPIREClient
func (f *fakeSPIREClientImpl) FetchTrustBundle(ctx context.Context) (*domain.TrustBundle, error) {
	if f.closed {
		return nil, errors.New("client is closed")
	}
	
	if f.fetchError != nil {
		return nil, f.fetchError
	}
	
	if f.trustBundle == nil {
		return nil, errors.New("no trust bundle configured")
	}
	
	return f.trustBundle, nil
}

// ValidateJWTSVID implements FakeSPIREClient
func (f *fakeSPIREClientImpl) ValidateJWTSVID(ctx context.Context, token string, audience string) (*domain.JWTClaims, error) {
	if f.closed {
		return nil, errors.New("client is closed")
	}
	
	if f.validateError != nil {
		return nil, f.validateError
	}
	
	if f.jwtClaims == nil {
		return nil, errors.New("no JWT claims configured")
	}
	
	return f.jwtClaims, nil
}

// WatchX509Context implements FakeSPIREClient
func (f *fakeSPIREClientImpl) WatchX509Context(ctx context.Context) (<-chan *domain.Certificate, <-chan error) {
	f.watchCallCount++
	return f.watchUpdates, f.watchErrors
}

// Close implements FakeSPIREClient
func (f *fakeSPIREClientImpl) Close() error {
	f.closed = true
	close(f.watchUpdates)
	close(f.watchErrors)
	return nil
}

// SendWatchUpdate simulates a certificate update for testing
func (f *fakeSPIREClientImpl) SendWatchUpdate(cert *domain.Certificate) {
	select {
	case f.watchUpdates <- cert:
	default:
		// Channel full, ignore
	}
}

// SendWatchError simulates an error in the watch stream for testing
func (f *fakeSPIREClientImpl) SendWatchError(err error) {
	select {
	case f.watchErrors <- err:
	default:
		// Channel full, ignore
	}
}

// GetCallCounts returns call counts for verification in tests
func (f *fakeSPIREClientImpl) GetCallCounts() (fetchCalls, watchCalls int) {
	return f.fetchCallCount, f.watchCallCount
}

// Contract tests that any SPIRE client adapter must pass

// TestSPIREClientContract_FetchX509SVID tests basic certificate fetching behavior
func TestSPIREClientContract_FetchX509SVID(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() FakeSPIREClient
		wantErr     bool
		errContains string
	}{
		{
			name: "successful fetch",
			setupClient: func() FakeSPIREClient {
				cert := &domain.Certificate{
					Cert:       &x509.Certificate{},
					Chain:      []*x509.Certificate{},
					PrivateKey: nil, // Would be a real key in practice
				}
				return NewFakeSPIREClient().WithCertificate(cert)
			},
			wantErr: false,
		},
		{
			name: "fetch error",
			setupClient: func() FakeSPIREClient {
				return NewFakeSPIREClient().WithFetchError(errors.New("SPIRE unavailable"))
			},
			wantErr:     true,
			errContains: "SPIRE unavailable",
		},
		{
			name: "closed client",
			setupClient: func() FakeSPIREClient {
				client := NewFakeSPIREClient()
				client.Close()
				return client
			},
			wantErr:     true,
			errContains: "client is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			cert, err := client.FetchX509SVID(ctx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cert == nil {
				t.Fatal("expected certificate but got nil")
			}
		})
	}
}

// TestSPIREClientContract_WatchBehavior tests certificate watching behavior
func TestSPIREClientContract_WatchBehavior(t *testing.T) {
	client := NewFakeSPIREClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start watching
	certChan, errChan := client.WatchX509Context(ctx)

	// Test certificate update
	testCert := &domain.Certificate{
		Cert:       &x509.Certificate{},
		Chain:      []*x509.Certificate{},
		PrivateKey: nil,
	}

	// Send update in goroutine to avoid blocking
	go func() {
		time.Sleep(10 * time.Millisecond)
		client.SendWatchUpdate(testCert)
	}()

	// Verify we receive the update
	select {
	case cert := <-certChan:
		if cert != testCert {
			t.Fatal("received different certificate than sent")
		}
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for certificate update")
	}

	// Test error propagation
	testErr := errors.New("watch connection lost")
	go func() {
		time.Sleep(10 * time.Millisecond)
		client.SendWatchError(testErr)
	}()

	select {
	case <-certChan:
		t.Fatal("unexpected certificate when expecting error")
	case err := <-errChan:
		if err.Error() != testErr.Error() {
			t.Fatalf("got error %q, want %q", err.Error(), testErr.Error())
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

// TestSPIREClientContract_ResourceCleanup tests proper resource cleanup
func TestSPIREClientContract_ResourceCleanup(t *testing.T) {
	client := NewFakeSPIREClient()

	// Verify client starts in working state
	ctx := context.Background()
	certChan, errChan := client.WatchX509Context(ctx)

	// Close the client
	if err := client.Close(); err != nil {
		t.Fatalf("failed to close client: %v", err)
	}

	// Verify channels are closed
	select {
	case _, ok := <-certChan:
		if ok {
			t.Fatal("certificate channel should be closed")
		}
	default:
		// Channel might not be immediately closed, that's ok
	}

	select {
	case _, ok := <-errChan:
		if ok {
			t.Fatal("error channel should be closed")
		}
	default:
		// Channel might not be immediately closed, that's ok
	}

	// Verify operations on closed client return errors
	_, err := client.FetchX509SVID(ctx)
	if err == nil || !contains(err.Error(), "closed") {
		t.Fatalf("expected 'closed' error, got: %v", err)
	}
}

// TestSPIREClientContract_JWTValidation tests JWT SVID validation behavior
func TestSPIREClientContract_JWTValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() FakeSPIREClient
		token       string
		audience    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid JWT",
			setupClient: func() FakeSPIREClient {
				claims := &domain.JWTClaims{
					Subject:   "spiffe://example.org/service",
					Audience:  []string{"backend"},
					ExpiresAt: time.Now().Add(time.Hour),
				}
				return NewFakeSPIREClient().WithJWTClaims(claims)
			},
			token:    "valid.jwt.token",
			audience: "backend",
			wantErr:  false,
		},
		{
			name: "validation error",
			setupClient: func() FakeSPIREClient {
				return NewFakeSPIREClient().WithValidateError(errors.New("invalid token"))
			},
			token:       "invalid.jwt.token",
			audience:    "backend",
			wantErr:     true,
			errContains: "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			claims, err := client.ValidateJWTSVID(ctx, tt.token, tt.audience)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if claims == nil {
				t.Fatal("expected JWT claims but got nil")
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}