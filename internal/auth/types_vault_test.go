package auth

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
