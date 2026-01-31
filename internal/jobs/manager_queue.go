// Package jobs provides a job manager for coordinating scraping, crawling, and research tasks.
//
// This file extends the Manager with pluggable queue backend support.
// It provides factory functions for creating managers with different queue backends.
package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/distributed"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/queue"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/redis/go-redis/v9"
)

// QueueBackend is the interface for pluggable queue backends.
// This is an alias for queue.Backend for convenience.
type QueueBackend = queue.Backend

// ManagerWithQueue extends Manager with pluggable queue backend support.
type ManagerWithQueue struct {
	*Manager
	queueBackend QueueBackend
	workerID     string
	registry     distributed.Registry
}

// NewManagerWithQueue creates a new job manager with the specified queue backend.
func NewManagerWithQueue(
	store *store.Store,
	dataDir, userAgent string,
	requestTimeout time.Duration,
	maxConcurrency int,
	rateLimitQPS int,
	rateLimitBurst int,
	maxRetries int,
	retryBase time.Duration,
	maxResponseBytes int64,
	usePlaywright bool,
	cbConfig fetch.CircuitBreakerConfig,
	adaptiveConfig *fetch.AdaptiveConfig,
	queueBackend QueueBackend,
) *ManagerWithQueue {
	// Create base manager
	baseManager := NewManager(
		store, dataDir, userAgent, requestTimeout,
		maxConcurrency, rateLimitQPS, rateLimitBurst,
		maxRetries, retryBase, maxResponseBytes,
		usePlaywright, cbConfig, adaptiveConfig,
	)

	return &ManagerWithQueue{
		Manager:      baseManager,
		queueBackend: queueBackend,
		workerID:     generateWorkerID(),
	}
}

// NewManagerFromConfig creates a job manager from configuration.
// It automatically selects the appropriate queue backend based on config.
func NewManagerFromConfig(
	store *store.Store,
	cfg config.Config,
	cbConfig fetch.CircuitBreakerConfig,
	adaptiveConfig *fetch.AdaptiveConfig,
) (*ManagerWithQueue, error) {
	var queueBackend QueueBackend

	switch cfg.QueueBackend {
	case "redis":
		redisOpts := queue.RedisOptions{
			Addr:       cfg.RedisAddr,
			Password:   cfg.RedisPassword,
			DB:         cfg.RedisDB,
			KeyPrefix:  cfg.RedisKeyPrefix,
			StreamName: "jobs",
			GroupName:  "spartan-workers",
			ConsumerID: generateWorkerID(),
		}
		redisQueue, err := queue.NewRedis(redisOpts)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create Redis queue", err)
		}
		queueBackend = redisQueue
		slog.Info("using Redis queue backend", "addr", cfg.RedisAddr)

	case "memory":
		fallthrough
	default:
		queueBackend = queue.NewMemory(128)
		slog.Info("using in-memory queue backend")
	}

	requestTimeout := time.Duration(cfg.RequestTimeoutSecs) * time.Second
	retryBase := time.Duration(cfg.RetryBaseMs) * time.Millisecond

	return NewManagerWithQueue(
		store,
		cfg.DataDir,
		cfg.UserAgent,
		requestTimeout,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		retryBase,
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
		cbConfig,
		adaptiveConfig,
		queueBackend,
	), nil
}

