package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewFileProvider(t *testing.T) {
	provider := config.NewFileProvider()
	if provider == nil {
		t.Error("config.NewFileProvider() returned nil")
	}
}

//nolint:cyclop // Test functions can have high complexity due to multiple test cases
func TestFileProvider_LoadConfiguration(t *testing.T) {
	provider := config.NewFileProvider()
	ctx := t.Context()

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
		setup      func(t *testing.T) (string, func())
	}{
		{
			name:       "nil context",
			configPath: "",
			wantErr:    true,
		},
		{
			name:       "empty config path",
			configPath: "",
			wantErr:    true,
		},
		{
			name:       "nonexistent file",
			configPath: "/nonexistent/path/config.yaml",
			wantErr:    true,
		},
		{
			name:    "valid config file",
			wantErr: false,
			setup: func(t *testing.T) (string, func()) {
				t.Helper()
				// Create a temporary config file
				tmpDir := t.TempDir()

				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `
service:
  name: "test-service"
  domain: "example.com"

spiffe:
  socketPath: "/tmp/spire-agent/public/api.sock"
`
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				return configPath, func() {}
			},
		},
		{
			name:    "invalid yaml format",
			wantErr: true,
			setup: func(t *testing.T) (string, func()) {
				t.Helper()
				tmpDir := t.TempDir()

				configPath := filepath.Join(tmpDir, "config.yaml")
				invalidContent := `invalid: yaml: content: [[[`
				err := os.WriteFile(configPath, []byte(invalidContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				return configPath, func() {}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			var cleanup func()

			if tt.setup != nil {
				configPath, cleanup = tt.setup(t)
				defer cleanup()
			} else {
				configPath = tt.configPath
			}

			var config *ports.Configuration
			var err error
			if tt.name == "nil context" {
				// Test with canceled context to simulate nil context behavior
				nilCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately to make it behave like nil
				config, err = provider.LoadConfiguration(nilCtx, configPath)
			} else {
				testCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				config, err = provider.LoadConfiguration(testCtx, configPath)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadConfiguration() returned nil config without error")
			}

			// Additional validation for successful cases
			if !tt.wantErr && config != nil {
				if err := config.Validate(); err != nil {
					t.Errorf("Loaded configuration is invalid: %v", err)
				}
			}
		})
	}
}

func TestFileProvider_GetDefaultConfiguration(t *testing.T) {
	provider := config.NewFileProvider()
	ctx := t.Context()

	tests := []struct {
		name   string
		useCtx bool
	}{
		{
			name:   "valid context",
			useCtx: true,
		},
		{
			name:   "nil context",
			useCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config *ports.Configuration
			if tt.useCtx {
				testCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				config = provider.GetDefaultConfiguration(testCtx)
			} else {
				// Test with canceled context to simulate nil context behavior
				nilCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately to make it behave like nil
				config = provider.GetDefaultConfiguration(nilCtx)
			}

			// Default configuration should be returned even with nil context
			if config == nil {
				t.Error("GetDefaultConfiguration() returned nil")
				return
			}

			// Default configuration should be valid
			if err := config.Validate(); err != nil {
				t.Errorf("Default configuration is invalid: %v", err)
			}

			// Verify default values
			if config.Agent == nil {
				t.Error("Default configuration missing Agent section")
			}
		})
	}
}

func TestFileProvider_Integration(t *testing.T) {
	// Integration test showing typical usage pattern
	provider := config.NewFileProvider()
	ctx := t.Context()

	// First try to load a config file, fall back to default
	_, err := provider.LoadConfiguration(ctx, "/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}

	// Get default configuration as fallback
	defaultConfig := provider.GetDefaultConfiguration(ctx)
	if defaultConfig == nil {
		t.Fatal("Default configuration should not be nil")
	}

	// Validate the default configuration
	if err := defaultConfig.Validate(); err != nil {
		t.Errorf("Default configuration validation failed: %v", err)
	}
}

func BenchmarkConfigProvider_GetDefaultConfiguration(b *testing.B) {
	provider := config.NewFileProvider()
	ctx := b.Context()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := provider.GetDefaultConfiguration(ctx)
		if config == nil {
			b.Error("GetDefaultConfiguration returned nil")
		}
	}
}

func BenchmarkConfigProvider_LoadConfiguration(b *testing.B) {
	provider := config.NewFileProvider()
	ctx := b.Context()

	// Create a temporary config file
	tmpDir := b.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
service:
  name: "benchmark-service"
  domain: "test.example.com"

spiffe:
  socketPath: "/tmp/spire-agent/public/api.sock"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.LoadConfiguration(ctx, configPath)
		if err != nil {
			b.Errorf("LoadConfiguration failed: %v", err)
		}
	}
}
