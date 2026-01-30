// Package api provides unit tests for error handling utilities.
// Tests cover writeError helper for correct HTTP status codes and error classification.
// Does NOT test API handler error responses (covered in other test files).
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Validation",
			err:            apperrors.Validation("invalid input"),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid input",
		},
		{
			name:           "NotFound",
			err:            apperrors.NotFound("not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
		{
			name:           "Permission",
			err:            apperrors.Permission("denied"),
			expectedStatus: http.StatusForbidden,
			expectedBody:   "denied",
		},
		{
			name:           "MethodNotAllowed",
			err:            apperrors.MethodNotAllowed("bad method"),
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "bad method",
		},
		{
			name:           "UnsupportedMediaType",
			err:            apperrors.UnsupportedMediaType("bad media"),
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedBody:   "bad media",
		},
		{
			name:           "Internal",
			err:            apperrors.Internal("something failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "something failed",
		},
		{
			name:           "GenericError",
			err:            errors.New("generic failure"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "generic failure",
		},
		{
			name:           "RedactionPath",
			err:            apperrors.Validation("failed at /Users/mitch/secret"),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed at [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			writeError(w, r, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Error != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, resp.Error)
			}

			if resp.RequestID != "" {
				t.Errorf("expected empty RequestID, got %q", resp.RequestID)
			}
		})
	}
}

func TestWriteErrorWithRequestID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, apperrors.Validation("test error"))
	})

	middleware := requestIDMiddleware(handler)

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("X-Request-ID", "test-req-id-123")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "test error" {
		t.Errorf("expected error message %q, got %q", "test error", resp.Error)
	}

	if resp.RequestID != "test-req-id-123" {
		t.Errorf("expected RequestID %q, got %q", "test-req-id-123", resp.RequestID)
	}

	if rr.Header().Get("X-Request-ID") != "test-req-id-123" {
		t.Errorf("expected X-Request-ID header %q, got %q", "test-req-id-123", rr.Header().Get("X-Request-ID"))
	}
}

func TestWriteErrorLogging(t *testing.T) {
	var logBuf bytes.Buffer
	handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError})
	logger := slog.New(handler)

	oldDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldDefault)

	tests := []struct {
		name         string
		err          error
		expectedKind string
		expectedMsg  string
	}{
		{
			name:         "Validation error",
			err:          apperrors.Validation("invalid input"),
			expectedKind: string(apperrors.KindValidation),
			expectedMsg:  "invalid input",
		},
		{
			name:         "Error with redacted secrets",
			err:          apperrors.Validation("failed with token=secret123"),
			expectedKind: string(apperrors.KindValidation),
			expectedMsg:  "failed with token=[REDACTED]",
		},
		{
			name:         "NotFound error",
			err:          apperrors.NotFound("resource not found"),
			expectedKind: string(apperrors.KindNotFound),
			expectedMsg:  "resource not found",
		},
		{
			name:         "Permission error",
			err:          apperrors.Permission("access denied"),
			expectedKind: string(apperrors.KindPermission),
			expectedMsg:  "access denied",
		},
		{
			name:         "MethodNotAllowed error",
			err:          apperrors.MethodNotAllowed("bad method"),
			expectedKind: string(apperrors.KindMethodNotAllowed),
			expectedMsg:  "bad method",
		},
		{
			name:         "UnsupportedMediaType error",
			err:          apperrors.UnsupportedMediaType("bad media"),
			expectedKind: string(apperrors.KindUnsupportedMediaType),
			expectedMsg:  "bad media",
		},
		{
			name:         "Internal error",
			err:          apperrors.Internal("something failed"),
			expectedKind: string(apperrors.KindInternal),
			expectedMsg:  "something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			writeError(w, r, tt.err)

			logOutput := logBuf.String()
			if logOutput == "" {
				t.Error("expected log output, got empty string")
			}

			if !strings.Contains(logOutput, "error_kind="+tt.expectedKind) {
				t.Errorf("log missing error_kind=%s, got: %s", tt.expectedKind, logOutput)
			}
			if !strings.Contains(logOutput, `error_message="`+tt.expectedMsg+`"`) {
				t.Errorf("log missing error_message=%s, got: %s", tt.expectedMsg, logOutput)
			}
			if !strings.Contains(logOutput, "request_id=") {
				t.Error("log missing request_id field")
			}
			if !strings.Contains(logOutput, "api error") {
				t.Error("log missing 'api error' message")
			}
		})
	}

	t.Run("Nil error should not log", func(t *testing.T) {
		logBuf.Reset()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		writeError(w, r, nil)

		logOutput := logBuf.String()
		if logOutput != "" {
			t.Errorf("expected no log output for nil error, got: %s", logOutput)
		}
	})
}
