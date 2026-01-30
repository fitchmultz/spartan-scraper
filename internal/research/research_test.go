package research

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello World!", []string{"hello", "world"}},
		{"Multiple tokens, with symbols.", []string{"multiple", "tokens", "with", "symbols"}},
		{"Duplicate duplicate", []string{"duplicate"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("tokenize(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSimHash(t *testing.T) {
	h1 := computeSimHash("this is a test")
	h2 := computeSimHash("this is another test")
	h3 := computeSimHash("completely different text")

	d12 := hammingDistance(h1, h2)
	d13 := hammingDistance(h1, h3)

	if d12 >= d13 {
		t.Errorf("expected similar text to have smaller hamming distance: d12=%d, d13=%d", d12, d13)
	}
}

func TestDedupEvidence(t *testing.T) {
	items := []Evidence{
		{URL: "u1", SimHash: computeSimHash("duplicate text")},
		{URL: "u2", SimHash: computeSimHash("duplicate text")},
		{URL: "u3", SimHash: computeSimHash("unique text")},
	}

	deduped := dedupEvidence(items, 3)
	if len(deduped) != 2 {
		t.Errorf("expected 2 items after dedup, got %d", len(deduped))
	}
}

func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Results for test</h1><p>This is a test page with evidence.</p></body></html>`)
	}))
	defer srv.Close()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{srv.URL},
		MaxDepth: 0, // scrape only
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	result, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Query != "test evidence" {
		t.Errorf("expected query 'test evidence', got %q", result.Query)
	}

	if len(result.Evidence) == 0 {
		t.Errorf("expected evidence, got 0 items")
	}

	if result.Summary == "" {
		t.Errorf("expected summary, got empty")
	}
}

func TestRunContextCancellation(t *testing.T) {
	// Create a server that delays responses
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, `<html><body><h1>Results for test</h1><p>This is a test page with evidence.</p></body></html>`)
	}))
	defer srv.Close()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start the research in a goroutine and cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{srv.URL, srv.URL, srv.URL},
		MaxDepth: 0,
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	_, err := Run(ctx, req)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}

	// Verify it's a context cancellation error
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

func TestRunAllTargetsFail(t *testing.T) {
	// Use invalid URLs that will cause connection errors
	req := Request{
		Query:    "test evidence",
		URLs:     []string{"http://localhost:1", "http://localhost:2"},
		MaxDepth: 0,
		Timeout:  1 * time.Second,
		DataDir:  t.TempDir(),
	}

	_, err := Run(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when all targets fail, got nil")
	}

	expectedMsg := "all research targets failed"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestRunPartialFailure(t *testing.T) {
	// Create two servers: one that succeeds, one that fails
	successSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Success</h1><p>This is successful evidence.</p></body></html>`)
	}))
	defer successSrv.Close()

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{failSrv.URL, successSrv.URL, failSrv.URL},
		MaxDepth: 0,
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	result, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error with partial failure, got %v", err)
	}

	if len(result.Evidence) == 0 {
		t.Error("expected evidence from successful target, got none")
	}
}
