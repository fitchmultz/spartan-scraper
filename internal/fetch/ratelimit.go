// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitInfo holds parsed rate limit data from various header formats.
// It represents the server's rate limit policy and current state.
type RateLimitInfo struct {
	Limit     int           // Maximum requests allowed in the window
	Remaining int           // Requests remaining in current window
	Reset     time.Time     // When the rate limit window resets
	Window    time.Duration // Optional: window duration if known
}

// ParseRateLimitHeader parses the RFC 9440 RateLimit header.
// Format: RateLimit: limit=100, remaining=50, reset=60
// The reset value can be:
//   - Delta-seconds (integer): seconds until reset
//   - Unix timestamp (integer >= 1e9): absolute reset time
//
// See: https://datatracker.ietf.org/doc/html/rfc9440
func ParseRateLimitHeader(header string) (RateLimitInfo, error) {
	var info RateLimitInfo
	if header == "" {
		return info, nil
	}

	pairs := parseHeaderPairs(header)

	if limitStr, ok := pairs["limit"]; ok {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			info.Limit = limit
		}
	}

	if remainingStr, ok := pairs["remaining"]; ok {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			info.Remaining = remaining
		}
	}

	if resetStr, ok := pairs["reset"]; ok {
		info.Reset = parseResetValue(resetStr)
	}

	return info, nil
}

// ParseXRateLimitHeaders parses common X-RateLimit-* header variants.
// Supports GitHub, Twitter, and other common API patterns:
//   - X-RateLimit-Limit: maximum requests allowed
//   - X-RateLimit-Remaining: requests remaining in current window
//   - X-RateLimit-Reset: reset time (Unix timestamp or HTTP date)
//
// Some APIs use different prefixes (e.g., x-ratelimit-* lowercase).
// This function checks both canonical and lowercase forms.
func ParseXRateLimitHeaders(headers http.Header) (RateLimitInfo, error) {
	var info RateLimitInfo

	// Try canonical form first, then lowercase variants
	limitStr := getHeaderValue(headers, "X-RateLimit-Limit", "x-ratelimit-limit", "x-rate-limit-limit")
	remainingStr := getHeaderValue(headers, "X-RateLimit-Remaining", "x-ratelimit-remaining", "x-rate-limit-remaining")
	resetStr := getHeaderValue(headers, "X-RateLimit-Reset", "x-ratelimit-reset", "x-rate-limit-reset")

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			info.Limit = limit
		}
	}

	if remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			info.Remaining = remaining
		}
	}

	if resetStr != "" {
		info.Reset = parseResetValue(resetStr)
	}

	return info, nil
}

// ParseRateLimitPolicyHeader parses the RateLimit-Policy header (RFC 9440).
// Format: RateLimit-Policy: 100;w=60
// Where 100 is the limit and w=60 specifies a 60-second window.
// This can provide window duration even when RateLimit header is not present.
func ParseRateLimitPolicyHeader(header string) (limit int, window time.Duration) {
	if header == "" {
		return 0, 0
	}

	// Parse format: "100;w=60" or "100"
	parts := strings.Split(header, ";")
	if len(parts) == 0 {
		return 0, 0
	}

	// Parse limit (first part)
	limitStr := strings.TrimSpace(parts[0])
	if limitParsed, err := strconv.Atoi(limitStr); err == nil && limitParsed > 0 {
		limit = limitParsed
	}

	// Parse window parameter if present
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "w=") {
			windowStr := strings.TrimPrefix(part, "w=")
			if windowSeconds, err := strconv.Atoi(windowStr); err == nil && windowSeconds > 0 {
				window = time.Duration(windowSeconds) * time.Second
			}
			break
		}
	}

	return limit, window
}

