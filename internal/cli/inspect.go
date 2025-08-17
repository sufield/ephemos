// Package cli provides command-line interface for SPIRE certificate and trust bundle inspection
// using SPIRE's built-in CLI tools and go-spiffe/v2 SDK rather than custom implementation.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect SPIRE certificates and trust bundles using built-in tools",
	Long: `Inspect SPIRE certificates and trust bundles using SPIRE's built-in CLI tools
and go-spiffe/v2 SDK rather than implementing custom inspection logic.

This command leverages:
- spire-agent api fetch x509 for SVID inspection
- spire-server bundle show for trust bundle inspection  
- go-spiffe/v2 library for programmatic access

Available subcommands:
  svid         Inspect X.509 SVID using SPIRE agent API
  bundle       Inspect trust bundle using SPIRE server API
  authorities  Inspect local authorities using SPIRE server`,
}

var inspectSvidCmd = &cobra.Command{
	Use:   "svid",
	Short: "Inspect X.509 SVID using SPIRE agent Workload API",
	Long: `Inspect the current X.509 SVID using SPIRE's Workload API through go-spiffe/v2.
This leverages SPIRE's built-in certificate management without custom parsing.

The command uses workloadapi.X509Source to fetch and display SVID information
including SPIFFE ID, certificate validity, and chain details.`,
	RunE: runInspectSvid,
}

