// Package aiauthoring provides focused unit tests for shared page helpers.
//
// Purpose:
// - Verify low-level page helper validation matches operator-facing URL expectations.
//
// Responsibilities:
// - Confirm HTTP URL validation accepts trimmed hostful URLs.
// - Confirm hostless or non-HTTP URLs are rejected before fetch-time failures.
//
// Scope:
// - `validateHTTPURL` only.
//
// Usage:
// - Run with `go test ./internal/aiauthoring`.
//
// Invariants/Assumptions:
// - Operator-facing authoring flows require concrete `http` or `https` hosts.
// - Invalid URL syntax should fail as validation, not later as an internal fetch error.
package aiauthoring

import "testing"

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "trimmed https url", raw: " https://example.com/path ", wantErr: false},
		{name: "localhost http url", raw: "http://127.0.0.1:8741/health", wantErr: false},
		{name: "non-http scheme", raw: "ftp://example.com/file.txt", wantErr: true},
		{name: "hostless https", raw: "https:", wantErr: true},
		{name: "garbage", raw: "not-a-url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPURL(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatalf("validateHTTPURL(%q) expected error", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateHTTPURL(%q) unexpected error: %v", tt.raw, err)
			}
		})
	}
}