// ExtractRateLimitInfo tries all known header formats and returns the best available data.
// Priority:
// 1. RFC 9440 RateLimit header (preferred standard)
// 2. X-RateLimit-* headers (common API patterns)
// 3. RateLimit-Policy header (for window info only)
//
// Returns (info, true) if any rate limit headers were found, (empty, false) otherwise.
func ExtractRateLimitInfo(headers http.Header) (RateLimitInfo, bool) {
	var info RateLimitInfo
	hasData := false

	// Try RFC 9440 RateLimit header first (preferred)
	rateLimitHeader := headers.Get("RateLimit")
	if rateLimitHeader != "" {
		if parsed, err := ParseRateLimitHeader(rateLimitHeader); err == nil {
			// Consider it valid if any field is populated
			if parsed.Limit > 0 || parsed.Remaining > 0 || !parsed.Reset.IsZero() {
				info = parsed
				hasData = true
			}
		}
	}

	// Try X-RateLimit-* headers if no RFC 9440 data or to supplement it
	xRateInfo, _ := ParseXRateLimitHeaders(headers)
	if xRateInfo.Limit > 0 || xRateInfo.Remaining > 0 || !xRateInfo.Reset.IsZero() {
		if !hasData {
			info = xRateInfo
			hasData = true
		} else {
			// Merge: prefer RFC 9440, but fill in gaps from X-RateLimit
			if info.Limit == 0 {
				info.Limit = xRateInfo.Limit
			}
			if info.Remaining == 0 {
				info.Remaining = xRateInfo.Remaining
			}
			if info.Reset.IsZero() {
				info.Reset = xRateInfo.Reset
			}
		}
	}

	// Try RateLimit-Policy for window information
	policyHeader := headers.Get("RateLimit-Policy")
	if policyHeader == "" {
		// Try common variants
		policyHeader = headers.Get("X-RateLimit-Policy")
	}
	if policyHeader != "" {
		policyLimit, window := ParseRateLimitPolicyHeader(policyHeader)
		if policyLimit > 0 && info.Limit == 0 {
			info.Limit = policyLimit
			hasData = true
		}
		if window > 0 && info.Window == 0 {
			info.Window = window
		}
	}

	return info, hasData
}

// IsRateLimited returns true if the rate limit has been exceeded (Remaining <= 0).
// Returns false if no rate limit information is available.
func (r *RateLimitInfo) IsRateLimited() bool {
	if r == nil {
		return false
	}
	return r.Remaining <= 0 && r.Limit > 0
}

// TimeUntilReset returns the duration until the rate limit resets.
// Returns 0 if reset time is not set or has already passed.
func (r *RateLimitInfo) TimeUntilReset() time.Duration {
	if r == nil || r.Reset.IsZero() {
		return 0
	}
	until := time.Until(r.Reset)
	if until < 0 {
		return 0
	}
	return until
}

// UsagePercent returns the percentage of rate limit used (0-100).
// Returns -1 if limit information is not available.
func (r *RateLimitInfo) UsagePercent() float64 {
	if r == nil || r.Limit <= 0 {
		return -1
	}
	used := r.Limit - r.Remaining
	if used < 0 {
		used = 0
	}
	return float64(used) / float64(r.Limit) * 100
}

// parseHeaderPairs parses a header value in the format "key=value, key2=value2".
// Handles quoted values and unquoted values.
func parseHeaderPairs(header string) map[string]string {
	result := make(map[string]string)
	if header == "" {
		return result
	}

	// Split by comma, but be careful with quoted strings
	var parts []string
	var current strings.Builder
	inQuotes := false

	for _, r := range header {
		switch r {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case ',':
			if inQuotes {
				current.WriteRune(r)
			} else {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(strings.ToLower(kv[0]))
			value := strings.TrimSpace(kv[1])
			// Remove quotes if present
			value = strings.Trim(value, `"`)
			result[key] = value
		}
	}

	return result
}

// parseResetValue parses a reset value which can be:
//   - Delta-seconds (integer < 1e9): seconds until reset
//   - Unix timestamp (integer >= 1e9): absolute reset time
//   - HTTP date: absolute reset time
func parseResetValue(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	// Try parsing as integer first
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		// Heuristic: if value is very large (>= 1e9), treat as Unix timestamp
		// Otherwise treat as delta seconds
		if seconds >= 1e9 {
			// Unix timestamp (seconds since epoch)
			return time.Unix(seconds, 0)
		}
		// Delta seconds
		return time.Now().Add(time.Duration(seconds) * time.Second)
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(value); err == nil {
		return t
	}

	return time.Time{}
}

// getHeaderValue returns the first non-empty header value from the given keys.
// Checks keys in order and returns the first match.
func getHeaderValue(headers http.Header, keys ...string) string {
	for _, key := range keys {
		if value := headers.Get(key); value != "" {
			return value
		}
	}
	return ""
}
