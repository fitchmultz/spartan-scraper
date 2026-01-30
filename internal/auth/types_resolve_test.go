// Package auth provides tests for ResolveInput and EnvOverrides types.
// Tests cover JSON serialization of resolution input structures.
// Does NOT test actual profile resolution logic.
package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEnvOverrides_JSONSerialization(t *testing.T) {
	tests := []struct {
		name      string
		overrides EnvOverrides
	}{
		{
			name: "fully populated",
			overrides: EnvOverrides{
				Basic:        "user:pass",
				Bearer:       "token123",
				APIKey:       "key123",
				APIKeyHeader: "X-API-Key",
				APIKeyQuery:  "api_key",
				APIKeyCookie: "apikey",
				Headers:      map[string]string{"X-Auth": "value"},
				Cookies:      map[string]string{"session": "val"},
			},
		},
		{
			name:      "empty overrides",
			overrides: EnvOverrides{},
		},
		{
			name: "with headers only",
			overrides: EnvOverrides{
				Headers: map[string]string{"X-Test": "test-value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.overrides)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded EnvOverrides
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.overrides) {
				t.Errorf("decoded = %v; want %v", decoded, tt.overrides)
			}
		})
	}
}

func TestResolveInput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input ResolveInput
	}{
		{
			name: "fully populated",
			input: ResolveInput{
				ProfileName: "profile1",
				URL:         "https://example.com",
				Headers:     []HeaderKV{{Key: "X-Auth", Value: "token"}},
				Cookies:     []Cookie{{Name: "session", Value: "abc"}},
				Tokens:      []Token{{Kind: TokenBearer, Value: "bearer"}},
				Login:       &LoginFlow{URL: "https://example.com/login"},
				Env:         &EnvOverrides{Basic: "user:pass"},
			},
		},
		{
			name: "minimal input",
			input: ResolveInput{
				ProfileName: "profile1",
				URL:         "https://example.com",
			},
		},
		{
			name: "with nil pointers",
			input: ResolveInput{
				ProfileName: "test",
				URL:         "https://example.com",
				Login:       nil,
				Env:         nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ResolveInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.ProfileName != tt.input.ProfileName {
				t.Errorf("ProfileName = %s; want %s", decoded.ProfileName, tt.input.ProfileName)
			}
			if decoded.URL != tt.input.URL {
				t.Errorf("URL = %s; want %s", decoded.URL, tt.input.URL)
			}
			if !reflect.DeepEqual(decoded.Headers, tt.input.Headers) {
				t.Errorf("Headers = %v; want %v", decoded.Headers, tt.input.Headers)
			}
			if !reflect.DeepEqual(decoded.Cookies, tt.input.Cookies) {
				t.Errorf("Cookies = %v; want %v", decoded.Cookies, tt.input.Cookies)
			}
			if !reflect.DeepEqual(decoded.Tokens, tt.input.Tokens) {
				t.Errorf("Tokens = %v; want %v", decoded.Tokens, tt.input.Tokens)
			}
			if !reflect.DeepEqual(decoded.Login, tt.input.Login) {
				t.Errorf("Login = %v; want %v", decoded.Login, tt.input.Login)
			}
			if !reflect.DeepEqual(decoded.Env, tt.input.Env) {
				t.Errorf("Env = %v; want %v", decoded.Env, tt.input.Env)
			}
		})
	}
}

func TestResolvedAuth_JSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		auth ResolvedAuth
	}{
		{
			name: "fully populated",
			auth: ResolvedAuth{
				Headers: map[string]string{"Authorization": "Bearer token"},
				Cookies: []string{"session=abc; domain=.example.com"},
				Query:   map[string]string{"api_key": "key123"},
				Basic:   "user:pass",
				Login:   &LoginFlow{URL: "https://example.com/login"},
			},
		},
		{
			name: "empty resolved auth",
			auth: ResolvedAuth{},
		},
		{
			name: "with headers only",
			auth: ResolvedAuth{
				Headers: map[string]string{"X-API-Key": "key123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.auth)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ResolvedAuth
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.auth) {
				t.Errorf("decoded = %v; want %v", decoded, tt.auth)
			}
		})
	}
}
