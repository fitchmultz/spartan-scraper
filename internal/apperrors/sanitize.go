// Package apperrors provides utilities for redacting sensitive information from error messages and URLs.
//
// This package defines:
// - RedactString(): Removes obvious secret-looking substrings and filesystem paths
// - SafeMessage(): Returns a redacted, user-facing message for any error
// - SanitizeURL(): Removes query parameters, fragments, and userinfo from URLs
//
// This package is responsible for:
// - Preventing accidental exposure of secrets in logs and user-facing error messages
// - Detecting and redacting common secret patterns (tokens, API keys, passwords)
// - Redacting filesystem paths that may contain user-specific information
//
// This package does NOT handle:
// - Complete secret detection (cannot catch all possible secrets)
// - Context-aware redaction (e.g., custom secrets per tenant)
// - Structured redaction of complex data structures
//
// Invariants:
// - RedactString() returns the original string if no patterns match
// - SafeMessage() returns an empty string for nil errors
// - SanitizeURL() returns the original string unchanged if URL parsing fails
// - All redaction functions are safe to call on empty or malformed input
//
// Supported redaction patterns:
// - Authorization tokens (Bearer, Basic)
// - Key-value secrets (password=, token=, api_key=, etc.)
// - JSON fields ("password":"...", "apiKey":"...", etc.)
// - Filesystem paths (Unix and Windows, including file:// URLs)
package apperrors

import (
	"net/url"
	"regexp"
)

var (
	// e.g. "Bearer abc123", "bearer eyJ...", "Basic Zm9vOmJhcg=="
	authzTokenRe = regexp.MustCompile(`(?i)\b(bearer|basic)\s+([^\s]+)`)

	// e.g. "token=abc", "api_key=abc", "password=abc"
	kvSecretRe = regexp.MustCompile(`(?i)\b(password|passwd|pass|token|api[_-]?key|secret)\s*=\s*([^\s&]+)`)

	// e.g. JSON fragments: "password":"...", "apiKey":"..."
	jsonSecretRe = regexp.MustCompile(`(?i)"(password|passwd|pass|token|apiKey|api_key|secret)"\s*:\s*"[^"]*"`)

	// Filesystem paths that should be redacted
	// Matches: Unix absolute paths (/Users, /home, /var, /tmp, /opt, /usr, /etc)
	//          Windows paths (C:\, D:\, etc.)
	//          file:// URLs
	pathRe = regexp.MustCompile(`(?i)(` +
		`file://[^\s"'<>]+|` + // file:// URLs (must have file: prefix)
		`(?:^|\s)[a-z]:/[^\s"'<>]+|` + // Windows paths with forward slashes (D:/...) at word boundary
		`\b[a-z]:\\[^\s"'<>]+|` + // Windows paths with backslashes (C:\...) - \b ensures word boundary
		`/Users/[^\s"'<>]+|` + // macOS user paths
		`/home/[^\s"'<>]+|` + // Linux home paths
		`/var/[^\s"'<>]+|` + // /var paths
		`/tmp/[^\s"'<>]+|` + // /tmp paths
		`/opt/[^\s"'<>]+|` + // /opt paths
		`/usr/[^\s"'<>]+|` + // /usr paths
		`/etc/[^\s"'<>]+` + // /etc paths
		`)`)
)

// RedactString redacts obvious secret-looking substrings and filesystem paths in s.
func RedactString(s string) string {
	if s == "" {
		return s
	}

	// Replace Authorization-like tokens
	s = authzTokenRe.ReplaceAllString(s, `$1 [REDACTED]`)

	// Replace key=value secrets
	s = kvSecretRe.ReplaceAllString(s, `$1=[REDACTED]`)

	// Replace JSON "key":"secret"
	s = jsonSecretRe.ReplaceAllString(s, `"$1":"[REDACTED]"`)

	// Replace filesystem paths
	s = pathRe.ReplaceAllString(s, `[REDACTED]`)

	return s
}

// SafeMessage returns a redacted, user-facing message for err.
func SafeMessage(err error) string {
	if err == nil {
		return ""
	}
	return RedactString(err.Error())
}

// SanitizeURL removes query parameters, fragments, and userinfo from URLs for safe logging.
// It preserves the scheme, host, and path for debugging purposes while stripping sensitive
// components that may contain tokens, credentials, or session identifiers.
//
// If the URL cannot be parsed, the original string is returned unchanged to avoid
// breaking logging for malformed input.
//
// Examples:
//   - "https://api.example.com/v1/users" → "https://api.example.com/v1/users"
//   - "https://api.example.com?token=secret" → "https://api.example.com"
//   - "https://user:pass@example.com/path" → "https://example.com/path"
//   - "https://example.com/page#section" → "https://example.com/page"
func SanitizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		// Return original on parse error; don't break logging
		return rawURL
	}

	// Strip sensitive components
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil

	return u.String()
}
