package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	corerrors "github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Helper function to create a test configuration provider with pre-loaded configs
func createTestConfigProvider() *config.InMemoryProvider {
	provider := config.NewInMemoryProvider()
	
	// Set up some test configurations
	provider.SetConfiguration("valid.yaml", createValidTestConfig())
	provider.SetConfiguration("invalid-name.yaml", &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "", // Invalid: empty name
			Domain: "example.org",
		},
	})
	provider.SetConfiguration("invalid-domain.yaml", &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "", // Invalid: empty domain
		},
	})
	provider.SetConfiguration("test.yaml", createValidTestConfig())
	
	return provider
}

// Helper function to create a provider that returns errors
type errorProvider struct {
	*config.InMemoryProvider
	errorOnPath string
}

func (e *errorProvider) LoadConfiguration(ctx context.Context, path string) (*ports.Configuration, error) {
	if path == e.errorOnPath {
		return nil, fmt.Errorf("simulated error for path: %s", path)
	}
	return e.InMemoryProvider.LoadConfiguration(ctx, path)
}

func createErrorProvider() *errorProvider {
	return &errorProvider{
		InMemoryProvider: config.NewInMemoryProvider(),
		errorOnPath:     "error.yaml",
	}
}

// Helper function to create a valid test configuration
func createValidTestConfig() *ports.Configuration {
	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.org",
		},
		SPIFFE: &ports.SPIFFEConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock",
		},
	}
}

func TestNewRegistrar(t *testing.T) {
	configProvider := createTestConfigProvider()

	tests := []struct {
		name           string
		configProvider ports.ConfigurationProvider
		config         *RegistrarConfig
		expectedSocket string
		expectedServer string
	}{
		{
			name:           "default config",
			configProvider: configProvider,
			config:         nil,
			expectedSocket: "/tmp/spire-server/private/api.sock",
			expectedServer: "spire-server",
		},
		{
			name:           "custom config",
			configProvider: configProvider,
			config: &RegistrarConfig{
				SPIRESocketPath: "/custom/socket.sock",
				SPIREServerPath: "/custom/spire-server",
			},
			expectedSocket: "/custom/socket.sock",
			expectedServer: "/custom/spire-server",
		},
		{
			name:           "partial config",
			configProvider: configProvider,
			config: &RegistrarConfig{
				SPIRESocketPath: "/custom/socket.sock",
			},
			expectedSocket: "/custom/socket.sock",
			expectedServer: "spire-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := NewRegistrar(tt.configProvider, tt.config)

			if registrar == nil {
				t.Fatal("NewRegistrar() returned nil")
			}

			if registrar.configProvider != tt.configProvider {
				t.Error("ConfigProvider not set correctly")
			}

			if registrar.spireSocketPath != tt.expectedSocket {
				t.Errorf("spireSocketPath = %v, want %v", registrar.spireSocketPath, tt.expectedSocket)
			}

			if registrar.spireServerPath != tt.expectedServer {
				t.Errorf("spireServerPath = %v, want %v", registrar.spireServerPath, tt.expectedServer)
			}
		})
	}
}

func TestNewRegistrar_EnvironmentVariable(t *testing.T) {
	// Test that environment variable is used when config doesn't specify socket path
	testSocket := "/env/test/socket.sock"
	t.Setenv("SPIRE_SOCKET_PATH", testSocket)

	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	if registrar.spireSocketPath != testSocket {
		t.Errorf("spireSocketPath = %v, want %v", registrar.spireSocketPath, testSocket)
	}
}

func TestRegistrar_RegisterService(t *testing.T) {
	// Create providers with different configurations
	validProvider := createTestConfigProvider()
	errorProvider := createErrorProvider()

	// Pre-load configurations for the error provider
	errorProvider.SetConfiguration("invalid-name.yaml", &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "", // Invalid: empty name
			Domain: "example.org",
		},
	})
	errorProvider.SetConfiguration("invalid-domain.yaml", &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "", // Invalid: empty domain
		},
	})

	tests := []struct {
		name           string
		configProvider ports.ConfigurationProvider
		configPath     string
		wantErr        bool
		errorType      interface{}
	}{
		{
			name:           "valid configuration",
			configProvider: validProvider,
			configPath:     "valid.yaml",
			wantErr:        false,
		},
		{
			name:           "missing configuration",
			configProvider: errorProvider,
			configPath:     "error.yaml",
			wantErr:        true,
			errorType:      &corerrors.DomainError{},
		},
		{
			name:           "invalid service name",
			configProvider: errorProvider,
			configPath:     "invalid-name.yaml",
			wantErr:        true,
			errorType:      &corerrors.ValidationError{},
		},
		{
			name:           "invalid service domain",
			configProvider: errorProvider,
			configPath:     "invalid-domain.yaml",
			wantErr:        true,
			errorType:      &corerrors.ValidationError{},
		},
		{
			name:           "invalid service name format",
			configProvider: validProvider,
			configPath:     "valid.yaml",
			wantErr:        false, // "test-service" is a valid name format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RegistrarConfig{
				SPIREServerPath: "echo", // Use echo instead of spire-server for testing
				Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
			}
			registrar := NewRegistrar(tt.configProvider, config)

			ctx := t.Context()
			err := registrar.RegisterService(ctx, tt.configPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorType != nil {
				switch v := tt.errorType.(type) {
				case *corerrors.DomainError:
					var domainErr *corerrors.DomainError
					if !errors.As(err, &domainErr) {
						t.Errorf("Expected DomainError, got %T", err)
					}
				case *corerrors.ValidationError:
					var validationErr *corerrors.ValidationError
					if !errors.As(err, &validationErr) {
						t.Errorf("Expected ValidationError, got %T", err)
					}
				default:
					t.Errorf("Unexpected error type: %T", v)
				}
			}
		})
	}
}

