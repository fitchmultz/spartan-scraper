// Package auth provides tests for Profile type.
// Tests cover JSON serialization of auth profiles with parents, headers, cookies.
// Does NOT test profile resolution or inheritance merging.
package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
