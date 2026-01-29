package apperrors

// Package apperrors provides centralized sentinel error definitions for stable error comparison.
//
// This package defines named error variables that serve as:
// - Stable error text for use with errors.Is() and errors.As()
// - Canonical error types migrated from other packages (validate, fetch, auth, jobs)
// - Pre-classified sentinel errors that can be wrapped with WithKind()
//
// This package is responsible for:
// - Centralizing sentinel errors to avoid duplication across the codebase
// - Providing stable error identities for testing and error handling
// - Maintaining consistent error text for common failure scenarios
//
// This package does NOT handle:
// - Creating new error instances dynamically (use New(), Wrap(), or the Validation()/NotFound() helpers)
// - Error classification (use WithKind() to classify sentinel errors)
// - Error wrapping or context addition (use Wrap() or errors.Wrap())
//
// Invariants:
// - All sentinel errors are defined as package-level variables using errors.New()
// - Sentinel error text is stable and should not change without justification
// - Sentinel errors should be compared using errors.Is(), not string equality
// - Use WithKind() to classify sentinel errors with a specific Kind if needed
//
// Sentinel error categories:
// - Validation errors: URL scheme, host, timeout, max depth, max pages, profile name
// - Fetch/browser errors: Chrome not found, Playwright not ready
// - Auth/vault errors: Invalid path
// - Jobs/queue errors: Queue full

import "errors"

var (
	// Validation-related sentinel errors (migrated from internal/validate).
	ErrInvalidURLScheme    = errors.New("invalid url: must be http or https and have a host")
	ErrInvalidURLHost      = errors.New("invalid url: must have a host")
	ErrInvalidTimeoutRange = errors.New("timeoutSeconds must be between 5 and 300")
	ErrInvalidMaxDepth     = errors.New("maxDepth must be between 1 and 10")
	ErrInvalidMaxPages     = errors.New("maxPages must be between 1 and 10000")
	ErrInvalidProfileName  = errors.New("invalid authProfile: only alphanumeric, hyphens, and underscores allowed")

	// Fetch/browser sentinel errors (migrated from internal/fetch).
	ErrChromeNotFound     = errors.New("chrome/chromium not found on PATH")
	ErrPlaywrightNotReady = errors.New("playwright drivers not installed or not found")

	// Auth/vault sentinel errors (migrated from internal/auth).
	ErrInvalidPath = errors.New("invalid path")

	// Jobs/queue sentinel errors (centralized).
	ErrQueueFull = errors.New("job queue full")
)
