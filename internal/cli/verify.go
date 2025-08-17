// Package cli provides command-line interface for SPIRE identity verification
// and diagnostics using SPIRE's built-in capabilities rather than custom implementation.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/adapters/secondary/verification"
	"github.com/sufield/ephemos/internal/core/ports"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Identity verification commands using SPIRE's built-in capabilities",
	Long: `Identity verification commands that leverage SPIRE's built-in identity verification
mechanisms through the go-spiffe/v2 library rather than implementing custom verification logic.

Available subcommands:
  identity     Verify a SPIFFE identity
  current      Get current workload identity
  connection   Validate connection to a specific SPIFFE ID
  refresh      Refresh workload identity`,
}

var verifyIdentityCmd = &cobra.Command{
	Use:   "identity <spiffe-id>",
	Short: "Verify a SPIFFE identity using the Workload API",
	Long: `Verify a SPIFFE identity using SPIRE's built-in Workload API verification.
This command leverages the go-spiffe/v2 library to verify that the current workload
identity matches the expected SPIFFE ID.

Example:
  ephemos verify identity spiffe://example.org/myservice`,
	Args: cobra.ExactArgs(1),
	RunE: runVerifyIdentity,
}

var verifyCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Get current workload identity from SPIRE",
	Long: `Get the current workload identity using SPIRE's Workload API.
This command uses go-spiffe/v2 to fetch the current SVID and trust bundle
from the SPIRE agent.`,
	RunE: runVerifyCurrent,
}

var verifyConnectionCmd = &cobra.Command{
	Use:   "connection <spiffe-id> <address>",
	Short: "Validate mTLS connection to a specific SPIFFE ID",
	Long: `Validate a mutual TLS connection to a service with the specified SPIFFE ID.
This command establishes a connection using SPIRE's built-in mTLS capabilities
and verifies the peer's identity.

Example:
  ephemos verify connection spiffe://example.org/backend localhost:8080`,
	Args: cobra.ExactArgs(2),
	RunE: runVerifyConnection,
}

var verifyRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh workload identity from SPIRE",
	Long: `Force a refresh of the workload identity from SPIRE.
This command closes the existing Workload API connection and re-establishes
it to get fresh identity material.`,
	RunE: runVerifyRefresh,
}

func init() {
	// Add persistent flags for verification
	verifyCmd.PersistentFlags().String("socket", "", "Workload API socket path (default: unix:///tmp/spire-agent/public/api.sock)")
	verifyCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Verification timeout")
	verifyCmd.PersistentFlags().String("trust-domain", "", "Expected trust domain")
	verifyCmd.PersistentFlags().StringSlice("allowed-ids", []string{}, "Allowed SPIFFE IDs")

	// Add subcommands
	verifyCmd.AddCommand(verifyIdentityCmd)
	verifyCmd.AddCommand(verifyCurrentCmd)
	verifyCmd.AddCommand(verifyConnectionCmd)
	verifyCmd.AddCommand(verifyRefreshCmd)
}

func runVerifyIdentity(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	expectedIDStr := args[0]

	// Parse expected SPIFFE ID
	expectedID, err := spiffeid.FromString(expectedIDStr)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE ID %s: %w", expectedIDStr, err)
	}

	// Create verifier
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	// Verify identity
	result, err := verifier.VerifyIdentity(ctx, expectedID)
	if err != nil {
		return fmt.Errorf("identity verification failed: %w", err)
	}

	return outputVerificationResult(cmd, result)
}

func runVerifyCurrent(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create verifier
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	// Get current identity
	identity, err := verifier.GetCurrentIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current identity: %w", err)
	}

	return outputIdentityInfo(cmd, identity)
}

func runVerifyConnection(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	targetIDStr := args[0]
	address := args[1]

	// Parse target SPIFFE ID
	targetID, err := spiffeid.FromString(targetIDStr)
	if err != nil {
		return fmt.Errorf("invalid target SPIFFE ID %s: %w", targetIDStr, err)
	}

	// Create verifier
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	// Validate connection
	result, err := verifier.ValidateConnection(ctx, targetID, address)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	return outputVerificationResult(cmd, result)
}

func runVerifyRefresh(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create verifier
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	// Refresh identity
	identity, err := verifier.RefreshIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh identity: %w", err)
	}

	return outputIdentityInfo(cmd, identity)
}

