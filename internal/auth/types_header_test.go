package auth

import (
	"encoding/json"
	"testing"
)

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
