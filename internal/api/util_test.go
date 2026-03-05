// Package api provides unit tests for utility functions.
// Tests contentTypeForExtension helper function and middleware.
// Does NOT test API handlers or integration behavior.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
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

// TestRequestIDMiddlewareOrder verifies that requestIDMiddleware runs before
// loggingMiddleware so that request IDs are available in logs.
func TestRequestIDMiddlewareOrder(t *testing.T) {
	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = contextRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Correct order: requestIDMiddleware outermost
	wrappedHandler := requestIDMiddleware(loggingMiddleware(handler))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-order-123")
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if capturedRequestID != "test-order-123" {
		t.Errorf("request ID not propagated to handler context: got %q, want %q", capturedRequestID, "test-order-123")
	}

	if headerReqID := rr.Header().Get("X-Request-ID"); headerReqID != "test-order-123" {
		t.Errorf("X-Request-ID header not set: got %q, want %q", headerReqID, "test-order-123")
	}
}

// TestRequestIDResponseWriterWrite ensures X-Request-ID header is set
// when handler only calls Write() without WriteHeader().
func TestRequestIDResponseWriterWrite(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only call Write, not WriteHeader
		w.Write([]byte("response body"))
	})

	middleware := requestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-write-123")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	if body := rr.Body.String(); body != "response body" {
		t.Errorf("expected body 'response body', got: %s", body)
	}

	if reqID := rr.Header().Get("X-Request-ID"); reqID != "test-write-123" {
		t.Errorf("X-Request-ID header not set on Write(): got %q, want %q", reqID, "test-write-123")
	}
}

// TestRequestIDResponseWriterWriteHeader ensures X-Request-ID header is set
// when handler calls WriteHeader().
func TestRequestIDResponseWriterWriteHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	middleware := requestIDMiddleware(handler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Request-ID", "test-header-123")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status 201, got %d", status)
	}

	if reqID := rr.Header().Get("X-Request-ID"); reqID != "test-header-123" {
		t.Errorf("X-Request-ID header not set on WriteHeader(): got %q, want %q", reqID, "test-header-123")
	}
}

// TestRecoveryMiddlewareWithRequestID verifies that panic recovery logs include request ID.
func TestRecoveryMiddlewareWithRequestID(t *testing.T) {
	var logBuf strings.Builder
	handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError})
	logger := slog.New(handler)

	oldDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldDefault)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with request id")
	})

	// Correct middleware order: requestIDMiddleware outermost
	handlerChain := requestIDMiddleware(recoveryMiddleware(panicHandler))

	req := httptest.NewRequest("GET", "/panic", nil)
	req.Header.Set("X-Request-ID", "panic-req-123")
	rr := httptest.NewRecorder()

	assertNotPanics(t, func() {
		handlerChain.ServeHTTP(rr, req)
	})

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", status)
	}

	// Check that panic log includes request_id
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "request_id=panic-req-123") {
		t.Errorf("panic log should contain request_id, got: %s", logOutput)
	}

	// Check that error response includes request ID
	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RequestID != "panic-req-123" {
		t.Errorf("error response should include request ID: got %q, want %q", resp.RequestID, "panic-req-123")
	}
}

// TestFullMiddlewareChainRequestID verifies request ID propagation through full middleware chain.
func TestFullMiddlewareChainRequestID(t *testing.T) {
	var capturedRequestID string
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = contextRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Correct order: requestIDMiddleware outermost (first on request, last on response)
	handler := requestIDMiddleware(loggingMiddleware(recoveryMiddleware(finalHandler)))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "full-chain-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRequestID != "full-chain-123" {
		t.Errorf("request ID not propagated through chain: got %q, want %q", capturedRequestID, "full-chain-123")
	}

	if reqID := rr.Header().Get("X-Request-ID"); reqID != "full-chain-123" {
		t.Errorf("X-Request-ID header not set in response: got %q, want %q", reqID, "full-chain-123")
	}
}

// TestGeneratedRequestIDPropagation verifies that generated request IDs are consistent.
func TestGeneratedRequestIDPropagation(t *testing.T) {
	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = contextRequestID(r.Context())
		w.Write([]byte("ok"))
	})

	middleware := requestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-Request-ID header, so one should be generated
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	responseReqID := rr.Header().Get("X-Request-ID")
	if responseReqID == "" {
		t.Error("X-Request-ID header should be generated when not provided")
	}

	if capturedRequestID != responseReqID {
		t.Errorf("request ID mismatch: context has %q, response header has %q", capturedRequestID, responseReqID)
	}

	// Verify it's a valid UUID
	uuidRegex := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	if !uuidRegex.MatchString(responseReqID) {
		t.Errorf("generated request ID should be valid UUID v4, got %q", responseReqID)
	}
}

// TestLoggingMiddlewareRedactsQuerySecrets verifies that sensitive query parameters
// are redacted in request logs to prevent secret leakage.
func TestLoggingMiddlewareRedactsQuerySecrets(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		shouldContain    []string // strings that should be in the redacted output
		shouldNotContain []string // strings that should NOT be in the redacted output (secrets)
	}{
		{
			name:             "token in query",
			query:            "token=secret123&foo=bar",
			shouldContain:    []string{"foo=bar", "[REDACTED]"},
			shouldNotContain: []string{"secret123"},
		},
		{
			name:             "api_key in query",
			query:            "api_key=mykey456&baz=qux",
			shouldContain:    []string{"baz=qux", "[REDACTED]"},
			shouldNotContain: []string{"mykey456"},
		},
		{
			name:             "password in query",
			query:            "password=mypass789&user=admin",
			shouldContain:    []string{"user=admin", "[REDACTED]"},
			shouldNotContain: []string{"mypass789"},
		},
		{
			name:             "multiple secrets in query",
			query:            "token=abc&api_key=def&password=ghi&normal=value",
			shouldContain:    []string{"normal=value", "[REDACTED]"},
			shouldNotContain: []string{"abc", "def", "ghi"},
		},
		{
			name:             "no secrets in query",
			query:            "foo=bar&baz=qux",
			shouldContain:    []string{"foo=bar", "baz=qux"},
			shouldNotContain: []string{"[REDACTED]"},
		},
		{
			name:             "empty query",
			query:            "",
			shouldContain:    []string{""},
			shouldNotContain: []string{},
		},
		{
			name:             "api-key with hyphen",
			query:            "api-key=secretkey&other=value",
			shouldContain:    []string{"other=value", "[REDACTED]"},
			shouldNotContain: []string{"secretkey"},
		},
		{
			name:             "secret in query",
			query:            "secret=mysecret&data=info",
			shouldContain:    []string{"data=info", "[REDACTED]"},
			shouldNotContain: []string{"mysecret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf strings.Builder
			handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
			logger := slog.New(handler)

			oldDefault := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldDefault)

			normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handlerChain := loggingMiddleware(normalHandler)

			path := "/test"
			if tt.query != "" {
				path = path + "?" + tt.query
			}
			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()

			handlerChain.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("expected status 200, got %d", status)
			}

			logOutput := logBuf.String()

			// Check that expected strings are present
			for _, s := range tt.shouldContain {
				if s != "" && !strings.Contains(logOutput, s) {
					t.Errorf("log output should contain %q, got: %s", s, logOutput)
				}
			}

			// Check that secrets are NOT present
			for _, s := range tt.shouldNotContain {
				if strings.Contains(logOutput, s) {
					t.Errorf("log output should NOT contain secret %q, got: %s", s, logOutput)
				}
			}
		})
	}
}
