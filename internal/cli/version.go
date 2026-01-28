// Package cli implements the command-line interface for Spartan.
// This file specifically handles the 'version' command.
//
// Responsibilities:
// - Print formatted version information to stdout.
// - Include build metadata (Commit, Date) and runtime environment (Go version, OS/Arch).
//
// Does NOT handle:
// - Checking for updates or contacting remote servers.
// - Machine-readable version output (e.g., JSON).
//
// Invariants/Assumptions:
// - Relies on the buildinfo package being populated (either by default or via linker flags).
package cli

import (
	"fmt"
	"runtime"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
)

// RunVersion prints the application version information.
func RunVersion() error {
	fmt.Printf("Spartan version: %s\n", buildinfo.Version)
	fmt.Printf("Commit:          %s\n", buildinfo.Commit)
	fmt.Printf("Build date:      %s\n", buildinfo.Date)
	fmt.Printf("Go version:      %s\n", runtime.Version())
	fmt.Printf("OS/Arch:         %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return nil
}
