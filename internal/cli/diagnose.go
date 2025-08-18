// Package cli provides command-line interface for SPIRE diagnostics
// using SPIRE's built-in CLI tools and APIs rather than custom implementation.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/adapters/secondary/verification"
	"github.com/sufield/ephemos/internal/core/ports"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "SPIRE diagnostics commands using built-in capabilities",
	Long: `SPIRE diagnostics commands that leverage SPIRE's built-in CLI tools and APIs
rather than implementing custom diagnostic functionality from scratch.

Available subcommands:
  server       Get SPIRE server diagnostics
  agent        Get SPIRE agent diagnostics
  entries      List registration entries
  bundles      Show trust bundle information
  agents       List connected agents
  version      Get component versions`,
}

var diagnoseServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Get SPIRE server diagnostics using built-in CLI tools",
	Long: `Get comprehensive SPIRE server diagnostic information using SPIRE's
built-in CLI commands and health endpoints.`,
	PreRunE: validateDiagnoseEnvironment,
	RunE:    runDiagnoseServer,
}

var diagnoseAgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Get SPIRE agent diagnostics using built-in CLI tools",
	Long: `Get comprehensive SPIRE agent diagnostic information using SPIRE's
built-in CLI commands and Workload API.`,
	PreRunE: validateDiagnoseEnvironment,
	RunE:    runDiagnoseAgent,
}

var diagnoseEntriesCmd = &cobra.Command{
	Use:   "entries",
	Short: "List registration entries using SPIRE CLI",
	Long: `List all registration entries using SPIRE's built-in 'spire-server entry show'
command with JSON output for structured data.`,
	RunE: runDiagnoseEntries,
}

