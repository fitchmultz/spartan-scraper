// Package auth provides tests for LoginFlow type.
// Tests cover JSON serialization of login automation configuration.
// Does NOT test actual login automation execution.
package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
