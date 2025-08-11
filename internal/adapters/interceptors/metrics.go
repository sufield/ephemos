package interceptors

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultResultCode = "success"

// MetricsCollector defines the interface for collecting gRPC metrics.
type MetricsCollector interface {
	// IncRequestsTotal increments the total number of requests
	IncRequestsTotal(method, service, code string)

	// ObserveRequestDuration records request duration
	ObserveRequestDuration(method, service, code string, duration time.Duration)

	// IncActiveRequests increments active requests counter
	IncActiveRequests(method, service string)

	// DecActiveRequests decrements active requests counter
	DecActiveRequests(method, service string)

	// IncStreamMessagesTotal increments stream message counter
	IncStreamMessagesTotal(method, service, direction string)

	// IncAuthenticationTotal increments authentication attempts
	IncAuthenticationTotal(service, result string)

	// ObservePayloadSize records payload sizes
	ObservePayloadSize(method, service, direction string, size int)
}

// DefaultMetricsCollector provides a no-op implementation.
type DefaultMetricsCollector struct{}

// IncRequestsTotal is a no-op implementation.
func (d *DefaultMetricsCollector) IncRequestsTotal(_, _, _ string) {}

// ObserveRequestDuration is a no-op implementation.
func (d *DefaultMetricsCollector) ObserveRequestDuration(_, _, _ string, _ time.Duration) {}

// IncActiveRequests is a no-op implementation.
func (d *DefaultMetricsCollector) IncActiveRequests(_, _ string) {}

// DecActiveRequests is a no-op implementation.
func (d *DefaultMetricsCollector) DecActiveRequests(_, _ string) {}

// IncStreamMessagesTotal is a no-op implementation.
func (d *DefaultMetricsCollector) IncStreamMessagesTotal(_, _, _ string) {}

// IncAuthenticationTotal is a no-op implementation.
func (d *DefaultMetricsCollector) IncAuthenticationTotal(_, _ string) {}

// ObservePayloadSize is a no-op implementation.
func (d *DefaultMetricsCollector) ObservePayloadSize(_, _, _ string, _ int) {}

// MetricsConfig configures metrics collection behavior.
type MetricsConfig struct {
	// MetricsCollector to use for collecting metrics
	MetricsCollector MetricsCollector

	// ServiceName to use in metrics labels
	ServiceName string

	// EnablePayloadSize enables payload size metrics collection
	EnablePayloadSize bool

	// EnableActiveRequests enables active requests tracking
	EnableActiveRequests bool
}

// MetricsInterceptor provides metrics collection for gRPC services.
type MetricsInterceptor struct {
	config *MetricsConfig
}

