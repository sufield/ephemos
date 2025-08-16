package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `Generate shell completion scripts for the ephemos CLI.

The completion script for each shell will be output to stdout.
You can source it directly or save it to a file and source that.

Examples:
  # Bash completion (requires bash-completion package)
  ephemos completion bash > /etc/bash_completion.d/ephemos
  
  # Zsh completion
  ephemos completion zsh > "${fpath[1]}/_ephemos"
  
  # Fish completion
  ephemos completion fish > ~/.config/fish/completions/ephemos.fish
  
  # PowerShell completion
  ephemos completion powershell > ephemos.ps1`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runCompletion,
}

func runCompletion(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("%w: unsupported shell %q", ErrUsage, shell)
	}
}

var manCmd = &cobra.Command{
	Use:   "man [directory]",
	Short: "Generate manual pages",
	Long: `Generate manual pages for the ephemos CLI.

If no directory is specified, manual pages will be generated in the current directory.
The generated manual pages can be installed in your system's manual page directories.

Example:
  ephemos man /usr/local/share/man/man1`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMan,
}

func runMan(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	header := &doc.GenManHeader{
		Title:   "EPHEMOS",
		Section: "1",
		Source:  "Ephemos CLI " + Version,
		Manual:  "Ephemos Manual",
	}

	if err := doc.GenManTree(cmd.Root(), header, dir); err != nil {
		return fmt.Errorf("%w: failed to generate manual pages: %v", ErrInternal, err)
	}

	fmt.Fprintf(os.Stderr, "Manual pages generated in directory: %s\n", dir)
	return nil
}

func init() {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(manCmd)
}