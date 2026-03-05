// Package fetch provides tests for HTTP fetcher method handling.
// Tests cover POST, PUT, DELETE, PATCH, and default GET methods with various body types.
// Does NOT test WebDAV methods or custom HTTP verbs.
package fetch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPFetch_POSTWithJSONBody verifies POST requests with JSON body and Content-Type header.
func TestHTTPFetch_POSTWithJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"key":"value"}` {
			t.Errorf("unexpected body: %s", string(body))
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:         server.URL,
		Method:      "POST",
		Body:        []byte(`{"key":"value"}`),
		ContentType: "application/json",
		Timeout:     5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}

// TestHTTPFetch_PUTRequest verifies PUT requests with body.
func TestHTTPFetch_PUTRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "update data" {
			t.Errorf("unexpected body: %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:         server.URL,
		Method:      "PUT",
		Body:        []byte("update data"),
		ContentType: "text/plain",
		Timeout:     5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}

// TestHTTPFetch_DELETERequest verifies DELETE requests.
func TestHTTPFetch_DELETERequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Method:  "DELETE",
		Timeout: 5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", result.Status)
	}
}

// TestHTTPFetch_PATCHRequest verifies PATCH requests with body.
func TestHTTPFetch_PATCHRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != `[{"op": "replace", "path": "/name", "value": "new"}]` {
			t.Errorf("unexpected body: %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:         server.URL,
		Method:      "PATCH",
		Body:        []byte(`[{"op": "replace", "path": "/name", "value": "new"}]`),
		ContentType: "application/json-patch+json",
		Timeout:     5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}

// TestHTTPFetch_DefaultMethodIsGET verifies that an empty method defaults to GET.
func TestHTTPFetch_DefaultMethodIsGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
		// Method is intentionally empty
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}

// TestHTTPFetch_BodyWithoutContentType verifies that body is sent even without explicit Content-Type.
func TestHTTPFetch_BodyWithoutContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw body" {
			t.Errorf("unexpected body: %s", string(body))
		}
		// Content-Type header should not be set by client
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("expected no Content-Type, got %s", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Method:  "POST",
		Body:    []byte("raw body"),
		Timeout: 5 * time.Second,
		// ContentType is intentionally empty
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}

// TestHTTPFetch_FormEncodedBody verifies POST with form-encoded body.
func TestHTTPFetch_FormEncodedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "name=value&foo=bar" {
			t.Errorf("unexpected body: %s", string(body))
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", ct)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:         server.URL,
		Method:      "POST",
		Body:        []byte("name=value&foo=bar"),
		ContentType: "application/x-www-form-urlencoded",
		Timeout:     5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}
}
