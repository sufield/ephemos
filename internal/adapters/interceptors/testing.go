package interceptors

import (
	"context"
	"log/slog"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
)

// MockIdentityProvider is a mock implementation of ports.IdentityProvider for testing.
type MockIdentityProvider struct {
	ServiceName   string
	ServiceDomain string
	ServiceURI    string
}

// NewMockIdentityProvider creates a new mock identity provider with default values.
func NewMockIdentityProvider() *MockIdentityProvider {
	return &MockIdentityProvider{
		ServiceName:   "test-service",
		ServiceDomain: "test.example.com",
		ServiceURI:    "spiffe://test.example.com/test-service",
	}
}

// GetServiceIdentity returns a mock service identity.
func (m *MockIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	return domain.NewServiceIdentity(m.ServiceName, m.ServiceDomain), nil
}

// GetCertificate returns a mock certificate (not implemented for testing).
func (m *MockIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	return nil, nil
}

// GetTrustBundle returns a mock trust bundle (not implemented for testing).
func (m *MockIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	return nil, nil
}

// Close closes the mock provider.
func (m *MockIdentityProvider) Close() error {
	return nil
}

// MockMetricsCollector is a mock implementation of MetricsCollector for testing.
type MockMetricsCollector struct {
	PropagationSuccesses   []PropagationSuccess
	PropagationFailures    []PropagationFailure
	ExtractionSuccesses    []ExtractionSuccess
	CallChainDepths        []int
	CircularCallDetections []string
}

// PropagationSuccess records a successful propagation event.
type PropagationSuccess struct {
	Method    string
	RequestID string
}

// PropagationFailure records a failed propagation event.
type PropagationFailure struct {
	Method string
	Reason string
	Error  error
}

// ExtractionSuccess records a successful extraction event.
type ExtractionSuccess struct {
	Method    string
	RequestID string
}

// NewMockMetricsCollector creates a new mock metrics collector.
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		PropagationSuccesses:   []PropagationSuccess{},
		PropagationFailures:    []PropagationFailure{},
		ExtractionSuccesses:    []ExtractionSuccess{},
		CallChainDepths:        []int{},
		CircularCallDetections: []string{},
	}
}

// RecordPropagationSuccess records successful identity propagation.
func (m *MockMetricsCollector) RecordPropagationSuccess(method string, requestID string) {
	m.PropagationSuccesses = append(m.PropagationSuccesses, PropagationSuccess{
		Method:    method,
		RequestID: requestID,
	})
}

// RecordPropagationFailure records failed identity propagation.
func (m *MockMetricsCollector) RecordPropagationFailure(method string, reason string, err error) {
	m.PropagationFailures = append(m.PropagationFailures, PropagationFailure{
		Method: method,
		Reason: reason,
		Error:  err,
	})
}

// RecordExtractionSuccess records successful identity extraction.
func (m *MockMetricsCollector) RecordExtractionSuccess(method string, requestID string) {
	m.ExtractionSuccesses = append(m.ExtractionSuccesses, ExtractionSuccess{
		Method:    method,
		RequestID: requestID,
	})
}

// RecordCallChainDepth records call chain depth for monitoring.
func (m *MockMetricsCollector) RecordCallChainDepth(depth int) {
	m.CallChainDepths = append(m.CallChainDepths, depth)
}

// RecordCircularCallDetected records detection of circular calls.
func (m *MockMetricsCollector) RecordCircularCallDetected(identity string) {
	m.CircularCallDetections = append(m.CircularCallDetections, identity)
}

// TestingConfig creates a complete configuration for testing with all features enabled.
func TestingConfig() *IdentityPropagationConfig {
	return &IdentityPropagationConfig{
		IdentityProvider:        NewMockIdentityProvider(),
		PropagateOriginalCaller: true,
		PropagateCallChain:      true,
		MaxCallChainDepth:       5, // Lower for testing
		CustomHeaders:           []string{"x-test-header", "x-trace-id"},
		Logger:                  slog.Default(),
		Clock:                   func() time.Time { return time.Unix(1640995200, 0) }, // Fixed time for testing
		IDGen:                   func() string { return "test-req-123" },              // Fixed ID for testing
		MetricsCollector:        NewMockMetricsCollector(),
	}
}

// CreateTestContext creates a context with sample propagated identity for testing.
func CreateTestContext(ctx context.Context) context.Context {
	identity := &PropagatedIdentity{
		OriginalCaller:    "spiffe://test.example.com/original-service",
		CallChain:         "original-service -> intermediate-service",
		CallerTrustDomain: "test.example.com",
		CallerService:     "intermediate-service",
		RequestID:         "test-req-456",
		Timestamp:         1640995200000,
	}

	return context.WithValue(ctx, propagatedIdentityKey, identity)
}

// ExtractMetrics is a helper function to extract metrics from a mock collector.
func ExtractMetrics(collector *MockMetricsCollector) (int, int, int) {
	return len(collector.PropagationSuccesses),
		len(collector.PropagationFailures),
		len(collector.ExtractionSuccesses)
}
