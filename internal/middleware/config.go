// Package middleware provides internal interceptor and metrics configuration.
package middleware

import "log/slog"

// MetricsConfig configures metrics collection.
type MetricsConfig struct {
	AuthMetricsCollector interface{}
}

// InterceptorConfig configures server interceptors.
type InterceptorConfig struct {
	// EnableAuth enables authentication interceptor
	EnableAuth bool
	// EnableLogging enables audit logging interceptor
	EnableLogging bool
	// EnableMetrics enables metrics collection interceptor
	EnableMetrics bool
	// EnableIdentityPropagation enables identity propagation between services
	EnableIdentityPropagation bool
	// Logger for interceptor logging
	Logger *slog.Logger
	// MetricsConfig for metrics configuration
	MetricsConfig *MetricsConfig
	// CustomInterceptors allows adding custom interceptors
	CustomInterceptors []interface{}
}

// NewDefaultConfig creates a default interceptor configuration.
func NewDefaultConfig() *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: false,
		Logger:                    slog.Default(),
		MetricsConfig:             &MetricsConfig{},
	}
}

// NewProductionConfig creates a production-optimized interceptor configuration.
func NewProductionConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: true,
		Logger:                    logger,
		MetricsConfig:             &MetricsConfig{},
	}
}

// NewDevelopmentConfig creates a development-friendly interceptor configuration.
func NewDevelopmentConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                false, // Disabled for easier development
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: true, // Enabled for development testing
		Logger:                    logger,
		MetricsConfig:             &MetricsConfig{},
	}
}