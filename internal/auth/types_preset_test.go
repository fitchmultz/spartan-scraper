// Package auth provides tests for TargetPreset type.
// Tests cover JSON serialization of target presets with host patterns.
// Does NOT test preset matching or profile resolution.
package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