// NewMetricsInterceptor creates a new metrics interceptor.
func NewMetricsInterceptor(config *MetricsConfig) *MetricsInterceptor {
	if config.MetricsCollector == nil {
		config.MetricsCollector = &DefaultMetricsCollector{}
	}

	return &MetricsInterceptor{
		config: config,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for metrics collection.
func (m *MetricsInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		method := info.FullMethod
		service := m.config.ServiceName

		// Track active requests
		if m.config.EnableActiveRequests {
			m.config.MetricsCollector.IncActiveRequests(method, service)
			defer m.config.MetricsCollector.DecActiveRequests(method, service)
		}

		// Track payload size if enabled
		if m.config.EnablePayloadSize {
			if size := estimatePayloadSize(req); size > 0 {
				m.config.MetricsCollector.ObservePayloadSize(method, service, "request", size)
			}
		}

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Determine status code
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		// Collect metrics
		codeStr := code.String()
		m.config.MetricsCollector.IncRequestsTotal(method, service, codeStr)
		m.config.MetricsCollector.ObserveRequestDuration(method, service, codeStr, duration)

		// Track response payload size if enabled and no error
		if m.config.EnablePayloadSize && err == nil {
			if size := estimatePayloadSize(resp); size > 0 {
				m.config.MetricsCollector.ObservePayloadSize(method, service, "response", size)
			}
		}

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = "failure"
			}
			m.config.MetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for metrics collection.
func (m *MetricsInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		method := info.FullMethod
		service := m.config.ServiceName
		ctx := ss.Context()

		// Track active requests
		if m.config.EnableActiveRequests {
			m.config.MetricsCollector.IncActiveRequests(method, service)
			defer m.config.MetricsCollector.DecActiveRequests(method, service)
		}

		// Wrap the stream to collect message metrics
		wrappedStream := &metricsServerStream{
			ServerStream: ss,
			method:       method,
			service:      service,
			collector:    m.config.MetricsCollector,
			enableSizes:  m.config.EnablePayloadSize,
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		// Calculate duration
		duration := time.Since(start)

		// Determine status code
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		// Collect metrics
		codeStr := code.String()
		m.config.MetricsCollector.IncRequestsTotal(method, service, codeStr)
		m.config.MetricsCollector.ObserveRequestDuration(method, service, codeStr, duration)

		// Track authentication metrics if identity is available
		if identity, ok := GetIdentityFromContext(ctx); ok {
			result := defaultResultCode
			if err != nil {
				result = "failure"
			}
			m.config.MetricsCollector.IncAuthenticationTotal(identity.ServiceName, result)
		}

		return err
	}
}

// metricsServerStream wraps a grpc.ServerStream to collect message metrics.
type metricsServerStream struct {
	grpc.ServerStream
	method      string
	service     string
	collector   MetricsCollector
	enableSizes bool
}

// SendMsg collects metrics for outgoing messages.
func (s *metricsServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.collector.IncStreamMessagesTotal(s.method, s.service, "sent")

		if s.enableSizes {
			if size := estimatePayloadSize(m); size > 0 {
				s.collector.ObservePayloadSize(s.method, s.service, "response", size)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// RecvMsg collects metrics for incoming messages.
func (s *metricsServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.collector.IncStreamMessagesTotal(s.method, s.service, "received")

		if s.enableSizes {
			if size := estimatePayloadSize(m); size > 0 {
				s.collector.ObservePayloadSize(s.method, s.service, "request", size)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}
	return nil
}

// Client-side interceptors

// UnaryClientInterceptor returns a gRPC unary client interceptor for metrics collection.
func (m *MetricsInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		service := m.config.ServiceName

		// Track active requests
		if m.config.EnableActiveRequests {
			m.config.MetricsCollector.IncActiveRequests(method, service)
			defer m.config.MetricsCollector.DecActiveRequests(method, service)
		}

		// Track request payload size if enabled
		if m.config.EnablePayloadSize {
			if size := estimatePayloadSize(req); size > 0 {
				m.config.MetricsCollector.ObservePayloadSize(method, service, "request", size)
			}
		}

		// Make the call
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Determine status code
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		// Collect metrics
		codeStr := code.String()
		m.config.MetricsCollector.IncRequestsTotal(method, service, codeStr)
		m.config.MetricsCollector.ObserveRequestDuration(method, service, codeStr, duration)

		// Track response payload size if enabled and no error
		if m.config.EnablePayloadSize && err == nil {
			if size := estimatePayloadSize(reply); size > 0 {
				m.config.MetricsCollector.ObservePayloadSize(method, service, "response", size)
			}
		}

		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor for metrics collection.
func (m *MetricsInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()
		service := m.config.ServiceName

		// Track active requests
		if m.config.EnableActiveRequests {
			m.config.MetricsCollector.IncActiveRequests(method, service)
			// Note: DecActiveRequests will be called when stream is closed
		}

		// Create the stream
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			if m.config.EnableActiveRequests {
				m.config.MetricsCollector.DecActiveRequests(method, service)
			}
			return nil, err
		}

		// Wrap the stream for metrics collection
		wrappedStream := &metricsClientStream{
			ClientStream: clientStream,
			method:       method,
			service:      service,
			collector:    m.config.MetricsCollector,
			enableSizes:  m.config.EnablePayloadSize,
			start:        start,
			enableActive: m.config.EnableActiveRequests,
		}

		return wrappedStream, nil
	}
}

// metricsClientStream wraps a grpc.ClientStream to collect message metrics.
type metricsClientStream struct {
	grpc.ClientStream
	method       string
	service      string
	collector    MetricsCollector
	enableSizes  bool
	start        time.Time
	enableActive bool
	closed       bool
}

// SendMsg collects metrics for outgoing messages.
func (s *metricsClientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err == nil {
		s.collector.IncStreamMessagesTotal(s.method, s.service, "sent")

		if s.enableSizes {
			if size := estimatePayloadSize(m); size > 0 {
				s.collector.ObservePayloadSize(s.method, s.service, "request", size)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// RecvMsg collects metrics for incoming messages.
func (s *metricsClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err == nil {
		s.collector.IncStreamMessagesTotal(s.method, s.service, "received")

		if s.enableSizes {
			if size := estimatePayloadSize(m); size > 0 {
				s.collector.ObservePayloadSize(s.method, s.service, "response", size)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}
	return nil
}

// CloseSend collects final metrics when the stream is closed.
func (s *metricsClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	s.collectFinalMetrics(err)
	if err != nil {
		return fmt.Errorf("failed to close send: %w", err)
	}
	return nil
}

// collectFinalMetrics collects metrics when the stream ends.
func (s *metricsClientStream) collectFinalMetrics(err error) {
	if s.closed {
		return
	}
	s.closed = true

	// Calculate duration
	duration := time.Since(s.start)

	// Determine status code
	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	// Collect final metrics
	codeStr := code.String()
	s.collector.IncRequestsTotal(s.method, s.service, codeStr)
	s.collector.ObserveRequestDuration(s.method, s.service, codeStr, duration)

	// Decrement active requests
	if s.enableActive {
		s.collector.DecActiveRequests(s.method, s.service)
	}
}

// estimatePayloadSize estimates the size of a payload.
// This is a simple implementation that could be enhanced with more accurate sizing.
func estimatePayloadSize(payload interface{}) int {
	if payload == nil {
		return 0
	}

	// This is a simplified estimation - in practice you might want to use
	// protocol buffer's Size() method or other serialization-specific methods
	switch v := payload.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	default:
		// Rough estimate - could be enhanced with reflection or proto.Size()
		const defaultPayloadSize = 64
		return defaultPayloadSize // Default estimate
	}
}

// DefaultMetricsConfig returns a default metrics configuration.
func DefaultMetricsConfig(serviceName string) *MetricsConfig {
	return &MetricsConfig{
		MetricsCollector:     &DefaultMetricsCollector{},
		ServiceName:          serviceName,
		EnablePayloadSize:    false, // Disabled by default for performance
		EnableActiveRequests: true,
	}
}
