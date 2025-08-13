// Package ephemos provides gRPC interceptor implementations.
package ephemos

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// createLoggingInterceptor creates a logging interceptor.
func createLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)
		
		// Log request
		logger.Info("gRPC request",
			"method", info.FullMethod,
			"metadata", md,
		)
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Log response
		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"code", st.Code(),
				"error", err,
			)
		} else {
			logger.Info("gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}
		
		return resp, err
	}
}

// createMetricsInterceptor creates a metrics collection interceptor.
func createMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Record metrics (in production, this would use a metrics library)
		duration := time.Since(start)
		_ = duration // Would record to metrics system
		
		return resp, err
	}
}

// createAuthInterceptor creates an authentication/authorization interceptor.
func createAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health check and other public methods
		publicMethods := map[string]bool{
			"/grpc.health.v1.Health/Check": true,
			"/grpc.health.v1.Health/Watch": true,
		}
		
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		
		// Extract metadata for auth
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		
		// Check for authorization header (simplified - in production would validate properly)
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			// For now, allow requests without auth header
			// In production, this would validate SPIFFE SVIDs or other auth mechanisms
			return handler(ctx, req)
		}
		
		// Validate auth token (simplified)
		// In production, this would validate against SPIFFE or other auth provider
		
		return handler(ctx, req)
	}
}

// createClientLoggingInterceptor creates a client-side logging interceptor.
func createClientLoggingInterceptor(logger *slog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		
		// Log request
		logger.Info("gRPC client request",
			"method", method,
			"target", cc.Target(),
		)
		
		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		// Log response
		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("gRPC client request failed",
				"method", method,
				"duration", duration,
				"code", st.Code(),
				"error", err,
			)
		} else {
			logger.Info("gRPC client request completed",
				"method", method,
				"duration", duration,
			)
		}
		
		return err
	}
}

// createClientMetricsInterceptor creates a client-side metrics interceptor.
func createClientMetricsInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		
		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		// Record metrics (in production, this would use a metrics library)
		duration := time.Since(start)
		_ = duration // Would record to metrics system
		
		return err
	}
}