// Package model provides job sanitization utilities for safe API/MCP responses.
//
// This file handles redaction of sensitive data from Job objects before they
// are serialized to untrusted outputs (HTTP API responses, MCP tool results).
//
// It does NOT handle:
// - Job persistence or storage operations
// - In-place mutation of stored Job objects
// - Business logic for job execution
//
// Invariants:
// - SanitizeJob returns a deep copy; original Job is never modified
// - All sensitive keys in Params are recursively redacted
// - ResultPath is always cleared (never exposed to clients)
// - Error strings are passed through apperrors.RedactString
package model

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// redactedPlaceholder is the string used to replace sensitive values.
const redactedPlaceholder = "[REDACTED]"

// sensitiveParamKeysExact contains parameter keys that should be redacted
// using exact matching (case-insensitive).
var sensitiveParamKeysExact = []string{
	"auth",
	"login",
	"headers",
	"cookies",
	"token",
	"tokens",
	"password",
	"secret",
	"credential",
	"authorization",
	"proxy",
}

// sensitiveParamKeysSuffix contains key suffixes that indicate sensitive data.
// Keys ending with these (case-insensitive) will be redacted.
var sensitiveParamKeysSuffix = []string{
	"password",
	"passwd",
	"pass",
	"token",
	"tokens",
	"secret",
	"apikey",
	"api_key",
	"key", // Must be checked last as it's most generic
}

// sensitiveParamKeysPrefix contains key prefixes that indicate sensitive data.
// Keys starting with these followed by an uppercase letter or separator will be redacted.
// These are more specific prefixes that clearly indicate sensitive data.
var sensitiveParamKeysPrefix = []string{
	"token",
}

// sensitiveHeaderValues contains header names whose values should be redacted.
// These are matched case-insensitively.
var sensitiveHeaderValues = []string{
	"authorization",
	"cookie",
	"set-cookie",
	"x-api-key",
	"x-auth-token",
	"proxy-authorization",
	"x-access-token",
	"x-token",
}

// SanitizeJob returns a redacted copy of the job safe for API/MCP responses.
// The original job is not modified.
func SanitizeJob(job Job) Job {
	// Create a copy with ResultPath cleared
	safe := Job{
		ID:        job.ID,
		Kind:      job.Kind,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
		// ResultPath is intentionally omitted - never expose filesystem paths
		ResultPath: "",
		// Error is sanitized to remove secrets and paths
		Error: apperrors.RedactString(job.Error),
	}

	// Deep copy and sanitize Params
	if job.Params != nil {
		safe.Params = sanitizeParams(job.Params)
	}

	return safe
}

// SanitizeJobs maps a slice of jobs through SanitizeJob.
func SanitizeJobs(jobs []Job) []Job {
	if jobs == nil {
		return nil
	}
	result := make([]Job, len(jobs))
	for i, job := range jobs {
		result[i] = SanitizeJob(job)
	}
	return result
}

// sanitizeParams creates a deep copy of params with sensitive values redacted.
func sanitizeParams(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}

	result := make(map[string]interface{}, len(params))
	for key, value := range params {
		// Special handling for headers - don't redact the whole thing
		if strings.EqualFold(key, "headers") {
			result[key] = sanitizeHeaders(value)
		} else if isSensitiveKey(key) {
			result[key] = redactedPlaceholder
		} else {
			result[key] = sanitizeAny(value)
		}
	}
	return result
}

// sanitizeAny recursively sanitizes arbitrary values.
func sanitizeAny(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		return sanitizeParams(val)
	case []interface{}:
		return sanitizeSlice(val)
	case string:
		// Sanitize strings that might contain paths using apperrors
		return apperrors.RedactString(val)
	case map[string]string:
		return sanitizeStringMap(val)
	case []string:
		return sanitizeStringSlice(val)
	default:
		return v
	}
}

// sanitizeStringMap sanitizes map[string]string values.
func sanitizeStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = apperrors.RedactString(v)
	}
	return result
}