var inspectBundleCmd = &cobra.Command{
	Use:   "bundle [trust-domain]",
	Short: "Inspect trust bundle using SPIRE built-in tools",
	Long: `Inspect trust bundle using SPIRE's built-in bundle management.
If no trust domain is specified, shows the local trust domain bundle.

This command can use either:
1. go-spiffe/v2 Workload API for programmatic access
2. spire-server bundle show CLI command for detailed output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInspectBundle,
}

var inspectAuthoritiesCmd = &cobra.Command{
	Use:   "authorities",
	Short: "Inspect local X.509 authorities using SPIRE server CLI",
	Long: `Inspect local X.509 certificate authorities using SPIRE's built-in
'spire-server localauthority x509 show' command.

This displays information about active and prepared CAs including
expiration dates and authority IDs.`,
	RunE: runInspectAuthorities,
}

func init() {
	// Add persistent flags for inspection
	inspectCmd.PersistentFlags().String("socket", "", "Workload API socket path (default: unix:///tmp/spire-agent/public/api.sock)")
	inspectCmd.PersistentFlags().String("server-socket", "", "SPIRE server socket path (default: unix:///tmp/spire-server/private/api.sock)")
	inspectCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Operation timeout")
	inspectCmd.PersistentFlags().Bool("use-cli", false, "Use SPIRE CLI commands instead of SDK")

	// Add subcommands
	inspectCmd.AddCommand(inspectSvidCmd)
	inspectCmd.AddCommand(inspectBundleCmd)
	inspectCmd.AddCommand(inspectAuthoritiesCmd)
}

func runInspectSvid(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	useCLI, _ := cmd.Flags().GetBool("use-cli")

	if useCLI {
		return runInspectSvidCLI(cmd)
	}
	return runInspectSvidSDK(ctx, cmd)
}

func runInspectSvidSDK(ctx context.Context, cmd *cobra.Command) error {
	socket, _ := cmd.Flags().GetString("socket")
	if socket == "" {
		socket = "unix:///tmp/spire-agent/public/api.sock"
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use go-spiffe/v2 SDK directly
	clientOptions := workloadapi.WithClientOptions(
		workloadapi.WithAddr(socket),
	)

	source, err := workloadapi.NewX509Source(ctxWithTimeout, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create X509Source: %w", err)
	}
	defer source.Close()

	// Get SVID using built-in SDK
	svid, err := source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	return outputSvidInfo(cmd, svid)
}

func runInspectSvidCLI(cmd *cobra.Command) error {
	socket, _ := cmd.Flags().GetString("socket")
	if socket == "" {
		socket = "/tmp/spire-agent/public/api.sock"
	}

	// Use SPIRE's built-in CLI command
	spireCmd := exec.Command("spire-agent", "api", "fetch", "x509",
		"-socketPath", strings.TrimPrefix(socket, "unix://"))

	output, err := spireCmd.Output()
	if err != nil {
		return fmt.Errorf("spire-agent api fetch failed: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		// Parse and structure the CLI output
		return outputSvidCLIAsJSON(output)
	default:
		// Display raw CLI output
		fmt.Print(string(output))
	}

	return nil
}

func runInspectBundle(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	useCLI, _ := cmd.Flags().GetBool("use-cli")

	var trustDomain spiffeid.TrustDomain
	if len(args) > 0 {
		td, err := spiffeid.TrustDomainFromString(args[0])
		if err != nil {
			return fmt.Errorf("invalid trust domain %s: %w", args[0], err)
		}
		trustDomain = td
	}

	if useCLI {
		return runInspectBundleCLI(cmd, trustDomain)
	}
	return runInspectBundleSDK(ctx, cmd, trustDomain)
}

func runInspectBundleSDK(ctx context.Context, cmd *cobra.Command, trustDomain spiffeid.TrustDomain) error {
	socket, _ := cmd.Flags().GetString("socket")
	if socket == "" {
		socket = "unix:///tmp/spire-agent/public/api.sock"
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use go-spiffe/v2 SDK directly
	clientOptions := workloadapi.WithClientOptions(
		workloadapi.WithAddr(socket),
	)

	source, err := workloadapi.NewX509Source(ctxWithTimeout, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create X509Source: %w", err)
	}
	defer source.Close()

	// If no trust domain specified, get it from current SVID
	if trustDomain.IsZero() {
		svid, err := source.GetX509SVID()
		if err != nil {
			return fmt.Errorf("failed to get SVID for trust domain: %w", err)
		}
		trustDomain = svid.ID.TrustDomain()
	}

	// Get trust bundle using built-in SDK
	bundle, err := source.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		return fmt.Errorf("failed to get trust bundle for %s: %w", trustDomain, err)
	}

	return outputBundleInfo(cmd, trustDomain, bundle)
}

func runInspectBundleCLI(cmd *cobra.Command, trustDomain spiffeid.TrustDomain) error {
	serverSocket, _ := cmd.Flags().GetString("server-socket")
	if serverSocket == "" {
		serverSocket = "/tmp/spire-server/private/api.sock"
	}

	// Use SPIRE's built-in bundle show command
	args := []string{"bundle", "show", "-format", "pem"}
	if serverSocket != "" {
		args = append(args, "-socketPath", strings.TrimPrefix(serverSocket, "unix://"))
	}

	spireCmd := exec.Command("spire-server", args...)
	output, err := spireCmd.Output()
	if err != nil {
		return fmt.Errorf("spire-server bundle show failed: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return outputBundleCLIAsJSON(output, trustDomain)
	default:
		fmt.Print(string(output))
	}

	return nil
}

func runInspectAuthorities(cmd *cobra.Command, args []string) error {
	serverSocket, _ := cmd.Flags().GetString("server-socket")
	if serverSocket == "" {
		serverSocket = "/tmp/spire-server/private/api.sock"
	}

	// Use SPIRE's built-in localauthority command
	args = []string{"localauthority", "x509", "show"}
	if serverSocket != "" {
		args = append(args, "-socketPath", strings.TrimPrefix(serverSocket, "unix://"))
	}

	spireCmd := exec.Command("spire-server", args...)
	output, err := spireCmd.Output()
	if err != nil {
		return fmt.Errorf("spire-server localauthority x509 show failed: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return outputAuthoritiesCLIAsJSON(output)
	default:
		fmt.Print(string(output))
	}

	return nil
}

func outputSvidInfo(cmd *cobra.Command, svid *x509svid.SVID) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		svidInfo := map[string]interface{}{
			"spiffe_id":         svid.ID.String(),
			"trust_domain":      svid.ID.TrustDomain().String(),
			"certificate_count": len(svid.Certificates),
			"has_private_key":   svid.PrivateKey != nil,
		}

		if len(svid.Certificates) > 0 {
			cert := svid.Certificates[0]
			svidInfo["not_before"] = cert.NotBefore
			svidInfo["not_after"] = cert.NotAfter
			svidInfo["serial_number"] = cert.SerialNumber.String()
			svidInfo["subject"] = cert.Subject.String()
			svidInfo["issuer"] = cert.Issuer.String()
		}

		return json.NewEncoder(os.Stdout).Encode(svidInfo)
	default:
		return outputSvidInfoText(svid, quiet, noEmoji)
	}
}

func outputSvidInfoText(svid *x509svid.SVID, quiet, noEmoji bool) error {
	status := "📋"
	if noEmoji {
		status = "[SVID]"
	}

	fmt.Printf("%s X.509 SVID Information\n", status)

	if !quiet {
		fmt.Printf("SPIFFE ID: %s\n", svid.ID)
		fmt.Printf("Trust Domain: %s\n", svid.ID.TrustDomain())
		fmt.Printf("Certificate Count: %d\n", len(svid.Certificates))
		fmt.Printf("Has Private Key: %t\n", svid.PrivateKey != nil)

		if len(svid.Certificates) > 0 {
			cert := svid.Certificates[0]
			fmt.Printf("Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
			fmt.Printf("Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))
			fmt.Printf("Serial Number: %s\n", cert.SerialNumber.String())
			fmt.Printf("Subject: %s\n", cert.Subject.String())
			fmt.Printf("Issuer: %s\n", cert.Issuer.String())
		}
	}

	return nil
}

func outputBundleInfo(cmd *cobra.Command, trustDomain spiffeid.TrustDomain, bundle *x509bundle.Bundle) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		authorities := bundle.X509Authorities()
		bundleInfo := map[string]interface{}{
			"trust_domain":      trustDomain.String(),
			"certificate_count": len(authorities),
			"authorities":       make([]map[string]interface{}, len(authorities)),
		}

		for i, cert := range authorities {
			bundleInfo["authorities"].([]map[string]interface{})[i] = map[string]interface{}{
				"subject":       cert.Subject.String(),
				"serial_number": cert.SerialNumber.String(),
				"not_before":    cert.NotBefore,
				"not_after":     cert.NotAfter,
				"is_ca":         cert.IsCA,
			}
		}

		return json.NewEncoder(os.Stdout).Encode(bundleInfo)
	default:
		return outputBundleInfoText(trustDomain, bundle, quiet, noEmoji)
	}
}

func outputBundleInfoText(trustDomain spiffeid.TrustDomain, bundle *x509bundle.Bundle, quiet, noEmoji bool) error {
	status := "🔐"
	if noEmoji {
		status = "[BUNDLE]"
	}

	fmt.Printf("%s Trust Bundle Information\n", status)

	if !quiet {
		fmt.Printf("Trust Domain: %s\n", trustDomain)

		authorities := bundle.X509Authorities()
		fmt.Printf("Certificate Count: %d\n", len(authorities))

		for i, cert := range authorities {
			fmt.Printf("\nAuthority %d:\n", i+1)
			fmt.Printf("  Subject: %s\n", cert.Subject.String())
			fmt.Printf("  Serial: %s\n", cert.SerialNumber.String())
			fmt.Printf("  Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
			fmt.Printf("  Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))
			fmt.Printf("  Is CA: %t\n", cert.IsCA)
		}
	}

	return nil
}

func outputSvidCLIAsJSON(output []byte) error {
	// Parse spire-agent CLI output and convert to JSON
	// This is a simplified implementation - real parsing would be more complex
	result := map[string]interface{}{
		"raw_output": string(output),
		"source":     "spire-agent-cli",
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func outputBundleCLIAsJSON(output []byte, trustDomain spiffeid.TrustDomain) error {
	// Parse spire-server bundle show output and convert to JSON
	result := map[string]interface{}{
		"trust_domain": trustDomain.String(),
		"raw_output":   string(output),
		"source":       "spire-server-cli",
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func outputAuthoritiesCLIAsJSON(output []byte) error {
	// Parse spire-server localauthority output and convert to JSON
	result := map[string]interface{}{
		"raw_output": string(output),
		"source":     "spire-server-cli",
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
