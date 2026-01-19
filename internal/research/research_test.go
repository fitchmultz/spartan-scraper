package research

import (
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

	result, err := Run(req)
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
