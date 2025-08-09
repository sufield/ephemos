// Package config provides configuration management for Ephemos.
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

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

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config ports.Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in file %s: %w", path, err)
	}

	return &config, nil
}

// GetDefaultConfiguration gets default.
func (p *FileProvider) GetDefaultConfiguration(ctx context.Context) *ports.Configuration {
	// GetDefaultConfiguration should always return a default configuration
	// regardless of context state, as it doesn't perform any blocking operations

	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "ephemos-service", // Default service name
			Domain: "",                // Empty domain uses SPIRE trust domain
		},
		SPIFFE: &ports.SPIFFEConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock", // Standard SPIRE agent socket path
		},
		AuthorizedClients: []string{}, // Empty list - no client restrictions by default
		TrustedServers:    []string{}, // Empty list - trust all servers by default (not recommended for production)
	}
}
