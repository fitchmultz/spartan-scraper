package api

import (
	"encoding/json"
	"errors"
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
			writeError(w, tt.err)

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
		writeError(w, apperrors.Validation("test error"))
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
