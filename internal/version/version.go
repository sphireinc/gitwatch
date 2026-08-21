// Package version contains build-time identity for gitwatch.
package version

var (
	Version   = "1.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string { return Version + " (" + Commit + ", " + BuildDate + ")" }