// sanitizeStringSlice sanitizes []string values.
func sanitizeStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}
	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = apperrors.RedactString(v)
	}
	return result
}

// sanitizeSlice sanitizes each element in a slice.
func sanitizeSlice(slice []interface{}) []interface{} {
	if slice == nil {
		return nil
	}
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = sanitizeAny(v)
	}
	return result
}

// sanitizeHeaders handles header maps specially, preserving header names
// but redacting sensitive header values.
func sanitizeHeaders(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for headerName, headerValue := range val {
			if isSensitiveHeader(headerName) {
				result[headerName] = redactedPlaceholder
			} else if strVal, ok := headerValue.(string); ok {
				result[headerName] = apperrors.RedactString(strVal)
			} else {
				result[headerName] = headerValue
			}
		}
		return result
	case map[string]string:
		result := make(map[string]string, len(val))
		for headerName, headerValue := range val {
			if isSensitiveHeader(headerName) {
				result[headerName] = redactedPlaceholder
			} else {
				result[headerName] = apperrors.RedactString(headerValue)
			}
		}
		return result
	case []interface{}:
		// Handle array of headers (e.g., ["Authorization: Bearer token"])
		result := make([]interface{}, len(val))
		for i, item := range val {
			if str, ok := item.(string); ok {
				result[i] = sanitizeHeaderString(str)
			} else {
				result[i] = item
			}
		}
		return result
	case []string:
		// Handle []string headers
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = sanitizeHeaderString(item)
		}
		return result
	default:
		return v
	}
}

// sanitizeHeaderString redacts sensitive values from header strings like "Authorization: Bearer token".
func sanitizeHeaderString(header string) string {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return header
	}

	headerName := strings.TrimSpace(parts[0])
	if isSensitiveHeader(headerName) {
		return headerName + ": " + redactedPlaceholder
	}
	return headerName + ": " + apperrors.RedactString(strings.TrimSpace(parts[1]))
}

// isSensitiveKey checks if a parameter key should be redacted.
// Uses exact matching for broad terms and suffix/prefix matching for compound keys.
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)

	// Check exact matches for broad terms
	for _, sensitive := range sensitiveParamKeysExact {
		if lowerKey == sensitive {
			return true
		}
	}

	// Check prefix matches with separator (auth_, auth-, auth. or AuthX)
	for _, prefix := range sensitiveParamKeysPrefix {
		if strings.HasPrefix(lowerKey, prefix) {
			// Must be followed by separator or uppercase (for camelCase)
			suffix := lowerKey[len(prefix):]
			if len(suffix) > 0 {
				firstChar := suffix[0]
				if firstChar == '_' || firstChar == '-' || firstChar == '.' || (key[len(prefix)] >= 'A' && key[len(prefix)] <= 'Z') {
					return true
				}
			}
		}
	}

	// Check suffix matches (key ends with sensitive suffix)
	// Split on separators to get the last token
	tokens := strings.FieldsFunc(lowerKey, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	if len(tokens) > 0 {
		lastToken := tokens[len(tokens)-1]
		for _, suffix := range sensitiveParamKeysSuffix {
			if lastToken == suffix {
				return true
			}
		}
	}

	// Check camelCase suffixes (e.g., userPassword, myApiKey)
	// by looking for the suffix preceded by lowercase and at end
	for _, suffix := range sensitiveParamKeysSuffix {
		if idx := strings.Index(lowerKey, suffix); idx != -1 {
			suffixEnd := idx + len(suffix)
			// Must be at the end of the string
			if suffixEnd == len(lowerKey) && idx > 0 {
				// Check if preceded by lowercase letter (camelCase)
				prevChar := key[idx-1]
				if prevChar >= 'a' && prevChar <= 'z' {
					return true
				}
			}
		}
	}

	return false
}

// isSensitiveHeader checks if a header name has a sensitive value.
func isSensitiveHeader(name string) bool {
	lowerName := strings.ToLower(name)
	for _, sensitive := range sensitiveHeaderValues {
		if lowerName == sensitive {
			return true
		}
	}
	return false
}
