package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		})
	}
}
