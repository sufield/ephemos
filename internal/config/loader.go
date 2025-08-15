// Package config provides internal configuration loading utilities.
package config

import (
	"context"
	"os"
	"strconv"
)

// GetBoolEnv returns a boolean environment variable value with a default.
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Environment variable names for configuration.
const (
	EnvServiceName     = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain     = "EPHEMOS_TRUST_DOMAIN"
	EnvRequireAuth     = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel        = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress     = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion   = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled    = "EPHEMOS_DEBUG_ENABLED"
)

// Configuration represents internal configuration structure.
type Configuration struct {
	Service   ServiceConfig
	Transport TransportConfig
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	Name   string
	Domain string
}

// TransportConfig contains transport layer configuration.
type TransportConfig struct {
	Type    string
	Address string
	TLS     *TLSConfig
}

// TLSConfig contains TLS/SSL configuration settings.
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// LoadFromEnvironment creates a configuration from environment variables.
func LoadFromEnvironment() (*Configuration, error) {
	config := &Configuration{}

	// Set values from environment variables if present
	if serviceName := os.Getenv(EnvServiceName); serviceName != "" {
		config.Service.Name = serviceName
	} else {
		config.Service.Name = "ephemos-service"
	}

	if trustDomain := os.Getenv(EnvTrustDomain); trustDomain != "" {
		config.Service.Domain = trustDomain
	} else {
		config.Service.Domain = "default.local"
	}

	// Set transport defaults
	config.Transport.Type = "http"
	if bindAddress := os.Getenv(EnvBindAddress); bindAddress != "" {
		config.Transport.Address = bindAddress
	} else {
		config.Transport.Address = ":8080"
	}

	return config, nil
}

// LoadFromYAML loads configuration from a YAML file.
func LoadFromYAML(ctx context.Context, path string) (*Configuration, error) {
	// For now, fall back to environment loading
	// In a real implementation, this would parse YAML
	envConfig, err := LoadFromEnvironment()
	if err != nil {
		// If env loading fails, return default config
		return GetDefault(), nil
	}
	return envConfig, nil
}

// GetDefault returns a configuration with sensible defaults.
func GetDefault() *Configuration {
	return &Configuration{
		Service: ServiceConfig{
			Name:   "ephemos-service",
			Domain: "default.local",
		},
		Transport: TransportConfig{
			Type:    "http",
			Address: ":8080",
			TLS: &TLSConfig{
				Enabled: true,
			},
		},
	}
}

// Provider defines the interface for loading configurations internally.
type Provider interface {
	LoadConfiguration(ctx context.Context, path string) (*Configuration, error)
	GetDefaultConfiguration(ctx context.Context) *Configuration
}