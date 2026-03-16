// Package submission verifies shared auth resolution helpers used during request conversion.
//
// Purpose:
//   - Prove submission-layer auth resolution preserves transport overrides while still
//     validating the final fetch auth configuration.
//
// Responsibilities:
// - Assert proxy and OAuth2 transport overrides survive canonical auth resolution.
// - Assert conflicting proxy transport overrides fail fast.
//
// Scope:
// - Submission-layer auth resolution only.
//
// Usage:
// - Run with `go test ./internal/submission`.
//
// Invariants/Assumptions:
// - Request conversion should not silently drop transport overrides.
package submission

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestResolveAuthForRequest_PreservesTransportOverrides(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	override := &fetch.AuthOptions{
		Headers: map[string]string{"X-Test": "value"},
		Proxy: &fetch.ProxyConfig{
			URL:      "http://proxy.example:8080",
			Username: "user",
			Password: "pass",
		},
		OAuth2: &fetch.OAuth2AuthConfig{
			ProfileName: "acme",
			AccessToken: "token-123",
			TokenType:   "Bearer",
		},
	}

	got, err := resolveAuthForRequest(cfg, "https://example.com", "", override)
	if err != nil {
		t.Fatalf("resolveAuthForRequest() error = %v", err)
	}

	if got.Headers["X-Test"] != "value" {
		t.Fatalf("expected headers to be preserved, got %#v", got.Headers)
	}
	if got.Proxy == nil || got.Proxy.URL != override.Proxy.URL || got.Proxy.Username != override.Proxy.Username || got.Proxy.Password != override.Proxy.Password {
		t.Fatalf("expected proxy override to be preserved, got %#v", got.Proxy)
	}
	if got.OAuth2 == nil || got.OAuth2.ProfileName != "acme" || got.OAuth2.AccessToken != "token-123" || got.OAuth2.TokenType != "Bearer" {
		t.Fatalf("expected oauth2 override to be preserved, got %#v", got.OAuth2)
	}
}

func TestResolveAuthForRequest_RejectsConflictingProxyOverrides(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	override := &fetch.AuthOptions{
		Proxy: &fetch.ProxyConfig{URL: "http://proxy.example:8080"},
		ProxyHints: &fetch.ProxySelectionHints{
			PreferredRegion: "us-east",
		},
	}

	_, err := resolveAuthForRequest(cfg, "https://example.com", "", override)
	if err == nil {
		t.Fatal("expected conflicting proxy overrides to fail")
	}
}
