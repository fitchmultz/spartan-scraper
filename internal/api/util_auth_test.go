package api

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
		ProxyPool: "primary",
		ProxyHints: &fetch.ProxySelectionHints{
			PreferredRegion: "us-east",
			RequiredTags:    []string{"residential", "sticky"},
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
	if got.ProxyPool != "primary" {
		t.Fatalf("expected proxy pool name to be preserved, got %q", got.ProxyPool)
	}
	if got.ProxyHints == nil || got.ProxyHints.PreferredRegion != "us-east" || len(got.ProxyHints.RequiredTags) != 2 {
		t.Fatalf("expected proxy hints to be preserved, got %#v", got.ProxyHints)
	}
	if got.OAuth2 == nil || got.OAuth2.ProfileName != "acme" || got.OAuth2.AccessToken != "token-123" || got.OAuth2.TokenType != "Bearer" {
		t.Fatalf("expected oauth2 override to be preserved, got %#v", got.OAuth2)
	}
}
