package apperrors

import "regexp"

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
