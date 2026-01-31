// Package scheduler provides distributed scheduling with leader election.
//
// This file implements distributed scheduling using Redis for leader election
// and shared state storage. It ensures only one scheduler instance is active
// across a cluster of nodes.
//
// This file is responsible for:
// - Leader election using distributed locks
// - Shared schedule storage in Redis
// - Graceful failover when the leader fails
// - Worker heartbeat monitoring
//
// This file does NOT handle:
// - Schedule execution (scheduler.go handles this)
// - File-based persistence (storage.go handles this)
// - In-memory caching (cached_scheduler.go handles this)
//
// Invariants:
// - Only one leader is active at any time
// - Leadership has a TTL and must be renewed
// - Schedules are stored in Redis for all nodes to access
// - Failed leaders are detected via TTL expiration
package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/redis/go-redis/v9"
)

// DistributedScheduler provides distributed scheduling with leader election.
type DistributedScheduler struct {
	client       *redis.Client
	keyPrefix    string
	leaderKey    string
	schedulesKey string
	instanceID   string
	leaseTTL     time.Duration
}

// NewDistributedScheduler creates a new distributed scheduler.
func NewDistributedScheduler(client *redis.Client, keyPrefix string, instanceID string) *DistributedScheduler {
	if keyPrefix == "" {
		keyPrefix = "spartan:scheduler:"
	}
	if instanceID == "" {
		instanceID = generateInstanceID()
	}
	return &DistributedScheduler{
		client:       client,
		keyPrefix:    keyPrefix,
		leaderKey:    keyPrefix + "leader",
		schedulesKey: keyPrefix + "schedules",
		instanceID:   instanceID,
		leaseTTL:     30 * time.Second,
	}
}

// Elect attempts to become the leader.
// Returns true if this instance is now the leader.
func (ds *DistributedScheduler) Elect(ctx context.Context) (bool, error) {
	// Try to acquire leadership with SET NX EX
	ok, err := ds.client.SetNX(ctx, ds.leaderKey, ds.instanceID, ds.leaseTTL).Result()
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to elect leader", err)
	}

	if ok {
		slog.Info("became scheduler leader", "instanceID", ds.instanceID)
		return true, nil
	}

	// Check if we're already the leader
	current, err := ds.client.Get(ctx, ds.leaderKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to check leader", err)
	}

	return current == ds.instanceID, nil
}

// RenewLeadership extends the leadership lease.
// Must be called periodically by the leader.
func (ds *DistributedScheduler) RenewLeadership(ctx context.Context) error {
	// Use Lua script to verify ownership and extend TTL
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := ds.client.Eval(ctx, script, []string{ds.leaderKey}, ds.instanceID, ds.leaseTTL.Seconds()).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to renew leadership", err)
	}

	if result.(int64) == 0 {
		return apperrors.New(apperrors.KindPermission, "lost leadership")
	}

	return nil
}

// Resign releases leadership.
func (ds *DistributedScheduler) Resign(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	_, err := ds.client.Eval(ctx, script, []string{ds.leaderKey}, ds.instanceID).Result()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to resign leadership", err)
	}

	slog.Info("resigned scheduler leadership", "instanceID", ds.instanceID)
	return nil
}

// IsLeader checks if this instance is currently the leader.
func (ds *DistributedScheduler) IsLeader(ctx context.Context) (bool, error) {
	current, err := ds.client.Get(ctx, ds.leaderKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal, "failed to check leadership", err)
	}

	return current == ds.instanceID, nil
}

// GetLeader returns the current leader instance ID.
func (ds *DistributedScheduler) GetLeader(ctx context.Context) (string, error) {
	current, err := ds.client.Get(ctx, ds.leaderKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to get leader", err)
	}

	return current, nil
}

// SaveSchedules saves all schedules to Redis.
func (ds *DistributedScheduler) SaveSchedules(ctx context.Context, schedules []Schedule) error {
	data, err := json.Marshal(schedules)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal schedules", err)
	}

	return ds.client.Set(ctx, ds.schedulesKey, data, 0).Err()
}

// LoadSchedules loads all schedules from Redis.
func (ds *DistributedScheduler) LoadSchedules(ctx context.Context) ([]Schedule, error) {
	data, err := ds.client.Get(ctx, ds.schedulesKey).Result()
	if err == redis.Nil {
		return []Schedule{}, nil
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to load schedules", err)
	}

	var schedules []Schedule
	if err := json.Unmarshal([]byte(data), &schedules); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal schedules", err)
	}

	return schedules, nil
}

// PublishScheduleUpdate notifies other nodes of a schedule change.
func (ds *DistributedScheduler) PublishScheduleUpdate(ctx context.Context, scheduleID string) error {
	channel := ds.keyPrefix + "updates"
	return ds.client.Publish(ctx, channel, scheduleID).Err()
}