var diagnoseBundlesCmd = &cobra.Command{
	Use:   "bundles [trust-domain]",
	Short: "Show trust bundle information using SPIRE CLI",
	Long: `Show trust bundle information using SPIRE's built-in 'spire-server bundle show'
command. If trust-domain is not specified, shows all bundles.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiagnoseBundles,
}

var diagnoseAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List connected agents using SPIRE CLI",
	Long: `List all connected SPIRE agents using SPIRE's built-in 'spire-server agent list'
command with JSON output for structured data.`,
	RunE: runDiagnoseAgents,
}

var diagnoseVersionCmd = &cobra.Command{
	Use:   "version <component>",
	Short: "Get SPIRE component version",
	Long: `Get the version of a SPIRE component (server or agent) using the
built-in -version flag.

Example:
  ephemos diagnose version spire-server
  ephemos diagnose version spire-agent`,
	Args:       cobra.ExactArgs(1),
	ValidArgs:  []string{"spire-server", "spire-agent"},
	ArgAliases: []string{"server", "agent"},
	PreRunE:    validateComponentArg,
	RunE:       runDiagnoseVersion,
}

func init() {
	// Add persistent flags for diagnostics
	diagnoseCmd.PersistentFlags().String("server-socket", "", "SPIRE server socket path (default: unix:///tmp/spire-server/private/api.sock)")
	diagnoseCmd.PersistentFlags().String("agent-socket", "", "SPIRE agent socket path (default: unix:///tmp/spire-agent/public/api.sock)")
	diagnoseCmd.PersistentFlags().String("server-address", "", "SPIRE server API address")
	diagnoseCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Diagnostics timeout")
	diagnoseCmd.PersistentFlags().Bool("use-api", false, "Use server API instead of CLI commands")
	diagnoseCmd.PersistentFlags().String("api-token", "", "Server API token for authentication")

	// Create flag dependencies and mutual exclusions
	diagnoseCmd.MarkFlagsMutuallyExclusive("server-socket", "server-address")
	diagnoseCmd.MarkFlagsRequiredTogether("use-api", "api-token")

	// When using API, server-address is required
	diagnoseCmd.MarkFlagsRequiredTogether("use-api", "server-address")

	// Add subcommands
	diagnoseCmd.AddCommand(diagnoseServerCmd)
	diagnoseCmd.AddCommand(diagnoseAgentCmd)
	diagnoseCmd.AddCommand(diagnoseEntriesCmd)
	diagnoseCmd.AddCommand(diagnoseBundlesCmd)
	diagnoseCmd.AddCommand(diagnoseAgentsCmd)
	diagnoseCmd.AddCommand(diagnoseVersionCmd)
}

func runDiagnoseServer(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	// Get server diagnostics
	diagnostics, err := provider.GetServerDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get server diagnostics: %w", err)
	}

	return outputDiagnosticInfo(cmd, diagnostics)
}

func runDiagnoseAgent(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	// Get agent diagnostics
	diagnostics, err := provider.GetAgentDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get agent diagnostics: %w", err)
	}

	return outputDiagnosticInfo(cmd, diagnostics)
}

func runDiagnoseEntries(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	// List registration entries
	entries, err := provider.ListRegistrationEntries(ctx)
	if err != nil {
		return fmt.Errorf("failed to list registration entries: %w", err)
	}

	return outputRegistrationEntries(cmd, entries)
}

func runDiagnoseBundles(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	var trustDomain spiffeid.TrustDomain
	if len(args) > 0 {
		td, err := spiffeid.TrustDomainFromString(args[0])
		if err != nil {
			return fmt.Errorf("invalid trust domain %s: %w", args[0], err)
		}
		trustDomain = td
	} else {
		// Use default trust domain if available
		trustDomain = spiffeid.RequireTrustDomainFromString("example.org")
	}

	// Show trust bundle
	bundleInfo, err := provider.ShowTrustBundle(ctx, trustDomain)
	if err != nil {
		return fmt.Errorf("failed to show trust bundle: %w", err)
	}

	return outputTrustBundleInfo(cmd, bundleInfo)
}

func runDiagnoseAgents(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	// List agents
	agents, err := provider.ListAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	return outputAgents(cmd, agents)
}

func runDiagnoseVersion(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	component := args[0]

	// Create diagnostics provider
	provider, err := createDiagnosticsProvider(cmd)
	if err != nil {
		return fmt.Errorf("failed to create diagnostics provider: %w", err)
	}

	// Get component version
	version, err := provider.GetComponentVersion(ctx, component)
	if err != nil {
		return fmt.Errorf("failed to get %s version: %w", component, err)
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		result := map[string]string{
			"component": component,
			"version":   version,
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		fmt.Printf("%s version: %s\n", component, version)
	}

	return nil
}

func createDiagnosticsProvider(cmd *cobra.Command) (*verification.SpireDiagnosticsProvider, error) {
	config := &ports.DiagnosticsConfig{}

	// Get socket paths
	if serverSocket, _ := cmd.Flags().GetString("server-socket"); serverSocket != "" {
		config.ServerSocketPath = serverSocket
	}
	if agentSocket, _ := cmd.Flags().GetString("agent-socket"); agentSocket != "" {
		config.AgentSocketPath = agentSocket
	}

	// Get server address
	if serverAddress, _ := cmd.Flags().GetString("server-address"); serverAddress != "" {
		config.ServerAddress = serverAddress
	}

	// Get timeout
	if timeout, _ := cmd.Flags().GetDuration("timeout"); timeout > 0 {
		config.Timeout = timeout
	}

	// Get API usage preference
	if useAPI, _ := cmd.Flags().GetBool("use-api"); useAPI {
		config.UseServerAPI = true
	}

	// Get API token
	if apiToken, _ := cmd.Flags().GetString("api-token"); apiToken != "" {
		config.ServerAPIToken = apiToken
	}

	return verification.NewSpireDiagnosticsProvider(config), nil
}

func outputDiagnosticInfo(cmd *cobra.Command, info *ports.DiagnosticInfo) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(info)
	default:
		return outputDiagnosticInfoText(info, quiet, noEmoji)
	}
}

func outputDiagnosticInfoText(info *ports.DiagnosticInfo, quiet, noEmoji bool) error {
	// Status indicator
	status := "🔍"
	if noEmoji {
		status = "[DIAG]"
	}

	fmt.Printf("%s %s Diagnostics\n", status, info.Component)

	if !quiet {
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Status: %s\n", info.Status)
		if !info.TrustDomain.IsZero() {
			fmt.Printf("Trust Domain: %s\n", info.TrustDomain)
		}
		fmt.Printf("Collected At: %s\n", info.CollectedAt.Format(time.RFC3339))

		// Output entries info if available
		if info.Entries != nil {
			fmt.Printf("\nRegistration Entries:\n")
			fmt.Printf("  Total: %d\n", info.Entries.Total)
			fmt.Printf("  Recent: %d\n", info.Entries.Recent)
			if len(info.Entries.BySelector) > 0 {
				fmt.Printf("  By Selector:\n")
				for selector, count := range info.Entries.BySelector {
					fmt.Printf("    %s: %d\n", selector, count)
				}
			}
		}

		// Output agent info if available
		if info.Agents != nil {
			fmt.Printf("\nAgents:\n")
			fmt.Printf("  Total: %d\n", info.Agents.Total)
			fmt.Printf("  Active: %d\n", info.Agents.Active)
			fmt.Printf("  Inactive: %d\n", info.Agents.Inactive)
			fmt.Printf("  Banned: %d\n", info.Agents.Banned)
		}

		// Output additional details
		if len(info.Details) > 0 {
			fmt.Printf("\nDetails:\n")
			for key, value := range info.Details {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	}

	return nil
}

func outputRegistrationEntries(cmd *cobra.Command, entries []*ports.RegistrationEntry) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(entries)
	default:
		return outputRegistrationEntriesText(entries, quiet, noEmoji)
	}
}

func outputRegistrationEntriesText(entries []*ports.RegistrationEntry, quiet, noEmoji bool) error {
	// Status indicator
	status := "📋"
	if noEmoji {
		status = "[ENTRIES]"
	}

	fmt.Printf("%s Registration Entries (%d total)\n", status, len(entries))

	if !quiet {
		for i, entry := range entries {
			fmt.Printf("\n%d. ID: %s\n", i+1, entry.ID)
			fmt.Printf("   SPIFFE ID: %s\n", entry.SPIFFEID)
			fmt.Printf("   Parent ID: %s\n", entry.ParentID)
			fmt.Printf("   Selectors: %v\n", entry.Selectors)
			if entry.TTL > 0 {
				fmt.Printf("   TTL: %ds\n", entry.TTL)
			}
			if len(entry.FederatesWith) > 0 {
				fmt.Printf("   Federates With: %v\n", entry.FederatesWith)
			}
			if len(entry.DNSNames) > 0 {
				fmt.Printf("   DNS Names: %v\n", entry.DNSNames)
			}
			fmt.Printf("   Admin: %t\n", entry.Admin)
			fmt.Printf("   Created: %s\n", entry.CreatedAt.Format(time.RFC3339))
		}
	}

	return nil
}

func outputTrustBundleInfo(cmd *cobra.Command, bundleInfo *ports.TrustBundleInfo) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(bundleInfo)
	default:
		return outputTrustBundleInfoText(bundleInfo, quiet, noEmoji)
	}
}

func outputTrustBundleInfoText(bundleInfo *ports.TrustBundleInfo, quiet, noEmoji bool) error {
	// Status indicator
	status := "🔐"
	if noEmoji {
		status = "[BUNDLE]"
	}

	fmt.Printf("%s Trust Bundle Information\n", status)

	if !quiet {
		if bundleInfo.Local != nil {
			fmt.Printf("\nLocal Bundle:\n")
			fmt.Printf("  Trust Domain: %s\n", bundleInfo.Local.TrustDomain)
			fmt.Printf("  Certificate Count: %d\n", bundleInfo.Local.CertificateCount)
			fmt.Printf("  Last Updated: %s\n", bundleInfo.Local.LastUpdated.Format(time.RFC3339))
			fmt.Printf("  Expires At: %s\n", bundleInfo.Local.ExpiresAt.Format(time.RFC3339))
		}

		if len(bundleInfo.Federated) > 0 {
			fmt.Printf("\nFederated Bundles:\n")
			for domain, bundle := range bundleInfo.Federated {
				fmt.Printf("  %s:\n", domain)
				fmt.Printf("    Certificate Count: %d\n", bundle.CertificateCount)
				fmt.Printf("    Last Updated: %s\n", bundle.LastUpdated.Format(time.RFC3339))
				fmt.Printf("    Expires At: %s\n", bundle.ExpiresAt.Format(time.RFC3339))
			}
		}
	}

	return nil
}

func outputAgents(cmd *cobra.Command, agents []*ports.Agent) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(agents)
	default:
		return outputAgentsText(agents, quiet, noEmoji)
	}
}

func outputAgentsText(agents []*ports.Agent, quiet, noEmoji bool) error {
	// Status indicator
	status := "🤖"
	if noEmoji {
		status = "[AGENTS]"
	}

	fmt.Printf("%s SPIRE Agents (%d total)\n", status, len(agents))

	if !quiet {
		for i, agent := range agents {
			fmt.Printf("\n%d. ID: %s\n", i+1, agent.ID)
			fmt.Printf("   Attestation Type: %s\n", agent.AttestationType)
			fmt.Printf("   Serial Number: %s\n", agent.SerialNumber)
			fmt.Printf("   Expires At: %s\n", agent.ExpiresAt.Format(time.RFC3339))
			fmt.Printf("   Banned: %t\n", agent.Banned)
			fmt.Printf("   Can Reattest: %t\n", agent.CanReattest)
			if len(agent.Selectors) > 0 {
				fmt.Printf("   Selectors: %v\n", agent.Selectors)
			}
		}
	}

	return nil
}

// validateDiagnoseEnvironment validates that SPIRE components are accessible for diagnostics
func validateDiagnoseEnvironment(cmd *cobra.Command, args []string) error {
	useAPI, _ := cmd.Flags().GetBool("use-api")

	if useAPI {
		// When using API, ensure we have required credentials
		apiToken, _ := cmd.Flags().GetString("api-token")
		if apiToken == "" {
			return fmt.Errorf("API token is required when using --use-api flag")
		}

		serverAddress, _ := cmd.Flags().GetString("server-address")
		if serverAddress == "" {
			return fmt.Errorf("server address is required when using --use-api flag")
		}
	} else {
		// When using CLI commands, check if they're available
		if _, err := exec.LookPath("spire-server"); err != nil {
			return fmt.Errorf("spire-server CLI not found in PATH: %w", err)
		}
		if _, err := exec.LookPath("spire-agent"); err != nil {
			return fmt.Errorf("spire-agent CLI not found in PATH: %w", err)
		}
	}

	// Validate socket paths format if provided
	if serverSocket, _ := cmd.Flags().GetString("server-socket"); serverSocket != "" {
		if !strings.HasPrefix(serverSocket, "unix://") && !strings.HasPrefix(serverSocket, "/") {
			return fmt.Errorf("server socket path must be absolute or start with unix://")
		}
	}

	if agentSocket, _ := cmd.Flags().GetString("agent-socket"); agentSocket != "" {
		if !strings.HasPrefix(agentSocket, "unix://") && !strings.HasPrefix(agentSocket, "/") {
			return fmt.Errorf("agent socket path must be absolute or start with unix://")
		}
	}

	return nil
}

// validateComponentArg validates the component argument for version command
func validateComponentArg(cmd *cobra.Command, args []string) error {
	// First run common environment validation
	if err := validateDiagnoseEnvironment(cmd, args); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("component argument is required")
	}

	component := args[0]
	validComponents := []string{"spire-server", "spire-agent", "server", "agent"}

	for _, valid := range validComponents {
		if component == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid component %s: must be one of %v", component, validComponents)
}
