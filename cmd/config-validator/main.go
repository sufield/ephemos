// Package main provides a configuration validation tool for Ephemos.
// This tool helps validate configuration settings before deployment to production.
//
//nolint:forbidigo // CLI tool requires direct output to stdout
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		envOnly    = flag.Bool("env-only", false, "Validate environment variables only (most secure)")
		production = flag.Bool("production", false, "Perform production readiness validation")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)

	setupUsage()
	flag.Parse()

	fmt.Println("Ephemos Configuration Validator")
	fmt.Println("==============================")
	fmt.Println()

	// Load configuration
	cfg := loadConfiguration(*envOnly, *configFile)
	fmt.Println("✅ Configuration loaded successfully")
	fmt.Println()

	// Display configuration details if verbose
	if *verbose {
		displayConfiguration(cfg)
		fmt.Println()
	}

	// Validate configuration
	validateConfiguration(cfg, *production)

	// Environment variable recommendations
	fmt.Println()
	fmt.Println("💡 Security recommendations:")
	printSecurityRecommendations(*envOnly)

	fmt.Println()
	fmt.Println("🎉 Configuration validation completed successfully!")

	if *production {
		fmt.Println("✅ Configuration is ready for production deployment")
	}
}

func displayConfiguration(cfg *ports.Configuration) {
	fmt.Println("📋 Configuration Details:")
	fmt.Printf("   Service Name: %s\n", cfg.Service.Name)
	fmt.Printf("   Trust Domain: %s\n", cfg.Service.Domain)

	if cfg.SPIFFE != nil {
		fmt.Printf("   SPIFFE Socket: %s\n", cfg.SPIFFE.SocketPath)
	}

	if len(cfg.AuthorizedClients) > 0 {
		fmt.Printf("   Authorized Clients: %d configured\n", len(cfg.AuthorizedClients))
		for i, client := range cfg.AuthorizedClients {
			fmt.Printf("     %d. %s\n", i+1, client)
		}
	} else {
		fmt.Println("   Authorized Clients: None (allows all clients)")
	}

	if len(cfg.TrustedServers) > 0 {
		fmt.Printf("   Trusted Servers: %d configured\n", len(cfg.TrustedServers))
		for i, server := range cfg.TrustedServers {
			fmt.Printf("     %d. %s\n", i+1, server)
		}
	} else {
		fmt.Println("   Trusted Servers: None (trusts all servers)")
	}
}

func printProductionTips(err error) {
	errorMsg := err.Error()

	if strings.Contains(errorMsg, "example.org") {
		fmt.Println("  • Set EPHEMOS_TRUST_DOMAIN to your production domain (e.g., 'prod.company.com')")
	}

	if strings.Contains(errorMsg, "localhost") {
		fmt.Println("  • Set EPHEMOS_TRUST_DOMAIN to a proper domain (not localhost)")
	}

	if strings.Contains(errorMsg, "example") || strings.Contains(errorMsg, "demo") {
		fmt.Println("  • Set EPHEMOS_SERVICE_NAME to your production service name (not example/demo)")
	}

	if strings.Contains(errorMsg, "debug mode") {
		fmt.Println("  • Set EPHEMOS_DEBUG_ENABLED=false for production")
	}

	if strings.Contains(errorMsg, "wildcard") {
		fmt.Println("  • Use specific SPIFFE IDs instead of wildcards in EPHEMOS_AUTHORIZED_CLIENTS")
	}

	if strings.Contains(errorMsg, "socket should be in a secure directory") {
		fmt.Println("  • Set EPHEMOS_SPIFFE_SOCKET to a secure path like '/run/spire/sockets/api.sock'")
	}
}

func printSecurityRecommendations(envOnly bool) {
	if !envOnly {
		fmt.Println("  🔒 Use environment variables for production (--env-only)")
		fmt.Println("     Environment variables are more secure than config files")
	}

	fmt.Println("  🔐 Required environment variables for production:")
	fmt.Printf("     export %s=\"your-service-name\"\n", ports.EnvServiceName)
	fmt.Printf("     export %s=\"your.production.domain\"\n", ports.EnvTrustDomain)

	fmt.Println("  🛡️ Optional security environment variables:")
	fmt.Printf("     export %s=\"/run/spire/sockets/api.sock\"\n", ports.EnvSPIFFESocket)
	fmt.Printf("     export %s=\"spiffe://your.domain/client1,spiffe://your.domain/client2\"\n", ports.EnvAuthorizedClients)
	fmt.Printf("     export %s=\"false\"\n", ports.EnvDebugEnabled)

	fmt.Println("  📚 For more details, see:")
	fmt.Println("     docs/security/CONFIGURATION_SECURITY.md")
	fmt.Println("     config/README.md")
}

// setupUsage configures the custom usage function for flag parsing.
func setupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Validates Ephemos configuration for security and production readiness.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Validate environment variables (most secure):\n")
		fmt.Fprintf(os.Stderr, "  %s --env-only --production\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Validate config file with environment override:\n")
		fmt.Fprintf(os.Stderr, "  %s --config config/production.yaml --production\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Verbose validation:\n")
		fmt.Fprintf(os.Stderr, "  %s --env-only --verbose\n\n", os.Args[0])
	}
}

// loadConfiguration loads the configuration based on the provided flags.
func loadConfiguration(envOnly bool, configFile string) *ports.Configuration {
	var cfg *ports.Configuration
	var err error

	switch {
	case envOnly:
		fmt.Println("🔒 Loading configuration from environment variables (secure mode)")
		cfg, err = ports.LoadFromEnvironment()
		if err != nil {
			fmt.Printf("❌ Failed to load configuration from environment: %v\n", err)
			os.Exit(1)
		}
	case configFile != "":
		cfg = loadConfigurationFromFile(configFile)
	default:
		fmt.Println("❌ Either --config or --env-only must be specified")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

// loadConfigurationFromFile loads configuration from a file and merges with environment.
func loadConfigurationFromFile(configFile string) *ports.Configuration {
	fmt.Printf("📁 Loading configuration from file: %s\n", configFile)
	provider := config.NewFileProvider()
	cfg, err := provider.LoadConfiguration(context.Background(), configFile)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration file: %v\n", err)
		os.Exit(1)
	}

	// Merge with environment variables (environment takes precedence)
	fmt.Println("🔄 Merging with environment variables (environment overrides file)")
	if err := cfg.MergeWithEnvironment(); err != nil {
		fmt.Printf("❌ Failed to merge with environment: %v\n", err)
		os.Exit(1)
	}

	return cfg
}

// validateConfiguration performs basic and optional production validation.
func validateConfiguration(cfg *ports.Configuration, production bool) {
	// Perform basic validation
	fmt.Println("🔍 Performing basic validation...")
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Basic validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Basic validation passed")

	// Perform production readiness validation if requested
	if production {
		fmt.Println()
		fmt.Println("🏭 Performing production readiness validation...")
		if err := cfg.IsProductionReady(); err != nil {
			fmt.Printf("❌ Production validation failed: %v\n", err)
			fmt.Println()
			fmt.Println("💡 Production readiness tips:")
			printProductionTips(err)
			os.Exit(1)
		}
		fmt.Println("✅ Production validation passed")
	}
}
