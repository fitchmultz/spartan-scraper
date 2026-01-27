// Package api provides utility functions for the Spartan Scraper HTTP API.
// These include JSON encoding helpers, parameter parsing, auth resolution,
// and content type detection.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"spartan-scraper/internal/apperrors"
	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/fetch"
)

const maxRequestBodySize = 1024 * 1024

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	redacted := apperrors.RedactString(message)
	errResp := ErrorResponse{Error: redacted}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("failed to encode json error response", "error", err)
	}
}

// writeError maps error kinds to appropriate HTTP status codes and writes the response.
// Error messages are redacted using SafeMessage to prevent secret leakage.
func writeError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	var status int
	switch apperrors.KindOf(err) {
	case apperrors.KindValidation:
		status = http.StatusBadRequest
	case apperrors.KindNotFound:
		status = http.StatusNotFound
	case apperrors.KindPermission:
		status = http.StatusForbidden
	default:
		status = http.StatusInternalServerError
	}

	message := apperrors.SafeMessage(err)
	writeJSONError(w, status, message)
}

func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil || val < 0 {
		return defaultVal
	}
	return val
}

func resolveAuthForRequest(cfg config.Config, url string, profile string, override *fetch.AuthOptions) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	if override != nil {
		input.Headers = toHeaderKVs(override.Headers)
		input.Cookies = toCookies(override.Cookies)
		input.Tokens = tokensFromOverride(*override)
		if login := loginFromOverride(*override); login != nil {
			input.Login = login
		}
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}

func toHeaderKVs(headers map[string]string) []auth.HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]auth.HeaderKV, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, auth.HeaderKV{Key: key, Value: value})
	}
	return out
}

func toCookies(cookies []string) []auth.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]auth.Cookie, 0, len(cookies))
	for _, raw := range cookies {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		out = append(out, auth.Cookie{Name: name, Value: value})
	}
	return out
}

func tokensFromOverride(override fetch.AuthOptions) []auth.Token {
	out := []auth.Token{}
	if override.Basic != "" {
		out = append(out, auth.Token{Kind: auth.TokenBasic, Value: override.Basic})
	}
	for key, value := range override.Query {
		out = append(out, auth.Token{Kind: auth.TokenApiKey, Value: value, Query: key})
	}
	return out
}

func loginFromOverride(override fetch.AuthOptions) *auth.LoginFlow {
	if override.LoginURL == "" && override.LoginUserSelector == "" && override.LoginPassSelector == "" && override.LoginSubmitSelector == "" && override.LoginUser == "" && override.LoginPass == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            override.LoginURL,
		UserSelector:   override.LoginUserSelector,
		PassSelector:   override.LoginPassSelector,
		SubmitSelector: override.LoginSubmitSelector,
		Username:       override.LoginUser,
		Password:       override.LoginPass,
	}
}

func contentTypeForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".jsonl":
		return "application/x-ndjson"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return ""
	}
}
