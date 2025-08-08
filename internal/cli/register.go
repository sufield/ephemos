package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/sufield/ephemos/internal/adapters/primary/cli"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/spf13/cobra"
)

var (
	configFile   string
	serviceName  string
	serviceDomain string
	selector     string
)

var registerCmd = &cobra.Command{
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

func init() {
	registerCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	registerCmd.Flags().StringVarP(&serviceName, "name", "n", "", "Service name")
	registerCmd.Flags().StringVarP(&serviceDomain, "domain", "d", "example.org", "Service domain")
	registerCmd.Flags().StringVarP(&selector, "selector", "s", "", "Custom selector (default: executable path)")
}

func runRegister(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create configuration
	var cfg *ports.Configuration
	
	if configFile != "" {
		// Load from config file
		configProvider := config.NewConfigProvider()
		var err error
		cfg, err = configProvider.LoadConfiguration(ctx, configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	} else if serviceName != "" {
		// Create from command line arguments
		cfg = &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   serviceName,
				Domain: serviceDomain,
			},
		}
		
		// Save to temporary config file for the registrar
		tempFile, err := os.CreateTemp("", "ephemos-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create temp config: %w", err)
		}
		defer os.Remove(tempFile.Name())
		
		// Write config to temp file
		configContent := fmt.Sprintf(`service:
  name: %s
  domain: %s
`, serviceName, serviceDomain)
		
		if _, err := tempFile.WriteString(configContent); err != nil {
			return fmt.Errorf("failed to write temp config: %w", err)
		}
		tempFile.Close()
		
		configFile = tempFile.Name()
	} else {
		return fmt.Errorf("either --config or --name must be provided")
	}
	
	// Create registrar
	registrarConfig := &cli.RegistrarConfig{
		SPIRESocketPath: os.Getenv("SPIRE_SOCKET_PATH"),
		Logger:          slog.Default(),
	}
	
	configProvider := config.NewConfigProvider()
	registrar := cli.NewRegistrar(configProvider, registrarConfig)
	
	// Register the service
	if err := registrar.RegisterService(ctx, configFile); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	
	fmt.Printf("Successfully registered service '%s' in domain '%s'\n", cfg.Service.Name, cfg.Service.Domain)
	fmt.Printf("SPIFFE ID: spiffe://%s/%s\n", cfg.Service.Domain, cfg.Service.Name)
	
	if selector != "" {
		fmt.Printf("Selector: %s\n", selector)
	} else {
		execPath, err := os.Executable()
		if err == nil {
			fmt.Printf("Selector: unix:path:%s\n", execPath)
		} else {
			fmt.Printf("Selector: (auto-determined)\n")
		}
	}
	
	return nil
}