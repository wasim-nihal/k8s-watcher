package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// GetVersion returns the formatted version string
func GetVersion() string {
	return fmt.Sprintf("%s (commit: %s, date: %s, go: %s)", Version, Commit, Date, runtime.Version())
}
