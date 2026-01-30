// Package auth provides authentication profile management and credential resolution.
// It handles profile inheritance, preset matching, environment variable overrides,
// profile persistence (Load/Save vault), and CRUD operations.
// It does NOT handle authentication execution.
package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateAPIKey(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		keyName     string
		permissions APIKeyPermission
		expiresAt   *time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid read_write key",
			keyName:     "Test Key",
			permissions: APIKeyPermissionReadWrite,
			expiresAt:   nil,
			wantErr:     false,
		},
		{
			name:        "valid read_only key",
			keyName:     "Read Only Key",
			permissions: APIKeyPermissionReadOnly,
			expiresAt:   nil,
			wantErr:     false,
		},
		{
			name:        "key with expiration",
			keyName:     "Expiring Key",
			permissions: APIKeyPermissionReadWrite,
			expiresAt:   func() *time.Time { tm := time.Now().Add(24 * time.Hour); return &tm }(),
			wantErr:     false,
		},
		{
			name:        "empty name fails",
			keyName:     "",
			permissions: APIKeyPermissionReadWrite,
			expiresAt:   nil,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid permissions fails",
			keyName:     "Test",
			permissions: "invalid",
			expiresAt:   nil,
			wantErr:     true,
			errContains: "permissions must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateAPIKey(tmpDir, tt.keyName, tt.permissions, tt.expiresAt)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateAPIKey() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GenerateAPIKey() error = %v, want containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateAPIKey() unexpected error = %v", err)
				return
			}

			// Verify key format
			if !strings.HasPrefix(key, apiKeyPrefix) {
				t.Errorf("GenerateAPIKey() key = %v, want prefix %v", key, apiKeyPrefix)
			}

			// Verify key length (prefix + base64url(32 bytes))
			if len(key) < len(apiKeyPrefix)+10 {
				t.Errorf("GenerateAPIKey() key too short = %v", key)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid key
	validKey, err := GenerateAPIKey(tmpDir, "Valid Key", APIKeyPermissionReadWrite, nil)
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}

	// Create an expired key
	expiredTime := time.Now().Add(-24 * time.Hour)
	expiredKey, err := GenerateAPIKey(tmpDir, "Expired Key", APIKeyPermissionReadOnly, &expiredTime)
	if err != nil {
		t.Fatalf("Failed to create expired key: %v", err)
	}

	tests := []struct {
		name        string
		key         string
		wantErr     bool
		errContains string
		wantPerm    APIKeyPermission
	}{
		{
			name:     "valid key",
			key:      validKey,
			wantErr:  false,
			wantPerm: APIKeyPermissionReadWrite,
		},
		{
			name:        "expired key",
			key:         expiredKey,
			wantErr:     true,
			errContains: "expired",
		},
		{
			name:        "empty key",
			key:         "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid key",
			key:         "ss_invalid_key",
			wantErr:     true,
			errContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := ValidateAPIKey(tmpDir, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAPIKey() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateAPIKey() error = %v, want containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateAPIKey() unexpected error = %v", err)
				return
			}

			if apiKey.Permissions != tt.wantPerm {
				t.Errorf("ValidateAPIKey() permissions = %v, want %v", apiKey.Permissions, tt.wantPerm)
			}

			// Verify LastUsedAt was updated
			if apiKey.LastUsedAt == nil {
				t.Errorf("ValidateAPIKey() LastUsedAt should be set")
			}
		})
	}
}

// clearAPIKeysCache clears the in-memory cache for testing
func clearAPIKeysCache() {
	apiKeysCacheMu.Lock()
	apiKeysCache = nil
	apiKeysCacheTime = time.Time{}
	apiKeysCacheMu.Unlock()
}

func TestListAPIKeys(t *testing.T) {
	tmpDir := t.TempDir()
	clearAPIKeysCache()

	// Initially empty
	keys, err := ListAPIKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("ListAPIKeys() = %v keys, want 0", len(keys))
	}

	// Create some keys
	_, err = GenerateAPIKey(tmpDir, "Key 1", APIKeyPermissionReadWrite, nil)
	if err != nil {
		t.Fatalf("Failed to create key 1: %v", err)
	}
	_, err = GenerateAPIKey(tmpDir, "Key 2", APIKeyPermissionReadOnly, nil)
	if err != nil {
		t.Fatalf("Failed to create key 2: %v", err)
	}

	// List should return 2 keys with masked values
	keys, err = ListAPIKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListAPIKeys() = %v keys, want 2", len(keys))
	}

	// Verify keys are masked
	for _, key := range keys {
		if !strings.HasSuffix(key.Key, "***") {
			t.Errorf("ListAPIKeys() key = %v, want masked with ***", key.Key)
		}
	}
}

func TestRevokeAPIKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a key
	key, err := GenerateAPIKey(tmpDir, "To Revoke", APIKeyPermissionReadWrite, nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	// Verify key exists
	_, err = ValidateAPIKey(tmpDir, key)
	if err != nil {
		t.Errorf("ValidateAPIKey() before revoke error = %v", err)
	}

	// Revoke the key
	err = RevokeAPIKey(tmpDir, key)
	if err != nil {
		t.Errorf("RevokeAPIKey() error = %v", err)
	}

	// Verify key no longer exists
	_, err = ValidateAPIKey(tmpDir, key)
	if err == nil {
		t.Errorf("ValidateAPIKey() after revoke should fail")
	}

	// Revoke non-existent key should fail
	err = RevokeAPIKey(tmpDir, "ss_nonexistent")
	if err == nil {
		t.Errorf("RevokeAPIKey() non-existent key should fail")
	}
}

func TestSaveAndLoadAPIKeys(t *testing.T) {
	tmpDir := t.TempDir()

	store := APIKeysStore{
		Version: apiKeysVersion,
		Keys: []APIKey{
			{
				Key:         "ss_test_key_1",
				Name:        "Test Key 1",
				Permissions: APIKeyPermissionReadWrite,
				CreatedAt:   time.Now(),
			},
			{
				Key:         "ss_test_key_2",
				Name:        "Test Key 2",
				Permissions: APIKeyPermissionReadOnly,
				CreatedAt:   time.Now(),
			},
		},
	}

	// Save
	err := SaveAPIKeys(tmpDir, store)
	if err != nil {
		t.Fatalf("SaveAPIKeys() error = %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, apiKeysFilename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("SaveAPIKeys() file not created")
	}

	// Load
	loaded, err := LoadAPIKeys(tmpDir)
	if err != nil {
		t.Fatalf("LoadAPIKeys() error = %v", err)
	}

	if loaded.Version != apiKeysVersion {
		t.Errorf("LoadAPIKeys() version = %v, want %v", loaded.Version, apiKeysVersion)
	}

	if len(loaded.Keys) != 2 {
		t.Errorf("LoadAPIKeys() keys = %v, want 2", len(loaded.Keys))
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ss_abcdef123456", "ss_abcde***"},
		{"short", "***"},
		{"ss_abc", "***"},
		{"", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := maskKey(tt.input)
			if got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadAPIKeysNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	clearAPIKeysCache()

	// Load from non-existent file should return empty store
	store, err := LoadAPIKeys(tmpDir)
	if err != nil {
		t.Errorf("LoadAPIKeys() error = %v", err)
	}

	if store.Version != apiKeysVersion {
		t.Errorf("LoadAPIKeys() version = %v, want %v", store.Version, apiKeysVersion)
	}

	if len(store.Keys) != 0 {
		t.Errorf("LoadAPIKeys() keys = %v, want 0", len(store.Keys))
	}
}
