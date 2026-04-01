// Package jobs provides jobs functionality for Spartan Scraper.
//
// Purpose:
// - Implement manager support for package jobs.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `jobs` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

/*
Purpose: Coordinate local scrape, crawl, and research job execution against the persistent store.
Responsibilities: Own queueing state, worker lifecycle, execution dependencies, and operator-facing job events.
Scope: Manager construction and lifecycle helpers only; create-time and run-time specifics live in sibling files.
Usage: Build via `NewManager(...)`, configure optional integrations, then call `Start(...)`.
Invariants/Assumptions: Manager methods are safe for concurrent use where explicitly locked, and job persistence remains store-backed.
*/
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
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// Manager coordinates the execution of scraping, crawling, and research jobs.
type Manager struct {
	store             *store.Store
	DataDir           string
	userAgent         string
	requestTimeout    time.Duration
	maxConcurrency    int
	limiter           *fetch.HostLimiter
	maxRetries        int
	retryBase         time.Duration
	maxResponseBytes  int64
	usePlaywright     bool
	queue             chan model.Job
	pipelineRegistry  *pipeline.Registry
	jsRegistry        *pipeline.JSRegistry
	templateRegistry  *extract.TemplateRegistry
	wg                sync.WaitGroup
	activeJobs        map[string]context.CancelFunc
	mu                sync.Mutex
	runtimeCtx        context.Context
	runtimeCancel     context.CancelFunc
	runtimeMu         sync.RWMutex
	eventSubscribers  []chan<- JobEvent
	subscribersMu     sync.RWMutex
	metricsCallback   func(duration time.Duration, success bool, fetcherType, url string)
	webhookDispatcher *webhook.Dispatcher
	proxyPool         *fetch.ProxyPool
	aiExtractor       *extract.AIExtractor
	exportTrigger     ExportTriggerInterface
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
func NewManager(store *store.Store, dataDir, userAgent string, requestTimeout time.Duration, maxConcurrency int, rateLimitQPS int, rateLimitBurst int, maxRetries int, retryBase time.Duration, maxResponseBytes int64, usePlaywright bool, cbConfig fetch.CircuitBreakerConfig, adaptiveConfig *fetch.AdaptiveConfig) *Manager {
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
		DataDir:          dataDir,
		userAgent:        userAgent,
		requestTimeout:   requestTimeout,
		maxConcurrency:   maxConcurrency,
		limiter:          createLimiter(rateLimitQPS, rateLimitBurst, cbConfig, adaptiveConfig),
		maxRetries:       maxRetries,
		retryBase:        retryBase,
		maxResponseBytes: maxResponseBytes,
		usePlaywright:    usePlaywright,
		queue:            make(chan model.Job, 128),
		pipelineRegistry: pipeline.NewRegistry(),
		jsRegistry:       jsRegistry,
		templateRegistry: templateRegistry,
		activeJobs:       make(map[string]context.CancelFunc),
		runtimeCtx:       context.Background(),
	}
}

func (m *Manager) setRuntimeContext(ctx context.Context) {
	baseCtx := context.Background()
	if ctx != nil {
		baseCtx = context.WithoutCancel(ctx)
	}
	runtimeCtx, cancel := context.WithCancel(baseCtx)

	m.runtimeMu.Lock()
	if m.runtimeCancel != nil {
		m.runtimeCancel()
	}
	m.runtimeCtx = runtimeCtx
	m.runtimeCancel = cancel
	m.runtimeMu.Unlock()
}

func (m *Manager) runtimeContext() context.Context {
	m.runtimeMu.RLock()
	ctx := m.runtimeCtx
	m.runtimeMu.RUnlock()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (m *Manager) sideEffectContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(m.runtimeContext(), timeout)
}

// createLimiter creates a HostLimiter with optional circuit breaker and adaptive rate limiting.
func createLimiter(qps, burst int, cbConfig fetch.CircuitBreakerConfig, adaptiveConfig *fetch.AdaptiveConfig) *fetch.HostLimiter {
	// Create circuit breaker if enabled
	var cb *fetch.CircuitBreaker
	if cbConfig.Enabled {
		cb = fetch.NewCircuitBreaker(cbConfig)
	}

	// Create limiter based on which features are enabled
	if adaptiveConfig != nil && adaptiveConfig.Enabled && cb != nil {
		return fetch.NewAdaptiveHostLimiterWithCircuitBreaker(qps, burst, adaptiveConfig, cb)
	}
	if adaptiveConfig != nil && adaptiveConfig.Enabled {
		return fetch.NewAdaptiveHostLimiter(qps, burst, adaptiveConfig)
	}
	if cb != nil {
		return fetch.NewHostLimiterWithCircuitBreaker(qps, burst, cb)
	}
	return fetch.NewHostLimiter(qps, burst)
}

