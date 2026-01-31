// Package distributed provides distributed coordination primitives.
//
// This file implements distributed locking using Redis with the Redlock algorithm.
// It uses SET NX EX for single-Redis locking (sufficient for most use cases).
package distributed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/redis/go-redis/v9"
)

// RedisLock implements distributed locking using Redis.
// It uses SET NX EX for atomic lock acquisition with TTL.
type RedisLock struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisLock creates a new Redis-based distributed lock.
func NewRedisLock(client *redis.Client, keyPrefix string) *RedisLock {
	if keyPrefix == "" {
		keyPrefix = "spartan:lock:"
	}
	return &RedisLock{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Acquire attempts to acquire the lock with given TTL.
// Returns true and a token if acquired, false if already held.
func (rl *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, string, error) {
	if ttl <= 0 {
		return false, "", apperrors.Validation("ttl must be positive")
	}

	token := generateToken()
	fullKey := rl.keyPrefix + key

	// SET key token NX EX ttl
	// NX = only if not exists
	// EX = expire time in seconds
	ok, err := rl.client.SetNX(ctx, fullKey, token, ttl).Result()
	if err != nil {
		return false, "", apperrors.Wrap(apperrors.KindInternal, "failed to acquire lock", err)
	}

	if !ok {
		// Lock is held by another
		return false, "", nil
	}

	return true, token, nil
}

// Release explicitly releases the lock.
// The token must match the one returned by Acquire.
func (rl *RedisLock) Release(ctx context.Context, key string, token string) error {
	fullKey := rl.keyPrefix + key

	// Use Lua script to check token and delete atomically
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := rl.client.Eval(ctx, script, []string{fullKey}, token).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to release lock", err)
	}

	if result.(int64) == 0 {
		return apperrors.New(apperrors.KindPermission, "lock not held or token mismatch")
	}

	return nil
}

// Renew extends the lock TTL.
// The token must match the one returned by Acquire.
func (rl *RedisLock) Renew(ctx context.Context, key string, token string, ttl time.Duration) error {
	if ttl <= 0 {
		return apperrors.Validation("ttl must be positive")
	}

	fullKey := rl.keyPrefix + key

	// Use Lua script to check token and expire atomically
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := rl.client.Eval(ctx, script, []string{fullKey}, token, ttl.Seconds()).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to renew lock", err)
	}

	if result.(int64) == 0 {
		return apperrors.New(apperrors.KindPermission, "lock not held or token mismatch")
	}

	return nil
}

// generateToken creates a random token for lock ownership.
func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RedisRegistry implements worker registration using Redis.
type RedisRegistry struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

// NewRedisRegistry creates a new Redis-based worker registry.
func NewRedisRegistry(client *redis.Client, keyPrefix string, ttl time.Duration) *Registry {
	if keyPrefix == "" {
		keyPrefix = "spartan:worker:"
	}
	if ttl == 0 {
		ttl = 30 * time.Second
	}
	r := &RedisRegistry{
		client:    client,
		keyPrefix: keyPrefix,
		ttl:       ttl,
	}
	var reg Registry = r
	return &reg
}

// Register registers a new worker.
func (rr *RedisRegistry) Register(ctx context.Context, worker Worker) error {
	key := rr.keyPrefix + worker.ID
	data, err := json.Marshal(worker)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal worker", err)
	}

	return rr.client.Set(ctx, key, data, rr.ttl).Err()
}

// Heartbeat updates the worker's last seen timestamp.
func (rr *RedisRegistry) Heartbeat(ctx context.Context, workerID string) error {
	key := rr.keyPrefix + workerID

	// Get existing worker data
	data, err := rr.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return apperrors.New(apperrors.KindNotFound, "worker not registered")
	}
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to get worker", err)
	}

	var worker Worker
	if err := json.Unmarshal([]byte(data), &worker); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal worker", err)
	}

	worker.LastHeartbeat = time.Now()
	newData, err := json.Marshal(worker)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal worker", err)
	}

	return rr.client.Set(ctx, key, newData, rr.ttl).Err()
}

// UpdateStatus updates the worker's status.
func (rr *RedisRegistry) UpdateStatus(ctx context.Context, workerID string, status WorkerStatus) error {
	key := rr.keyPrefix + workerID

	data, err := rr.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return apperrors.New(apperrors.KindNotFound, "worker not registered")
	}
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to get worker", err)
	}

	var worker Worker
	if err := json.Unmarshal([]byte(data), &worker); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal worker", err)
	}

	worker.Status = status
	worker.LastHeartbeat = time.Now()
	newData, err := json.Marshal(worker)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal worker", err)
	}

	return rr.client.Set(ctx, key, newData, rr.ttl).Err()
}

