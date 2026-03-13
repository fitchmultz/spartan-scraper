// Package extract provides caching for AI extraction results to reduce API costs.
package extract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultAICacheTTL is the default time-to-live for cache entries.
const DefaultAICacheTTL = 24 * time.Hour

// cacheEntry stores a cached result with metadata.
type cacheEntry struct {
	Result           AIExtractResult `json:"result"`
	CreatedAt        time.Time       `json:"created_at"`
	RouteFingerprint string          `json:"route_fingerprint,omitempty"`
	KeyHash          string          `json:"key_hash"`
}

// FileAICache implements AICache with file-based storage.
type FileAICache struct {
	dataDir string
	ttl     time.Duration
	mu      sync.RWMutex
	memory  map[string]*cacheEntry
}

// NewFileAICache creates a new file-based AI cache.
func NewFileAICache(dataDir string, ttl time.Duration) *FileAICache {
	if ttl <= 0 {
		ttl = DefaultAICacheTTL
	}
	c := &FileAICache{
		dataDir: dataDir,
		ttl:     ttl,
		memory:  make(map[string]*cacheEntry),
	}
	// Load existing cache from disk
	_ = c.loadFromDisk()
	return c
}

// Get retrieves a cached result if it exists and is not expired.
func (c *FileAICache) Get(key string) (*AIExtractResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.memory[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.CreatedAt) > c.ttl {
		return nil, false
	}

	// Return a copy with Cached flag set
	result := entry.Result
	result.Cached = true
	return &result, true
}

// Set stores a result in the cache.
func (c *FileAICache) Set(key string, result *AIExtractResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memory[key] = &cacheEntry{
		Result:    *result,
		CreatedAt: time.Now(),
		KeyHash:   hashKey(key),
	}

	// Persist to disk asynchronously
	go c.saveToDisk()
}

// GenerateCacheKey creates a deterministic cache key from request and configured route fingerprint.
func GenerateCacheKey(req AIExtractRequest, routeFingerprint string) string {
	// Include content hash, mode, fields, and configured route order.
	h := sha256.New()

	// Hash the HTML content (truncated)
	content := req.HTML
	if req.MaxContentChars > 0 && len(content) > req.MaxContentChars {
		content = content[:req.MaxContentChars]
	}
	h.Write([]byte(content))

	// Hash the mode
	h.Write([]byte(req.Mode))

	// Hash the prompt (if natural language mode)
	if req.Prompt != "" {
		h.Write([]byte(req.Prompt))
	}

	// Hash the fields
	for _, f := range req.Fields {
		h.Write([]byte(f))
	}

	// Include route fingerprint to avoid stale results when routing changes.
	h.Write([]byte(routeFingerprint))

	return hex.EncodeToString(h.Sum(nil))
}

// hashKey creates a shortened hash for display purposes.
func hashKey(key string) string {
	if len(key) <= 16 {
		return key
	}
	return key[:16]
}

// cacheFilePath returns the path to the cache file.
func (c *FileAICache) cacheFilePath() string {
	return filepath.Join(c.dataDir, "ai_cache.json")
}

// loadFromDisk loads cached entries from disk.
func (c *FileAICache) loadFromDisk() error {
	path := c.cacheFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var entries map[string]*cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Filter out expired entries
	now := time.Now()
	for key, entry := range entries {
		if now.Sub(entry.CreatedAt) <= c.ttl {
			c.memory[key] = entry
		}
	}

	return nil
}

// saveToDisk persists cache to disk.
func (c *FileAICache) saveToDisk() error {
	c.mu.RLock()
	entries := make(map[string]*cacheEntry, len(c.memory))
	for k, v := range c.memory {
		entries[k] = v
	}
	c.mu.RUnlock()

	// Ensure data directory exists
	if err := os.MkdirAll(c.dataDir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	path := c.cacheFilePath()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cached entries.
func (c *FileAICache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memory = make(map[string]*cacheEntry)
	_ = os.Remove(c.cacheFilePath())
}

// Stats returns cache statistics.
func (c *FileAICache) Stats() (total int, expired int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total = len(c.memory)
	now := time.Now()
	for _, entry := range c.memory {
		if now.Sub(entry.CreatedAt) > c.ttl {
			expired++
		}
	}
	return total, expired
}
