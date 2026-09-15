// Package version contains build-time identity for gitwatch.
package version

import (
	"regexp"
	"runtime/debug"
)

var (
	// Version is the release or development version embedded at build time.
	Version   = "1.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var (
	taggedModuleVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	pseudoModuleVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-(?:0\.)?[0-9]{14}-[0-9a-f]+$`)
)

// resolvedVersion returns the linker-provided value when one is present. A
// source install of a tagged module cannot receive the release linker flags,
// but Go records that module version in build information; use it so
// `go install github.com/sphireinc/git-watch/cmd/gitwatch@vX.Y.Z` reports the
// version it actually installed.
func resolvedVersion(linked, module string) string {
	if linked != "1.0.0-dev" {
		return linked
	}
	if taggedModuleVersion.MatchString(module) && !pseudoModuleVersion.MatchString(module) {
		return module[1:]
	}
	return linked
}

func currentVersion() string {
	module := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		module = info.Main.Version
	}
	return resolvedVersion(Version, module)
}

// String returns the complete human-readable build identity.
func String() string { return currentVersion() + " (" + Commit + ", " + BuildDate + ")" }
