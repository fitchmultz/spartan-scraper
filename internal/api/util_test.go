// Package api provides unit tests for utility functions.
// Tests contentTypeForExtension helper function and middleware.
// Does NOT test API handlers or integration behavior.
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestContentTypeForExtension(t *testing.T) {
	tests := []struct {
		name         string
		ext          string
		expectedType string
	}{
		{name: "jsonl", ext: ".jsonl", expectedType: "application/x-ndjson"},
		{name: "JSONL uppercase", ext: ".JSONL", expectedType: "application/x-ndjson"},
		{name: "json", ext: ".json", expectedType: "application/json"},
		{name: "JSON uppercase", ext: ".JSON", expectedType: "application/json"},
		{name: "csv", ext: ".csv", expectedType: "text/csv"},
		{name: "xml", ext: ".xml", expectedType: "application/xml"},
		{name: "txt", ext: ".txt", expectedType: "text/plain; charset=utf-8"},
		{name: "unknown extension", ext: ".unknown", expectedType: ""},
		{name: "no extension", ext: "", expectedType: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentTypeForExtension(tt.ext)
			if result != tt.expectedType {
				t.Errorf("contentTypeForExtension(%q) = %q, want %q", tt.ext, result, tt.expectedType)
			}
		})
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		resource string
		expected string
	}{
		{name: "simple job id", path: "/v1/jobs/abc", resource: "jobs", expected: "abc"},
		{name: "job id with trailing slash", path: "/v1/jobs/abc/", resource: "jobs", expected: "abc"},
		{name: "job id with results suffix", path: "/v1/jobs/abc/results", resource: "jobs", expected: "abc"},
		{name: "job id with results and trailing slash", path: "/v1/jobs/abc/results/", resource: "jobs", expected: "abc"},
		{name: "auth profile profile name", path: "/v1/auth/profiles/my-profile", resource: "profiles", expected: "my-profile"},
		{name: "auth profile with trailing slash", path: "/v1/auth/profiles/my-profile/", resource: "profiles", expected: "my-profile"},
		{name: "empty id", path: "/v1/jobs/", resource: "jobs", expected: ""},
		{name: "missing id", path: "/v1/jobs", resource: "jobs", expected: ""},
		{name: "unknown resource", path: "/unknown/abc", resource: "jobs", expected: ""},
		{name: "nested resource", path: "/v1/orgs/o1/jobs/j1", resource: "orgs", expected: "o1"},
		{name: "nested resource 2", path: "/v1/orgs/o1/jobs/j1", resource: "jobs", expected: "j1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractID(tt.path, tt.resource)
			if result != tt.expected {
				t.Errorf("extractID(%q, %q) = %q, want %q", tt.path, tt.resource, result, tt.expected)
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := recoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	assertNotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", status)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "\"error\"") {
		t.Errorf("expected error response, got: %s", body)
	}
}

func TestRecoveryMiddlewareNormalHandler(t *testing.T) {
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := recoveryMiddleware(normalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	if body := rr.Body.String(); body != "ok" {
		t.Errorf("expected body 'ok', got: %s", body)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	handler := loggingMiddleware(normalHandler)

	req := httptest.NewRequest("POST", "/test?foo=bar", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status 201, got %d", status)
	}

	if body := rr.Body.String(); body != "created" {
		t.Errorf("expected body 'created', got: %s", body)
	}
}

func TestMiddlewareChain(t *testing.T) {
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := loggingMiddleware(recoveryMiddleware(finalHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	if body := rr.Body.String(); body != "success" {
		t.Errorf("expected body 'success', got: %s", body)
	}
}

func assertNotPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic, got %v", r)
		}
	}()
	f()
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		expectNewUUID bool
	}{
		{
			name:          "with custom request id header",
			headerValue:   "custom-request-id-123",
			expectNewUUID: false,
		},
		{
			name:          "without request id header",
			headerValue:   "",
			expectNewUUID: true,
		},
		{
			name:          "with empty request id header",
			headerValue:   "",
			expectNewUUID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Request-ID", tt.headerValue)
			}

			reqID := getRequestID(req)

			if tt.expectNewUUID {
				uuidRegex := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
				if !uuidRegex.MatchString(reqID) {
					t.Errorf("expected UUID v4 format, got %q", reqID)
				}
			} else {
				if reqID != tt.headerValue {
					t.Errorf("getRequestID() = %q, want %q", reqID, tt.headerValue)
				}
			}
		})
	}
}

func TestContextRequestID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "with request id in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), requestIDKey, "test-request-id")
			},
			expected: "test-request-id",
		},
		{
			name: "without request id in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "with non-string value in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), requestIDKey, 123)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqID := contextRequestID(tt.setupCtx())
			if reqID != tt.expected {
				t.Errorf("contextRequestID() = %q, want %q", reqID, tt.expected)
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := contextRequestID(r.Context())
		if reqID != "" {
			w.Header().Set("X-Request-ID-From-Context", reqID)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	middleware := requestIDMiddleware(handler)

	t.Run("with custom request id header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-id-123")
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}

		if reqID := rr.Header().Get("X-Request-ID"); reqID != "custom-id-123" {
			t.Errorf("X-Request-ID header = %q, want %q", reqID, "custom-id-123")
		}

		if reqID := rr.Header().Get("X-Request-ID-From-Context"); reqID != "custom-id-123" {
			t.Errorf("request ID not propagated to context: %q, want %q", reqID, "custom-id-123")
		}
	})

	t.Run("without request id header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}

		reqID := rr.Header().Get("X-Request-ID")
		if reqID == "" {
			t.Error("X-Request-ID header should be generated when not provided")
		}

		if reqID := rr.Header().Get("X-Request-ID-From-Context"); reqID == "" {
			t.Error("request ID not propagated to context")
		}
	})
}

func TestLoggingMiddlewareWithRequestID(t *testing.T) {
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	handler := requestIDMiddleware(loggingMiddleware(normalHandler))

	t.Run("with request id", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test?foo=bar", nil)
		req.Header.Set("X-Request-ID", "test-req-123")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("expected status 201, got %d", status)
		}

		if reqID := rr.Header().Get("X-Request-ID"); reqID != "test-req-123" {
			t.Errorf("X-Request-ID header = %q, want %q", reqID, "test-req-123")
		}
	})
}
