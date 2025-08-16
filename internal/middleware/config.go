// Package middleware provides internal interceptor and metrics configuration.
package middleware

import "log/slog"

// InterceptorConfig configures server interceptors.
type InterceptorConfig struct {
	// EnableAuth enables authentication interceptor
	EnableAuth bool
	// EnableLogging enables audit logging interceptor
	EnableLogging bool
	// EnableIdentityPropagation enables identity propagation between services
	EnableIdentityPropagation bool
	// Logger for interceptor logging
	Logger *slog.Logger
	// CustomInterceptors allows adding custom interceptors
	CustomInterceptors []interface{}
}

// NewDefaultConfig creates a default interceptor configuration.
func NewDefaultConfig() *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableIdentityPropagation: false,
		Logger:                    slog.Default(),
	}
}

// NewProductionConfig creates a production-optimized interceptor configuration.
func NewProductionConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableIdentityPropagation: true,
		Logger:                    logger,
	}
}

// NewDevelopmentConfig creates a development-friendly interceptor configuration.
func NewDevelopmentConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                false, // Disabled for easier development
		EnableLogging:             true,
		EnableIdentityPropagation: true, // Enabled for development testing
		Logger:                    logger,
	}
}