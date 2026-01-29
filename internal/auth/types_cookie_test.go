package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
