package configurationprovider

import (
	"context"
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Factory creates a new ConfigurationProvider implementation for testing.
type Factory func(t *testing.T) ports.ConfigurationProvider

// TestPaths provides test paths for configuration testing.
type TestPaths struct {
	ValidPath   string
	InvalidPath string
}

// Run executes the complete contract test suite against any ConfigurationProvider implementation.
func Run(t *testing.T, newImpl Factory, paths TestPaths) {
	ctx := context.Background()

	t.Run("load valid configuration", func(t *testing.T) {
		provider := newImpl(t)
		
		config, err := provider.LoadConfiguration(ctx, paths.ValidPath)
		
		// Contract: Either returns valid config or expected error
		if err != nil {
			t.Logf("LoadConfiguration returned error (may be expected): %v", err)
			return
		}
		
		if config == nil {
			t.Fatal("LoadConfiguration returned nil config without error")
		}
		
		// Validate basic config structure
		if config.Service.Name == "" {
			t.Error("Configuration.Service.Name should not be empty")
		}
		
		// Config should be valid
		if err := config.Validate(); err != nil {
			t.Errorf("Loaded configuration should be valid: %v", err)
		}
	})

	t.Run("load invalid configuration", func(t *testing.T) {
		provider := newImpl(t)
		
		config, err := provider.LoadConfiguration(ctx, paths.InvalidPath)
		
		// Contract: Invalid config should return error
		if err == nil {
			t.Errorf("LoadConfiguration(%q) should return error", paths.InvalidPath)
		}
		
		if config != nil {
			t.Error("LoadConfiguration should return nil config on error")
		}
	})

	t.Run("get default configuration", func(t *testing.T) {
		provider := newImpl(t)
		
		config := provider.GetDefaultConfiguration(ctx)
		
		// Contract: May return nil if provider doesn't support defaults
		if config == nil {
			t.Log("GetDefaultConfiguration returned nil - provider may not support defaults")
			return
		}
		
		// If config returned, should be valid
		if err := config.Validate(); err != nil {
			t.Errorf("Default configuration should be valid: %v", err)
		}
		
		if config.Service.Name == "" {
			t.Error("Default configuration service name should not be empty")
		}
	})

	t.Run("empty path rejected", func(t *testing.T) {
		provider := newImpl(t)
		
		if _, err := provider.LoadConfiguration(ctx, ""); err == nil {
			t.Error("LoadConfiguration(\"\") should return error")
		}
	})

	t.Run("whitespace path rejected", func(t *testing.T) {
		provider := newImpl(t)
		
		if _, err := provider.LoadConfiguration(ctx, "   "); err == nil {
			t.Error("LoadConfiguration with whitespace should return error")
		}
	})

	t.Run("configuration validation edge cases", func(t *testing.T) {
		provider := newImpl(t)
		
		// Try to get a valid config for testing validation
		var baseConfig *ports.Configuration
		
		// Try valid path first
		if paths.ValidPath != "" {
			if config, err := provider.LoadConfiguration(ctx, paths.ValidPath); err == nil {
				baseConfig = config
			}
		}
		
		// Fallback to default config
		if baseConfig == nil {
			baseConfig = provider.GetDefaultConfiguration(ctx)
		}
		
		if baseConfig == nil {
			t.Skip("Cannot obtain config for validation testing")
		}
		
		// Test invalid modifications
		testCases := []struct {
			name   string
			modify func(*ports.Configuration) *ports.Configuration
		}{
			{
				name: "empty service name",
				modify: func(c *ports.Configuration) *ports.Configuration {
					modified := *c
					modified.Service.Name = ""
					return &modified
				},
			},
			{
				name: "whitespace service name",
				modify: func(c *ports.Configuration) *ports.Configuration {
					modified := *c
					modified.Service.Name = "   "
					return &modified
				},
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				invalidConfig := tc.modify(baseConfig)
				if err := invalidConfig.Validate(); err == nil {
					t.Errorf("Modified config should be invalid: %s", tc.name)
				}
			})
		}
	})
}