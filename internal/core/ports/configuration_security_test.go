package ports_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestLoadFromEnvironment(t *testing.T) {
	// Clear environment for test
	for _, env := range []string{
		ports.EnvServiceName,
		ports.EnvTrustDomain,
		ports.EnvAgentSocket,
		ports.EnvDebugEnabled,
	} {
		t.Setenv(env, "")
	}

	tests := []struct {
		name          string
		envVars       map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid production configuration",
			envVars: map[string]string{
				ports.EnvServiceName: "payment-service",
				ports.EnvTrustDomain: "prod.company.com",
				ports.EnvAgentSocket: "/run/spire/sockets/api.sock",
			},
			expectError: false,
		},
		{
			name: "missing required service name",
			envVars: map[string]string{
				ports.EnvTrustDomain: "prod.company.com",
			},
			expectError:   true,
			errorContains: "service name is required",
		},
		{
			name: "production security - example.org domain",
			envVars: map[string]string{
				ports.EnvServiceName: "payment-service",
				ports.EnvTrustDomain: "example.org",
			},
			expectError:   true,
			errorContains: "trust domain contains 'example.org'",
		},
		{
			name: "production security - localhost domain",
			envVars: map[string]string{
				ports.EnvServiceName: "payment-service",
				ports.EnvTrustDomain: "localhost.local",
			},
			expectError:   true,
			errorContains: "trust domain contains 'localhost'",
		},
		{
			name: "production security - example service name",
			envVars: map[string]string{
				ports.EnvServiceName: "example-service",
				ports.EnvTrustDomain: "prod.company.com",
			},
			expectError:   true,
			errorContains: "service name contains demo/example values",
		},
		{
			name: "production security - debug enabled",
			envVars: map[string]string{
				ports.EnvServiceName:  "payment-service",
				ports.EnvTrustDomain:  "prod.company.com",
				ports.EnvDebugEnabled: "true",
			},
			expectError:   true,
			errorContains: "debug mode is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			config, err := ports.LoadFromEnvironment()

			if tt.expectError {
				// Some tests expect LoadFromEnvironment to fail (e.g., missing required fields)
				// Others expect it to succeed but fail production validation
				if err != nil {
					// LoadFromEnvironment failed as expected for validation errors
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
					assert.Nil(t, config)
				} else {
					// LoadFromEnvironment succeeded, check production readiness
					assert.NotNil(t, config)
					prodErr := config.IsProductionReady()
					assert.Error(t, prodErr, "IsProductionReady should fail for non-production configs")
					if tt.errorContains != "" {
						assert.Contains(t, prodErr.Error(), tt.errorContains)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)

				// Verify configuration values
				assert.Equal(t, tt.envVars[ports.EnvServiceName], config.Service.Name)
				if domain, ok := tt.envVars[ports.EnvTrustDomain]; ok {
					assert.Equal(t, domain, config.Service.Domain)
				}
			}
		})
	}
}

func TestMergeWithEnvironment(t *testing.T) {
	// Clear environment for test
	for _, env := range []string{
		ports.EnvServiceName,
		ports.EnvTrustDomain,
	} {
		t.Setenv(env, "")
	}

	// Create initial configuration
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   domain.NewServiceNameUnsafe("file-service"),
			Domain: "file.domain.com",
		},
		Agent: &ports.AgentConfig{
			SocketPath: domain.NewSocketPathUnsafe("/tmp/file/socket"),
		},
	}

	// Set environment variables that should override
	t.Setenv(ports.EnvServiceName, "env-service")
	t.Setenv(ports.EnvTrustDomain, "env.domain.com")

	err := config.MergeWithEnvironment()
	assert.NoError(t, err)

	// Verify environment variables override file values
	assert.Equal(t, "env-service", config.Service.Name)
	assert.Equal(t, "env.domain.com", config.Service.Domain)

	// Verify file values remain where no environment override
	assert.Equal(t, "/tmp/file/socket", config.Agent.SocketPath.Value())
}