// Start launches the worker pool to process enqueued jobs.
// It uses the configured queue backend instead of the in-memory channel.
func (m *ManagerWithQueue) Start(ctx context.Context) {
	slog.Info("starting job manager with queue backend", "concurrency", m.maxConcurrency, "workerID", m.workerID)

	// Recover any queued jobs from previous runs
	if err := m.recoverQueuedJobs(ctx); err != nil {
		slog.Error("failed to recover queued jobs", "error", err)
		// Continue startup anyway - new jobs will still work
	}

	// Start workers that consume from the queue backend
	for i := 0; i < m.maxConcurrency; i++ {
		m.wg.Add(1)
		go func(workerID int) {
			defer m.wg.Done()
			slog.Debug("starting worker", "workerID", workerID, "backend", "queue")

			// Subscribe to queue and process messages
			err := m.queueBackend.Subscribe(ctx, func(msgCtx context.Context, msg queue.Message) error {
				// Deserialize job from message
				var job model.Job
				if err := json.Unmarshal(msg.Body, &job); err != nil {
					slog.Error("failed to unmarshal job from queue", "error", err, "msgID", msg.ID)
					return err
				}

				slog.Debug("worker picked up job", "workerID", workerID, "jobID", job.ID, "kind", job.Kind)

				// Run the job
				if err := m.run(msgCtx, job); err != nil {
					slog.Error("job failed", "jobID", job.ID, "error", err)
					return err
				}

				return nil
			})

			if err != nil && err != context.Canceled {
				slog.Error("queue subscription error", "workerID", workerID, "error", err)
			}
		}(i)
	}

	// Start periodic database checkpoint goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		slog.Info("started periodic db checkpoint", "interval", "5m")
		for {
			select {
			case <-ctx.Done():
				slog.Debug("stopping periodic db checkpoint")
				return
			case <-ticker.C:
				if err := m.store.Checkpoint(ctx); err != nil {
					slog.Warn("periodic db checkpoint failed", "error", err)
				} else {
					slog.Debug("periodic db checkpoint succeeded")
				}
			}
		}
	}()
}

// Enqueue adds a job to the processing queue using the configured backend.
func (m *ManagerWithQueue) Enqueue(job model.Job) error {
	slog.Debug("enqueuing job", "jobID", job.ID, "kind", job.Kind)

	// Serialize job to JSON
	body, err := json.Marshal(job)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal job", err)
	}

	msg := queue.Message{
		ID:   job.ID,
		Body: body,
		Headers: map[string]string{
			"job_kind": string(job.Kind),
		},
	}

	// Publish to queue backend
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.queueBackend.Publish(ctx, msg); err != nil {
		if err == queue.ErrQueueFull {
			return apperrors.ErrQueueFull
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to enqueue job", err)
	}

	// Publish event for WebSocket subscribers
	m.publishEvent(JobEvent{
		Type: JobEventCreated,
		Job:  job,
	})

	return nil
}

// Status returns the current status of the job manager.
func (m *ManagerWithQueue) Status() ManagerStatus {
	m.mu.Lock()
	active := len(m.activeJobs)
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	depth, err := m.queueBackend.QueueDepth(ctx)
	if err != nil {
		slog.Debug("failed to get queue depth", "error", err)
		depth = 0
	}

	return ManagerStatus{
		QueuedJobs: int(depth),
		ActiveJobs: active,
	}
}

// Close cleanly shuts down the manager and its queue backend.
func (m *ManagerWithQueue) Close() error {
	slog.Info("closing job manager")
	if err := m.queueBackend.Close(); err != nil {
		slog.Error("failed to close queue backend", "error", err)
		return err
	}
	return nil
}

// SetRegistry sets the worker registry for distributed coordination.
func (m *ManagerWithQueue) SetRegistry(registry distributed.Registry) {
	m.registry = registry
}

// generateWorkerID creates a unique worker identifier.
func generateWorkerID() string {
	return "worker-" + time.Now().Format("20060102-150405-") + randomString(6)
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

// Ensure ManagerWithQueue can be used where *Manager is expected
// by providing access to the embedded Manager.

// CreateRedisClient creates a Redis client from configuration.
func CreateRedisClient(cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to connect to Redis", err)
	}

	return client, nil
}

// LoadTemplateRegistry loads the template registry for the manager.
func LoadTemplateRegistry(dataDir string) (*extract.TemplateRegistry, error) {
	return extract.LoadTemplateRegistry(dataDir)
}

// LoadJSRegistry loads the JS registry for the manager.
func LoadJSRegistry(dataDir string) (*pipeline.JSRegistry, error) {
	return pipeline.LoadJSRegistry(dataDir)
}
