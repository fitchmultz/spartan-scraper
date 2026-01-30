// Package jobs provides a job manager for coordinating scraping, crawling, and research tasks.
// It handles job queuing, worker management, concurrency control, and status tracking
// using an underlying persistent store.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// Manager coordinates the execution of scraping, crawling, and research jobs.
type Manager struct {
	store            *store.Store
	dataDir          string
	userAgent        string
	requestTimeout   time.Duration
	maxConcurrency   int
	limiter          *fetch.HostLimiter
	maxRetries       int
	retryBase        time.Duration
	maxResponseBytes int64
	usePlaywright    bool
	queue            chan model.Job
	pipelineRegistry *pipeline.Registry
	jsRegistry       *pipeline.JSRegistry
	templateRegistry *extract.TemplateRegistry
	wg               sync.WaitGroup
	activeJobs       map[string]context.CancelFunc
	mu               sync.Mutex
	eventSubscribers []chan<- JobEvent
	subscribersMu    sync.RWMutex
	metricsCallback  func(duration time.Duration, success bool, fetcherType, url string)
}

// JobEventType represents the type of job lifecycle event.
type JobEventType string

const (
	JobEventCreated   JobEventType = "created"
	JobEventStarted   JobEventType = "started"
	JobEventStatus    JobEventType = "status"
	JobEventCompleted JobEventType = "completed"
)

// JobEvent represents a job lifecycle event for subscribers.
type JobEvent struct {
	Type       JobEventType
	Job        model.Job
	PrevStatus model.Status
}

// ManagerStatus represents the current state of the job manager.
type ManagerStatus struct {
	QueuedJobs int `json:"queuedJobs"`
	ActiveJobs int `json:"activeJobs"`
}

// Status returns the current status of the job manager.
func (m *Manager) Status() ManagerStatus {
	m.mu.Lock()
	active := len(m.activeJobs)
	m.mu.Unlock()
	return ManagerStatus{
		QueuedJobs: len(m.queue),
		ActiveJobs: active,
	}
}

// NewManager creates a new job manager with the specified configuration.
func NewManager(store *store.Store, dataDir, userAgent string, requestTimeout time.Duration, maxConcurrency int, rateLimitQPS int, rateLimitBurst int, maxRetries int, retryBase time.Duration, maxResponseBytes int64, usePlaywright bool) *Manager {
	jsRegistry, err := pipeline.LoadJSRegistry(dataDir)
	if err != nil {
		slog.Warn("failed to load JS registry", "error", err)
	}
	templateRegistry, err := extract.LoadTemplateRegistry(dataDir)
	if err != nil {
		slog.Warn("failed to load template registry", "error", err)
	}
	return &Manager{
		store:            store,
		dataDir:          dataDir,
		userAgent:        userAgent,
		requestTimeout:   requestTimeout,
		maxConcurrency:   maxConcurrency,
		limiter:          fetch.NewHostLimiter(rateLimitQPS, rateLimitBurst),
		maxRetries:       maxRetries,
		retryBase:        retryBase,
		maxResponseBytes: maxResponseBytes,
		usePlaywright:    usePlaywright,
		queue:            make(chan model.Job, 128),
		pipelineRegistry: pipeline.NewRegistry(),
		jsRegistry:       jsRegistry,
		templateRegistry: templateRegistry,
		activeJobs:       make(map[string]context.CancelFunc),
	}
}

// recoverQueuedJobs loads all queued jobs from the store and enqueues them.
func (m *Manager) recoverQueuedJobs(ctx context.Context) error {
	var totalRecovered int
	opts := store.ListByStatusOptions{Limit: 100}

	for {
		jobs, err := m.store.ListByStatus(ctx, model.StatusQueued, opts)
		if err != nil {
			return fmt.Errorf("failed to list queued jobs: %w", err)
		}

		if len(jobs) == 0 {
			break
		}

		slog.Info("recovering queued jobs", "count", len(jobs), "offset", opts.Offset)
		for _, job := range jobs {
			slog.Info("recovering queued job", "jobID", job.ID, "kind", job.Kind)
			if err := m.Enqueue(job); err != nil {
				slog.Error("failed to enqueue recovered job", "jobID", job.ID, "error", err)
			} else {
				totalRecovered++
			}
		}

		opts.Offset += len(jobs)
		if len(jobs) < opts.Limit {
			break
		}
	}

	slog.Info("job recovery complete", "totalRecovered", totalRecovered)
	return nil
}

