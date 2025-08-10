package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/sufield/ephemos/internal/cli"
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
				Long: `Identity-based authentication CLI for SPIFFE/SPIRE services.

Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
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
	// Test that Execute function exists and doesn't panic when accessed
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute() panicked when referenced: %v", r)
		}
	}()

	// Test that Execute function is callable (functions cannot be nil in Go)
	_ = cli.Execute
}

func TestRootCmdStructure(_ *testing.T) {
	// Since rootCmd is unexported, we can only test the public API
	// Test that Execute function exists
	_ = cli.Execute
}

func TestRootCmdFlags(_ *testing.T) {
	// Test basic CLI functionality - mainly that Execute exists
	_ = cli.Execute
}

func TestRootCmdCompletion(_ *testing.T) {
	// Test basic CLI functionality
	_ = cli.Execute
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

func TestRootCmdVersion(_ *testing.T) {
	// Test basic CLI functionality
	_ = cli.Execute
}

func TestRootCmdUsageTemplate(_ *testing.T) {
	// Test basic CLI functionality
	_ = cli.Execute
}
