// Package cli provides command-line interface for Ephemos.
package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/sufield/ephemos/internal/adapters/primary/cli"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
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
	PreRunE: validateRegisterFlags,
	RunE:    runRegister,
}

func init() {
	// Define flags
	registerCmd.Flags().StringP("config", "c", "", "Path to configuration file")
	registerCmd.Flags().StringP("name", "n", "", "Service name")
	registerCmd.Flags().StringP("domain", "d", "example.org", "Service domain")
	registerCmd.Flags().StringP("selector", "s", "", "Custom selector (default: executable path)")

	// Use Cobra's built-in flag validation
	registerCmd.MarkFlagFilename("config", "yaml", "yml")

	// Create mutually exclusive groups - either config OR name must be provided
	registerCmd.MarkFlagsMutuallyExclusive("config", "name")

	// When using name, domain is implicitly required (already has default)
	// No need for MarkFlagsRequiredTogether since domain has a default value

	// Add intelligent completions
	registerCmd.RegisterFlagCompletionFunc("config", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
	})

	registerCmd.RegisterFlagCompletionFunc("domain", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"example.org\tDefault domain",
			"localhost\tLocal development",
			"prod.company.com\tProduction domain",
			"staging.company.com\tStaging domain",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	registerCmd.RegisterFlagCompletionFunc("selector", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"unix:uid:1000\tUser ID 1000",
			"unix:gid:1000\tGroup ID 1000",
			"unix:path:/usr/bin/app\tExecutable path",
			"k8s:ns:default\tKubernetes namespace",
			"k8s:sa:default\tKubernetes service account",
		}, cobra.ShellCompDirectiveNoFileComp
	})
}

// validateRegisterFlags performs business logic validation after Cobra's built-in validation
func validateRegisterFlags(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	serviceName, _ := cmd.Flags().GetString("name")

	// Cobra handles the mutual exclusivity, but we need at least one
	if configFile == "" && serviceName == "" {
		return fmt.Errorf("either --config or --name must be provided")
	}

	// Validate service name format if provided
	if serviceName != "" && !isValidServiceName(serviceName) {
		return fmt.Errorf("invalid service name format: %s (must contain only alphanumeric characters, hyphens, and underscores)", serviceName)
	}

	return nil
}

func runRegister(cmd *cobra.Command, args []string) error {
	// Get flag values using Cobra's methods
	configFile, _ := cmd.Flags().GetString("config")
	serviceName, _ := cmd.Flags().GetString("name")
	serviceDomain, _ := cmd.Flags().GetString("domain")
	selector, _ := cmd.Flags().GetString("selector")

	// Prepare configuration
	var cfg *ports.Configuration
	var tempFile string

	if configFile != "" {
		// Use provided config file
		configProvider := config.NewFileProvider()
		loadedCfg, err := configProvider.LoadConfiguration(cmd.Context(), configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		cfg = loadedCfg
	} else {
		// Create configuration from flags
		cfg = &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   serviceName,
				Domain: serviceDomain,
			},
		}

		// Create temporary config file for the registrar
		tf, err := createTempConfig(serviceName, serviceDomain)
		if err != nil {
			return err
		}
		tempFile = tf
		defer os.Remove(tempFile)
		configFile = tempFile
	}

	// Perform registration
	registrarConfig := &cli.RegistrarConfig{
		SPIRESocketPath: os.Getenv("SPIRE_SOCKET_PATH"),
		Logger:          slog.Default(),
	}

	configProvider := config.NewFileProvider()
	registrar := cli.NewRegistrar(configProvider, registrarConfig)

	if err := registrar.RegisterService(cmd.Context(), configFile); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Output success message
	outputRegistrationSuccess(cmd, cfg, selector)
	return nil
}

// isValidServiceName validates service name format
func isValidServiceName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

// createTempConfig creates a temporary configuration file
func createTempConfig(serviceName, serviceDomain string) (string, error) {
	tempFile, err := os.CreateTemp("", "ephemos-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp config: %w", err)
	}

	configContent := fmt.Sprintf(`service:
  name: %s
  domain: %s
`, serviceName, serviceDomain)

	if _, err := tempFile.WriteString(configContent); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp config: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// outputRegistrationSuccess outputs success message based on format flag
func outputRegistrationSuccess(cmd *cobra.Command, cfg *ports.Configuration, selector string) {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")

	if quiet {
		return
	}

	spiffeID := fmt.Sprintf("spiffe://%s/%s", cfg.Service.Domain, cfg.Service.Name)

	if format == "json" {
		// JSON output
		output := map[string]string{
			"service":   cfg.Service.Name,
			"domain":    cfg.Service.Domain,
			"spiffe_id": spiffeID,
		}
		if selector != "" {
			output["selector"] = selector
		} else if execPath, err := os.Executable(); err == nil {
			output["selector"] = fmt.Sprintf("unix:path:%s", execPath)
		}

		// Simple JSON output (error ignored as we're just printing)
		fmt.Printf(`{"service":"%s","domain":"%s","spiffe_id":"%s"`,
			cfg.Service.Name, cfg.Service.Domain, spiffeID)
		if selector != "" {
			fmt.Printf(`,"selector":"%s"`, selector)
		}
		fmt.Println("}")
	} else {
		// Text output
		noEmoji, _ := cmd.Flags().GetBool("no-emoji")
		icon := "✅"
		if noEmoji {
			icon = "[OK]"
		}

		fmt.Printf("%s Successfully registered service\n", icon)
		fmt.Printf("  Service: %s\n", cfg.Service.Name)
		fmt.Printf("  Domain: %s\n", cfg.Service.Domain)
		fmt.Printf("  SPIFFE ID: %s\n", spiffeID)

		if selector != "" {
			fmt.Printf("  Selector: %s\n", selector)
		} else if execPath, err := os.Executable(); err == nil {
			fmt.Printf("  Selector: unix:path:%s (auto-determined)\n", execPath)
		}
	}
}
