// Package store provides persistent storage for crawl states used in deduplication.
//
// This file implements the crawl state store using Redis as an alternative to SQLite.
// It enables distributed crawling by sharing crawl state across multiple nodes.
//
// This file is responsible for:
// - Crawl state CRUD operations using Redis Hash
// - Crawl state listing with cursor-based pagination
// - Counting crawl states
// - Automatic expiration of old crawl states via Redis TTL
//
// This file does NOT handle:
// - Connection management (caller provides Redis client)
// - Job operations (store_jobs.go handles this)
// - Store initialization (store_init.go handles this)
//
// Invariants:
// - URLs are hashed for Redis keys to handle arbitrary URL lengths
// - Timestamps are stored as RFC3339Nano strings
// - Empty state returns (not error) when crawl state not found
package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/redis/go-redis/v9"
)

// RedisCrawlStateStore provides Redis-based storage for crawl states.
type RedisCrawlStateStore struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

// NewRedisCrawlStateStore creates a new Redis-based crawl state store.
func NewRedisCrawlStateStore(client *redis.Client, keyPrefix string, ttl time.Duration) *RedisCrawlStateStore {
	if keyPrefix == "" {
		keyPrefix = "spartan:crawl_state:"
	}
	return &RedisCrawlStateStore{
		client:    client,
		keyPrefix: keyPrefix,
		ttl:       ttl,
	}
}

// GetCrawlState retrieves the crawl state for a given URL.
// Returns empty state if not found (no error).
func (r *RedisCrawlStateStore) GetCrawlState(ctx context.Context, url string) (model.CrawlState, error) {
	key := r.keyForURL(url)

	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return model.CrawlState{}, apperrors.Wrap(apperrors.KindInternal, "failed to get crawl state", err)
	}

	if len(data) == 0 {
		return model.CrawlState{}, nil
	}

	return r.parseCrawlState(url, data)
}

// UpsertCrawlState inserts or updates the crawl state for a URL.
func (r *RedisCrawlStateStore) UpsertCrawlState(ctx context.Context, state model.CrawlState) error {
	key := r.keyForURL(state.URL)

	data := map[string]string{
		"url":              state.URL,
		"etag":             state.ETag,
		"last_modified":    state.LastModified,
		"content_hash":     state.ContentHash,
		"last_scraped":     state.LastScraped.Format(time.RFC3339Nano),
		"depth":            strconv.Itoa(state.Depth),
		"job_id":           state.JobID,
		"previous_content": state.PreviousContent,
		"content_snapshot": state.ContentSnapshot,
	}

	// Use pipeline for atomic operation
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, data)

	// Set expiration if configured
	if r.ttl > 0 {
		pipe.Expire(ctx, key, r.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to upsert crawl state", err)
	}

	return nil
}

// DeleteCrawlState removes a specific crawl state by URL.
func (r *RedisCrawlStateStore) DeleteCrawlState(ctx context.Context, url string) error {
	key := r.keyForURL(url)
	return r.client.Del(ctx, key).Err()
}

// DeleteAllCrawlStates removes all crawl states from the store.
// Uses SCAN to find and delete keys in batches.
func (r *RedisCrawlStateStore) DeleteAllCrawlStates(ctx context.Context) error {
	pattern := r.keyPrefix + "*"
	var cursor uint64
	var batch []string

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to scan crawl states", err)
		}

		batch = append(batch, keys...)

		// Delete in batches of 1000
		if len(batch) >= 1000 {
			if err := r.client.Del(ctx, batch...).Err(); err != nil {
				return apperrors.Wrap(apperrors.KindInternal, "failed to delete crawl states", err)
			}
			batch = batch[:0]
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Delete remaining keys
	if len(batch) > 0 {
		if err := r.client.Del(ctx, batch...).Err(); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to delete crawl states", err)
		}
	}

	return nil
}

// ListCrawlStates returns all crawl states with cursor-based pagination.
// Note: Redis doesn't support offset-based pagination efficiently.
// This implementation uses SCAN and may return fewer results than limit.
func (r *RedisCrawlStateStore) ListCrawlStates(ctx context.Context, opts ListCrawlStatesOptions) ([]model.CrawlState, error) {
	opts = opts.Defaults()
	pattern := r.keyPrefix + "*"

	var cursor uint64
	var results []model.CrawlState
	var skipped int

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, int64(opts.Limit)).Result()
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan crawl states", err)
		}

		for _, key := range keys {
			data, err := r.client.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}

			if len(data) == 0 {
				continue
			}

			// Extract URL from the hash
			url := data["url"]
			if url == "" {
				// Fallback: decode from key
				url = r.urlFromKey(key)
			}

			state, err := r.parseCrawlState(url, data)
			if err != nil {
				continue
			}

			// Handle offset by skipping
			if skipped < opts.Offset {
				skipped++
				continue
			}

			results = append(results, state)

			if len(results) >= opts.Limit {
				return results, nil
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return results, nil
}

