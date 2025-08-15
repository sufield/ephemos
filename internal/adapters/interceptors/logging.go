package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/sufield/ephemos/internal/adapters/logging"
)

const (
	// Default thresholds for logging configuration.
	defaultSlowThreshold = 500 * time.Millisecond
	debugSlowThreshold   = 100 * time.Millisecond
)

// LoggingConfig configures audit logging behavior.
type LoggingConfig struct {
	// Logger instance (will use secure logger with redaction)
	Logger *slog.Logger

	// LogRequests enables request logging
	LogRequests bool

	// LogResponses enables response logging (be careful with sensitive data)
	LogResponses bool

	// LogPayloads includes request/response payloads in logs
	LogPayloads bool

	// SlowRequestThreshold logs requests that take longer than this duration
	SlowRequestThreshold time.Duration

	// ExcludeMethods are methods that should not be logged (e.g., health checks)
	ExcludeMethods []string

	// IncludeHeaders are specific headers to include in logs
	IncludeHeaders []string
}

// LoggingInterceptor provides structured audit logging for gRPC services.
type LoggingInterceptor struct {
	config *LoggingConfig
	logger *slog.Logger
}

// NewLoggingInterceptor creates a new logging interceptor with secure redaction.
func NewLoggingInterceptor(config *LoggingConfig) *LoggingInterceptor {
	logger := config.Logger
	if logger == nil {
		// Use secure logger with automatic redaction
		logger = slog.New(logging.NewRedactorHandler(slog.Default().Handler()))
	}

	// Set default slow request threshold
	if config.SlowRequestThreshold == 0 {
		const defaultSlowThreshold = 500 * time.Millisecond
		config.SlowRequestThreshold = defaultSlowThreshold
	}

	return &LoggingInterceptor{
		config: config,
		logger: logger,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for logging.
func (l *LoggingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if method should be excluded from logging
		if l.shouldExcludeMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		start := time.Now()

		// Extract identity and request information
		logEntry := l.createBaseLogEntry(ctx, info.FullMethod, "unary")

		// Log request if enabled
		if l.config.LogRequests {
			requestEntry := logEntry.With("event", "request_received")
			if l.config.LogPayloads {
				requestEntry = requestEntry.With("request_payload", req)
			}
			requestEntry.Info("gRPC request received")
		}

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Determine log level based on error and duration
		logLevel := slog.LevelInfo
		if err != nil {
			logLevel = slog.LevelError
		} else if duration > l.config.SlowRequestThreshold {
			logLevel = slog.LevelWarn
		}

		// Create response log entry
		responseEntry := logEntry.With(
			"event", "request_completed",
			"duration_ms", duration.Milliseconds(),
			"success", err == nil,
		)

		// Add error information if present
		if err != nil {
			grpcStatus := status.Convert(err)
			responseEntry = responseEntry.With(
				"error_code", grpcStatus.Code().String(),
				"error_message", grpcStatus.Message(),
			)
		}

		// Add response payload if enabled and no error
		if l.config.LogResponses && err == nil && l.config.LogPayloads {
			responseEntry = responseEntry.With("response_payload", resp)
		}

		// Log the response
		responseEntry.Log(ctx, logLevel, "gRPC request completed")

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for logging.
func (l *LoggingInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if method should be excluded from logging
		if l.shouldExcludeMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()

		// Extract identity and request information
		logEntry := l.createBaseLogEntry(ctx, info.FullMethod, "stream")

		// Log stream start
		if l.config.LogRequests {
			logEntry.With("event", "stream_started").Info("gRPC stream started")
		}

		// Wrap the stream to count messages
		wrappedStream := &loggingServerStream{
			ServerStream: ss,
			logger:       l.logger,
			method:       info.FullMethod,
			logPayloads:  l.config.LogPayloads,
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		duration := time.Since(start)

		// Determine log level
		logLevel := slog.LevelInfo
		if err != nil {
			logLevel = slog.LevelError
		} else if duration > l.config.SlowRequestThreshold {
			logLevel = slog.LevelWarn
		}

		// Create completion log entry
		completionEntry := logEntry.With(
			"event", "stream_completed",
			"duration_ms", duration.Milliseconds(),
			"success", err == nil,
			"messages_sent", wrappedStream.messagesSent,
			"messages_received", wrappedStream.messagesReceived,
		)

		// Add error information if present
		if err != nil {
			grpcStatus := status.Convert(err)
			completionEntry = completionEntry.With(
				"error_code", grpcStatus.Code().String(),
				"error_message", grpcStatus.Message(),
			)
		}

		// Log stream completion
		completionEntry.Log(ctx, logLevel, "gRPC stream completed")

		return err
	}
}

// loggingServerStream wraps a grpc.ServerStream to log message flow.
type loggingServerStream struct {
	grpc.ServerStream
	logger           *slog.Logger
	method           string
	logPayloads      bool
	messagesSent     int
	messagesReceived int
}

// SendMsg logs outgoing messages.
func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.messagesSent++
		if s.logPayloads {
			s.logger.Debug("Stream message sent",
				"method", s.method,
				"message_count", s.messagesSent,
				"payload", m)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// RecvMsg logs incoming messages.
func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.messagesReceived++
		if s.logPayloads {
			s.logger.Debug("Stream message received",
				"method", s.method,
				"message_count", s.messagesReceived,
				"payload", m)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}
	return nil
}

// createBaseLogEntry creates a base log entry with common fields.
func (l *LoggingInterceptor) createBaseLogEntry(ctx context.Context, method, requestType string) *slog.Logger {
	entry := l.logger.With(
		"method", method,
		"request_type", requestType,
		"timestamp", time.Now().Unix(),
	)

	// Add identity information if available
	if identity, ok := GetIdentityFromContext(ctx); ok {
		entry = entry.With(
			"client_spiffe_id", identity.SPIFFEID,
			"client_service", identity.ServiceName,
			"client_trust_domain", identity.TrustDomain,
		)
	}

	// Add propagated identity information
	if originalCaller, ok := GetOriginalCaller(ctx); ok {
		entry = entry.With("original_caller", originalCaller)
	}

	if callChain, ok := GetCallChain(ctx); ok {
		entry = entry.With("call_chain", callChain)
	}

	if requestID, ok := GetRequestID(ctx); ok {
		entry = entry.With("request_id", requestID)
	}

	return entry
}

// shouldExcludeMethod checks if a method should be excluded from logging.
func (l *LoggingInterceptor) shouldExcludeMethod(method string) bool {
	for _, excludeMethod := range l.config.ExcludeMethods {
		if method == excludeMethod {
			return true
		}
	}
	return false
}

// DefaultLoggingConfig returns a default logging configuration.
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Logger:               nil, // Will use secure logger
		LogRequests:          true,
		LogResponses:         true,
		LogPayloads:          false, // Disabled by default for security
		SlowRequestThreshold: defaultSlowThreshold,
		ExcludeMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		},
		IncludeHeaders: []string{},
	}
}

// NewSecureLoggingConfig creates a logging config with payload logging disabled for security.
func NewSecureLoggingConfig() *LoggingConfig {
	config := DefaultLoggingConfig()
	config.LogPayloads = false // Ensure payloads are not logged
	return config
}

// NewDebugLoggingConfig creates a logging config suitable for development/debugging.
func NewDebugLoggingConfig() *LoggingConfig {
	config := DefaultLoggingConfig()
	config.LogPayloads = true
	config.SlowRequestThreshold = debugSlowThreshold
	config.ExcludeMethods = []string{} // Log everything in debug mode
	return config
}
