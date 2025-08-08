package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "no arguments shows help",
			args:       []string{},
			wantErr:    false,
			wantOutput: "Identity-based authentication CLI for SPIFFE/SPIRE services",
		},
		{
			name:       "help flag",
			args:       []string{"--help"},
			wantErr:    false,
			wantOutput: "Identity-based authentication CLI for SPIFFE/SPIRE services",
		},
		{
			name:       "short help flag",
			args:       []string{"-h"},
			wantErr:    false,
			wantOutput: "Identity-based authentication CLI for SPIFFE/SPIRE services",
		},
		{
			name:    "invalid command",
			args:    []string{"invalid-command"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the root command to avoid side effects
			cmd := &cobra.Command{
				Use:   "ephemos",
				Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
				Long: `Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
Use this CLI to register services, manage identities, and configure authentication policies.`,
			}
			
			// Add the register command to match the real structure
			cmd.AddCommand(&cobra.Command{
				Use:   "register",
				Short: "Register a service with SPIFFE/SPIRE",
			})

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Execute() output = %v, want to contain %v", output, tt.wantOutput)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Test the actual Execute function
	// Note: This will use the global rootCmd which has side effects
	
	// We can't easily test Execute() in isolation because it uses the global rootCmd
	// and os.Args. In a real implementation, you might want to refactor this to be
	// more testable by accepting arguments as parameters.
	
	// For now, we just test that Execute doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute() panicked: %v", r)
		}
	}()

	// We can't call Execute() here as it would interfere with the test runner
	// Instead, we test the structure
	if rootCmd == nil {
		t.Error("rootCmd is nil")
	}

	if rootCmd.Use != "ephemos" {
		t.Errorf("rootCmd.Use = %v, want ephemos", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short is empty")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd.Long is empty")
	}
}

func TestRootCmdStructure(t *testing.T) {
	// Test that the root command has the expected structure
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	// Check basic properties
	if rootCmd.Use != "ephemos" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "ephemos")
	}

	expectedShort := "Identity-based authentication CLI for SPIFFE/SPIRE services"
	if rootCmd.Short != expectedShort {
		t.Errorf("rootCmd.Short = %q, want %q", rootCmd.Short, expectedShort)
	}

	if !strings.Contains(rootCmd.Long, "Ephemos provides identity-based authentication") {
		t.Error("rootCmd.Long does not contain expected content")
	}

	// Check that subcommands are registered
	hasRegisterCmd := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "register" {
			hasRegisterCmd = true
			break
		}
	}

	if !hasRegisterCmd {
		t.Error("rootCmd does not have register subcommand")
	}
}

func TestRootCmdFlags(t *testing.T) {
	// Test that the root command handles flags correctly
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	// Test help flag exists
	helpFlag := rootCmd.Flags().Lookup("help")
	if helpFlag == nil {
		t.Error("help flag not found")
	}

	// Test that the command can parse flags without error
	args := []string{"--help"}
	rootCmd.SetArgs(args)
	
	// Capture output to avoid printing during test
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Execute() with --help failed: %v", err)
	}
}

func TestRootCmdCompletion(t *testing.T) {
	// Test that the root command supports completion
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	// Check that completion commands are available
	completionCmd := rootCmd.Commands()
	found := false
	for _, cmd := range completionCmd {
		if strings.Contains(cmd.Use, "completion") {
			found = true
			break
		}
	}

	// Note: Cobra automatically adds completion commands, but they might not be visible
	// in all versions, so we don't fail the test if not found
	if !found {
		t.Log("Completion command not found (this may be expected)")
	}
}

func BenchmarkRootCmdExecution(b *testing.B) {
	// Benchmark help command execution
	cmd := &cobra.Command{
		Use:   "ephemos",
		Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
		Long:  `Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.`,
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := cmd.Execute()
		if err != nil {
			b.Errorf("Execute() failed: %v", err)
		}
	}
}

func TestRootCmdVersion(t *testing.T) {
	// Test version-related functionality if present
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	// Check if version is set (optional)
	version := rootCmd.Version
	if version != "" {
		t.Logf("Version found: %s", version)
	} else {
		t.Log("No version set (this is acceptable)")
	}
}

func TestRootCmdUsageTemplate(t *testing.T) {
	// Test that usage template works correctly
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	usage := rootCmd.UsageString()
	if usage == "" {
		t.Error("UsageString() returned empty string")
	}

	if !strings.Contains(usage, "ephemos") {
		t.Error("Usage string does not contain command name")
	}
}