// CountCrawlStates returns the approximate number of crawl states.
// Note: This uses SCAN and may be slow for large datasets.
func (r *RedisCrawlStateStore) CountCrawlStates(ctx context.Context) (int, error) {
	pattern := r.keyPrefix + "*"
	var cursor uint64
	var count int

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, apperrors.Wrap(apperrors.KindInternal, "failed to scan crawl states", err)
		}

		count += len(keys)

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// keyForURL generates a Redis key for a URL using SHA256 hash.
// This ensures the key length is bounded regardless of URL length.
func (r *RedisCrawlStateStore) keyForURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return r.keyPrefix + hex.EncodeToString(hash[:])
}

// urlFromKey attempts to extract the URL from a Redis key.
// This is a best-effort fallback and may not always work.
func (r *RedisCrawlStateStore) urlFromKey(key string) string {
	// The key is a hash, so we can't reverse it
	// Return the key as-is for debugging
	return key
}

// parseCrawlState converts Redis hash data to a CrawlState.
func (r *RedisCrawlStateStore) parseCrawlState(url string, data map[string]string) (model.CrawlState, error) {
	state := model.CrawlState{
		URL:             url,
		ETag:            data["etag"],
		LastModified:    data["last_modified"],
		ContentHash:     data["content_hash"],
		JobID:           data["job_id"],
		PreviousContent: data["previous_content"],
		ContentSnapshot: data["content_snapshot"],
	}

	if depthStr := data["depth"]; depthStr != "" {
		if depth, err := strconv.Atoi(depthStr); err == nil {
			state.Depth = depth
		}
	}

	if lastScraped := data["last_scraped"]; lastScraped != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastScraped); err == nil {
			state.LastScraped = t
		}
	}

	return state, nil
}

// SetTTL updates the TTL for a crawl state.
// This can be used to extend or reduce the expiration time.
func (r *RedisCrawlStateStore) SetTTL(ctx context.Context, url string, ttl time.Duration) error {
	key := r.keyForURL(url)
	return r.client.Expire(ctx, key, ttl).Err()
}

// GetTTL returns the remaining TTL for a crawl state.
// Returns 0 if the key doesn't exist or has no TTL.
func (r *RedisCrawlStateStore) GetTTL(ctx context.Context, url string) (time.Duration, error) {
	key := r.keyForURL(url)
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to get TTL", err)
	}
	return ttl, nil
}

// BatchUpsertCrawlStates performs a batch upsert of crawl states.
// This is more efficient than individual upserts for large datasets.
func (r *RedisCrawlStateStore) BatchUpsertCrawlStates(ctx context.Context, states []model.CrawlState) error {
	pipe := r.client.Pipeline()

	for _, state := range states {
		key := r.keyForURL(state.URL)

		data := map[string]string{
			"url":              state.URL,
			"etag":             state.ETag,
			"last_modified":    state.LastModified,
			"content_hash":     state.ContentHash,
			"last_scraped":     state.LastScraped.Format(time.RFC3339Nano),
			"depth":            strconv.Itoa(state.Depth),
			"job_id":           state.JobID,
			"previous_content": state.PreviousContent,
			"content_snapshot": state.ContentSnapshot,
		}

		pipe.HSet(ctx, key, data)

		if r.ttl > 0 {
			pipe.Expire(ctx, key, r.ttl)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to batch upsert crawl states", err)
	}

	return nil
}

// ExportToJSON exports all crawl states to JSON format.
// This can be used for backup or migration purposes.
func (r *RedisCrawlStateStore) ExportToJSON(ctx context.Context) ([]byte, error) {
	states, err := r.ListCrawlStates(ctx, ListCrawlStatesOptions{Limit: 10000})
	if err != nil {
		return nil, err
	}

	return json.Marshal(states)
}

// ImportFromJSON imports crawl states from JSON format.
// This can be used for restore or migration purposes.
func (r *RedisCrawlStateStore) ImportFromJSON(ctx context.Context, data []byte) error {
	var states []model.CrawlState
	if err := json.Unmarshal(data, &states); err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid JSON data", err)
	}

	return r.BatchUpsertCrawlStates(ctx, states)
}

// Ensure RedisCrawlStateStore implements the same interface as Store's crawl state methods.
// This is checked at compile time.
var _ CrawlStateStore = (*RedisCrawlStateStore)(nil)

// CrawlStateStore defines the interface for crawl state storage.
// This matches the methods in store_crawl_states.go.
type CrawlStateStore interface {
	GetCrawlState(ctx context.Context, url string) (model.CrawlState, error)
	UpsertCrawlState(ctx context.Context, state model.CrawlState) error
	DeleteCrawlState(ctx context.Context, url string) error
	DeleteAllCrawlStates(ctx context.Context) error
	ListCrawlStates(ctx context.Context, opts ListCrawlStatesOptions) ([]model.CrawlState, error)
	CountCrawlStates(ctx context.Context) (int, error)
}
