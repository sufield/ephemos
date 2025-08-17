// Package services provides core business logic services.
package services

// MetricsReporter defines contract for reporting identity service metrics.
type MetricsReporter interface {
	RecordCacheHit(cacheType string)
	RecordCacheMiss(cacheType string)
	RecordRefresh(reason string, duration float64)
	UpdateCertExpiry(serviceName string, expiryTime float64)
	RecordValidation(success bool)
	RecordRetry(providerType string, attempt int)
}

// NoOpMetrics implements MetricsReporter with no-op methods for when metrics are disabled.
type NoOpMetrics struct{}

// RecordCacheHit no-op implementation.
func (m *NoOpMetrics) RecordCacheHit(cacheType string) {}

// RecordCacheMiss no-op implementation.
func (m *NoOpMetrics) RecordCacheMiss(cacheType string) {}

// RecordRefresh no-op implementation.
func (m *NoOpMetrics) RecordRefresh(reason string, duration float64) {}

// UpdateCertExpiry no-op implementation.
func (m *NoOpMetrics) UpdateCertExpiry(serviceName string, expiryTime float64) {}

// RecordValidation no-op implementation.
func (m *NoOpMetrics) RecordValidation(success bool) {}

// RecordRetry no-op implementation.
func (m *NoOpMetrics) RecordRetry(providerType string, attempt int) {}
