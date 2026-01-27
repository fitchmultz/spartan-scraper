// Package auth provides authentication profile management and credential resolution.
// It handles profile inheritance, preset matching, environment variable overrides,
// profile persistence (Load/Save vault), and CRUD operations.
// It does NOT handle authentication execution.
package auth

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"spartan-scraper/internal/fetch"
)

func Resolve(dataDir string, in ResolveInput) (ResolvedAuth, error) {
	vault, err := LoadVault(dataDir)
	if err != nil {
		return ResolvedAuth{}, err
	}

	var base Profile
	if in.ProfileName != "" {
		base, err = MergeProfiles(vault, in.ProfileName, map[string]bool{})
		if err != nil {
			return ResolvedAuth{}, err
		}
	}

	resolved := ResolvedAuth{
		Headers: map[string]string{},
		Query:   map[string]string{},
	}
	applyProfile(&resolved, base)

	if in.URL != "" {
		if preset, ok := matchPresetInProfile(base, in.URL); ok {
			if err := applyPresetWithProfile(&resolved, vault, preset); err != nil {
				return ResolvedAuth{}, err
			}
		}
		if preset, ok := MatchPreset(vault, in.URL); ok {
			if err := applyPresetWithProfile(&resolved, vault, *preset); err != nil {
				return ResolvedAuth{}, err
			}
		}
	}

	applyHeaders(&resolved, in.Headers)
	applyCookies(&resolved, in.Cookies)
	applyTokens(&resolved, in.Tokens)
	if in.Login != nil {
		resolved.Login = mergeLogin(resolved.Login, in.Login)
	}

	if in.Env != nil {
		resolved = ApplyEnvOverrides(resolved, *in.Env)
	}

	return normalizeResolved(resolved), nil
}

func MergeProfiles(vault Vault, name string, visited map[string]bool) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		return Profile{}, errors.New("profile name is required")
	}
	if visited[name] {
		return Profile{}, fmt.Errorf("auth profile cycle detected: %s", name)
	}
	visited[name] = true
	defer delete(visited, name)

	profile, found := findProfile(vault, name)
	if !found {
		return Profile{}, fmt.Errorf("auth profile not found: %s", name)
	}

	merged := Profile{}
	for _, parent := range profile.Parents {
		parent = strings.TrimSpace(parent)
		if parent == "" {
			continue
		}
		parentProfile, err := MergeProfiles(vault, parent, visited)
		if err != nil {
			return Profile{}, err
		}
		merged = mergeProfile(merged, parentProfile)
	}

	merged = mergeProfile(merged, profile)
	return merged, nil
}

func MatchPreset(vault Vault, rawURL string) (*TargetPreset, bool) {
	host := extractHost(rawURL)
	if host == "" {
		return nil, false
	}
	for _, preset := range vault.Presets {
		if hostMatchesAnyPattern(host, preset.HostPatterns) {
			return &preset, true
		}
	}
	return nil, false
}

func ApplyEnvOverrides(res ResolvedAuth, env EnvOverrides) ResolvedAuth {
	if env.Basic != "" {
		res.Basic = env.Basic
	}
	if env.Bearer != "" {
		if res.Headers == nil {
			res.Headers = map[string]string{}
		}
		res.Headers["Authorization"] = "Bearer " + env.Bearer
	}
	if env.APIKey != "" {
		switch {
		case env.APIKeyHeader != "":
			if res.Headers == nil {
				res.Headers = map[string]string{}
			}
			res.Headers[env.APIKeyHeader] = env.APIKey
		case env.APIKeyQuery != "":
			if res.Query == nil {
				res.Query = map[string]string{}
			}
			res.Query[env.APIKeyQuery] = env.APIKey
		case env.APIKeyCookie != "":
			res.Cookies = upsertCookieValue(res.Cookies, env.APIKeyCookie, env.APIKey)
		default:
			if res.Headers == nil {
				res.Headers = map[string]string{}
			}
			res.Headers["Authorization"] = env.APIKey
		}
	}
	for key, value := range env.Headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		if res.Headers == nil {
			res.Headers = map[string]string{}
		}
		res.Headers[key] = value
	}
	for name, value := range env.Cookies {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		res.Cookies = upsertCookieValue(res.Cookies, name, value)
	}
	return res
}

