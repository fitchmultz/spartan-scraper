// Package api provides utility functions for the Spartan Scraper HTTP API.
// These include JSON encoding helpers, parameter parsing, auth resolution,
// and content type detection.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/google/uuid"
)

const maxRequestBodySize = 1024 * 1024

var requestIDKey = &struct{}{}

func getRequestID(r *http.Request) string {
	if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
		return reqID
	}
	return uuid.New().String()
}

func contextRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

type requestIDResponseWriter struct {
	http.ResponseWriter
	requestID string
}

func (rw *requestIDResponseWriter) WriteHeader(code int) {
	if rw.requestID != "" {
		rw.Header().Set("X-Request-ID", rw.requestID)
	}
	rw.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errResp := ErrorResponse{Error: message, RequestID: requestID}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("failed to encode json error response", "error", err)
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := getRequestID(r)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		rw := &requestIDResponseWriter{ResponseWriter: w, requestID: reqID}
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
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
	case apperrors.KindMethodNotAllowed:
		status = http.StatusMethodNotAllowed
	case apperrors.KindUnsupportedMediaType:
		status = http.StatusUnsupportedMediaType
	default:
		status = http.StatusInternalServerError
	}

	message := apperrors.SafeMessage(err)
	var requestID string
	if rw, ok := w.(*requestIDResponseWriter); ok {
		requestID = rw.requestID
	}
	writeJSONError(w, status, message, requestID)
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

// extractID extracts the ID from a URL path given the resource name.
// It splits the path by "/" and returns the segment immediately following the resource name.
// If the segment is empty or does not exist, it returns an empty string.
func extractID(path, resource string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			id := parts[i+1]
			if id != "" {
				return id
			}
		}
	}
	return ""
}

// recoveryMiddleware recovers from panics in handlers and returns a 500 response.
// It logs the panic with stack trace.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				slog.Error("panic recovered",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprintf("%v", rec),
					"stack", string(stack),
				)
				writeError(w, apperrors.Internal("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs incoming requests and responses with duration and status code.
// It uses structured logging with slog for consistent log format.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		reqID := contextRequestID(r.Context())

		slog.Info("request started",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
			"request_id", reqID,
		)

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
		)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture the status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
