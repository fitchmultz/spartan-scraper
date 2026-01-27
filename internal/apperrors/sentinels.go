package apperrors

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
