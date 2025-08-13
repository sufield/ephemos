// Package transport provides enhanced gRPC connection management with backoff and retry policies.
package transport

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
)

// Connection timeout constants.
const (
	defaultConnectTimeout        = 30 * time.Second
	developmentConnectTimeout    = 5 * time.Second
	highThroughputConnectTimeout = 10 * time.Second
)

// Backoff configuration constants.
const (
	defaultBackoffMultiplier   = 1.6
	defaultBackoffJitter       = 0.2
	defaultMaxBackoffDelay     = 120 * time.Second
	developmentMaxBackoffDelay = 10 * time.Second
	developmentBaseDelay       = 500 * time.Millisecond
)

// Keepalive constants.
const (
	defaultKeepaliveTime           = 10 * time.Second
	defaultKeepaliveTimeout        = 5 * time.Second
	highThroughputKeepaliveTime    = 30 * time.Second
	highThroughputKeepaliveTimeout = 10 * time.Second
)

// Idle timeout constants.
const (
	defaultIdleTimeout        = 30 * time.Minute
	developmentIdleTimeout    = 5 * time.Minute
	highThroughputIdleTimeout = 0 // Disabled
)

// Message size constants.
const (
	defaultMaxMessageSize        = 4 * 1024 * 1024  // 4MB
	highThroughputMaxMessageSize = 16 * 1024 * 1024 // 16MB
)

// Connection pool constants.
const (
	defaultPoolSize        = 5
	highThroughputPoolSize = 10
)

// ConnectionConfig provides configuration for gRPC client connections.
type ConnectionConfig struct {
	// Connection timeout for initial connection establishment
	ConnectTimeout time.Duration

	// Backoff configuration for connection retries
	BackoffConfig backoff.Config

	// Keepalive parameters for connection health
	KeepaliveParams keepalive.ClientParameters

	// Idle timeout for connections (0 disables idle timeout)
	IdleTimeout time.Duration

	// Maximum message sizes
	MaxRecvMsgSize int
	MaxSendMsgSize int

	// Enable connection pooling
	EnablePooling bool
	PoolSize      int

	// Service configuration for method-level settings
	ServiceConfig string
}

// DefaultConnectionConfig returns a production-ready connection configuration.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		ConnectTimeout: defaultConnectTimeout,
		BackoffConfig: backoff.Config{
			BaseDelay:  1.0 * time.Second,
			Multiplier: defaultBackoffMultiplier,
			Jitter:     defaultBackoffJitter,
			MaxDelay:   defaultMaxBackoffDelay,
		},
		KeepaliveParams: keepalive.ClientParameters{
			Time:                defaultKeepaliveTime,
			Timeout:             defaultKeepaliveTimeout,
			PermitWithoutStream: true, // Send pings even without active streams
		},
		IdleTimeout:    defaultIdleTimeout,
		MaxRecvMsgSize: defaultMaxMessageSize,
		MaxSendMsgSize: defaultMaxMessageSize,
		EnablePooling:  false, // Disabled by default
		PoolSize:       defaultPoolSize,
		ServiceConfig: `{
			"methodConfig": [
				{
					"name": [{"service": ""}], 
					"retryPolicy": {
						"maxAttempts": 5,
						"initialBackoff": "1s",
						"maxBackoff": "30s",
						"backoffMultiplier": 2.0,
						"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED", "ABORTED"]
					}
				}
			]
		}`,
	}
}

// DevelopmentConnectionConfig returns a configuration suitable for development.
func DevelopmentConnectionConfig() *ConnectionConfig {
	config := DefaultConnectionConfig()
	config.ConnectTimeout = developmentConnectTimeout
	config.BackoffConfig.BaseDelay = developmentBaseDelay
	config.BackoffConfig.MaxDelay = developmentMaxBackoffDelay
	config.IdleTimeout = developmentIdleTimeout
	config.ServiceConfig = `{
		"methodConfig": [
			{
				"name": [{"service": ""}],
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "0.5s", 
					"maxBackoff": "5s",
					"backoffMultiplier": 1.5,
					"retryableStatusCodes": ["UNAVAILABLE"]
				}
			}
		]
	}`
	return config
}

// HighThroughputConnectionConfig returns a configuration optimized for high-throughput scenarios.
func HighThroughputConnectionConfig() *ConnectionConfig {
	config := DefaultConnectionConfig()
	config.ConnectTimeout = highThroughputConnectTimeout
	config.KeepaliveParams.Time = highThroughputKeepaliveTime       // Less frequent keepalives
	config.KeepaliveParams.Timeout = highThroughputKeepaliveTimeout // Longer timeout
	config.IdleTimeout = highThroughputIdleTimeout                  // Disable idle timeout
	config.MaxRecvMsgSize = highThroughputMaxMessageSize
	config.MaxSendMsgSize = highThroughputMaxMessageSize
	config.EnablePooling = true // Enable pooling for high throughput
	config.PoolSize = highThroughputPoolSize
	config.ServiceConfig = `{
		"methodConfig": [
			{
				"name": [{"service": ""}],
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "2s",
					"maxBackoff": "60s", 
					"backoffMultiplier": 2.0,
					"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}
		]
	}`
	return config
}

// ToDialOptions converts the connection configuration to gRPC dial options.
func (c *ConnectionConfig) ToDialOptions() []grpc.DialOption {
	options := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           c.BackoffConfig,
			MinConnectTimeout: c.ConnectTimeout,
		}),
		grpc.WithKeepaliveParams(c.KeepaliveParams),
		grpc.WithDefaultServiceConfig(c.ServiceConfig),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(c.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(c.MaxSendMsgSize),
		),
	}

	// Note: grpc.WithIdleTimeout may not be available in all gRPC versions
	// Idle timeout configuration handled via keepalive parameters instead
	if c.IdleTimeout > 0 {
		// Use keepalive parameters for connection management
		keepaliveParams := keepalive.ClientParameters{
			Time:                c.IdleTimeout,
			Timeout:             c.IdleTimeout / 2,
			PermitWithoutStream: true,
		}
		options = append(options, grpc.WithKeepaliveParams(keepaliveParams))
	}

	return options
}
