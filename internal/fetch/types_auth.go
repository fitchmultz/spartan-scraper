// Package fetch provides HTTP and headless browser content fetching capabilities.
// Authentication and proxy configuration types.
package fetch

import (
	"net/url"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// ProxyConfig defines proxy settings for fetch operations.
type ProxyConfig struct {
	URL      string `json:"url,omitempty"`      // Proxy URL (http://, https://, socks5://)
	Username string `json:"username,omitempty"` // Username for proxy authentication
	Password string `json:"password,omitempty"` // Password for proxy authentication
}

// AuthOptions contains authentication options for fetch operations.
type AuthOptions struct {
	Basic               string            `json:"basic,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	Cookies             []string          `json:"cookies,omitempty"`
	Query               map[string]string `json:"query,omitempty"`
	LoginURL            string            `json:"loginUrl,omitempty"`
	LoginUserSelector   string            `json:"loginUserSelector,omitempty"`
	LoginPassSelector   string            `json:"loginPassSelector,omitempty"`
	LoginSubmitSelector string            `json:"loginSubmitSelector,omitempty"`
	LoginUser           string            `json:"loginUser,omitempty"`
	LoginPass           string            `json:"loginPass,omitempty"`
	LoginAutoDetect     bool              `json:"loginAutoDetect,omitempty"`
	Proxy               *ProxyConfig      `json:"proxy,omitempty"`
	// ProxyHints provides hints for proxy selection when using the loaded proxy pool.
	ProxyHints *ProxySelectionHints `json:"proxyHints,omitempty"`
	// OAuth2 contains OAuth 2.0 configuration for automatic token management.
	// When set, the fetcher will use OAuth transport with automatic token refresh.
	OAuth2 *OAuth2AuthConfig `json:"oauth2,omitempty"`
}

// OAuth2AuthConfig defines OAuth 2.0 authentication configuration for fetch operations.
type OAuth2AuthConfig struct {
	// ProfileName is the name of the auth profile with OAuth2 configuration
	ProfileName string `json:"profileName,omitempty"`
	// AccessToken is a static access token (optional - if not set, will be loaded from store)
	AccessToken string `json:"accessToken,omitempty"`
	// TokenType is the token type (e.g., "Bearer"). Defaults to "Bearer" if not set.
	TokenType string `json:"tokenType,omitempty"`
}

// NormalizeProxySelectionHints trims and deduplicates proxy selection hints.
func NormalizeProxySelectionHints(hints *ProxySelectionHints) *ProxySelectionHints {
	if hints == nil {
		return nil
	}
	out := &ProxySelectionHints{
		PreferredRegion: strings.TrimSpace(hints.PreferredRegion),
		RequiredTags:    normalizeStringSlice(hints.RequiredTags),
		ExcludeProxyIDs: normalizeStringSlice(hints.ExcludeProxyIDs),
	}
	if out.PreferredRegion == "" && len(out.RequiredTags) == 0 && len(out.ExcludeProxyIDs) == 0 {
		return nil
	}
	return out
}

// NormalizeTransport trims proxy-related transport overrides in place.
func (a *AuthOptions) NormalizeTransport() {
	if a == nil {
		return
	}
	if a.Proxy != nil {
		a.Proxy.URL = strings.TrimSpace(a.Proxy.URL)
		a.Proxy.Username = strings.TrimSpace(a.Proxy.Username)
		a.Proxy.Password = strings.TrimSpace(a.Proxy.Password)
		if a.Proxy.URL == "" && a.Proxy.Username == "" && a.Proxy.Password == "" {
			a.Proxy = nil
		}
	}
	a.ProxyHints = NormalizeProxySelectionHints(a.ProxyHints)
}

// ValidateTransport rejects ambiguous or malformed proxy overrides.
func (a *AuthOptions) ValidateTransport() error {
	if a == nil {
		return nil
	}
	a.NormalizeTransport()
	hasDirectProxy := a.Proxy != nil && a.Proxy.URL != ""
	if a.Proxy != nil && a.Proxy.URL == "" && (a.Proxy.Username != "" || a.Proxy.Password != "") {
		return apperrors.Validation("proxy username/password require --proxy or auth.proxy.url")
	}
	if hasDirectProxy && a.ProxyHints != nil {
		return apperrors.Validation("direct proxy and proxy selection hints are mutually exclusive")
	}
	return nil
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ApplyAuthQuery applies authentication query parameters to a URL.
// If the query map is empty, the original URL is returned unchanged.
func ApplyAuthQuery(rawURL string, query map[string]string) string {
	if len(query) == 0 {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	values := parsed.Query()
	for key, value := range query {
		if key == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
