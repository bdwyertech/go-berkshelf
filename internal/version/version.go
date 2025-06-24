package version

import (
	"fmt"
	"runtime"
)

// Version information set at build time
var (
	// Version is the current version of go-berkshelf
	Version = "0.1.0-dev"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the date of the build
	BuildDate = "unknown"
)

// BuildInfo contains build information
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildDate string
	GoVersion string
	Platform  string
}

// GetBuildInfo returns the current build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (b BuildInfo) String() string {
	return fmt.Sprintf("go-berkshelf version %s (commit: %s, built: %s, go: %s, platform: %s)",
		b.Version, b.GitCommit, b.BuildDate, b.GoVersion, b.Platform)
}
