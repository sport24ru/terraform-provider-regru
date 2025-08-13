package version

import (
	"fmt"
	"time"
)

var (
	// Version is the version of the provider
	Version = "dev"
	// Commit is the git commit hash
	Commit = "unknown"
	// Date is the build date
	Date = "unknown"
)

// String returns the version string
func String() string {
	return fmt.Sprintf("v%s", Version)
}

// Full returns the full version information
func Full() string {
	return fmt.Sprintf("v%s (%s, built %s)", Version, Commit, Date)
}

// BuildDate returns the build date as a time.Time
func BuildDate() time.Time {
	if Date == "unknown" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, Date)
	return t
}
