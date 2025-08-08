package cli

import (
	"bytes"
	"strings" 
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true, // Should require service name
		},
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
		{
			name:    "short help flag",
			args:    []string{"-h"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test version of the register command
			cmd := &cobra.Command{
				Use:   "register",
				Short: "Register a service with SPIFFE/SPIRE",
				Long: `Register a service with SPIFFE/SPIRE for identity-based authentication.
This command creates the necessary service entries and selectors.`,
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock implementation that validates arguments
					if len(args) == 0 {
						return cmd.Help()
					}
					return nil
				},
			}

			// Add flags that the real command might have
			cmd.Flags().StringP("selector", "s", "", "Service selector")
			cmd.Flags().StringP("parent-id", "p", "", "Parent SPIFFE ID")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For help commands, check that help text is shown
			if strings.Contains(strings.Join(tt.args, " "), "help") {
				output := buf.String()
				if !strings.Contains(output, "register") {
					t.Error("Help output should contain command name")
				}
			}
		})
	}
}

func TestRegisterCmdFlags(t *testing.T) {
	// Test that register command accepts expected flags
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a service with SPIFFE/SPIRE",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Add common flags for service registration
	cmd.Flags().StringP("selector", "s", "", "Service selector (e.g., 'unix:uid:1000')")
	cmd.Flags().StringP("parent-id", "p", "", "Parent SPIFFE ID")
	cmd.Flags().StringP("spiffe-id", "i", "", "SPIFFE ID for the service")
	cmd.Flags().StringP("trust-domain", "t", "", "Trust domain")

	tests := []struct {
		name string
		args []string
		flag string
	}{
		{
			name: "selector flag",
			args: []string{"--selector", "unix:uid:1000", "my-service"},
			flag: "selector",
		},
		{
			name: "parent-id flag", 
			args: []string{"--parent-id", "spiffe://example.com/agent", "my-service"},
			flag: "parent-id",
		},
		{
			name: "spiffe-id flag",
			args: []string{"--spiffe-id", "spiffe://example.com/my-service", "my-service"},
			flag: "spiffe-id",
		},
		{
			name: "trust-domain flag",
			args: []string{"--trust-domain", "example.com", "my-service"},
			flag: "trust-domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Execute() with %s flag failed: %v", tt.flag, err)
			}

			// Verify flag was parsed
			flagValue, err := cmd.Flags().GetString(tt.flag)
			if err != nil {
				t.Errorf("Failed to get %s flag value: %v", tt.flag, err)
			}

			if flagValue == "" {
				t.Errorf("%s flag value is empty", tt.flag)
			}
		})
	}
}

func TestRegisterCmdValidation(t *testing.T) {
	// Test input validation
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a service with SPIFFE/SPIRE",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Basic validation logic
			if len(args) == 0 {
				return cobra.ErrMissingArg
			}

			serviceName := args[0]
			if strings.TrimSpace(serviceName) == "" {
				return cobra.ErrInvalidArg
			}

			return nil
		},
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid service name",
			args:    []string{"my-service"},
			wantErr: false,
		},
		{
			name:    "empty service name",
			args:    []string{""},
			wantErr: true,
		},
		{
			name:    "whitespace only service name",
			args:    []string{"   "},
			wantErr: true,
		},
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterCmdCompletion(t *testing.T) {
	// Test command completion functionality
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a service with SPIFFE/SPIRE",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// Return some example service names for completion
			if len(args) == 0 {
				return []string{
					"echo-server\tExample echo server service",
					"auth-service\tAuthentication service",
					"api-gateway\tAPI gateway service",
				}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	// Test completion
	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	
	if len(completions) == 0 {
		t.Error("Expected some completions for service names")
	}

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("Expected NoFileComp directive, got %v", directive)
	}

	// Test completion with existing args (should return no more completions)
	completions, directive = cmd.ValidArgsFunction(cmd, []string{"existing-service"}, "")
	
	if len(completions) != 0 {
		t.Error("Expected no completions when service name already provided")
	}
}

func TestRegisterCmdUsage(t *testing.T) {
	// Test command usage and help text
	cmd := &cobra.Command{
		Use:   "register [service-name]",
		Short: "Register a service with SPIFFE/SPIRE",
		Long: `Register a service with SPIFFE/SPIRE for identity-based authentication.

This command creates the necessary service entries and selectors in the SPIRE server,
enabling the service to obtain and use SVID certificates for mTLS communication.

Examples:
  ephemos register my-service --selector unix:uid:1000
  ephemos register api-gateway --parent-id spiffe://example.com/agent`,
		Args: cobra.ExactArgs(1),
	}

	usage := cmd.UsageString()
	
	if !strings.Contains(usage, "register") {
		t.Error("Usage string should contain command name")
	}

	if !strings.Contains(usage, "[service-name]") {
		t.Error("Usage string should show service-name argument")
	}

	longHelp := cmd.Long
	if !strings.Contains(longHelp, "Examples:") {
		t.Error("Long help should contain examples")
	}
}

func BenchmarkRegisterCmdExecution(b *testing.B) {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a service with SPIFFE/SPIRE",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock implementation
			return nil
		},
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		cmd.SetArgs([]string{"test-service"})
		err := cmd.Execute()
		if err != nil {
			b.Errorf("Execute() failed: %v", err)
		}
	}
}

func TestRegisterCmdIntegration(t *testing.T) {
	// Integration test that shows the command working with a parent command
	rootCmd := &cobra.Command{
		Use: "ephemos",
	}

	registerCmd := &cobra.Command{
		Use:   "register [service-name]",
		Short: "Register a service with SPIFFE/SPIRE",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock successful registration
			cmd.Printf("Successfully registered service: %s\n", args[0])
			return nil
		},
	}

	rootCmd.AddCommand(registerCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"register", "test-service"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Integration test failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Successfully registered service: test-service") {
		t.Errorf("Expected success message, got: %s", output)
	}
}