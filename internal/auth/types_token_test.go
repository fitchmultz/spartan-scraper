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