func applyPresetWithProfile(res *ResolvedAuth, vault Vault, preset TargetPreset) error {
	if preset.Profile != "" {
		profile, err := MergeProfiles(vault, preset.Profile, map[string]bool{})
		if err != nil {
			return err
		}
		applyProfile(res, profile)
	}
	applyHeaders(res, preset.Headers)
	applyCookies(res, preset.Cookies)
	applyTokens(res, preset.Tokens)
	return nil
}

func applyProfile(res *ResolvedAuth, profile Profile) {
	applyHeaders(res, profile.Headers)
	applyCookies(res, profile.Cookies)
	applyTokens(res, profile.Tokens)
	if profile.Login != nil {
		res.Login = mergeLogin(res.Login, profile.Login)
	}
}

func applyHeaders(res *ResolvedAuth, headers []HeaderKV) {
	if len(headers) == 0 {
		return
	}
	if res.Headers == nil {
		res.Headers = map[string]string{}
	}
	for _, header := range headers {
		key := strings.TrimSpace(header.Key)
		if key == "" {
			continue
		}
		res.Headers[key] = header.Value
	}
}

func applyCookies(res *ResolvedAuth, cookies []Cookie) {
	if len(cookies) == 0 {
		return
	}
	for _, cookie := range cookies {
		if strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		res.Cookies = upsertCookieValue(res.Cookies, cookie.Name, cookie.Value)
	}
}

func applyTokens(res *ResolvedAuth, tokens []Token) {
	if len(tokens) == 0 {
		return
	}
	for _, token := range tokens {
		if strings.TrimSpace(token.Value) == "" {
			continue
		}
		switch token.Kind {
		case TokenBasic:
			res.Basic = token.Value
		case TokenBearer:
			header := token.Header
			if header == "" {
				header = "Authorization"
			}
			if res.Headers == nil {
				res.Headers = map[string]string{}
			}
			value := token.Value
			if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
				value = "Bearer " + value
			}
			res.Headers[header] = value
		case TokenApiKey:
			switch {
			case token.Header != "":
				if res.Headers == nil {
					res.Headers = map[string]string{}
				}
				res.Headers[token.Header] = token.Value
			case token.Query != "":
				if res.Query == nil {
					res.Query = map[string]string{}
				}
				res.Query[token.Query] = token.Value
			case token.Cookie != "":
				res.Cookies = upsertCookieValue(res.Cookies, token.Cookie, token.Value)
			default:
				if res.Headers == nil {
					res.Headers = map[string]string{}
				}
				res.Headers["Authorization"] = token.Value
			}
		default:
			if res.Headers == nil {
				res.Headers = map[string]string{}
			}
			res.Headers["Authorization"] = token.Value
		}
	}
}

func mergeProfile(base Profile, overlay Profile) Profile {
	out := base
	if overlay.Name != "" {
		out.Name = overlay.Name
	}
	if len(overlay.Parents) > 0 {
		out.Parents = append([]string{}, overlay.Parents...)
	}
	out.Headers = mergeHeaders(out.Headers, overlay.Headers)
	out.Cookies = mergeCookies(out.Cookies, overlay.Cookies)
	out.Tokens = mergeTokens(out.Tokens, overlay.Tokens)
	out.Presets = mergePresets(out.Presets, overlay.Presets)
	out.Login = mergeLogin(out.Login, overlay.Login)
	return out
}

func mergeHeaders(base []HeaderKV, overlay []HeaderKV) []HeaderKV {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	values := map[string]string{}
	order := []string{}
	add := func(items []HeaderKV) {
		for _, item := range items {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				continue
			}
			if _, ok := values[key]; !ok {
				order = append(order, key)
			}
			values[key] = item.Value
		}
	}
	add(base)
	add(overlay)

	out := make([]HeaderKV, 0, len(order))
	for _, key := range order {
		out = append(out, HeaderKV{Key: key, Value: values[key]})
	}
	return out
}

func mergeCookies(base []Cookie, overlay []Cookie) []Cookie {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	values := map[string]Cookie{}
	order := []string{}
	add := func(items []Cookie) {
		for _, item := range items {
			key := strings.TrimSpace(item.Name)
			if key == "" {
				continue
			}
			if _, ok := values[key]; !ok {
				order = append(order, key)
			}
			values[key] = item
		}
	}
	add(base)
	add(overlay)

	out := make([]Cookie, 0, len(order))
	for _, key := range order {
		out = append(out, values[key])
	}
	return out
}

