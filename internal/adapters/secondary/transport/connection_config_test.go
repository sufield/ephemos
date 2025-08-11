package transport_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"

	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
)

func TestDefaultConnectionConfig(t *testing.T) {
	config := transport.DefaultConnectionConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.ConnectTimeout)
	assert.Equal(t, 1.0*time.Second, config.BackoffConfig.BaseDelay)
	assert.Equal(t, 1.6, config.BackoffConfig.Multiplier)
	assert.Equal(t, 0.2, config.BackoffConfig.Jitter)
	assert.Equal(t, 120*time.Second, config.BackoffConfig.MaxDelay)
	assert.Equal(t, 10*time.Second, config.KeepaliveParams.Time)
	assert.Equal(t, 5*time.Second, config.KeepaliveParams.Timeout)
	assert.True(t, config.KeepaliveParams.PermitWithoutStream)
	assert.Equal(t, 30*time.Minute, config.IdleTimeout)
	assert.Equal(t, 4*1024*1024, config.MaxRecvMsgSize)
	assert.Equal(t, 4*1024*1024, config.MaxSendMsgSize)
	assert.False(t, config.EnablePooling)
	assert.Equal(t, 5, config.PoolSize)
	assert.Contains(t, config.ServiceConfig, "retryPolicy")
	assert.Contains(t, config.ServiceConfig, "UNAVAILABLE")
}

func TestDevelopmentConnectionConfig(t *testing.T) {
	config := transport.DevelopmentConnectionConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 5*time.Second, config.ConnectTimeout)
	assert.Equal(t, 500*time.Millisecond, config.BackoffConfig.BaseDelay)
	assert.Equal(t, 10*time.Second, config.BackoffConfig.MaxDelay)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
	assert.Contains(t, config.ServiceConfig, "maxAttempts\": 3")
	assert.Contains(t, config.ServiceConfig, "initialBackoff\": \"0.5s\"")
}

func TestHighThroughputConnectionConfig(t *testing.T) {
	config := transport.HighThroughputConnectionConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10*time.Second, config.ConnectTimeout)
	assert.Equal(t, 30*time.Second, config.KeepaliveParams.Time)
	assert.Equal(t, 10*time.Second, config.KeepaliveParams.Timeout)
	assert.Equal(t, time.Duration(0), config.IdleTimeout) // Disabled
	assert.Equal(t, 16*1024*1024, config.MaxRecvMsgSize)
	assert.Equal(t, 16*1024*1024, config.MaxSendMsgSize)
	assert.True(t, config.EnablePooling)
	assert.Equal(t, 10, config.PoolSize)
}

func TestConnectionConfigToDialOptions(t *testing.T) {
	config := &transport.ConnectionConfig{
		ConnectTimeout: 10 * time.Second,
		BackoffConfig: backoff.Config{
			BaseDelay:  2 * time.Second,
			Multiplier: 2.0,
			Jitter:     0.1,
			MaxDelay:   60 * time.Second,
		},
		KeepaliveParams: keepalive.ClientParameters{
			Time:                15 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: false,
		},
		IdleTimeout:    45 * time.Minute,
		MaxRecvMsgSize: 8 * 1024 * 1024,
		MaxSendMsgSize: 8 * 1024 * 1024,
		ServiceConfig:  `{"loadBalancingConfig": [{"round_robin": {}}]}`,
	}

	options := config.ToDialOptions()

	// Should have multiple dial options
	assert.GreaterOrEqual(t, len(options), 4)

	// Verify options can be used (basic smoke test)
	assert.NotNil(t, options)
}

func TestConnectionConfigToDialOptionsWithoutIdleTimeout(t *testing.T) {
	config := &transport.ConnectionConfig{
		ConnectTimeout: 5 * time.Second,
		BackoffConfig: backoff.Config{
			BaseDelay:  1 * time.Second,
			Multiplier: 1.5,
			Jitter:     0.2,
			MaxDelay:   30 * time.Second,
		},
		KeepaliveParams: keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		},
		IdleTimeout:    0, // Disabled
		MaxRecvMsgSize: 2 * 1024 * 1024,
		MaxSendMsgSize: 2 * 1024 * 1024,
		ServiceConfig:  `{}`,
	}

	options := config.ToDialOptions()

	// Should not include idle timeout option when set to 0
	assert.NotNil(t, options)
	assert.GreaterOrEqual(t, len(options), 3)
}

func TestConnectionConfigDefaults(t *testing.T) {
	tests := []struct {
		name            string
		configFunc      func() *transport.ConnectionConfig
		expectedPooled  bool
		expectedTimeout time.Duration
	}{
		{
			name:            "default_config",
			configFunc:      transport.DefaultConnectionConfig,
			expectedPooled:  false,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "development_config",
			configFunc:      transport.DevelopmentConnectionConfig,
			expectedPooled:  false,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "high_throughput_config",
			configFunc:      transport.HighThroughputConnectionConfig,
			expectedPooled:  true,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc()

			assert.Equal(t, tt.expectedPooled, config.EnablePooling)
			assert.Equal(t, tt.expectedTimeout, config.ConnectTimeout)
			assert.Greater(t, config.BackoffConfig.BaseDelay, time.Duration(0))
			assert.Greater(t, config.BackoffConfig.MaxDelay, config.BackoffConfig.BaseDelay)
			assert.Greater(t, config.KeepaliveParams.Time, time.Duration(0))
			assert.Greater(t, config.KeepaliveParams.Timeout, time.Duration(0))
			assert.Greater(t, config.MaxRecvMsgSize, 0)
			assert.Greater(t, config.MaxSendMsgSize, 0)
			assert.Greater(t, config.PoolSize, 0)
			assert.NotEmpty(t, config.ServiceConfig)
		})
	}
}

func TestConnectionConfigBackoffValidation(t *testing.T) {
	config := transport.DefaultConnectionConfig()

	// Validate backoff parameters are reasonable
	assert.Greater(t, config.BackoffConfig.Multiplier, 1.0)
	assert.LessOrEqual(t, config.BackoffConfig.Jitter, 1.0)
	assert.GreaterOrEqual(t, config.BackoffConfig.Jitter, 0.0)
	assert.Greater(t, config.BackoffConfig.MaxDelay, config.BackoffConfig.BaseDelay)
}

func TestConnectionConfigKeepaliveValidation(t *testing.T) {
	config := transport.DefaultConnectionConfig()

	// Validate keepalive parameters are reasonable
	assert.Greater(t, config.KeepaliveParams.Time, time.Duration(0))
	assert.Greater(t, config.KeepaliveParams.Timeout, time.Duration(0))
	assert.Less(t, config.KeepaliveParams.Timeout, config.KeepaliveParams.Time)
}
