package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

const (
	defaultResultCode = "success"
	failureResultCode = "failure"
)

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

	// EnablePayloadSize enables payload size tracking
	EnablePayloadSize bool

	// EnableActiveRequests enables active request counting
	EnableActiveRequests bool
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
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Call the handler
		resp, err := handler(ctx, req)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = failureResultCode
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
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// Call the handler
		err := handler(srv, ss)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = failureResultCode
			}
			m.config.AuthMetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return err
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor for authentication metrics collection.
func (m *AuthMetricsInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Call the method
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = failureResultCode
			}
			m.config.AuthMetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor for authentication metrics collection.
func (m *AuthMetricsInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Call the method
		stream, err := streamer(ctx, desc, cc, method, opts...)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = failureResultCode
			}
			m.config.AuthMetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return stream, err
	}
}

// DefaultAuthMetricsConfig returns a default authentication metrics configuration.
func DefaultAuthMetricsConfig(serviceName string) *AuthMetricsConfig {
	return &AuthMetricsConfig{
		AuthMetricsCollector: &DefaultAuthMetricsCollector{},
		ServiceName:          serviceName,
	}
}

// MetricsConfig is an alias for AuthMetricsConfig to maintain backward compatibility.
type MetricsConfig = AuthMetricsConfig

// MetricsInterceptor is an alias for AuthMetricsInterceptor to maintain backward compatibility.
type MetricsInterceptor = AuthMetricsInterceptor

// DefaultMetricsConfig returns a default metrics configuration.
func DefaultMetricsConfig(serviceName string) *MetricsConfig {
	return DefaultAuthMetricsConfig(serviceName)
}

// NewMetricsInterceptor creates a new metrics interceptor.
func NewMetricsInterceptor(config *MetricsConfig) *MetricsInterceptor {
	return NewAuthMetricsInterceptor(config)
}
