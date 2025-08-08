// Package cli provides command-line interface implementations for Ephemos
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// RegistrarConfig holds configuration for the Registrar
type RegistrarConfig struct {
	SPIRESocketPath string
	SPIREServerPath string  // Path to spire-server binary
	Logger          *slog.Logger
}

// Registrar handles service registration with SPIRE
type Registrar struct {
	configProvider  ports.ConfigurationProvider
	spireSocketPath string
	spireServerPath string
	logger          *slog.Logger
}

// NewRegistrar creates a new Registrar with proper dependency injection
func NewRegistrar(
	configProvider ports.ConfigurationProvider,
	config *RegistrarConfig,
) *Registrar {
	if config == nil {
		config = &RegistrarConfig{}
	}
	
	if config.SPIRESocketPath == "" {
		config.SPIRESocketPath = os.Getenv("SPIRE_SOCKET_PATH")
		if config.SPIRESocketPath == "" {
			config.SPIRESocketPath = "/tmp/spire-server/private/api.sock"
		}
	}
	
	if config.SPIREServerPath == "" {
		config.SPIREServerPath = "spire-server"
	}
	
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	
	return &Registrar{
		configProvider:  configProvider,
		spireSocketPath: config.SPIRESocketPath,
		spireServerPath: config.SPIREServerPath,
		logger:          config.Logger,
	}
}

// RegisterService registers a service with SPIRE based on its configuration
func (r *Registrar) RegisterService(ctx context.Context, configPath string) error {
	r.logger.Info("Starting service registration", "configPath", configPath)
	
	cfg, err := r.configProvider.LoadConfiguration(ctx, configPath)
	if err != nil {
		r.logger.Error("Failed to load configuration", "error", err)
		return errors.NewDomainError(errors.ErrMissingConfiguration, err)
	}
	
	if err := r.validateConfig(cfg); err != nil {
		r.logger.Error("Invalid configuration", "error", err)
		return err
	}
	
	// Create SPIRE registration entry
	if err := r.createSPIREEntry(ctx, cfg); err != nil {
		r.logger.Error("Failed to create SPIRE entry", "error", err)
		return errors.NewDomainError(errors.ErrSPIFFERegistration, err)
	}
	
	r.logger.Info("Service registration completed successfully", 
		"service", cfg.Service.Name,
		"domain", cfg.Service.Domain)
	
	return nil
}

// validateConfig performs comprehensive validation of the configuration
func (r *Registrar) validateConfig(cfg *ports.Configuration) error {
	// Validate service name
	if cfg.Service.Name == "" {
		return &errors.ValidationError{
			Field:   "Service.Name",
			Value:   cfg.Service.Name,
			Message: "service name is required",
		}
	}
	
	// Validate service name format (alphanumeric with hyphens)
	validName := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	if !validName.MatchString(cfg.Service.Name) {
		return &errors.ValidationError{
			Field:   "Service.Name",
			Value:   cfg.Service.Name,
			Message: "service name must be alphanumeric with optional hyphens",
		}
	}
	
	// Validate domain
	if cfg.Service.Domain == "" {
		return &errors.ValidationError{
			Field:   "Service.Domain",
			Value:   cfg.Service.Domain,
			Message: "service domain is required",
		}
	}
	
	// Validate domain format (basic DNS name validation)
	validDomain := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-\.]*[a-zA-Z0-9])?$`)
	if !validDomain.MatchString(cfg.Service.Domain) {
		return &errors.ValidationError{
			Field:   "Service.Domain",
			Value:   cfg.Service.Domain,
			Message: "service domain must be a valid DNS name",
		}
	}
	
	return nil
}

// createSPIREEntry creates a registration entry in SPIRE
// NOTE: In production, you should use the SPIRE Server API directly via gRPC
// This implementation uses the CLI for simplicity, as the SPIRE Server API
// requires additional proto definitions not included in go-spiffe
func (r *Registrar) createSPIREEntry(ctx context.Context, cfg *ports.Configuration) error {
	// Build SPIFFE IDs
	spiffeID, err := spiffeid.FromString(fmt.Sprintf("spiffe://%s/%s", cfg.Service.Domain, cfg.Service.Name))
	if err != nil {
		return fmt.Errorf("failed to parse SPIFFE ID: %w", err)
	}
	
	parentID, err := spiffeid.FromString(fmt.Sprintf("spiffe://%s/spire-agent", cfg.Service.Domain))
	if err != nil {
		return fmt.Errorf("failed to parse parent ID: %w", err)
	}
	
	// Get service selector
	selector, err := r.getServiceSelector()
	if err != nil {
		return fmt.Errorf("failed to determine service selector: %w", err)
	}
	
	r.logger.Debug("Creating SPIRE entry",
		"spiffeID", spiffeID.String(),
		"parentID", parentID.String(),
		"selector", selector,
		"socketPath", r.spireSocketPath)
	
	// Use spire-server CLI command
	// In production, use the SPIRE Server API directly
	cmd := exec.CommandContext(ctx, r.spireServerPath, "entry", "create",
		"-socketPath", r.spireSocketPath,
		"-spiffeID", spiffeID.String(),
		"-parentID", parentID.String(),
		"-selector", selector,
		"-ttl", "3600",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		
		// Check if entry already exists (not an error)
		if strings.Contains(outputStr, "already exists") {
			r.logger.Info("Registration entry already exists", "service", cfg.Service.Name)
			return nil
		}
		
		// Check for specific error conditions
		if strings.Contains(outputStr, "permission denied") {
			return fmt.Errorf("permission denied: ensure you have access to SPIRE socket at %s", r.spireSocketPath)
		}
		
		if strings.Contains(outputStr, "connection refused") || strings.Contains(outputStr, "no such file") {
			return fmt.Errorf("SPIRE server not running or not accessible at %s", r.spireSocketPath)
		}
		
		return fmt.Errorf("SPIRE registration failed: %w\nOutput: %s", err, outputStr)
	}
	
	r.logger.Info("Created SPIRE registration entry", 
		"service", cfg.Service.Name,
		"output", string(output))
	
	return nil
}

// getServiceSelector determines the service selector for SPIRE
// For service-to-service authentication, we use path-based selectors
func (r *Registrar) getServiceSelector() (string, error) {
	// Get the current executable path for the selector
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Use the executable path as the selector for service identification
	return fmt.Sprintf("unix:path:%s", execPath), nil
}

