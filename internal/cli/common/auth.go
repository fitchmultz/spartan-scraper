// Package common provides auth-related CLI helpers.
// It is responsible for translating CLI flags into auth.ResolveInput / fetch.AuthOptions.
//
// It does NOT manage auth vault on disk (internal/auth does that).
package common

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

type LoginFlowInput struct {
	URL            string
	UserSelector   string
	PassSelector   string
	SubmitSelector string
	Username       string
	Password       string
	AutoDetect     bool
}

func BuildLoginFlow(input LoginFlowInput) *auth.LoginFlow {
	if input.URL == "" &&
		input.UserSelector == "" &&
		input.PassSelector == "" &&
		input.SubmitSelector == "" &&
		input.Username == "" &&
		input.Password == "" &&
		!input.AutoDetect {
		return nil
	}
	return &auth.LoginFlow{
		URL:            input.URL,
		UserSelector:   input.UserSelector,
		PassSelector:   input.PassSelector,
		SubmitSelector: input.SubmitSelector,
		Username:       input.Username,
		Password:       input.Password,
		AutoDetect:     input.AutoDetect,
	}
}

func ParseTokenKind(kind string) auth.TokenKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "basic":
		return auth.TokenBasic
	case "api_key", "api-key", "apikey":
		return auth.TokenApiKey
	default:
		return auth.TokenBearer
	}
}

func BuildTokens(basic string, tokens []string, kind string, header string, query string, cookie string) []auth.Token {
	out := make([]auth.Token, 0, len(tokens)+1)
	if strings.TrimSpace(basic) != "" {
		out = append(out, auth.Token{Kind: auth.TokenBasic, Value: basic})
	}
	tokenKind := ParseTokenKind(kind)
	for _, value := range tokens {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, auth.Token{
			Kind:   tokenKind,
			Value:  value,
			Header: header,
			Query:  query,
			Cookie: cookie,
		})
	}
	return out
}

func ToHeaderKVs(headers map[string]string) []auth.HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]auth.HeaderKV, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, auth.HeaderKV{Key: key, Value: value})
	}
	return out
}

func ToCookies(cookies []string) []auth.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]auth.Cookie, 0, len(cookies))
	for _, raw := range cookies {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		out = append(out, auth.Cookie{Name: name, Value: value})
	}
	return out
}

// ResolveInput fills in required fields for auth.Resolve.
func ResolveInput(cfg config.Config, url string, profile string, overrides auth.ResolveInput) auth.ResolveInput {
	overrides.ProfileName = profile
	overrides.URL = url
	overrides.Env = &cfg.AuthOverrides
	return overrides
}

// ResolveAuthForRequest resolves auth (profile + env + overrides) into fetch.AuthOptions.
func ResolveAuthForRequest(cfg config.Config, url string, profile string, overrides auth.ResolveInput) (fetch.AuthOptions, error) {
	input := ResolveInput(cfg, url, profile, overrides)
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	authOptions := auth.ToFetchOptions(resolved)
	authOptions.NormalizeTransport()
	if err := authOptions.ValidateTransport(); err != nil {
		return fetch.AuthOptions{}, err
	}
	return authOptions, nil
}

type ProxyFlagConfig struct {
	ProxyURL        string
	ProxyUsername   string
	ProxyPassword   string
	PreferredRegion string
	RequiredTags    []string
	ExcludeProxyIDs []string
}

func ApplyProxyOverrides(authOptions *fetch.AuthOptions, cfg ProxyFlagConfig) error {
	if authOptions == nil {
		return nil
	}
	proxyURL := strings.TrimSpace(cfg.ProxyURL)
	if proxyURL != "" {
		authOptions.Proxy = &fetch.ProxyConfig{
			URL:      proxyURL,
			Username: strings.TrimSpace(cfg.ProxyUsername),
			Password: strings.TrimSpace(cfg.ProxyPassword),
		}
	}
	authOptions.ProxyHints = fetch.NormalizeProxySelectionHints(&fetch.ProxySelectionHints{
		PreferredRegion: cfg.PreferredRegion,
		RequiredTags:    cfg.RequiredTags,
		ExcludeProxyIDs: cfg.ExcludeProxyIDs,
	})
	authOptions.NormalizeTransport()
	return authOptions.ValidateTransport()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ResolveAuthFromCommonFlags(cfg config.Config, url string, cf *CommonFlags) (fetch.AuthOptions, error) {
	authOverrides := auth.ResolveInput{
		Headers: ToHeaderKVs(cf.Headers.ToMap()),
		Cookies: ToCookies([]string(cf.Cookies)),
		Tokens:  BuildTokens(*cf.AuthBasic, []string(cf.TokenValues), *cf.TokenKind, *cf.TokenHeader, *cf.TokenQuery, *cf.TokenCookie),
		Login: BuildLoginFlow(LoginFlowInput{
			URL:            *cf.LoginURL,
			UserSelector:   *cf.LoginUserSelector,
			PassSelector:   *cf.LoginPassSelector,
			SubmitSelector: *cf.LoginSubmitSelector,
			Username:       *cf.LoginUser,
			Password:       *cf.LoginPass,
			AutoDetect:     *cf.LoginAutoDetect,
		}),
	}

	// Resolve auth first
	authOptions, err := ResolveAuthForRequest(cfg, url, *cf.ProfileName, authOverrides)
	if err != nil {
		return fetch.AuthOptions{}, err
	}

	if err := ApplyProxyOverrides(&authOptions, ProxyFlagConfig{
		ProxyURL:        firstNonEmpty(*cf.ProxyURL, cfg.ProxyURL),
		ProxyUsername:   firstNonEmpty(*cf.ProxyUsername, cfg.ProxyUsername),
		ProxyPassword:   firstNonEmpty(*cf.ProxyPassword, cfg.ProxyPassword),
		PreferredRegion: *cf.ProxyRegion,
		RequiredTags:    []string(cf.ProxyRequiredTags),
		ExcludeProxyIDs: []string(cf.ProxyExcludeProxyID),
	}); err != nil {
		return fetch.AuthOptions{}, err
	}

	return authOptions, nil
}
