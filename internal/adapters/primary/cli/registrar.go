// Package cli provides command-line interface implementations for Ephemos
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// RegistrarConfig holds configuration for the Registrar.
type RegistrarConfig struct {
	SPIRESocketPath string
	SPIREServerPath string // Path to spire-server binary
	Logger          *slog.Logger
}

// Registrar handles service registration with SPIRE.
type Registrar struct {
	configProvider  ports.ConfigurationProvider
	spireSocketPath string
	spireServerPath string
	logger          *slog.Logger
}

// isValidPath validates that a path is safe to use.
func isValidPath(path string) bool {
	if path == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	// Prevent path traversal
	if strings.Contains(cleanPath, "..") {
		return false
	}
	// Ensure it's within allowed paths for SPIRE
	allowedPaths := []string{"/usr/bin/", "/usr/local/bin/", "/opt/spire/bin/", "./bin/"}
	for _, allowed := range allowedPaths {
		if strings.HasPrefix(cleanPath, allowed) || cleanPath == "spire-server" {
			return true
		}
	}
	return false
}

// isValidSocketPath validates that a socket path is safe to use.
func isValidSocketPath(path string) bool {
	if path == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	// Prevent path traversal
	if strings.Contains(cleanPath, "..") {
		return false
	}
	// Must be a socket file (typically .sock extension or in /tmp)
	return strings.HasSuffix(cleanPath, ".sock") || strings.HasPrefix(cleanPath, "/tmp/") || strings.HasPrefix(cleanPath, "/var/run/")
}

// isValidSelector validates that a selector is in the correct format.
func isValidSelector(selector string) bool {
	if selector == "" {
		return false
	}
	// Basic validation - selectors should be in format "type:key:value"
	parts := strings.Split(selector, ":")
	if len(parts) < 2 {
		return false
	}
	// First part should be a known selector type
	validTypes := []string{"unix", "k8s", "docker"}
	for _, validType := range validTypes {
		if parts[0] == validType {
			return true
		}
	}
	return false
}

// NewRegistrar creates a new Registrar with proper dependency injection.
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

// RegisterService registers a service with SPIRE based on its configuration.
func (r *Registrar) RegisterService(ctx context.Context, configPath string) error {
	r.logger.Info("Starting service registration", "configPath", configPath)

	cfg, err := r.configProvider.LoadConfiguration(ctx, configPath)
	if err != nil {
		r.logger.Error("Failed to load configuration", "error", err)
		return fmt.Errorf("failed to load configuration: %w", errors.NewDomainError(errors.ErrMissingConfiguration, err))
	}

	if err := r.validateConfig(cfg); err != nil {
		r.logger.Error("Invalid configuration", "error", err)
		return err
	}

	// Create SPIRE registration entry
	if err := r.createSPIREEntry(ctx, cfg); err != nil {
		r.logger.Error("Failed to create SPIRE entry", "error", err)
		return fmt.Errorf("SPIFFE registration failed: %w", errors.NewDomainError(errors.ErrSPIFFERegistration, err))
	}

	r.logger.Info("Service registration completed successfully",
		"service", cfg.Service.Name,
		"domain", cfg.Service.Domain)

	return nil
}

// validateConfig performs comprehensive validation of the configuration.
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
	validDomain := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-.]*[a-zA-Z0-9])?$`)
	if !validDomain.MatchString(cfg.Service.Domain) {
		return &errors.ValidationError{
			Field:   "Service.Domain",
			Value:   cfg.Service.Domain,
			Message: "service domain must be a valid DNS name",
		}
	}

	return nil
}

// createSPIREEntry creates a registration entry in SPIRE.
// NOTE: In production, you should use the SPIRE Server API directly via gRPC
// This implementation uses the CLI for simplicity, as the SPIRE Server API
// requires additional proto definitions not included in go-spiffe
//
//nolint:cyclop,funlen // Function has inherent complexity from validation and command execution
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
	selector := r.getServiceSelector()

	r.logger.Debug("Creating SPIRE entry",
		"spiffeID", spiffeID.String(),
		"parentID", parentID.String(),
		"selector", selector,
		"socketPath", r.spireSocketPath)

	// Validate paths for security
	if !isValidPath(r.spireServerPath) {
		return fmt.Errorf("invalid spire-server path: %s", r.spireServerPath)
	}

	// Validate socket path for security
	if !isValidSocketPath(r.spireSocketPath) {
		return fmt.Errorf("invalid socket path: %s", r.spireSocketPath)
	}

	// Validate selector format for security
	if !isValidSelector(selector) {
		return fmt.Errorf("invalid selector format: %s", selector)
	}

	// Use spire-server CLI command
	// In production, use the SPIRE Server API directly
	//nolint:gosec // G204: Input is validated above for security
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

// getServiceSelector determines the service selector for SPIRE.
// For demo purposes, we use unix:uid selector (running as current user).
// In production, use more specific selectors like k8s:pod-label or unix:path.
func (r *Registrar) getServiceSelector() string {
	// For demo, use unix:uid selector with current user
	// This works when services run as the same user
	uid := os.Getuid()
	return fmt.Sprintf("unix:uid:%d", uid)
}

// GetConfigProvider returns the configuration provider (for testing).
func (r *Registrar) GetConfigProvider() ports.ConfigurationProvider {
	return r.configProvider
}

// GetSPIRESocketPath returns the SPIRE socket path (for testing).
func (r *Registrar) GetSPIRESocketPath() string {
	return r.spireSocketPath
}

// GetSPIREServerPath returns the SPIRE server path (for testing).
func (r *Registrar) GetSPIREServerPath() string {
	return r.spireServerPath
}

// ValidateConfig exposes validateConfig for testing.
func (r *Registrar) ValidateConfig(cfg *ports.Configuration) error {
	return r.validateConfig(cfg)
}

// CreateSPIREEntry exposes createSPIREEntry for testing.
func (r *Registrar) CreateSPIREEntry(ctx context.Context, cfg *ports.Configuration) error {
	return r.createSPIREEntry(ctx, cfg)
}

// GetServiceSelector exposes getServiceSelector for testing.
func (r *Registrar) GetServiceSelector() string {
	return r.getServiceSelector()
}