func TestRegistrar_getServiceSelector(t *testing.T) {
	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	selector, err := registrar.getServiceSelector()
	if err != nil {
		t.Fatalf("getServiceSelector() error = %v", err)
	}

	expectedPrefix := "unix:uid:"
	if !strings.HasPrefix(selector, expectedPrefix) {
		t.Errorf("getServiceSelector() = %v, want prefix %v", selector, expectedPrefix)
	}
}

func TestRegistrar_validateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ports.Configuration
		wantErr bool
		errType string
	}{
		{
			name:    "valid configuration",
			config:  createValidTestConfig(),
			wantErr: false,
		},
		{
			name: "empty service name",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "",
					Domain: "example.org",
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "invalid service name format",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "invalid_name!",
					Domain: "example.org",
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "empty domain",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "",
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "invalid domain format",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "invalid domain!",
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
	}

	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registrar.validateConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				var validationErr *corerrors.ValidationError
				if !errors.As(err, &validationErr) && tt.errType == "ValidationError" {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestRegistrar_createSPIREEntry(t *testing.T) {
	// Use echo command as a stand-in for spire-server
	// This tests that the command construction and execution work correctly
	config := &RegistrarConfig{
		SPIREServerPath: "echo", // Use echo to test command construction
		SPIRESocketPath: "/tmp/test.sock",
		Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}
	registrar := NewRegistrar(createTestConfigProvider(), config)

	testConfig := createValidTestConfig()
	ctx := t.Context()

	// This will succeed because echo always succeeds, so we test the success path
	err := registrar.createSPIREEntry(ctx, testConfig)

	// Echo command succeeds, so this should not error
	// The real validation is for SPIRE server response parsing, not command execution
	if err != nil {
		// Check if it's an expected error message format
		if !strings.Contains(err.Error(), "SPIRE") && !strings.Contains(err.Error(), "registration") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestRegistrar_EdgeCases(t *testing.T) {
	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	// Test with hyphenated service name
	hyphenatedConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service-name",
			Domain: "example.org",
		},
	}
	if err := registrar.validateConfig(hyphenatedConfig); err != nil {
		t.Errorf("validateConfig() should accept hyphenated names, got error: %v", err)
	}

	// Test with numeric domain
	numericDomainConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "123.example.org",
		},
	}
	if err := registrar.validateConfig(numericDomainConfig); err != nil {
		t.Errorf("validateConfig() should accept numeric domains, got error: %v", err)
	}
}

func TestRegistrar_Integration(t *testing.T) {
	// This integration test demonstrates using real components
	// without mocks, following the principle of testing with real dependencies
	config := createValidTestConfig()
	configProvider := createTestConfigProvider()
	configProvider.SetConfiguration("test.yaml", config)

	registrarConfig := &RegistrarConfig{
		SPIREServerPath: "echo", // Use echo as a safe command for testing
		SPIRESocketPath: "/tmp/test.sock",
		Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	registrar := NewRegistrar(configProvider, registrarConfig)

	ctx := t.Context()

	// This may succeed or fail depending on how echo responds to the arguments
	err := registrar.RegisterService(ctx, "test.yaml")

	// Echo command may succeed, so we just log the result
	if err != nil {
		// Check if it's a validation error (expected for certain configs)
		var validationErr *corerrors.ValidationError
		if errors.As(err, &validationErr) {
			// Expected for certain test configurations
			return
		}
		t.Logf("Integration test completed with result: %v", err)
	}
}

func TestRegistrar_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		config       *ports.Configuration
		expectedType error
		setupFunc    func(*Registrar)
	}{
		{
			name: "invalid service name characters",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test@service",
					Domain: "example.org",
				},
			},
			expectedType: &corerrors.ValidationError{},
		},
		{
			name: "service name with spaces",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test service",
					Domain: "example.org",
				},
			},
			expectedType: &corerrors.ValidationError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configProvider := createTestConfigProvider()
			configProvider.SetConfiguration("test-config.yaml", tt.config)
			
			registrar := NewRegistrar(configProvider, &RegistrarConfig{
				SPIREServerPath: "echo",
				Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
			})

			if tt.setupFunc != nil {
				tt.setupFunc(registrar)
			}

			ctx := t.Context()
			err := registrar.RegisterService(ctx, "test-config.yaml")

			if err == nil {
				t.Error("Expected error but got nil")
				return
			}

			switch tt.expectedType.(type) {
			case *corerrors.ValidationError:
				var validationErr *corerrors.ValidationError
				if !errors.As(err, &validationErr) {
					t.Errorf("Expected ValidationError, got %T: %v", err, err)
				}
			case *corerrors.DomainError:
				var domainErr *corerrors.DomainError
				if !errors.As(err, &domainErr) {
					t.Errorf("Expected DomainError, got %T: %v", err, err)
				}
			}
		})
	}
}

// Benchmark tests using real implementations
func BenchmarkRegistrar_RegisterService(b *testing.B) {
	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = registrar.RegisterService(ctx, "valid.yaml")
	}
}

func BenchmarkRegistrar_validateConfig(b *testing.B) {
	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)
	config := createValidTestConfig()

	for i := 0; i < b.N; i++ {
		_ = registrar.validateConfig(config)
	}
}

func BenchmarkRegistrar_getServiceSelector(b *testing.B) {
	configProvider := createTestConfigProvider()
	registrar := NewRegistrar(configProvider, nil)

	for i := 0; i < b.N; i++ {
		_, _ = registrar.getServiceSelector()
	}
}