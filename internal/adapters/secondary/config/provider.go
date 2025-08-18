// Package config provides configuration management for Ephemos.
package config

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// FileProvider provides configs from files.
type FileProvider struct{}

// NewFileProvider creates provider.
func NewFileProvider() *FileProvider {
	return &FileProvider{}
}

// LoadConfiguration loads config.
func (p *FileProvider) LoadConfiguration(ctx context.Context, path string) (*ports.Configuration, error) {
	// Validate and clean input path
	if strings.TrimSpace(path) == "" {
		return nil, &errors.ValidationError{
			Field:   "path",
			Value:   path,
			Message: "configuration file path cannot be empty or whitespace",
		}
	}

	// Clean path first
	cleanPath := filepath.Clean(path)

	// Convert to absolute path to properly validate
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config file path: %w", err)
	}

	// Use the absolute path for reading
	cleanPath = absPath

	// Check for context cancellation
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("configuration loading canceled: %w", ctx.Err())
		default:
		}
	}

	// Use Viper for multi-format configuration loading
	v := viper.New()
	v.SetConfigFile(cleanPath)

	// Also read from environment (env vars take precedence)
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	p.setConfigDefaults(v)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Unmarshal configuration
	var config ports.Configuration
	if err := v.Unmarshal(&config, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in file %s: %w", path, err)
	}

	return &config, nil
}

// GetDefaultConfiguration gets default.
func (p *FileProvider) GetDefaultConfiguration(_ context.Context) *ports.Configuration {
	// GetDefaultConfiguration should always return a default configuration
	// regardless of context state, as it doesn't perform any blocking operations

	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   domain.NewServiceNameUnsafe("ephemos-service"), // Default service name
			Domain: "",                                           // Empty domain uses SPIRE trust domain
		},
		Agent: &ports.AgentConfig{
			SocketPath: domain.NewSocketPathUnsafe("/run/sockets/agent.sock"), // Standard agent socket path
		},
	}
}

// setConfigDefaults sets default values for configuration.
func (p *FileProvider) setConfigDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "ephemos-service")
	v.SetDefault("service.domain", "")
	v.SetDefault("agent.socketpath", "/run/sockets/agent.sock")
	v.SetDefault("service.cache.ttl_minutes", 30)
	v.SetDefault("service.cache.proactive_refresh_minutes", 10)
}
