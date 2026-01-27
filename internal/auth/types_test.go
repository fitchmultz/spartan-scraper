package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTokenKind_Constants(t *testing.T) {
	tests := []struct {
		name     string
		kind     TokenKind
		expected string
	}{
		{"TokenBearer", TokenBearer, "bearer"},
		{"TokenBasic", TokenBasic, "basic"},
		{"TokenApiKey", TokenApiKey, "api_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.kind)
			if got != tt.expected {
				t.Errorf("TokenKind value = %s; want %s", got, tt.expected)
			}
		})
	}
}

func TestToken_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		token Token
	}{
		{
			name: "all fields populated",
			token: Token{
				Kind:   TokenBearer,
				Value:  "secret-token",
				Header: "Authorization",
				Query:  "token",
				Cookie: "session",
			},
		},
		{
			name: "only required fields",
			token: Token{
				Kind:  TokenApiKey,
				Value: "api-key-123",
			},
		},
		{
			name: "with header field only",
			token: Token{
				Kind:   TokenBearer,
				Value:  "token-value",
				Header: "X-Auth-Token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.token)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded Token
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.token) {
				t.Errorf("decoded = %v; want %v", decoded, tt.token)
			}
		})
	}
}

func TestToken_OmitEmpty(t *testing.T) {
	token := Token{
		Kind:  TokenBearer,
		Value: "test-token",
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	optionalFields := []string{"header", "query", "cookie"}
	for _, field := range optionalFields {
		if _, ok := jsonMap[field]; ok {
			t.Errorf("field %s should be omitted when empty", field)
		}
	}

	if jsonMap["kind"] != string(TokenBearer) {
		t.Errorf("kind = %v; want %s", jsonMap["kind"], TokenBearer)
	}
	if jsonMap["value"] != "test-token" {
		t.Errorf("value = %v; want test-token", jsonMap["value"])
	}
}

func TestHeaderKV_JSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		h    HeaderKV
	}{
		{
			name: "standard header",
			h:    HeaderKV{Key: "Authorization", Value: "Bearer token"},
		},
		{
			name: "custom header",
			h:    HeaderKV{Key: "X-API-Key", Value: "key-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.h)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded HeaderKV
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Key != tt.h.Key {
				t.Errorf("Key = %s; want %s", decoded.Key, tt.h.Key)
			}
			if decoded.Value != tt.h.Value {
				t.Errorf("Value = %s; want %s", decoded.Value, tt.h.Value)
			}
		})
	}
}

func TestCookie_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		cookie Cookie
	}{
		{
			name: "all fields populated",
			cookie: Cookie{
				Name:   "session",
				Value:  "abc123",
				Domain: "example.com",
				Path:   "/",
			},
		},
		{
			name: "only required fields",
			cookie: Cookie{
				Name:  "session",
				Value: "xyz789",
			},
		},
		{
			name: "with domain only",
			cookie: Cookie{
				Name:   "token",
				Value:  "value123",
				Domain: "api.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cookie)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded Cookie
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.cookie) {
				t.Errorf("decoded = %v; want %v", decoded, tt.cookie)
			}
		})
	}
}

func TestCookie_OmitEmpty(t *testing.T) {
	cookie := Cookie{
		Name:  "session",
		Value: "value",
	}

	data, err := json.Marshal(cookie)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := jsonMap["domain"]; ok {
		t.Error("field domain should be omitted when empty")
	}
	if _, ok := jsonMap["path"]; ok {
		t.Error("field path should be omitted when empty")
	}
}

func TestLoginFlow_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		login *LoginFlow
	}{
		{
			name: "all fields populated",
			login: &LoginFlow{
				URL:            "https://example.com/login",
				UserSelector:   "#username",
				PassSelector:   "#password",
				SubmitSelector: "button[type='submit']",
				Username:       "user",
				Password:       "pass",
			},
		},
		{
			name:  "nil pointer",
			login: nil,
		},
		{
			name: "partial fields",
			login: &LoginFlow{
				URL:          "https://example.com/auth",
				UserSelector: "input[name='user']",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.login)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded *LoginFlow
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if tt.login == nil {
				if decoded != nil {
					t.Errorf("decoded = %v; want nil", decoded)
				}
			} else {
				if !reflect.DeepEqual(decoded, tt.login) {
					t.Errorf("decoded = %v; want %v", decoded, tt.login)
				}
			}
		})
	}
}