func createIdentityVerifier(cmd *cobra.Command) (*verification.SpireIdentityVerifier, error) {
	config := &ports.VerificationConfig{}

	// Get socket path
	if socket, _ := cmd.Flags().GetString("socket"); socket != "" {
		config.WorkloadAPISocket = socket
	}

	// Get timeout
	if timeout, _ := cmd.Flags().GetDuration("timeout"); timeout > 0 {
		config.Timeout = timeout
	}

	// Get trust domain
	if trustDomain, _ := cmd.Flags().GetString("trust-domain"); trustDomain != "" {
		td, err := spiffeid.TrustDomainFromString(trustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain %s: %w", trustDomain, err)
		}
		config.TrustDomain = td
	}

	// Get allowed SPIFFE IDs
	if allowedIDs, _ := cmd.Flags().GetStringSlice("allowed-ids"); len(allowedIDs) > 0 {
		for _, idStr := range allowedIDs {
			id, err := spiffeid.FromString(idStr)
			if err != nil {
				return nil, fmt.Errorf("invalid allowed SPIFFE ID %s: %w", idStr, err)
			}
			config.AllowedSPIFFEIDs = append(config.AllowedSPIFFEIDs, id)
		}
	}

	return verification.NewSpireIdentityVerifier(config)
}

func outputVerificationResult(cmd *cobra.Command, result *ports.IdentityVerificationResult) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		return outputVerificationResultText(result, quiet, noEmoji)
	}
}

func outputVerificationResultText(result *ports.IdentityVerificationResult, quiet, noEmoji bool) error {
	// Status indicator
	status := "❌"
	if noEmoji {
		status = "[FAIL]"
	}
	if result.Valid {
		status = "✅"
		if noEmoji {
			status = "[PASS]"
		}
	}

	fmt.Printf("%s Identity Verification\n", status)
	
	if !quiet {
		fmt.Printf("Identity: %s\n", result.Identity)
		fmt.Printf("Trust Domain: %s\n", result.TrustDomain)
		fmt.Printf("Valid: %t\n", result.Valid)
		fmt.Printf("Message: %s\n", result.Message)
		fmt.Printf("Verified At: %s\n", result.VerifiedAt.Format(time.RFC3339))
		
		if !result.NotBefore.IsZero() {
			fmt.Printf("Not Before: %s\n", result.NotBefore.Format(time.RFC3339))
		}
		if !result.NotAfter.IsZero() {
			fmt.Printf("Not After: %s\n", result.NotAfter.Format(time.RFC3339))
		}
		if result.SerialNumber != "" {
			fmt.Printf("Serial Number: %s\n", result.SerialNumber)
		}
		// Key usage details available via 'ephemos inspect svid' command
	}

	return nil
}

func outputIdentityInfo(cmd *cobra.Command, identity *ports.IdentityInfo) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	switch format {
	case "json":
		// Create a JSON-safe version of identity info
		jsonIdentity := struct {
			SPIFFEID    string    `json:"spiffe_id"`
			TrustDomain string    `json:"trust_domain"`
			FetchedAt   time.Time `json:"fetched_at"`
			Source      string    `json:"source"`
			HasSVID     bool      `json:"has_svid"`
			HasBundle   bool      `json:"has_bundle"`
		}{
			SPIFFEID:    identity.SPIFFEID.String(),
			TrustDomain: identity.SPIFFEID.TrustDomain().String(),
			FetchedAt:   identity.FetchedAt,
			Source:      identity.Source,
			HasSVID:     identity.SVID != nil,
			HasBundle:   identity.TrustBundle != nil,
		}
		return json.NewEncoder(os.Stdout).Encode(jsonIdentity)
	default:
		return outputIdentityInfoText(identity, quiet, noEmoji)
	}
}

func outputIdentityInfoText(identity *ports.IdentityInfo, quiet, noEmoji bool) error {
	// Status indicator
	status := "🆔"
	if noEmoji {
		status = "[ID]"
	}

	fmt.Printf("%s Current Identity\n", status)
	
	if !quiet {
		fmt.Printf("SPIFFE ID: %s\n", identity.SPIFFEID)
		fmt.Printf("Trust Domain: %s\n", identity.SPIFFEID.TrustDomain())
		fmt.Printf("Source: %s\n", identity.Source)
		fmt.Printf("Fetched At: %s\n", identity.FetchedAt.Format(time.RFC3339))
		fmt.Printf("Has SVID: %t\n", identity.SVID != nil)
		fmt.Printf("Has Trust Bundle: %t\n", identity.TrustBundle != nil)

		if identity.SVID != nil && len(identity.SVID.Certificates) > 0 {
			cert := identity.SVID.Certificates[0]
			fmt.Printf("Certificate Expires: %s\n", cert.NotAfter.Format(time.RFC3339))
			fmt.Printf("Certificate Serial: %s\n", cert.SerialNumber.String())
		}
	}

	return nil
}