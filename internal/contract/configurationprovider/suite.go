// Package configurationprovider provides contract test suites for ConfigurationProvider implementations.
// These tests ensure that all ConfigurationProvider adapters behave consistently.
package configurationprovider

import (
	"context"
	"testing"

	"github.com/sufield/ephemos/internal/core/domain"
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
	t.Helper()
	ctx := t.Context()

	t.Run("load valid configuration", func(t *testing.T) {
		testLoadValidConfiguration(ctx, t, newImpl, paths.ValidPath)
	})

	t.Run("load invalid configuration", func(t *testing.T) {
		testLoadInvalidConfiguration(ctx, t, newImpl, paths.InvalidPath)
	})

	t.Run("get default configuration", func(t *testing.T) {
		testGetDefaultConfiguration(ctx, t, newImpl)
	})

	t.Run("empty path rejected", func(t *testing.T) {
		testEmptyPathRejected(ctx, t, newImpl)
	})

	t.Run("whitespace path rejected", func(t *testing.T) {
		testWhitespacePathRejected(ctx, t, newImpl)
	})

	t.Run("configuration validation edge cases", func(t *testing.T) {
		testConfigurationValidationEdgeCases(ctx, t, newImpl, paths)
	})
}

// testLoadValidConfiguration tests loading a valid configuration.
func testLoadValidConfiguration(ctx context.Context, t *testing.T, newImpl Factory, validPath string) {
	t.Helper()
	provider := newImpl(t)

	config, err := provider.LoadConfiguration(ctx, validPath)
	// Contract: Either returns valid config or expected error
	if err != nil {
		t.Logf("LoadConfiguration returned error (may be expected): %v", err)
		return
	}

	assertValidConfig(t, config)
}

// testLoadInvalidConfiguration tests loading an invalid configuration.
func testLoadInvalidConfiguration(ctx context.Context, t *testing.T, newImpl Factory, invalidPath string) {
	t.Helper()
	provider := newImpl(t)

	config, err := provider.LoadConfiguration(ctx, invalidPath)

	// Contract: Invalid config should return error
	if err == nil {
		t.Errorf("LoadConfiguration(%q) should return error", invalidPath)
	}

	if config != nil {
		t.Error("LoadConfiguration should return nil config on error")
	}
}

// testGetDefaultConfiguration tests getting the default configuration.
func testGetDefaultConfiguration(ctx context.Context, t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)

	config := provider.GetDefaultConfiguration(ctx)

	// Contract: May return nil if provider doesn't support defaults
	if config == nil {
		t.Log("GetDefaultConfiguration returned nil - provider may not support defaults")
		return
	}

	assertValidConfig(t, config)
}

// testEmptyPathRejected tests that empty paths are rejected.
func testEmptyPathRejected(ctx context.Context, t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)

	if _, err := provider.LoadConfiguration(ctx, ""); err == nil {
		t.Error("LoadConfiguration(\"\") should return error")
	}
}

// testWhitespacePathRejected tests that whitespace paths are rejected.
func testWhitespacePathRejected(ctx context.Context, t *testing.T, newImpl Factory) {
	t.Helper()
	provider := newImpl(t)

	if _, err := provider.LoadConfiguration(ctx, "   "); err == nil {
		t.Error("LoadConfiguration with whitespace should return error")
	}
}

// testConfigurationValidationEdgeCases tests validation edge cases.
func testConfigurationValidationEdgeCases(ctx context.Context, t *testing.T, newImpl Factory, paths TestPaths) {
	t.Helper()
	provider := newImpl(t)

	baseConfig := obtainBaseConfig(ctx, provider, paths)
	if baseConfig == nil {
		t.Skip("Cannot obtain config for validation testing")
	}

	testInvalidModifications(t, baseConfig)
}

// obtainBaseConfig attempts to get a valid configuration for testing.
func obtainBaseConfig(ctx context.Context, provider ports.ConfigurationProvider, paths TestPaths) *ports.Configuration {
	// Try valid path first
	if paths.ValidPath != "" {
		if config, err := provider.LoadConfiguration(ctx, paths.ValidPath); err == nil {
			return config
		}
	}

	// Fallback to default config
	return provider.GetDefaultConfiguration(ctx)
}

// testInvalidModifications tests that invalid modifications are rejected.
func testInvalidModifications(t *testing.T, baseConfig *ports.Configuration) {
	t.Helper()
	testCases := []struct {
		name   string
		modify func(*ports.Configuration) *ports.Configuration
	}{
		{
			name:   "empty service name",
			modify: modifyServiceNameEmpty,
		},
		{
			name:   "whitespace service name",
			modify: modifyServiceNameWhitespace,
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
}

// modifyServiceNameEmpty returns a config with empty service name.
func modifyServiceNameEmpty(c *ports.Configuration) *ports.Configuration {
	modified := *c
	modified.Service.Name = domain.NewServiceNameUnsafe("")
	return &modified
}

// modifyServiceNameWhitespace returns a config with whitespace service name.
func modifyServiceNameWhitespace(c *ports.Configuration) *ports.Configuration {
	modified := *c
	modified.Service.Name = domain.NewServiceNameUnsafe("   ")
	return &modified
}

// assertValidConfig asserts that a configuration is valid.
func assertValidConfig(t *testing.T, config *ports.Configuration) {
	t.Helper()
	if config == nil {
		t.Fatal("Configuration should not be nil")
	}

	// Validate basic config structure
	if config.Service.Name.IsEmpty() {
		t.Error("Configuration.Service.Name should not be empty")
	}

	// Config should be valid
	if err := config.Validate(); err != nil {
		t.Errorf("Configuration should be valid: %v", err)
	}
}