func TestValidateProductionSecurity(t *testing.T) {
	tests := []struct {
		name          string
		config        *ports.Configuration
		expectError   bool
		errorContains string
	}{
		{
			name: "valid production config",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("payment-service"),
					Domain: "prod.company.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/run/spire/sockets/api.sock"),
				},
			},
			expectError: false,
		},
		{
			name: "example.org domain",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("payment-service"),
					Domain: "example.org",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/run/spire/sockets/api.sock"),
				},
			},
			expectError:   true,
			errorContains: "trust domain contains 'example.org'",
		},
		{
			name: "localhost domain",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("payment-service"),
					Domain: "localhost",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/run/spire/sockets/api.sock"),
				},
			},
			expectError:   true,
			errorContains: "trust domain contains 'localhost'",
		},
		{
			name: "example.com domain",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("payment-service"),
					Domain: "test.example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/run/spire/sockets/api.sock"),
				},
			},
			expectError:   true,
			errorContains: "trust domain contains 'example.com'",
		},
		{
			name: "demo service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("demo-service"),
					Domain: "prod.company.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/run/spire/sockets/api.sock"),
				},
			},
			expectError:   true,
			errorContains: "service name contains demo/example values",
		},
		{
			name: "insecure socket path",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("payment-service"),
					Domain: "prod.company.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/home/user/spire.sock"),
				},
			},
			expectError:   true,
			errorContains: "SPIFFE socket should be in a secure directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.IsProductionReady()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	originalValue := os.Getenv("TEST_BOOL_ENV")
	defer func() {
		if originalValue != "" {
			t.Setenv("TEST_BOOL_ENV", originalValue)
		} else {
			os.Unsetenv("TEST_BOOL_ENV")
		}
	}()

	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true value",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false value",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "1 value",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "0 value",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "invalid value uses default",
			envValue:     "invalid",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "empty value uses default",
			envValue:     "",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "unset uses default",
			envValue:     "", // Will be unset
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TEST_BOOL_ENV", tt.envValue)
			} else {
				os.Unsetenv("TEST_BOOL_ENV")
			}

			result := ports.GetBoolEnv("TEST_BOOL_ENV", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentVariableConstants(t *testing.T) {
	// Verify environment variable names follow consistent naming convention
	expectedVars := map[string]string{
		"EPHEMOS_SERVICE_NAME":           ports.EnvServiceName,
		"EPHEMOS_TRUST_DOMAIN":           ports.EnvTrustDomain,
		"EPHEMOS_AGENT_SOCKET":           ports.EnvAgentSocket,
		"EPHEMOS_REQUIRE_AUTHENTICATION": ports.EnvRequireAuth,
		"EPHEMOS_LOG_LEVEL":              ports.EnvLogLevel,
		"EPHEMOS_BIND_ADDRESS":           ports.EnvBindAddress,
		"EPHEMOS_TLS_MIN_VERSION":        ports.EnvTLSMinVersion,
		"EPHEMOS_DEBUG_ENABLED":          ports.EnvDebugEnabled,
	}

	for expected, actual := range expectedVars {
		assert.Equal(t, expected, actual, "Environment variable constant should match expected name")
		assert.True(t, strings.HasPrefix(actual, "EPHEMOS_"), "All environment variables should start with EPHEMOS_")
	}
}

func TestProductionSecurityIntegration(t *testing.T) {
	// Store original environment
	originalDebug := os.Getenv(ports.EnvDebugEnabled)
	defer func() {
		if originalDebug != "" {
			t.Setenv(ports.EnvDebugEnabled, originalDebug)
		} else {
			os.Unsetenv(ports.EnvDebugEnabled)
		}
	}()

	// Test that debug mode detection works in production validation
	t.Setenv(ports.EnvServiceName, "payment-service")
	t.Setenv(ports.EnvTrustDomain, "prod.company.com")
	t.Setenv(ports.EnvDebugEnabled, "true")

	config, err := ports.LoadFromEnvironment()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "debug mode is enabled")

	// Test that production config works when debug is disabled
	t.Setenv(ports.EnvDebugEnabled, "false")
	config, err = ports.LoadFromEnvironment()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Clean up
	os.Unsetenv(ports.EnvServiceName)
	os.Unsetenv(ports.EnvTrustDomain)
	os.Unsetenv(ports.EnvDebugEnabled)
}
