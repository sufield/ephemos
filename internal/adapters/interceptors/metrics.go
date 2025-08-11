package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

const defaultResultCode = "success"

// AuthMetricsCollector defines the interface for collecting authentication metrics.
type AuthMetricsCollector interface {
	// IncAuthenticationTotal increments authentication attempts
	IncAuthenticationTotal(service, result string)
}

// DefaultAuthMetricsCollector provides a no-op implementation.
type DefaultAuthMetricsCollector struct{}

// IncAuthenticationTotal is a no-op implementation.
func (d *DefaultAuthMetricsCollector) IncAuthenticationTotal(_, _ string) {}

// AuthMetricsConfig configures authentication metrics collection.
type AuthMetricsConfig struct {
	// AuthMetricsCollector to use for collecting authentication metrics
	AuthMetricsCollector AuthMetricsCollector

	// ServiceName to use in metrics labels
	ServiceName string
}

// AuthMetricsInterceptor provides authentication metrics collection for gRPC services.
type AuthMetricsInterceptor struct {
	config *AuthMetricsConfig
}

// NewAuthMetricsInterceptor creates a new authentication metrics interceptor.
func NewAuthMetricsInterceptor(config *AuthMetricsConfig) *AuthMetricsInterceptor {
	if config.AuthMetricsCollector == nil {
		config.AuthMetricsCollector = &DefaultAuthMetricsCollector{}
	}

	return &AuthMetricsInterceptor{
		config: config,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for authentication metrics collection.
func (m *AuthMetricsInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Call the handler
		resp, err := handler(ctx, req)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = "failure"
			}
			m.config.AuthMetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for authentication metrics collection.
func (m *AuthMetricsInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// Call the handler
		err := handler(srv, ss)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = "failure"
			}
			m.config.AuthMetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return err
	}
}

// DefaultAuthMetricsConfig returns a default authentication metrics configuration.
func DefaultAuthMetricsConfig(serviceName string) *AuthMetricsConfig {
	return &AuthMetricsConfig{
		AuthMetricsCollector: &DefaultAuthMetricsCollector{},
		ServiceName:          serviceName,
	}
}