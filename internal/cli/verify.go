// Package cli provides command-line interface for SPIRE identity verification
// and diagnostics using SPIRE's built-in capabilities rather than custom implementation.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/adapters/secondary/verification"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Output templates for verification results
const verifyResultTemplate = `{{if .Valid}}✅{{else}}❌{{end}} Identity Verification
Identity: {{.Identity}}
Trust Domain: {{.TrustDomain}}
Valid: {{.Valid}}
Message: {{.Message}}
Verified At: {{.VerifiedAt.Format "2006-01-02 15:04:05"}}
{{if not .NotBefore.IsZero}}Not Before: {{.NotBefore.Format "2006-01-02 15:04:05"}}{{end}}
{{if not .NotAfter.IsZero}}Not After: {{.NotAfter.Format "2006-01-02 15:04:05"}}{{end}}
{{if .SerialNumber}}Serial Number: {{.SerialNumber}}{{end}}`

const identityInfoTemplate = `🆔 Current Identity
SPIFFE ID: {{.SPIFFEID}}
Trust Domain: {{.SPIFFEID.TrustDomain}}
Source: {{.Source}}
Fetched At: {{.FetchedAt.Format "2006-01-02 15:04:05"}}
Has SVID: {{if .SVID}}true{{else}}false{{end}}
Has Trust Bundle: {{if .TrustBundle}}true{{else}}false{{end}}`

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
	Args:    cobra.ExactArgs(1),
	PreRunE: validateVerifyIdentityArgs,
	RunE:    runVerifyIdentity,
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
	Args:    cobra.ExactArgs(2),
	PreRunE: validateVerifyConnectionArgs,
	RunE:    runVerifyConnection,
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

	// Add flag validation - socket paths should be file paths
	verifyCmd.MarkPersistentFlagFilename("socket")

	// Add completions for common values
	verifyCmd.RegisterFlagCompletionFunc("trust-domain", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"example.org\tDefault domain",
			"localhost\tLocal development",
			"prod.company.com\tProduction",
			"staging.company.com\tStaging",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// Add subcommands
	verifyCmd.AddCommand(verifyIdentityCmd)
	verifyCmd.AddCommand(verifyCurrentCmd)
	verifyCmd.AddCommand(verifyConnectionCmd)
	verifyCmd.AddCommand(verifyRefreshCmd)
}

// validateVerifyIdentityArgs validates the SPIFFE ID argument
func validateVerifyIdentityArgs(cmd *cobra.Command, args []string) error {
	if _, err := spiffeid.FromString(args[0]); err != nil {
		return fmt.Errorf("invalid SPIFFE ID %s: %w", args[0], err)
	}
	return nil
}

// validateVerifyConnectionArgs validates connection arguments
func validateVerifyConnectionArgs(cmd *cobra.Command, args []string) error {
	if _, err := spiffeid.FromString(args[0]); err != nil {
		return fmt.Errorf("invalid target SPIFFE ID %s: %w", args[0], err)
	}
	// Basic address validation could be added here
	return nil
}

func runVerifyIdentity(cmd *cobra.Command, args []string) error {
	expectedID, _ := spiffeid.FromString(args[0]) // Already validated in PreRunE

	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	result, err := verifier.VerifyIdentity(cmd.Context(), expectedID)
	if err != nil {
		return fmt.Errorf("identity verification failed: %w", err)
	}

	return outputResult(cmd, result)
}

func runVerifyCurrent(cmd *cobra.Command, args []string) error {
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	identity, err := verifier.GetCurrentIdentity(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get current identity: %w", err)
	}

	return outputIdentity(cmd, identity)
}

func runVerifyConnection(cmd *cobra.Command, args []string) error {
	targetID, _ := spiffeid.FromString(args[0]) // Already validated in PreRunE
	address := args[1]

	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	result, err := verifier.ValidateConnection(cmd.Context(), targetID, address)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	return outputResult(cmd, result)
}

func runVerifyRefresh(cmd *cobra.Command, args []string) error {
	verifier, err := createIdentityVerifier(cmd)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	identity, err := verifier.RefreshIdentity(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to refresh identity: %w", err)
	}

	return outputIdentity(cmd, identity)
}

func createIdentityVerifier(cmd *cobra.Command) (*verification.SpireIdentityVerifier, error) {
	config := &ports.VerificationConfig{}

	// Get configuration from flags
	if socket, _ := cmd.Flags().GetString("socket"); socket != "" {
		config.WorkloadAPISocket = socket
	}

	if timeout, _ := cmd.Flags().GetDuration("timeout"); timeout > 0 {
		config.Timeout = timeout
	}

	if trustDomain, _ := cmd.Flags().GetString("trust-domain"); trustDomain != "" {
		td, err := spiffeid.TrustDomainFromString(trustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain %s: %w", trustDomain, err)
		}
		config.TrustDomain = td
	}

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

// outputJSONData is a utility function to encode any data as JSON to stdout
func outputJSONData(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// outputOptions holds common output configuration flags
type outputOptions struct {
	Format  string
	Quiet   bool
	NoEmoji bool
}

// getOutputOptions extracts common output flags from a Cobra command
func getOutputOptions(cmd *cobra.Command) outputOptions {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	return outputOptions{
		Format:  format,
		Quiet:   quiet,
		NoEmoji: noEmoji,
	}
}

// outputResult outputs verification result using template or JSON
func outputResult(cmd *cobra.Command, result *ports.IdentityVerificationResult) error {
	opts := getOutputOptions(cmd)

	if opts.Quiet && !result.Valid {
		return fmt.Errorf("verification failed")
	}

	if opts.Quiet {
		return nil
	}

	switch opts.Format {
	case "json":
		return outputJSONData(result)
	default:
		// Use template for text output
		tmplText := verifyResultTemplate
		if opts.NoEmoji {
			tmplText = replaceEmojis(tmplText)
		}

		tmpl := template.Must(template.New("result").Parse(tmplText))
		return tmpl.Execute(os.Stdout, result)
	}
}

// outputIdentity outputs identity info using template or JSON
func outputIdentity(cmd *cobra.Command, identity *ports.IdentityInfo) error {
	opts := getOutputOptions(cmd)

	if opts.Quiet {
		fmt.Println(identity.SPIFFEID.String())
		return nil
	}

	switch opts.Format {
	case "json":
		// Create a JSON-safe version
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
		return outputJSONData(jsonIdentity)
	default:
		// Use template for text output
		tmplText := identityInfoTemplate
		if opts.NoEmoji {
			tmplText = replaceEmojis(tmplText)
		}

		tmpl := template.Must(template.New("identity").Parse(tmplText))
		return tmpl.Execute(os.Stdout, identity)
	}
}

// replaceEmojis replaces emojis with text equivalents
func replaceEmojis(text string) string {
	replacements := map[string]string{
		"✅": "[PASS]",
		"❌": "[FAIL]",
		"🆔": "[ID]",
	}

	result := text
	for emoji, replacement := range replacements {
		result = strings.ReplaceAll(result, emoji, replacement)
	}
	return result
}
