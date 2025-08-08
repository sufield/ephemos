package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigProvider(t *testing.T) {
	provider := NewConfigProvider()
	if provider == nil {
		t.Error("NewConfigProvider() returned nil")
	}
}

func TestConfigProvider_LoadConfiguration(t *testing.T) {
	provider := NewConfigProvider()
	ctx := context.Background()

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
		setup      func() (string, func())
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
			setup: func() (string, func()) {
				// Create a temporary config file
				tmpDir, err := os.MkdirTemp("", "ephemos-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				
				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `
spiffe:
  domain: "test.example.com"
  socket_path: "/tmp/spire-agent/public/api.sock"
  trust_domain: "example.com"
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				
				return configPath, func() { os.RemoveAll(tmpDir) }
			},
		},
		{
			name:    "invalid yaml format",
			wantErr: true,
			setup: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "ephemos-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				
				configPath := filepath.Join(tmpDir, "config.yaml")
				invalidContent := `invalid: yaml: content: [[[`
				err = os.WriteFile(configPath, []byte(invalidContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				
				return configPath, func() { os.RemoveAll(tmpDir) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			var cleanup func()
			
			if tt.setup != nil {
				configPath, cleanup = tt.setup()
				defer cleanup()
			} else {
				configPath = tt.configPath
			}

			testCtx := ctx
			if tt.name == "nil context" {
				testCtx = nil
			}

			config, err := provider.LoadConfiguration(testCtx, configPath)
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

func TestConfigProvider_GetDefaultConfiguration(t *testing.T) {
	provider := NewConfigProvider()
	ctx := context.Background()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "valid context",
			ctx:  ctx,
		},
		{
			name: "nil context",
			ctx:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := provider.GetDefaultConfiguration(tt.ctx)
			
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
			if config.SPIFFE == nil {
				t.Error("Default configuration missing SPIFFE section")
			}
		})
	}
}

func TestConfigProvider_Integration(t *testing.T) {
	// Integration test showing typical usage pattern
	provider := NewConfigProvider()
	ctx := context.Background()

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
	provider := NewConfigProvider()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := provider.GetDefaultConfiguration(ctx)
		if config == nil {
			b.Error("GetDefaultConfiguration returned nil")
		}
	}
}

func BenchmarkConfigProvider_LoadConfiguration(b *testing.B) {
	provider := NewConfigProvider()
	ctx := context.Background()

	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "ephemos-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
spiffe:
  domain: "test.example.com" 
  socket_path: "/tmp/spire-agent/public/api.sock"
  trust_domain: "example.com"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
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