func mergeTokens(base []Token, overlay []Token) []Token {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	values := map[string]Token{}
	order := []string{}
	add := func(items []Token) {
		for _, item := range items {
			key := tokenKey(item)
			if key == "" {
				continue
			}
			if _, ok := values[key]; !ok {
				order = append(order, key)
			}
			values[key] = item
		}
	}
	add(base)
	add(overlay)

	out := make([]Token, 0, len(order))
	for _, key := range order {
		out = append(out, values[key])
	}
	return out
}

func mergePresets(base []TargetPreset, overlay []TargetPreset) []TargetPreset {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	values := map[string]TargetPreset{}
	unnamed := []TargetPreset{}
	add := func(items []TargetPreset) {
		for _, item := range items {
			if strings.TrimSpace(item.Name) == "" {
				unnamed = append(unnamed, item)
				continue
			}
			values[item.Name] = item
		}
	}
	add(base)
	add(overlay)

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]TargetPreset, 0, len(unnamed)+len(names))
	out = append(out, unnamed...)
	for _, name := range names {
		out = append(out, values[name])
	}
	return out
}

func mergeLogin(base *LoginFlow, overlay *LoginFlow) *LoginFlow {
	if base == nil && overlay == nil {
		return nil
	}
	if base == nil {
		copy := *overlay
		return &copy
	}
	out := *base
	if overlay == nil {
		return &out
	}
	if overlay.URL != "" {
		out.URL = overlay.URL
	}
	if overlay.UserSelector != "" {
		out.UserSelector = overlay.UserSelector
	}
	if overlay.PassSelector != "" {
		out.PassSelector = overlay.PassSelector
	}
	if overlay.SubmitSelector != "" {
		out.SubmitSelector = overlay.SubmitSelector
	}
	if overlay.Username != "" {
		out.Username = overlay.Username
	}
	if overlay.Password != "" {
		out.Password = overlay.Password
	}
	return &out
}

func normalizeResolved(res ResolvedAuth) ResolvedAuth {
	if len(res.Headers) == 0 {
		res.Headers = nil
	}
	if len(res.Query) == 0 {
		res.Query = nil
	}
	if len(res.Cookies) == 0 {
		res.Cookies = nil
	}
	return res
}

func matchPresetInProfile(profile Profile, rawURL string) (TargetPreset, bool) {
	host := extractHost(rawURL)
	if host == "" {
		return TargetPreset{}, false
	}
	for _, preset := range profile.Presets {
		if hostMatchesAnyPattern(host, preset.HostPatterns) {
			return preset, true
		}
	}
	return TargetPreset{}, false
}

func findProfile(vault Vault, name string) (Profile, bool) {
	for _, profile := range vault.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}

func extractHost(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func hostMatchesAnyPattern(host string, patterns []string) bool {
	if host == "" || len(patterns) == 0 {
		return false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if host == pattern {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(host, prefix) {
				return true
			}
			continue
		}
	}
	return false
}

func tokenKey(token Token) string {
	if token.Kind == "" {
		return ""
	}
	return string(token.Kind) + "|" + token.Header + "|" + token.Query + "|" + token.Cookie
}

func upsertCookieValue(items []string, name string, value string) []string {
	if strings.TrimSpace(name) == "" {
		return items
	}
	updated := false
	out := make([]string, 0, len(items)+1)
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 0 {
			continue
		}
		if parts[0] == name {
			out = append(out, name+"="+value)
			updated = true
		} else {
			out = append(out, item)
		}
	}
	if !updated {
		out = append(out, name+"="+value)
	}
	return out
}

func ToFetchOptions(res ResolvedAuth) fetch.AuthOptions {
	out := fetch.AuthOptions{
		Basic:   res.Basic,
		Headers: res.Headers,
		Cookies: res.Cookies,
		Query:   res.Query,
	}
	if res.Login != nil {
		out.LoginURL = res.Login.URL
		out.LoginUserSelector = res.Login.UserSelector
		out.LoginPassSelector = res.Login.PassSelector
		out.LoginSubmitSelector = res.Login.SubmitSelector
		out.LoginUser = res.Login.Username
		out.LoginPass = res.Login.Password
	}
	return out
}