// Start launches the worker pool to process enqueued jobs.
// During shutdown, queued jobs are drained and executed with a fresh context
// to avoid running jobs with a canceled context.
func (m *Manager) Start(ctx context.Context) {
	slog.Info("starting job manager", "concurrency", m.maxConcurrency)

	// Recover any queued jobs from previous runs
	if err := m.recoverQueuedJobs(ctx); err != nil {
		slog.Error("failed to recover queued jobs", "error", err)
		// Continue startup anyway - new jobs will still work
	}

	for i := 0; i < m.maxConcurrency; i++ {
		m.wg.Add(1)
		go func(workerID int) {
			defer m.wg.Done()
			slog.Debug("starting worker", "workerID", workerID)
			for {
				select {
				case <-ctx.Done():
					slog.Debug("stopping worker, draining queue", "workerID", workerID)
					// Drain the queue before returning using a fresh context
					// to avoid running jobs with a canceled context
					drainCtx, drainCancel := context.WithTimeout(context.Background(), 30*time.Second)
					for {
						select {
						case job, ok := <-m.queue:
							if !ok {
								drainCancel()
								return
							}
							slog.Debug("worker picked up job (draining)", "workerID", workerID, "jobID", job.ID, "kind", job.Kind)
							if err := m.run(drainCtx, job); err != nil {
								slog.Error("job failed during drain", "jobID", job.ID, "error", err)
							}
						default:
							drainCancel()
							return
						}
					}
				case job, ok := <-m.queue:
					if !ok {
						slog.Debug("job queue closed, stopping worker", "workerID", workerID)
						return
					}
					slog.Debug("worker picked up job", "workerID", workerID, "jobID", job.ID, "kind", job.Kind)
					if err := m.run(ctx, job); err != nil {
						slog.Error("job failed", "jobID", job.ID, "error", err)
					}
				}
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

// Wait blocks until all active workers have finished processing.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// Enqueue adds a job to the processing queue. It returns an error if the queue is full.
func (m *Manager) Enqueue(job model.Job) error {
	slog.Debug("enqueuing job", "jobID", job.ID, "kind", job.Kind)
	select {
	case m.queue <- job:
		// Publish event for WebSocket subscribers
		m.publishEvent(JobEvent{
			Type: JobEventCreated,
			Job:  job,
		})
		return nil
	default:
		slog.Warn("job queue full", "jobID", job.ID)
		return apperrors.ErrQueueFull
	}
}

// SubscribeToEvents allows subscribers to receive job events.
// The subscriber should read from the channel until it's closed.
func (m *Manager) SubscribeToEvents(ch chan<- JobEvent) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	m.eventSubscribers = append(m.eventSubscribers, ch)
}

// UnsubscribeFromEvents removes a subscriber from receiving job events.
func (m *Manager) UnsubscribeFromEvents(ch chan<- JobEvent) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	for i, subscriber := range m.eventSubscribers {
		if subscriber == ch {
			m.eventSubscribers = append(m.eventSubscribers[:i], m.eventSubscribers[i+1:]...)
			break
		}
	}
}

// publishEvent broadcasts a job event to all subscribers.
// Non-blocking: slow subscribers will miss events.
func (m *Manager) publishEvent(event JobEvent) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()
	for _, ch := range m.eventSubscribers {
		select {
		case ch <- event:
		default:
			// Channel full or closed, skip this subscriber
		}
	}
}

// CancelJob attempts to cancel a running or queued job.
func (m *Manager) CancelJob(ctx context.Context, id string) error {
	m.mu.Lock()
	cancel, ok := m.activeJobs[id]
	m.mu.Unlock()

	if ok {
		slog.Info("canceling active job", "jobID", id)
		cancel()
	} else {
		slog.Info("job not active, marking as canceled in store", "jobID", id)
	}

	job, err := m.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status.IsTerminal() {
		slog.Info("job already in terminal state, not overwriting", "jobID", id, "status", job.Status)
		return nil
	}

	return m.store.UpdateStatus(ctx, id, model.StatusCanceled, "canceled by user")
}