// recoverQueuedJobs loads all queued jobs that are ready from the store and enqueues them.
// Only jobs with dependency_status = 'ready' are enqueued.
func (m *Manager) recoverQueuedJobs(ctx context.Context) error {
	var totalRecovered int

	// Get all queued jobs with ready dependency status
	jobs, err := m.store.GetJobsByDependencyStatus(ctx, model.DependencyStatusReady)
	if err != nil {
		return fmt.Errorf("failed to list ready jobs: %w", err)
	}

	for _, job := range jobs {
		if job.Status != model.StatusQueued {
			continue
		}
		slog.Info("recovering queued job", "jobID", job.ID, "kind", job.Kind)
		if err := m.Enqueue(job); err != nil {
			slog.Error("failed to enqueue recovered job", "jobID", job.ID, "error", err)
		} else {
			totalRecovered++
		}
	}

	slog.Info("job recovery complete", "totalRecovered", totalRecovered)
	return nil
}

// Start launches the worker pool to process enqueued jobs.
// During shutdown, queued jobs are drained and executed with a fresh context
// to avoid running jobs with a canceled context.
func (m *Manager) Start(ctx context.Context) {
	m.setRuntimeContext(ctx)
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
	m.runtimeMu.Lock()
	if m.runtimeCancel != nil {
		m.runtimeCancel()
		m.runtimeCancel = nil
	}
	m.runtimeMu.Unlock()
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
// Slow subscribers still miss events, while job-owned side effects stay bound to
// the manager lifecycle context.
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

	// Dispatch webhook if configured
	if m.webhookDispatcher != nil {
		if cfg := event.Job.ExtractWebhookConfig(); cfg != nil {
			m.dispatchWebhook(event, cfg)
		}
	}

	// Notify export trigger if configured
	if m.exportTrigger != nil {
		m.exportTrigger.HandleJobEvent(event)
	}
}

// dispatchWebhook sends a webhook notification for a job event.
func (m *Manager) dispatchWebhook(event JobEvent, cfg *model.WebhookSpec) {
	// Map JobEventType to webhook EventType
	var eventType webhook.EventType
	switch event.Type {
	case JobEventCreated:
		eventType = webhook.EventJobCreated
	case JobEventStarted:
		eventType = webhook.EventJobStarted
	case JobEventCompleted:
		eventType = webhook.EventJobCompleted
	default:
		// Don't send webhook for status updates that aren't terminal
		return
	}

	// Check if this event should be sent based on configured events
	if !webhook.ShouldSendEvent(eventType, string(event.Job.Status), cfg.Events) {
		return
	}

	payload := webhook.Payload{
		EventID:    event.Job.ID + "-" + string(event.Type),
		EventType:  eventType,
		Timestamp:  time.Now(),
		JobID:      event.Job.ID,
		JobKind:    string(event.Job.Kind),
		Status:     string(event.Job.Status),
		PrevStatus: string(event.PrevStatus),
		Error:      event.Job.Error,
	}
	if event.Job.ResultPath != "" {
		payload.ResultURL = "/v1/jobs/" + event.Job.ID + "/results"
	}

	if err := m.webhookDispatcher.Deliver(m.runtimeContext(), cfg.URL, payload, cfg.Secret); err != nil {
		slog.Warn("job webhook delivery failed",
			"jobID", event.Job.ID,
			"eventType", eventType,
			"url", webhook.SanitizeURL(cfg.URL),
			"error", apperrors.SafeMessage(err))
	}
}

// SetWebhookDispatcher sets the webhook dispatcher for the manager.
// This should be called before Start() if webhook notifications are desired.
func (m *Manager) SetWebhookDispatcher(dispatcher *webhook.Dispatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookDispatcher = dispatcher
}

// SetAIExtractor sets the AI extractor for the manager.
// This enables AI-powered extraction for jobs.
func (m *Manager) SetAIExtractor(extractor *extract.AIExtractor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aiExtractor = extractor
}

// ExportTriggerInterface defines the interface for export triggers.
// This is implemented by scheduler.ExportTrigger.
type ExportTriggerInterface interface {
	HandleJobEvent(event JobEvent)
}

// SetExportTrigger sets the export trigger for the manager.
// This should be called before Start() if automated export scheduling is desired.
func (m *Manager) SetExportTrigger(trigger ExportTriggerInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exportTrigger = trigger
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

// SetProxyPool sets the proxy pool for the manager.
func (m *Manager) SetProxyPool(pool *fetch.ProxyPool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proxyPool = pool
}

// GetProxyPool returns the current proxy pool, or nil if not set.
func (m *Manager) GetProxyPool() *fetch.ProxyPool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.proxyPool
}
