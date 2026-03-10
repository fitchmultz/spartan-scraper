// Package paramdecode provides tests for shared parameter decoding behavior.
//
// Purpose:
// - Lock in the shared coercion rules used by jobs and scheduler parameter maps.
//
// Responsibilities:
// - Verify fallback behavior for bool and positive-int reads.
// - Verify string-slice extraction from persisted JSON shapes.
// - Verify struct and byte decoding from round-tripped values.
//
// Scope:
// - Unit tests for the internal shared decoder package only.
//
// Usage:
// - Run with `go test ./internal/paramdecode`.
//
// Invariants/Assumptions:
// - Invalid values fall back instead of panicking.
// - JSON-compatible maps decode into typed structs.
// - Bytes preserve base64 round-tripped payloads.
package paramdecode

import (
	"encoding/base64"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestBoolDefault(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		fallback bool
		want     bool
	}{
		{name: "nil params use fallback", params: nil, fallback: true, want: true},
		{name: "missing key uses fallback", params: map[string]any{}, fallback: true, want: true},
		{name: "explicit false wins", params: map[string]any{"playwright": false}, fallback: true, want: false},
		{name: "invalid type uses fallback", params: map[string]any{"playwright": "yes"}, fallback: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BoolDefault(tt.params, "playwright", tt.fallback); got != tt.want {
				t.Fatalf("BoolDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		fallback int
		want     int
	}{
		{name: "typed int", params: map[string]any{"timeout": 30}, fallback: 10, want: 30},
		{name: "json number", params: map[string]any{"timeout": 45.0}, fallback: 10, want: 45},
		{name: "zero falls back", params: map[string]any{"timeout": 0}, fallback: 10, want: 10},
		{name: "negative falls back", params: map[string]any{"timeout": -1}, fallback: 10, want: 10},
		{name: "invalid type falls back", params: map[string]any{"timeout": "30"}, fallback: 10, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PositiveInt(tt.params, "timeout", tt.fallback); got != tt.want {
				t.Fatalf("PositiveInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStringSlice(t *testing.T) {
	params := map[string]any{
		"urls": []interface{}{"https://example.com", "https://example.com/docs", 3},
	}

	got := StringSlice(params, "urls")
	want := []string{"https://example.com", "https://example.com/docs"}
	if len(got) != len(want) {
		t.Fatalf("StringSlice() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("StringSlice()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDecodeAndDecodePtr(t *testing.T) {
	params := map[string]any{
		"auth": map[string]any{
			"basic":   "user:pass",
			"headers": map[string]any{"X-Test": "1"},
			"cookies": []any{"session=abc"},
		},
		"screenshot": map[string]any{
			"enabled": true,
			"format":  "jpeg",
			"quality": 85,
		},
	}

	auth := Decode[fetch.AuthOptions](params, "auth")
	if auth.Basic != "user:pass" {
		t.Fatalf("Decode auth basic = %q, want %q", auth.Basic, "user:pass")
	}
	if auth.Headers["X-Test"] != "1" {
		t.Fatalf("Decode auth header = %q, want %q", auth.Headers["X-Test"], "1")
	}

	screenshot := DecodePtr[fetch.ScreenshotConfig](params, "screenshot")
	if screenshot == nil {
		t.Fatal("DecodePtr screenshot returned nil")
	}
	if screenshot.Format != fetch.ScreenshotFormatJPEG || screenshot.Quality != 85 {
		t.Fatalf("DecodePtr screenshot = %+v, want jpeg quality 85", screenshot)
	}

	if invalid := DecodePtr[fetch.ScreenshotConfig](map[string]any{"screenshot": "invalid"}, "screenshot"); invalid != nil {
		t.Fatalf("DecodePtr invalid input = %+v, want nil", invalid)
	}
}

func TestBytes(t *testing.T) {
	raw := []byte(`{"hello":"world"}`)
	params := map[string]any{
		"body": base64.StdEncoding.EncodeToString(raw),
	}

	got := Bytes(params, "body")
	if string(got) != string(raw) {
		t.Fatalf("Bytes() = %q, want %q", got, raw)
	}
}
