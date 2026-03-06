// Package webassets contains repository-level checks for the static web shell.
//
// Purpose:
//   - Keep lightweight validation close to the versioned frontend entry assets.
//
// Responsibilities:
//   - Provide a stable home for tests that verify `web/index.html` and tracked
//     public assets without depending on browser tooling.
//
// Scope:
//   - Static asset validation only; it does not serve or build the web app.
//
// Usage:
//   - Runs through `go test ./...` as part of the standard CI gate.
//
// Invariants/Assumptions:
//   - The repository stores the Vite shell at `web/index.html`.
//   - Public static assets live under `web/public/`.
package webassets
