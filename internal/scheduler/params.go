// Package scheduler provides parameter extraction and loading for scheduled jobs.
//
// This file is responsible for:
// - Extracting extract.ExtractOptions from schedule parameters
// - Extracting pipeline.Options from schedule parameters
// - Extracting fetch.AuthOptions from schedule parameters (with auth resolution)
// - Type-safe parameter accessors (string, bool, int, string slice)
//
// This file does NOT handle:
// - Parameter validation (validation.go does this)
// - Schedule persistence or execution
// - Direct auth vault access (uses auth.Resolve)
//
// Invariants:
// - nil params return zero values for all types
// - Auth resolution uses auth.Resolve with the provided dataDir
// - Token kind defaults to Bearer if not recognized
package scheduler

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func loadExtract(params map[string]interface{}) extract.ExtractOptions {
	if params == nil {
		return extract.ExtractOptions{}
	}
	opts := extract.ExtractOptions{
		Template: stringParam(params, "extractTemplate"),
		Validate: boolParam(params, "extractValidate"),
	}

	if extractConfigPath := stringParam(params, "extractConfig"); extractConfigPath != "" {
		data, err := os.ReadFile(extractConfigPath)
		if err != nil {
			slog.Warn("failed to read extract config", "path", extractConfigPath, "error", err)
			return opts
		}
		var tmpl extract.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			slog.Warn("failed to parse extract config", "path", extractConfigPath, "error", err)
			return opts
		}
		opts.Inline = &tmpl
	}

	return opts
}

func loadPipeline(params map[string]interface{}) pipeline.Options {
	if params == nil {
		return pipeline.Options{}
	}
	raw, ok := params["pipeline"]
	if !ok || raw == nil {
		return pipeline.Options{}
	}
	if opts, ok := raw.(pipeline.Options); ok {
		return opts
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return pipeline.Options{}
	}
	var opts pipeline.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return pipeline.Options{}
	}
	return opts
}

func loadAuth(params map[string]interface{}, dataDir string, targetURL string, env auth.EnvOverrides) (fetch.AuthOptions, error) {
	if params == nil {
		return fetch.AuthOptions{}, nil
	}

	input := loadAuthOverrides(params)

	input.ProfileName = stringParam(params, "authProfile")
	input.URL = targetURL
	input.Env = &env

	resolved, err := auth.Resolve(dataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}

func stringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
}

func boolParam(params map[string]interface{}, key string) bool {
	if params == nil {
		return false
	}
	if value, ok := params[key].(bool); ok {
		return value
	}
	return false
}

func boolParamDefault(params map[string]interface{}, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	if _, ok := params[key]; !ok {
		return fallback
	}
	if value, ok := params[key].(bool); ok {
		return value
	}
	return fallback
}

func intParam(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	switch value := params[key].(type) {
	case int:
		if value <= 0 {
			return fallback
		}
		return value
	case float64:
		if int(value) <= 0 {
			return fallback
		}
		return int(value)
	default:
		return fallback
	}
}

func stringSliceParam(params map[string]interface{}, key string) []string {
	if params == nil {
		return nil
	}
	switch value := params[key].(type) {
	case []interface{}:
		items := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	case []string:
		return value
	default:
		return nil
	}
}

func loadAuthOverrides(params map[string]interface{}) auth.ResolveInput {
	if params == nil {
		return auth.ResolveInput{}
	}

	input := auth.ResolveInput{
		Headers: loadHeaderKVs(params),
		Cookies: loadCookies(params),
		Tokens:  loadTokens(params),
		Login:   loadLoginFlow(params),
	}

	return input
}

func loadHeaderKVs(params map[string]interface{}) []auth.HeaderKV {
	raw, ok := params["headers"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []auth.HeaderKV:
		return v
	case []interface{}:
		headers := make([]auth.HeaderKV, 0, len(v))
		for _, item := range v {
			if h, ok := item.(map[string]interface{}); ok {
				headers = append(headers, auth.HeaderKV{
					Key:   stringFromMap(h, "key"),
					Value: stringFromMap(h, "value"),
				})
			}
		}
		return headers
	}
	return nil
}

func loadCookies(params map[string]interface{}) []auth.Cookie {
	raw, ok := params["cookies"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []auth.Cookie:
		return v
	case []interface{}:
		cookies := make([]auth.Cookie, 0, len(v))
		for _, item := range v {
			if c, ok := item.(map[string]interface{}); ok {
				cookies = append(cookies, auth.Cookie{
					Name:   stringFromMap(c, "name"),
					Value:  stringFromMap(c, "value"),
					Domain: stringFromMap(c, "domain"),
					Path:   stringFromMap(c, "path"),
				})
			}
		}
		return cookies
	}
	return nil
}

func loadTokens(params map[string]interface{}) []auth.Token {
	if params == nil {
		return nil
	}

	tokens := make([]auth.Token, 0)

	if basic := stringParam(params, "authBasic"); basic != "" {
		tokens = append(tokens, auth.Token{Kind: auth.TokenBasic, Value: basic})
	}

	tokenKind := parseTokenKind(stringParam(params, "tokenKind"))
	tokenValues := stringSliceParam(params, "tokens")
	for _, value := range tokenValues {
		if strings.TrimSpace(value) == "" {
			continue
		}
		tokens = append(tokens, auth.Token{
			Kind:   tokenKind,
			Value:  value,
			Header: stringParam(params, "tokenHeader"),
			Query:  stringParam(params, "tokenQuery"),
			Cookie: stringParam(params, "tokenCookie"),
		})
	}

	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func loadLoginFlow(params map[string]interface{}) *auth.LoginFlow {
	loginURL := stringParam(params, "loginURL")
	if loginURL == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            loginURL,
		UserSelector:   stringParam(params, "loginUserSelector"),
		PassSelector:   stringParam(params, "loginPassSelector"),
		SubmitSelector: stringParam(params, "loginSubmitSelector"),
		Username:       stringParam(params, "loginUser"),
		Password:       stringParam(params, "loginPass"),
	}
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func parseTokenKind(kind string) auth.TokenKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "basic":
		return auth.TokenBasic
	case "api_key", "api-key", "apikey":
		return auth.TokenApiKey
	default:
		return auth.TokenBearer
	}
}
