// Package api implements the REST API server for Spartan Scraper.
// It provides endpoints for enqueuing jobs, managing auth profiles,
// and retrieving job status and results.
package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// apiKeyContextKey is the context key for storing API key info
type apiKeyContextKey struct{}

// apiKeyAuthMiddleware creates middleware that validates X-API-Key header
// when API authentication is enabled.
// - Skips validation for localhost when API_AUTH_ENABLED is not explicitly true
// - Returns 403 if key is missing
// - Returns 403 if key is invalid or expired
// - Returns 403 if key has read-only permission but request is not GET/HEAD/OPTIONS
func apiKeyAuthMiddleware(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for localhost if not explicitly enabled
		if !cfg.APIAuthEnabled && isLocalhost(cfg.BindAddr) {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health check endpoint
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key from header
		key := r.Header.Get("X-API-Key")
		if key == "" {
			writeError(w, r, apperrors.Permission("missing X-API-Key header"))
			return
		}

		// Validate key
		apiKey, err := auth.ValidateAPIKey(cfg.DataDir, key)
		if err != nil {
			writeError(w, r, apperrors.Permission("invalid API key"))
			return
		}

		// Check permissions for write operations
		if apiKey.Permissions == auth.APIKeyPermissionReadOnly {
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				writeError(w, r, apperrors.Permission("read-only API key cannot perform write operations"))
				return
			}
		}

		// Store key info in context for potential logging/auditing
		ctx := context.WithValue(r.Context(), apiKeyContextKey{}, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isLocalhost checks if the address is a localhost address
func isLocalhost(addr string) bool {
	return addr == "127.0.0.1" || addr == "localhost" || addr == "::1" || strings.HasPrefix(addr, "127.")
}

// GetAPIKeyFromContext retrieves the API key from the request context
// Returns the APIKey and true if found, zero value and false otherwise
func GetAPIKeyFromContext(ctx context.Context) (auth.APIKey, bool) {
	if key, ok := ctx.Value(apiKeyContextKey{}).(auth.APIKey); ok {
		return key, true
	}
	return auth.APIKey{}, false
}
