// Package config provides internal configuration loading utilities.
package config

import (
	"context"
	"strings"

	"github.com/spf13/viper"
)


// Environment variable names for configuration.
const (
	EnvServiceName   = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain   = "EPHEMOS_TRUST_DOMAIN"
	EnvRequireAuth   = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel      = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress   = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled  = "EPHEMOS_DEBUG_ENABLED"
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
	v := viper.New()
	
	// Configure viper for environment variables
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// Set defaults
	setConfigDefaults(v)
	
	// Unmarshal configuration
	var config Configuration
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// LoadFromYAML loads configuration from a YAML file.
func LoadFromYAML(ctx context.Context, path string) (*Configuration, error) {
	return LoadFromFile(ctx, path, "yaml")
}

// LoadFromJSON loads configuration from a JSON file.
func LoadFromJSON(ctx context.Context, path string) (*Configuration, error) {
	return LoadFromFile(ctx, path, "json")
}

// LoadFromTOML loads configuration from a TOML file.
func LoadFromTOML(ctx context.Context, path string) (*Configuration, error) {
	return LoadFromFile(ctx, path, "toml")
}

// LoadFromFile loads configuration from a file with automatic format detection.
// Supports YAML, JSON, and TOML formats.
func LoadFromFile(ctx context.Context, path string, configType string) (*Configuration, error) {
	v := viper.New()
	
	// Configure viper for file loading
	if configType != "" {
		v.SetConfigFile(path)
		v.SetConfigType(configType)
	} else {
		// Auto-detect format from file extension
		v.SetConfigFile(path)
	}
	
	// Also read from environment (env vars take precedence)
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// Set defaults
	setConfigDefaults(v)
	
	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		// If file loading fails, fall back to environment loading
		return LoadFromEnvironment()
	}
	
	// Unmarshal configuration
	var config Configuration
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// setConfigDefaults sets default values for configuration.
func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "ephemos-service")
	v.SetDefault("service.domain", "default.local")
	v.SetDefault("transport.type", "http")
	v.SetDefault("transport.address", ":8080")
	v.SetDefault("transport.tls.enabled", true)
	v.SetDefault("transport.tls.certfile", "")
	v.SetDefault("transport.tls.keyfile", "")
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
