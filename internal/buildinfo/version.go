// Package buildinfo provides information about the current build of the application.
// This information is typically injected at build time using linker flags (-X).
//
// Responsibilities:
// - Provide a central location for build-time metadata (Version, Commit, Date).
// - Allow other packages to access version information consistently.
//
// Does NOT handle:
// - Runtime configuration or environment variable management.
// - Persisting or updating version information (it is read-only at runtime).
//
// Invariants/Assumptions:
// - The variables Version, Commit, and Date are expected to be populated by the linker.
// - Default values are provided for development environments where linker flags might be absent.
package buildinfo

var (
	// Version is the current version of the application.
	Version = "1.0.0-rc1"
	// Commit is the git commit hash of the build.
	Commit = "none"
	// Date is the build date in UTC.
	Date = "unknown"
)
