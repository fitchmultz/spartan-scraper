package scrape

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Test Title</title></head><body><h1>Hello</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:       srv.URL,
		Timeout:   5 * time.Second,
		UserAgent: "SpartanTest/1.0",
		DataDir:   t.TempDir(),
	}

	result, err := Run(req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}

	if result.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", result.Title)
	}

	if result.Normalized.Title != "Test Title" {
		t.Errorf("expected normalized title 'Test Title', got %q", result.Normalized.Title)
	}
}
