// Package metrics provides Prometheus-based implementations of service metrics reporting.
package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sufield/ephemos/internal/core/services"
)

var (
	// Certificate cache metrics
	certCacheHitsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ephemos_cert_cache_hits_total",
		Help: "Total number of certificate cache hits",
	})

	certCacheMissesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ephemos_cert_cache_misses_total",
		Help: "Total number of certificate cache misses",
	})

	// Trust bundle cache metrics
	bundleCacheHitsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ephemos_bundle_cache_hits_total",
		Help: "Total number of trust bundle cache hits",
	})

	bundleCacheMissesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ephemos_bundle_cache_misses_total",
		Help: "Total number of trust bundle cache misses",
	})

	// Certificate refresh metrics
	certRefreshCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ephemos_cert_refresh_total",
		Help: "Total number of certificate refreshes",
	}, []string{"reason"}) // reason: expired, proactive, cache_miss

	certRefreshDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ephemos_cert_refresh_duration_seconds",
		Help:    "Duration of certificate refresh operations",
		Buckets: prometheus.DefBuckets,
	})

	// Certificate expiry gauge
	certExpiryTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ephemos_cert_expiry_timestamp_seconds",
		Help: "Unix timestamp when the cached certificate will expire",
	}, []string{"service_name"})

	// Certificate validation metrics
	certValidationCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ephemos_cert_validation_total",
		Help: "Total number of certificate validations",
	}, []string{"result"}) // result: success, failure

	// Retry metrics
	providerRetryCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ephemos_provider_retry_total",
		Help: "Total number of provider retry attempts",
	}, []string{"provider_type", "attempt"})
)

// PrometheusMetrics implements services.MetricsReporter using Prometheus.
type PrometheusMetrics struct{}

// NewPrometheusMetrics creates a new Prometheus metrics reporter.
func NewPrometheusMetrics() services.MetricsReporter {
	return &PrometheusMetrics{}
}

// RecordCacheHit records a cache hit.
func (m *PrometheusMetrics) RecordCacheHit(cacheType string) {
	switch cacheType {
	case "certificate":
		certCacheHitsCounter.Inc()
	case "bundle":
		bundleCacheHitsCounter.Inc()
	}
}

// RecordCacheMiss records a cache miss.
func (m *PrometheusMetrics) RecordCacheMiss(cacheType string) {
	switch cacheType {
	case "certificate":
		certCacheMissesCounter.Inc()
	case "bundle":
		bundleCacheMissesCounter.Inc()
	}
}

// RecordRefresh records a certificate refresh.
func (m *PrometheusMetrics) RecordRefresh(reason string, duration float64) {
	certRefreshCounter.WithLabelValues(reason).Inc()
	certRefreshDuration.Observe(duration)
}

// UpdateCertExpiry updates the certificate expiry timestamp.
func (m *PrometheusMetrics) UpdateCertExpiry(serviceName string, expiryTime float64) {
	certExpiryTimestamp.WithLabelValues(serviceName).Set(expiryTime)
}

// RecordValidation records a certificate validation result.
func (m *PrometheusMetrics) RecordValidation(success bool) {
	result := "failure"
	if success {
		result = "success"
	}
	certValidationCounter.WithLabelValues(result).Inc()
}

// RecordRetry records a provider retry attempt.
func (m *PrometheusMetrics) RecordRetry(providerType string, attempt int) {
	providerRetryCounter.WithLabelValues(providerType, strconv.Itoa(attempt)).Inc()
}