func TestProfile_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
	}{
		{
			name: "fully populated",
			profile: Profile{
				Name:    "test-profile",
				Parents: []string{"parent1", "parent2"},
				Headers: []HeaderKV{{Key: "X-Auth", Value: "token"}},
				Cookies: []Cookie{{Name: "session", Value: "abc"}},
				Tokens: []Token{
					{Kind: TokenBearer, Value: "bearer-token"},
				},
				Login: &LoginFlow{
					URL:          "https://example.com/login",
					UserSelector: "#username",
				},
				Presets: []TargetPreset{
					{Name: "preset1", HostPatterns: []string{"example.com"}},
				},
			},
		},
		{
			name: "minimal profile",
			profile: Profile{
				Name: "minimal",
			},
		},
		{
			name: "with nil login",
			profile: Profile{
				Name:  "test",
				Login: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.profile)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded Profile
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.profile.Name {
				t.Errorf("Name = %s; want %s", decoded.Name, tt.profile.Name)
			}
			if !reflect.DeepEqual(decoded.Parents, tt.profile.Parents) {
				t.Errorf("Parents = %v; want %v", decoded.Parents, tt.profile.Parents)
			}
			if !reflect.DeepEqual(decoded.Headers, tt.profile.Headers) {
				t.Errorf("Headers = %v; want %v", decoded.Headers, tt.profile.Headers)
			}
			if !reflect.DeepEqual(decoded.Cookies, tt.profile.Cookies) {
				t.Errorf("Cookies = %v; want %v", decoded.Cookies, tt.profile.Cookies)
			}
			if !reflect.DeepEqual(decoded.Tokens, tt.profile.Tokens) {
				t.Errorf("Tokens = %v; want %v", decoded.Tokens, tt.profile.Tokens)
			}
			if !reflect.DeepEqual(decoded.Login, tt.profile.Login) {
				t.Errorf("Login = %v; want %v", decoded.Login, tt.profile.Login)
			}
			if !reflect.DeepEqual(decoded.Presets, tt.profile.Presets) {
				t.Errorf("Presets = %v; want %v", decoded.Presets, tt.profile.Presets)
			}
		})
	}
}

func TestTargetPreset_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		preset TargetPreset
	}{
		{
			name: "fully populated",
			preset: TargetPreset{
				Name:         "preset1",
				HostPatterns: []string{"example.com", "*.example.org"},
				Profile:      "profile1",
				Headers:      []HeaderKV{{Key: "X-Test", Value: "value"}},
				Cookies:      []Cookie{{Name: "cookie", Value: "val"}},
				Tokens:       []Token{{Kind: TokenBasic, Value: "basic-auth"}},
			},
		},
		{
			name: "minimal preset",
			preset: TargetPreset{
				Name: "minimal",
			},
		},
		{
			name: "with profile reference",
			preset: TargetPreset{
				Name:    "with-profile",
				Profile: "some-profile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.preset)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded TargetPreset
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.preset) {
				t.Errorf("decoded = %v; want %v", decoded, tt.preset)
			}
		})
	}
}

func TestVault_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		vault Vault
	}{
		{
			name: "fully populated",
			vault: Vault{
				Version: "1.0",
				Profiles: []Profile{
					{Name: "profile1"},
					{Name: "profile2"},
				},
				Presets: []TargetPreset{
					{Name: "preset1"},
				},
			},
		},
		{
			name: "empty vault",
			vault: Vault{
				Version:  "1.0",
				Profiles: []Profile{},
			},
		},
		{
			name: "with nil presets",
			vault: Vault{
				Version:  "1.0",
				Profiles: []Profile{{Name: "profile1"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.vault)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded Vault
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Version != tt.vault.Version {
				t.Errorf("Version = %s; want %s", decoded.Version, tt.vault.Version)
			}
			if !reflect.DeepEqual(decoded.Profiles, tt.vault.Profiles) {
				t.Errorf("Profiles = %v; want %v", decoded.Profiles, tt.vault.Profiles)
			}
			if !reflect.DeepEqual(decoded.Presets, tt.vault.Presets) {
				t.Errorf("Presets = %v; want %v", decoded.Presets, tt.vault.Presets)
			}
		})
	}
}

func TestVault_PresetsOmitEmpty(t *testing.T) {
	vault := Vault{
		Version:  "1.0",
		Profiles: []Profile{{Name: "profile1"}},
	}

	data, err := json.Marshal(vault)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := jsonMap["presets"]; ok {
		t.Error("field presets should be omitted when empty or nil")
	}
}

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
