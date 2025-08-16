package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information - injected at compile time via ldflags/x_defs
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	BuildUser = "unknown"
	BuildHost = "unknown"
)

// VersionInfo contains detailed version and build information
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	BuildUser string `json:"build_user"`
	BuildHost string `json:"build_host"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"os"`
	GOARCH    string `json:"arch"`
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
		BuildHost: BuildHost,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display detailed version and build information for the Ephemos CLI.",
	RunE:  runVersion,
}

func runVersion(cmd *cobra.Command, args []string) error {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("%w: failed to get format flag: %v", ErrUsage, err)
	}

	info := GetVersionInfo()

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(info); err != nil {
			return fmt.Errorf("%w: failed to encode version info as JSON: %v", ErrInternal, err)
		}
	case "text":
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Commit: %s\n", info.Commit)
		fmt.Printf("Build Date: %s\n", info.BuildDate)
		fmt.Printf("Build User: %s\n", info.BuildUser)
		fmt.Printf("Build Host: %s\n", info.BuildHost)
		fmt.Printf("Go Version: %s\n", info.GoVersion)
		fmt.Printf("OS/Arch: %s/%s\n", info.GOOS, info.GOARCH)
	default:
		return fmt.Errorf("%w: unsupported format %q, use 'text' or 'json'", ErrUsage, format)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
}