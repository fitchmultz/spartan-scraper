// Package api provides shared HTTP utility functions for the Spartan Scraper API.
//
// Purpose:
// - Centralize low-level HTTP helpers that are reused across API handlers.
//
// Responsibilities:
// - Decode and encode JSON payloads consistently.
// - Parse common request parameters and normalize response metadata.
// - Provide shared error writing, request ID, and content-type helpers.
//
// Scope:
// - Transport-level API utilities only.
//
// Usage:
// - Imported implicitly across the api package by request handlers and tests.
//
// Invariants/Assumptions:
// - Error responses use the shared request ID envelope.
// - JSON decoding should reject malformed or oversized request bodies consistently.
package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
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

func (rw *requestIDResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *requestIDResponseWriter) WriteHeader(code int) {
	if rw.requestID != "" {
		rw.Header().Set("X-Request-ID", rw.requestID)
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *requestIDResponseWriter) Write(p []byte) (int, error) {
	if rw.requestID != "" {
		rw.Header().Set("X-Request-ID", rw.requestID)
	}
	return rw.ResponseWriter.Write(p)
}

func (rw *requestIDResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (rw *requestIDResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *requestIDResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (rw *requestIDResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if rw.requestID != "" {
		rw.Header().Set("X-Request-ID", rw.requestID)
	}
	if readFrom, ok := rw.ResponseWriter.(io.ReaderFrom); ok {
		return readFrom.ReadFrom(reader)
	}
	return io.Copy(rw.ResponseWriter, reader)
}

func writeJSON(w http.ResponseWriter, payload any) {
	setJSONContentType(w)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

func writeJSONStatus(w http.ResponseWriter, status int, payload any) {
	setJSONContentType(w)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func isJSONContentType(contentType string) bool {
	if strings.TrimSpace(contentType) == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	return decodeJSONBodyWithLimit(w, r, dst, maxRequestBodySize)
}

func decodeJSONBodyWithLimit(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		return apperrors.UnsupportedMediaType("content-type must be application/json")
	}
	if maxBytes <= 0 {
		maxBytes = maxRequestBodySize
	}
	if r.ContentLength > maxBytes {
		return apperrors.RequestEntityTooLarge("request body too large")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return apperrors.Wrap(apperrors.KindRequestEntityTooLarge, "request body too large", err)
		}
		return apperrors.Validation("invalid JSON: " + err.Error())
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return apperrors.Validation("invalid JSON: request body must contain a single JSON value")
		}
		return apperrors.Validation("invalid JSON: " + err.Error())
	}
	return nil
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
// The request parameter is used to extract the request ID from context for logging and response.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
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
	case apperrors.KindRequestEntityTooLarge:
		status = http.StatusRequestEntityTooLarge
	default:
		status = http.StatusInternalServerError
	}

	message := apperrors.SafeMessage(err)
	requestID := contextRequestID(r.Context())

	slog.Error("api error",
		"error_kind", apperrors.KindOf(err),
		"error_message", message,
		"request_id", requestID,
	)

	writeJSONError(w, status, message, requestID)
}

// parseIntParamStrict parses an integer parameter and returns an error for invalid input.
// Unlike parseIntParam, this does NOT silently default - it validates and reports errors.
// Returns validation error if:
//   - The string is non-empty but cannot be parsed as an integer
//   - The parsed value is negative
func parseIntParamStrict(s string, paramName string) (int, error) {
	if s == "" {
		return 0, nil
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, apperrors.Validation(fmt.Sprintf("invalid %s: must be a non-negative integer", paramName))
	}
	if val < 0 {
		return 0, apperrors.Validation(fmt.Sprintf("invalid %s: cannot be negative", paramName))
	}
	return val, nil
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

// validateJobID checks if the provided ID is a valid UUID format.
// Returns apperrors.Validation error if invalid.
func validateJobID(id string) error {
	if id == "" {
		return apperrors.Validation("id required")
	}
	// Validate UUID format (with or without hyphens)
	if _, err := uuid.Parse(id); err != nil {
		return apperrors.Validation(fmt.Sprintf("invalid job id format: %s", id))
	}
	return nil
}

// recoveryMiddleware recovers from panics in handlers and returns a 500 response.
// It logs the panic with stack trace and includes the request ID in logs.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				reqID := contextRequestID(r.Context())
				slog.Error("panic recovered",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprintf("%v", rec),
					"stack", string(stack),
					"request_id", reqID,
				)
				writeError(w, r, apperrors.Internal("internal server error"))
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
			"query", apperrors.RedactString(r.URL.RawQuery),
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

func (lrw *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return lrw.ResponseWriter
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(p []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	return lrw.ResponseWriter.Write(p)
}

func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := lrw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (lrw *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := lrw.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (lrw *loggingResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	if readFrom, ok := lrw.ResponseWriter.(io.ReaderFrom); ok {
		return readFrom.ReadFrom(reader)
	}
	return io.Copy(lrw.ResponseWriter, reader)
}
