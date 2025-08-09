// Package cli provides command-line interface for Ephemos.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/sufield/ephemos/internal/adapters/primary/cli"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

//nolint:gochecknoglobals // CLI flags require global variables in Cobra
var (
	configFile    string
	serviceName   string
	serviceDomain string
	selector      string
)

var registerCmd = &cobra.Command{ //nolint:gochecknoglobals // Cobra command pattern
	Use:   "register",
	Short: "Register a service with SPIRE",
	Long: `Register a service identity with SPIRE server.

You can either provide a config file or specify the service details directly.

Examples:
  # Using config file
  ephemos register --config service.yaml
  
  # Using command line arguments
  ephemos register --name echo-server --domain example.org
  ephemos register --name echo-server --domain example.org --selector unix:uid:1000`,
	RunE: runRegister,
}

func init() { //nolint:gochecknoinits // Cobra requires init for flag setup
	registerCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	registerCmd.Flags().StringVarP(&serviceName, "name", "n", "", "Service name")
	registerCmd.Flags().StringVarP(&serviceDomain, "domain", "d", "example.org", "Service domain")
	registerCmd.Flags().StringVarP(&selector, "selector", "s", "", "Custom selector (default: executable path)")
}

func runRegister(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := loadConfiguration(ctx)
	if err != nil {
		return err
	}

	if err := performRegistration(ctx); err != nil {
		return err
	}

	logRegistrationSuccess(cfg)
	return nil
}

func loadConfiguration(ctx context.Context) (*ports.Configuration, error) {
	switch {
	case configFile != "":
		return loadFromConfigFile(ctx)
	case serviceName != "":
		return createTempConfigFromFlags()
	default:
		return nil, fmt.Errorf("either --config or --name must be provided")
	}
}

func loadFromConfigFile(ctx context.Context) (*ports.Configuration, error) {
	configProvider := config.NewFileProvider()
	cfg, err := configProvider.LoadConfiguration(ctx, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

func createTempConfigFromFlags() (*ports.Configuration, error) {
	cfg := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   serviceName,
			Domain: serviceDomain,
		},
	}

	tempFile, err := os.CreateTemp("", "ephemos-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp config: %w", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			slog.Warn("Failed to remove temp file", "file", tempFile.Name(), "error", err)
		}
	}()

	configContent := fmt.Sprintf(`service:
  name: %s
  domain: %s
`, serviceName, serviceDomain)

	if _, err := tempFile.WriteString(configContent); err != nil {
		return nil, fmt.Errorf("failed to write temp config: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	configFile = tempFile.Name()
	return cfg, nil
}

func performRegistration(ctx context.Context) error {
	registrarConfig := &cli.RegistrarConfig{
		SPIRESocketPath: os.Getenv("SPIRE_SOCKET_PATH"),
		Logger:          slog.Default(),
	}

	configProvider := config.NewFileProvider()
	registrar := cli.NewRegistrar(configProvider, registrarConfig)

	if err := registrar.RegisterService(ctx, configFile); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	return nil
}

func logRegistrationSuccess(cfg *ports.Configuration) {
	slog.Info("Successfully registered service",
		"service", cfg.Service.Name,
		"domain", cfg.Service.Domain,
		"spiffe_id", fmt.Sprintf("spiffe://%s/%s", cfg.Service.Domain, cfg.Service.Name))

	if selector != "" {
		slog.Info("Service registered with selector", "selector", selector)
	} else {
		execPath, err := os.Executable()
		if err == nil {
			slog.Info("Service registered with auto-determined selector", "selector", fmt.Sprintf("unix:path:%s", execPath))
		} else {
			slog.Info("Service registered with auto-determined selector")
		}
	}
}
