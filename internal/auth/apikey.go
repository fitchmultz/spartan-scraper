// Package auth provides authentication profile management and credential resolution.
// It handles profile inheritance, preset matching, environment variable overrides,
// profile persistence (Load/Save vault), and CRUD operations.
// It does NOT handle authentication execution.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

const (
	apiKeysVersion  = "1"
	apiKeysFilename = "api_keys.json"
	apiKeyPrefix    = "ss_"
)

// APIKeysStore manages API keys storage
type APIKeysStore struct {
	Version string   `json:"version"`
	Keys    []APIKey `json:"keys"`
}

var (
	// apiKeysCache holds the in-memory cache of API keys
	apiKeysCache     *APIKeysStore
	apiKeysCacheMu   sync.RWMutex
	apiKeysCacheTime time.Time
	apiKeysCacheTTL  = 5 * time.Second
)

// LoadAPIKeys loads API keys from disk
func LoadAPIKeys(dataDir string) (APIKeysStore, error) {
	// Check cache first
	apiKeysCacheMu.RLock()
	if apiKeysCache != nil && time.Since(apiKeysCacheTime) < apiKeysCacheTTL {
		cacheCopy := *apiKeysCache
		apiKeysCacheMu.RUnlock()
		return cacheCopy, nil
	}
	apiKeysCacheMu.RUnlock()

	path := apiKeysPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return APIKeysStore{Version: apiKeysVersion, Keys: []APIKey{}}, nil
		}
		return APIKeysStore{}, err
	}

	var store APIKeysStore
	if err := json.Unmarshal(data, &store); err != nil {
		return APIKeysStore{}, err
	}
	if store.Version == "" {
		store.Version = apiKeysVersion
	}
	if store.Keys == nil {
		store.Keys = []APIKey{}
	}

	// Update cache
	apiKeysCacheMu.Lock()
	apiKeysCache = &store
	apiKeysCacheTime = time.Now()
	apiKeysCacheMu.Unlock()

	return store, nil
}

// SaveAPIKeys saves API keys to disk
func SaveAPIKeys(dataDir string, store APIKeysStore) error {
	if store.Version == "" {
		store.Version = apiKeysVersion
	}
	if err := fsutil.EnsureDataDir(dataDirOrDefault(dataDir)); err != nil {
		return err
	}
	path := apiKeysPath(dataDir)
	payload, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return err
	}

	// Invalidate cache
	apiKeysCacheMu.Lock()
	apiKeysCache = nil
	apiKeysCacheTime = time.Time{}
	apiKeysCacheMu.Unlock()

	return nil
}

// GenerateAPIKey creates a new API key with the given parameters
// Returns the generated key (only time it's visible)
func GenerateAPIKey(dataDir, name string, permissions APIKeyPermission, expiresAt *time.Time) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", apperrors.Validation("API key name is required")
	}

	if permissions != APIKeyPermissionReadOnly && permissions != APIKeyPermissionReadWrite {
		return "", apperrors.Validation("permissions must be 'read_only' or 'read_write'")
	}

	// Generate secure random key
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to generate API key", err)
	}

	// Create key with prefix for easy identification
	keyValue := apiKeyPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)

	store, err := LoadAPIKeys(dataDir)
	if err != nil {
		return "", err
	}

	apiKey := APIKey{
		Key:         keyValue,
		Name:        name,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	store.Keys = append(store.Keys, apiKey)

	if err := SaveAPIKeys(dataDir, store); err != nil {
		return "", err
	}

	return keyValue, nil
}

// ValidateAPIKey checks if the provided key is valid and returns the key details
// Updates LastUsedAt on successful validation
func ValidateAPIKey(dataDir, key string) (APIKey, error) {
	if key == "" {
		return APIKey{}, apperrors.Permission("API key is empty")
	}

	store, err := LoadAPIKeys(dataDir)
	if err != nil {
		return APIKey{}, err
	}

	for i := range store.Keys {
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(store.Keys[i].Key), []byte(key)) == 1 {
			// Check expiration
			if store.Keys[i].ExpiresAt != nil && store.Keys[i].ExpiresAt.Before(time.Now()) {
				return APIKey{}, apperrors.Permission("API key expired")
			}

			// Update LastUsedAt
			now := time.Now()
			store.Keys[i].LastUsedAt = &now
			if err := SaveAPIKeys(dataDir, store); err != nil {
				// Log but don't fail the validation
				// In a real system, we'd use structured logging here
				_ = err
			}

			return store.Keys[i], nil
		}
	}

	return APIKey{}, apperrors.Permission("invalid API key")
}

// ListAPIKeys returns all API keys (without the actual key values for security)
func ListAPIKeys(dataDir string) ([]APIKey, error) {
	store, err := LoadAPIKeys(dataDir)
	if err != nil {
		return nil, err
	}

	// Return keys with masked values for security
	maskedKeys := make([]APIKey, len(store.Keys))
	for i, key := range store.Keys {
		maskedKeys[i] = APIKey{
			Key:         maskKey(key.Key),
			Name:        key.Name,
			Permissions: key.Permissions,
			CreatedAt:   key.CreatedAt,
			ExpiresAt:   key.ExpiresAt,
			LastUsedAt:  key.LastUsedAt,
		}
	}

	return maskedKeys, nil
}

// RevokeAPIKey removes an API key by its key value
func RevokeAPIKey(dataDir, key string) error {
	if key == "" {
		return apperrors.Validation("API key is required")
	}

	store, err := LoadAPIKeys(dataDir)
	if err != nil {
		return err
	}

	filtered := make([]APIKey, 0, len(store.Keys))
	found := false
	for _, k := range store.Keys {
		if subtle.ConstantTimeCompare([]byte(k.Key), []byte(key)) == 1 {
			found = true
			continue
		}
		filtered = append(filtered, k)
	}

	if !found {
		return apperrors.NotFound("API key not found")
	}

	store.Keys = filtered
	return SaveAPIKeys(dataDir, store)
}

// maskKey masks the key value, showing only the prefix and first few characters
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}

func apiKeysPath(dataDir string) string {
	return filepath.Join(dataDirOrDefault(dataDir), apiKeysFilename)
}
