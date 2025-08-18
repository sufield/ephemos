package ports_test

import (
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *ports.Configuration
		wantErr       bool
		errorContains string
	}{
		{
			name: "valid configuration",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("test-service"),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr: false,
		},
		{
			name: "empty service config",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{}, // Empty service config to test validation
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr:       true,
			errorContains: "service name is required",
		},
		{
			name: "nil SPIFFE config",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("test-service"),
					Domain: "example.com",
				},
				Agent: nil,
			},
			wantErr:       false, // SPIFFE config is optional
			errorContains: "",
		},
		{
			name: "empty service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe(""),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr:       true,
			errorContains: "service name is required",
		},
		{
			name: "empty service domain",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("test-service"),
					Domain: "",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr:       false, // Domain is optional
			errorContains: "",
		},
		{
			name: "whitespace only service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("   "),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr:       true,
			errorContains: "service name is required",
		},
		{
			name: "empty socket path",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("test-service"),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe(""),
				},
			},
			wantErr:       true,
			errorContains: "socket path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errorContains)
				}
			}
		})
	}
}

func TestServiceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ports.ServiceConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ports.ServiceConfig{
				Name:   domain.NewServiceNameUnsafe("test-service"),
				Domain: "example.com",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: ports.ServiceConfig{
				Name:   domain.NewServiceNameUnsafe(""),
				Domain: "example.com",
			},
			wantErr: true,
		},
		{
			name: "empty domain",
			config: ports.ServiceConfig{
				Name:   domain.NewServiceNameUnsafe("test-service"),
				Domain: "",
			},
			wantErr: false, // Domain is optional
		},
		{
			name: "whitespace name",
			config: ports.ServiceConfig{
				Name:   domain.NewServiceNameUnsafe("   "),
				Domain: "example.com",
			},
			wantErr: true,
		},
		{
			name: "whitespace domain",
			config: ports.ServiceConfig{
				Name:   domain.NewServiceNameUnsafe("test-service"),
				Domain: "   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test service validation by creating a Configuration and validating it
			config := &ports.Configuration{
				Service: tt.config,
			}
			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ServiceConfig validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ports.AgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ports.AgentConfig{
				SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
			},
			wantErr: false,
		},
		{
			name: "empty socket path",
			config: &ports.AgentConfig{
				SocketPath: domain.NewSocketPathUnsafe(""),
			},
			wantErr: true,
		},
		{
			name: "whitespace socket path",
			config: &ports.AgentConfig{
				SocketPath: domain.NewSocketPathUnsafe("  /tmp/spire-agent/public/api.sock  "),
			},
			wantErr: false, // Should be trimmed and valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Agent config validation by creating a Configuration with it
			config := &ports.Configuration{
				Service: ports.ServiceConfig{
					Name: domain.NewServiceNameUnsafe("test-service"),
				},
				Agent: tt.config,
			}
			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SPIFFEConfig validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfiguration_DefaultValues(t *testing.T) {
	// Test that configuration provides reasonable defaults where appropriate
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   domain.NewServiceNameUnsafe("test-service"),
			Domain: "example.com",
		},
		Agent: &ports.AgentConfig{
			SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Valid configuration should not return error: %v", err)
	}
}

func TestConfiguration_EdgeCases(t *testing.T) {
	// Test edge cases and boundary conditions
	tests := []struct {
		name    string
		config  *ports.Configuration
		wantErr bool
	}{
		{
			name: "very long service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe(strings.Repeat("a", 1000)),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr: false, // Should be valid unless there's a length limit
		},
		{
			name: "unicode service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("测试服务"),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
				},
			},
			wantErr: true, // Unicode not allowed in service names
		},
		{
			name: "special characters in path",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   domain.NewServiceNameUnsafe("test-service"),
					Domain: "example.com",
				},
				Agent: &ports.AgentConfig{
					SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock?query=1"),
				},
			},
			wantErr: false, // Path validation may vary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkConfiguration_Validate(b *testing.B) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   domain.NewServiceNameUnsafe("test-service"),
			Domain: "example.com",
		},
		Agent: &ports.AgentConfig{
			SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Errorf("Validate() failed: %v", err)
		}
	}
}

func BenchmarkServiceConfig_Validate(b *testing.B) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   domain.NewServiceNameUnsafe("test-service"),
			Domain: "example.com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Errorf("Validate() failed: %v", err)
		}
	}
}

func BenchmarkSPIFFEConfig_Validate(b *testing.B) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name: domain.NewServiceNameUnsafe("test-service"),
		},
		Agent: &ports.AgentConfig{
			SocketPath: domain.NewSocketPathUnsafe("/tmp/spire-agent/public/api.sock"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Errorf("Validate() failed: %v", err)
		}
	}
}
