// Package version contains build-time identity for gitwatch.
package version

var (
	// Version is the release or development version embedded at build time.
	Version   = "1.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns the complete human-readable build identity.
func String() string { return Version + " (" + Commit + ", " + BuildDate + ")" }