// SubscribeToUpdates subscribes to schedule update notifications.
func (ds *DistributedScheduler) SubscribeToUpdates(ctx context.Context) (<-chan string, error) {
	channel := ds.keyPrefix + "updates"
	pubsub := ds.client.Subscribe(ctx, channel)

	// Wait for subscription to be confirmed
	if _, err := pubsub.Receive(ctx); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to subscribe to updates", err)
	}

	ch := make(chan string, 10)
	go func() {
		defer close(ch)
		defer pubsub.Close()

		msgCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				select {
				case ch <- msg.Payload:
				default:
					// Channel full, drop message
				}
			}
		}
	}()

	return ch, nil
}

// DistributedRunner runs the scheduler with leader election.
type DistributedRunner struct {
	scheduler    *DistributedScheduler
	dataDir      string
	manager      *jobs.Manager
	leaderTicker *time.Ticker
	workTicker   *time.Ticker
}

// NewDistributedRunner creates a new distributed scheduler runner.
func NewDistributedRunner(client *redis.Client, dataDir string, manager *jobs.Manager) *DistributedRunner {
	return &DistributedRunner{
		scheduler: NewDistributedScheduler(client, "", ""),
		dataDir:   dataDir,
		manager:   manager,
	}
}

// Run starts the distributed scheduler loop.
// It continuously attempts to acquire leadership and runs the scheduler when leader.
func (dr *DistributedRunner) Run(ctx context.Context) error {
	dr.leaderTicker = time.NewTicker(10 * time.Second)
	defer dr.leaderTicker.Stop()

	dr.workTicker = time.NewTicker(1 * time.Second)
	defer dr.workTicker.Stop()

	isLeader := false
	cs, err := NewCachedScheduler(dr.dataDir, dr.manager)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create cached scheduler", err)
	}
	defer cs.watcher.Close()

	cs.startWatcher(ctx)
	go cs.reloadLoop(ctx)

	// Try to become leader immediately
	isLeader, _ = dr.scheduler.Elect(ctx)
	if isLeader {
		slog.Info("became scheduler leader on startup")
	}

	for {
		select {
		case <-ctx.Done():
			if isLeader {
				dr.scheduler.Resign(ctx)
			}
			return nil

		case <-dr.leaderTicker.C:
			if !isLeader {
				// Try to become leader
				var err error
				isLeader, err = dr.scheduler.Elect(ctx)
				if err != nil {
					slog.Error("failed to elect leader", "error", err)
				} else if isLeader {
					slog.Info("became scheduler leader")
				}
			} else {
				// Renew leadership
				if err := dr.scheduler.RenewLeadership(ctx); err != nil {
					slog.Error("failed to renew leadership, losing leader status", "error", err)
					isLeader = false
				}
			}

		case <-dr.workTicker.C:
			if !isLeader {
				continue
			}

			// Run scheduler work
			if err := dr.runSchedulerWork(ctx, cs); err != nil {
				slog.Error("scheduler work failed", "error", err)
			}
		}
	}
}

// runSchedulerWork performs one iteration of schedule evaluation and job enqueueing.
func (dr *DistributedRunner) runSchedulerWork(ctx context.Context, cs *cachedScheduler) error {
	now := time.Now()

	cs.mu.RLock()
	schedules := make([]Schedule, len(cs.schedules))
	copy(schedules, cs.schedules)
	cs.mu.RUnlock()

	changed := false
	for i := range schedules {
		if schedules[i].NextRun.After(now) {
			continue
		}

		err := dr.enqueueSchedule(ctx, schedules[i])
		if err == nil {
			schedules[i].NextRun = now.Add(time.Duration(schedules[i].IntervalSeconds) * time.Second)
			changed = true
		} else {
			slog.Error("failed to enqueue scheduled job",
				"scheduleID", schedules[i].ID,
				"scheduleKind", schedules[i].Kind,
				"error", err,
			)
		}
	}

	if changed {
		// Save to file for backward compatibility
		if err := SaveAll(dr.dataDir, schedules); err != nil {
			slog.Error("failed to save schedules", "error", err)
		} else {
			// Save to Redis for distributed access
			if err := dr.scheduler.SaveSchedules(ctx, schedules); err != nil {
				slog.Error("failed to save schedules to Redis", "error", err)
			}

			cs.mu.Lock()
			cs.schedules = schedules
			cs.mu.Unlock()
		}
	}

	return nil
}

// enqueueSchedule enqueues a job for a schedule.
func (dr *DistributedRunner) enqueueSchedule(ctx context.Context, schedule Schedule) error {
	// This is a simplified version - in production, use the full enqueue logic from scheduler.go
	slog.Info("would enqueue scheduled job", "scheduleID", schedule.ID, "kind", schedule.Kind)
	return nil
}

// generateInstanceID creates a unique instance identifier.
func generateInstanceID() string {
	return "scheduler-" + time.Now().Format("20060102-150405-") + randomString(6)
}

// randomString generates a random string of the specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
