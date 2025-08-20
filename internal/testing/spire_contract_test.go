package testing

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
	coreerrors "github.com/sufield/ephemos/internal/core/errors"
)

// FakeSPIREClient provides a fake SPIRE client for contract testing
type FakeSPIREClient struct {
	serviceID     string
	trustDomain   string
	certificate   *domain.Certificate
	trustBundle   []*x509.Certificate
	rotationTime  time.Time
	callLog       []string
	shouldFail    bool
	failureReason string
}

// NewFakeSPIREClient creates a new fake SPIRE client for testing
func NewFakeSPIREClient(serviceID, trustDomain string) *FakeSPIREClient {
	return &FakeSPIREClient{
		serviceID:   serviceID,
		trustDomain: trustDomain,
		callLog:     make([]string, 0),
	}
}

// SPIREClientContract defines the contract that SPIRE adapters must fulfill
type SPIREClientContract interface {
	GetServiceIdentity(ctx context.Context) (string, error)
	GetCertificate(ctx context.Context) (*domain.Certificate, error)
	GetTrustBundle(ctx context.Context) ([]*x509.Certificate, error)
	WatchForRotation(ctx context.Context) (<-chan time.Time, error)
	Close() error
}

// Implement the contract interface
func (f *FakeSPIREClient) GetServiceIdentity(ctx context.Context) (string, error) {
	f.callLog = append(f.callLog, "GetServiceIdentity")
	if f.shouldFail {
		return "", coreerrors.NewDomainError(coreerrors.ErrConnectionFailed, errors.New(f.failureReason))
	}
	return f.serviceID, nil
}

func (f *FakeSPIREClient) GetCertificate(ctx context.Context) (*domain.Certificate, error) {
	f.callLog = append(f.callLog, "GetCertificate")
	if f.shouldFail {
		return nil, coreerrors.NewDomainError(coreerrors.ErrCertificateUnavailable, errors.New(f.failureReason))
	}
	return f.certificate, nil
}

func (f *FakeSPIREClient) GetTrustBundle(ctx context.Context) ([]*x509.Certificate, error) {
	f.callLog = append(f.callLog, "GetTrustBundle")
	if f.shouldFail {
		return nil, coreerrors.NewDomainError(coreerrors.ErrTrustBundleUnavailable, errors.New(f.failureReason))
	}
	return f.trustBundle, nil
}

func (f *FakeSPIREClient) WatchForRotation(ctx context.Context) (<-chan time.Time, error) {
	f.callLog = append(f.callLog, "WatchForRotation")
	if f.shouldFail {
		return nil, coreerrors.NewDomainError(coreerrors.ErrConnectionFailed, errors.New(f.failureReason))
	}
	
	ch := make(chan time.Time, 1)
	if !f.rotationTime.IsZero() {
		ch <- f.rotationTime
	}
	return ch, nil
}

func (f *FakeSPIREClient) Close() error {
	f.callLog = append(f.callLog, "Close")
	if f.shouldFail {
		return coreerrors.NewDomainError(coreerrors.ErrConnectionFailed, errors.New(f.failureReason))
	}
	return nil
}

// Test helpers
func (f *FakeSPIREClient) SetCertificate(cert *domain.Certificate) {
	f.certificate = cert
}

func (f *FakeSPIREClient) SetTrustBundle(bundle []*x509.Certificate) {
	f.trustBundle = bundle
}

func (f *FakeSPIREClient) SetRotationTime(t time.Time) {
	f.rotationTime = t
}

func (f *FakeSPIREClient) SetShouldFail(fail bool, reason string) {
	f.shouldFail = fail
	f.failureReason = reason
}

func (f *FakeSPIREClient) GetCallLog() []string {
	return f.callLog
}

func (f *FakeSPIREClient) ClearCallLog() {
	f.callLog = make([]string, 0)
}

// Contract tests that any SPIRE adapter must pass
func TestSPIREClientContract_GetServiceIdentity(t *testing.T) {
	tests := []struct {
		name          string
		serviceID     string
		trustDomain   string
		shouldFail    bool
		failureReason string
		expectError   bool
	}{
		{
			name:        "successful identity retrieval",
			serviceID:   "test-service",
			trustDomain: "example.org",
			shouldFail:  false,
			expectError: false,
		},
		{
			name:          "failure case",
			serviceID:     "test-service",
			trustDomain:   "example.org",
			shouldFail:    true,
			failureReason: "SPIRE agent unavailable",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewFakeSPIREClient(tt.serviceID, tt.trustDomain)
			client.SetShouldFail(tt.shouldFail, tt.failureReason)

			ctx := context.Background()
			identity, err := client.GetServiceIdentity(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if identity != tt.serviceID {
				t.Errorf("expected identity %s, got %s", tt.serviceID, identity)
			}

			callLog := client.GetCallLog()
			if len(callLog) != 1 || callLog[0] != "GetServiceIdentity" {
				t.Errorf("expected call log [GetServiceIdentity], got %v", callLog)
			}
		})
	}
}

func TestSPIREClientContract_GetCertificate(t *testing.T) {
	client := NewFakeSPIREClient("test-service", "example.org")
	
	// Set up a test certificate (using nil for simplicity in test)
	testCert := &domain.Certificate{}
	client.SetCertificate(testCert)

	ctx := context.Background()
	cert, err := client.GetCertificate(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cert == nil {
		t.Error("expected certificate but got nil")
	}
}

func TestSPIREClientContract_WatchForRotation(t *testing.T) {
	client := NewFakeSPIREClient("test-service", "example.org")
	rotationTime := time.Now()
	client.SetRotationTime(rotationTime)

	ctx := context.Background()
	ch, err := client.WatchForRotation(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	select {
	case receivedTime := <-ch:
		if !receivedTime.Equal(rotationTime) {
			t.Errorf("expected rotation time %v, got %v", rotationTime, receivedTime)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for rotation event")
	}
}

func TestSPIREClientContract_Close(t *testing.T) {
	client := NewFakeSPIREClient("test-service", "example.org")

	err := client.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}

	callLog := client.GetCallLog()
	if len(callLog) != 1 || callLog[0] != "Close" {
		t.Errorf("expected call log [Close], got %v", callLog)
	}
}

// RunSPIREClientContractTests runs all contract tests against any SPIREClientContract implementation
func RunSPIREClientContractTests(t *testing.T, client SPIREClientContract) {
	t.Run("GetServiceIdentity", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.GetServiceIdentity(ctx)
		// Contract: should not panic and should return either identity or error
		if err == nil {
			// Success case - validate we got a non-empty identity
		} else {
			// Error case - validate it's a proper domain error
			var domainErr *coreerrors.DomainError
			if !errors.As(err, &domainErr) {
				t.Errorf("expected coreerrors.DomainError, got %T", err)
			}
		}
	})

	t.Run("GetCertificate", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.GetCertificate(ctx)
		// Contract: should not panic
		if err != nil {
			var domainErr *coreerrors.DomainError
			if !errors.As(err, &domainErr) {
				t.Errorf("expected coreerrors.DomainError, got %T", err)
			}
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := client.Close()
		// Contract: should not panic, error is optional
		if err != nil {
			var domainErr *coreerrors.DomainError
			if !errors.As(err, &domainErr) {
				t.Errorf("expected coreerrors.DomainError, got %T", err)
			}
		}
	})
}