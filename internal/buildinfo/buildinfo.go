// Package buildinfo provides build-time information for Ephemos binaries.
// Build information is injected at compile time via ldflags/x_defs.
package buildinfo

// Build information variables - injected at compile time via ldflags/x_defs
var (
	Version   = "dev"
	CommitHash = "unknown"
	BuildTime = "unknown"
	BuildUser = "unknown"
	BuildHost = "unknown"
)

// Info returns a structured representation of the build information
type Info struct {
	Version   string `json:"version"`
	CommitHash string `json:"commit_hash"`
	BuildTime string `json:"build_time"`
	BuildUser string `json:"build_user"`
	BuildHost string `json:"build_host"`
}

// Get returns the current build information as a structured Info
func Get() Info {
	return Info{
		Version:   Version,
		CommitHash: CommitHash,
		BuildTime: BuildTime,
		BuildUser: BuildUser,
		BuildHost: BuildHost,
	}
}