// Unregister removes a worker from the registry.
func (rr *RedisRegistry) Unregister(ctx context.Context, workerID string) error {
	key := rr.keyPrefix + workerID
	return rr.client.Del(ctx, key).Err()
}

// ListWorkers returns all currently registered workers.
func (rr *RedisRegistry) ListWorkers(ctx context.Context) ([]Worker, error) {
	pattern := rr.keyPrefix + "*"
	keys, err := rr.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list workers", err)
	}

	workers := make([]Worker, 0, len(keys))
	for _, key := range keys {
		data, err := rr.client.Get(ctx, key).Result()
		if err != nil {
			// Worker may have expired between KEYS and GET
			continue
		}

		var worker Worker
		if err := json.Unmarshal([]byte(data), &worker); err != nil {
			slog.Warn("failed to unmarshal worker data", "key", key, "error", err)
			continue
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

// GetWorker returns a specific worker by ID.
func (rr *RedisRegistry) GetWorker(ctx context.Context, workerID string) (Worker, error) {
	key := rr.keyPrefix + workerID
	data, err := rr.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return Worker{}, apperrors.New(apperrors.KindNotFound, "worker not found")
	}
	if err != nil {
		return Worker{}, apperrors.Wrap(apperrors.KindInternal, "failed to get worker", err)
	}

	var worker Worker
	if err := json.Unmarshal([]byte(data), &worker); err != nil {
		return Worker{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal worker", err)
	}

	return worker, nil
}

// RedisLeaderElection implements leader election using Redis.
type RedisLeaderElection struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisLeaderElection creates a new Redis-based leader election.
func NewRedisLeaderElection(client *redis.Client, keyPrefix string) *RedisLeaderElection {
	if keyPrefix == "" {
		keyPrefix = "spartan:leader:"
	}
	return &RedisLeaderElection{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Elect attempts to become the leader for the given role.
func (rle *RedisLeaderElection) Elect(ctx context.Context, role string, instanceID string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, apperrors.Validation("ttl must be positive")
	}

	key := rle.keyPrefix + role

	// Try to acquire leadership
	ok, err := rle.client.SetNX(ctx, key, instanceID, ttl).Result()
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to elect leader", err)
	}

	if ok {
		return true, nil
	}

	// Check if we're already the leader
	current, err := rle.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to check leader", err)
	}

	return current == instanceID, nil
}

// RenewLeadership extends the leadership TTL.
func (rle *RedisLeaderElection) RenewLeadership(ctx context.Context, role string, instanceID string, ttl time.Duration) error {
	if ttl <= 0 {
		return apperrors.Validation("ttl must be positive")
	}

	key := rle.keyPrefix + role

	// Use Lua script to verify ownership and extend TTL
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := rle.client.Eval(ctx, script, []string{key}, instanceID, ttl.Seconds()).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to renew leadership", err)
	}

	if result.(int64) == 0 {
		return apperrors.New(apperrors.KindPermission, "not the current leader")
	}

	return nil
}

// Resign releases leadership.
func (rle *RedisLeaderElection) Resign(ctx context.Context, role string, instanceID string) error {
	key := rle.keyPrefix + role

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	_, err := rle.client.Eval(ctx, script, []string{key}, instanceID).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to resign leadership", err)
	}

	return nil
}

// IsLeader checks if this instance is currently the leader.
func (rle *RedisLeaderElection) IsLeader(ctx context.Context, role string, instanceID string) (bool, error) {
	key := rle.keyPrefix + role
	current, err := rle.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to check leadership", err)
	}

	return current == instanceID, nil
}

// GetLeader returns the current leader for a role.
func (rle *RedisLeaderElection) GetLeader(ctx context.Context, role string) (string, error) {
	key := rle.keyPrefix + role
	current, err := rle.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to get leader", err)
	}

	return current, nil
}

// Ensure RedisRegistry implements Registry interface
var _ Registry = (*RedisRegistry)(nil)

// Ensure RedisLeaderElection implements LeaderElection interface
var _ LeaderElection = (*RedisLeaderElection)(nil)

// Ensure RedisLock implements Lock interface
var _ Lock = (*RedisLock)(nil)
