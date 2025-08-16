package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// manCmd generates manual pages (keeping this as it's not built into Cobra)
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
	// Cobra automatically adds the completion command, so we don't need to add it manually
	rootCmd.AddCommand(manCmd)
}