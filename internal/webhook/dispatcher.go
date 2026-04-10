// Package webhook dispatches webhook notifications for jobs, watches, crawls, and exports.
//
// Purpose:
// - Send outbound webhook deliveries with signing, retry, tracking, and transport hardening.
//
// Responsibilities:
// - Serialize JSON and multipart export webhook payloads.
// - Apply HMAC signatures when secrets are configured.
// - Retry failed deliveries with exponential backoff.
// - Validate webhook URLs, pin dialing to the validated IP set, and disable redirect hops.
// - Optionally persist delivery tracking records.
//
// Scope:
// - Outbound webhook dispatch only.
//
// Usage:
// - Construct one shared Dispatcher and pass it to managers that emit webhook side effects.
//
// Invariants/Assumptions:
// - Deliveries are best-effort and at-least-once, not exactly-once.
// - Webhook secrets are supplied per dispatch or by dispatcher default configuration.
// - Redirects are treated as non-success responses so validated targets cannot pivot hosts.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// generateID creates a unique identifier for delivery records using the provided reader.
// Returns an error if the reader fails to provide sufficient random bytes.
func generateID(r io.Reader) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type dispatchTask struct {
	ctx     context.Context
	url     string
	request deliveryRequest
	secret  string
	result  chan error
}

// Stats summarizes dispatcher queue and backpressure state.
type Stats struct {
	Workers         int   `json:"workers"`
	Queued          int   `json:"queued"`
	QueueCapacity   int   `json:"queueCapacity"`
	Active          int   `json:"active"`
	Dropped         int64 `json:"dropped"`
	MaxRetries      int   `json:"maxRetries"`
	TimeoutMillis   int64 `json:"timeoutMillis"`
	AllowInternal   bool  `json:"allowInternal"`
	QueueWaitMillis int64 `json:"queueWaitMillis"`
}

// Config for webhook dispatcher.
type Config struct {
	// Secret is the default HMAC secret for signature generation (optional).
	// Per-webhook secrets can override this via the Dispatch secret parameter.
	Secret string

	// MaxRetries is the maximum number of delivery attempts (default: 3).
	MaxRetries int

	// BaseDelay is the initial retry delay (default: 1s).
	BaseDelay time.Duration

	// MaxDelay is the maximum retry delay (default: 30s).
	MaxDelay time.Duration

	// Timeout is the HTTP request timeout (default: 30s).
	Timeout time.Duration

	// AllowInternal allows webhooks to internal/private addresses (default: false).
	// When false, delivery resolves once, pins dialing to the validated IP set, blocks
	// private IPs/localhost/link-local answers, and refuses redirect hops.
	// WARNING: Only enable in trusted environments.
	AllowInternal bool

	// MaxConcurrentDispatches limits the number of concurrent webhook workers (default: 100).
	MaxConcurrentDispatches int

	// MaxQueuedDispatches bounds the number of deliveries waiting behind active workers.
	// Defaults to 4x MaxConcurrentDispatches.
	MaxQueuedDispatches int
}

// Dispatcher manages webhook delivery with retry logic.
type Dispatcher struct {
	secret           string
	maxRetries       int
	baseDelay        time.Duration
	maxDelay         time.Duration
	timeout          time.Duration
	allowInternal    bool
	store            *Store
	resolver         ipResolver
	dialContext      dialContextFunc
	tlsConfig        *tls.Config
	workers          int
	queue            chan dispatchTask
	queueWaitTimeout time.Duration
	activeCount      atomic.Int64
	droppedCount     atomic.Int64
	stopCh           chan struct{}
	stopOnce         sync.Once
	wg               sync.WaitGroup
}

// NewDispatcher creates a new webhook dispatcher with the given configuration.
// If cfg is zero-valued, sensible defaults are used.
func NewDispatcher(cfg Config) *Dispatcher {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	baseDelay := cfg.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 1 * time.Second
	}

	maxDelay := cfg.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	maxConcurrent := cfg.MaxConcurrentDispatches
	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}

	maxQueued := cfg.MaxQueuedDispatches
	if maxQueued <= 0 {
		maxQueued = maxConcurrent * 4
	}

	baseDialer := &net.Dialer{Timeout: timeout}
	d := &Dispatcher{
		secret:           cfg.Secret,
		maxRetries:       maxRetries,
		baseDelay:        baseDelay,
		maxDelay:         maxDelay,
		timeout:          timeout,
		allowInternal:    cfg.AllowInternal,
		store:            nil,
		resolver:         systemIPResolver{resolver: net.DefaultResolver},
		dialContext:      baseDialer.DialContext,
		tlsConfig:        nil,
		workers:          maxConcurrent,
		queue:            make(chan dispatchTask, maxQueued),
		queueWaitTimeout: 5 * time.Second,
		stopCh:           make(chan struct{}),
	}
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.runWorker()
	}
	return d
}

// NewDispatcherWithStore creates a new webhook dispatcher with the given configuration and store.
// If cfg is zero-valued, sensible defaults are used.
func NewDispatcherWithStore(cfg Config, store *Store) *Dispatcher {
	d := NewDispatcher(cfg)
	d.store = store
	return d
}

// Store returns the webhook store associated with this dispatcher.
// Returns nil if no store was configured.
func (d *Dispatcher) Store() *Store {
	return d.store
}

// Dispatch sends a JSON webhook notification asynchronously.
// The secret parameter overrides the dispatcher's default secret if non-empty.
// URLs are validated against SSRF attacks before dispatching.
// When the queue stays full beyond the wait timeout, dispatches are dropped.
func (d *Dispatcher) Dispatch(ctx context.Context, url string, payload Payload, secret string) {
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		slog.Warn("webhook dispatch request build failed",
			"jobID", payload.JobID,
			"eventType", payload.EventType,
			"error", err)
		return
	}
	if err := d.submit(ctx, url, request, secret, false); err != nil {
		slog.Warn("webhook dispatch enqueue failed",
			"jobID", request.JobID,
			"eventType", request.EventType,
			"url", SanitizeURL(url),
			"error", apperrors.SafeMessage(err))
	}
}

// Deliver sends a JSON webhook notification synchronously and returns the delivery result.
// The secret parameter overrides the dispatcher's default secret if non-empty.
// URLs are validated against SSRF attacks before dispatching.
// When the queue stays full beyond the wait timeout, delivery fails.
func (d *Dispatcher) Deliver(ctx context.Context, url string, payload Payload, secret string) error {
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		return err
	}
	return d.submit(ctx, url, request, secret, true)
}

// DispatchExport sends an export-completed webhook asynchronously using the shared
// multipart export-delivery contract.
func (d *Dispatcher) DispatchExport(ctx context.Context, url string, payload Payload, content []byte, secret string) {
	request, err := exportDeliveryRequest(payload, content)
	if err != nil {
		slog.Warn("export webhook dispatch request build failed",
			"jobID", payload.JobID,
			"eventType", payload.EventType,
			"error", err)
		return
	}
	if err := d.submit(ctx, url, request, secret, false); err != nil {
		slog.Warn("export webhook dispatch enqueue failed",
			"jobID", request.JobID,
			"eventType", request.EventType,
			"url", SanitizeURL(url),
			"error", apperrors.SafeMessage(err))
	}
}

// DeliverExport sends an export-completed webhook synchronously using the shared
// multipart export-delivery contract.
func (d *Dispatcher) DeliverExport(ctx context.Context, url string, payload Payload, content []byte, secret string) error {
	request, err := exportDeliveryRequest(payload, content)
	if err != nil {
		return err
	}
	return d.submit(ctx, url, request, secret, true)
}

func (d *Dispatcher) submit(ctx context.Context, url string, request deliveryRequest, secret string, wait bool) error {
	if ctx == nil {
		ctx = context.Background()
	}

	task := dispatchTask{
		ctx:     ctx,
		url:     url,
		request: request,
		secret:  secret,
	}
	if wait {
		task.result = make(chan error, 1)
	}

	timer := time.NewTimer(d.queueWaitTimeout)
	defer timer.Stop()

	select {
	case <-d.stopCh:
		return apperrors.Internal("webhook dispatcher is closed")
	case <-ctx.Done():
		return ctx.Err()
	case d.queue <- task:
		if !wait {
			return nil
		}
	case <-timer.C:
		d.droppedCount.Add(1)
		err := apperrors.Internal("webhook dispatch queue is full")
		slog.Error("webhook dispatch dropped - queue full",
			"jobID", request.JobID,
			"eventType", request.EventType,
			"url", SanitizeURL(url))
		return err
	}

	select {
	case <-d.stopCh:
		return apperrors.Internal("webhook dispatcher is closed")
	case <-ctx.Done():
		return ctx.Err()
	case err := <-task.result:
		return err
	}
}

func (d *Dispatcher) runWorker() {
	defer d.wg.Done()
	for {
		select {
		case <-d.stopCh:
			return
		case task := <-d.queue:
			err := d.executeDelivery(task.ctx, task.url, task.request, task.secret)
			if task.result != nil {
				task.result <- err
			}
		}
	}
}

func (d *Dispatcher) executeDelivery(ctx context.Context, url string, request deliveryRequest, secret string) error {
	target, err := resolveDeliveryTarget(ctx, url, d.allowInternal, d.resolver)
	if err != nil {
		slog.Error("webhook URL failed validation",
			"url", SanitizeURL(url),
			"jobID", request.JobID,
			"error", apperrors.SafeMessage(err))
		return err
	}

	d.activeCount.Add(1)
	defer d.activeCount.Add(-1)

	client, closeClient := d.clientForTarget(target)
	defer closeClient()

	return d.dispatchWithRetry(ctx, client, url, request, secret)
}

func (d *Dispatcher) clientForTarget(target deliveryTarget) (*http.Client, func()) {
	return newPinnedHTTPClient(d.timeout, target, d.dialContext, d.tlsConfig)
}

// Stats returns the current dispatcher queue and backpressure metrics.
func (d *Dispatcher) Stats() Stats {
	return Stats{
		Workers:         d.workers,
		Queued:          len(d.queue),
		QueueCapacity:   cap(d.queue),
		Active:          int(d.activeCount.Load()),
		Dropped:         d.droppedCount.Load(),
		MaxRetries:      d.maxRetries,
		TimeoutMillis:   d.timeout.Milliseconds(),
		AllowInternal:   d.allowInternal,
		QueueWaitMillis: d.queueWaitTimeout.Milliseconds(),
	}
}

// DroppedCount returns the number of webhooks dropped due to queue backpressure.
func (d *Dispatcher) DroppedCount() int64 {
	return d.droppedCount.Load()
}

// Close stops accepting new deliveries, drains queued sync callers with an error,
// and waits for active workers to finish.
func (d *Dispatcher) Close() error {
	d.stopOnce.Do(func() {
		close(d.stopCh)
		for {
			select {
			case task := <-d.queue:
				if task.result != nil {
					task.result <- apperrors.Internal("webhook dispatcher is closed")
				}
			default:
				return
			}
		}
	})
	d.wg.Wait()
	return nil
}

// dispatchWithRetry attempts delivery with exponential backoff.
// This method should be called in a goroutine as it blocks during retries.
func (d *Dispatcher) dispatchWithRetry(ctx context.Context, client *http.Client, url string, request deliveryRequest, secret string) error {
	// Use per-dispatch secret if provided, otherwise fall back to dispatcher secret
	useSecret := secret
	if useSecret == "" {
		useSecret = d.secret
	}

	// Create delivery record if store is available
	var record *DeliveryRecord
	if d.store != nil {
		id, err := generateID(rand.Reader)
		if err != nil {
			slog.Error("failed to generate delivery record ID, continuing without tracking",
				"error", err,
				"jobID", request.JobID,
				"eventType", request.EventType)
		} else {
			record = &DeliveryRecord{
				ID:        id,
				EventID:   request.EventID,
				EventType: request.EventType,
				JobID:     request.JobID,
				URL:       url,
				Status:    DeliveryStatusPending,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := d.store.CreateRecord(ctx, record); err != nil {
				slog.Warn("failed to create delivery record", "error", err, "jobID", request.JobID)
				// Continue without tracking
				record = nil
			}
		}
	}

	var (
		lastErr      error
		err          error
		responseCode int
	)
	delay := d.baseDelay

	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		err, responseCode = d.attemptDelivery(ctx, client, url, request, useSecret)
		if err == nil {
			slog.Debug("webhook delivered successfully",
				"jobID", request.JobID,
				"eventType", request.EventType,
				"attempt", attempt,
				"url", url,
			)
			// Update record on success
			if record != nil {
				record.Status = DeliveryStatusDelivered
				now := time.Now()
				record.DeliveredAt = &now
				record.Attempts = attempt
				record.ResponseCode = responseCode
				if err := d.store.UpdateRecord(ctx, record); err != nil {
					slog.Warn("failed to update delivery record", "error", err, "jobID", request.JobID)
				}
			}
			return nil
		}

		lastErr = err
		slog.Warn("webhook delivery failed",
			"jobID", request.JobID,
			"eventType", request.EventType,
			"attempt", attempt,
			"maxRetries", d.maxRetries,
			"error", err,
			"url", url,
		)

		// Update record on attempt
		if record != nil {
			record.Attempts = attempt
			record.LastError = err.Error()
			if err := d.store.UpdateRecord(ctx, record); err != nil {
				slog.Warn("failed to update delivery record", "error", err, "jobID", request.JobID)
			}
		}

		if attempt < d.maxRetries {
			select {
			case <-ctx.Done():
				slog.Debug("webhook delivery canceled", "jobID", request.JobID, "attempt", attempt)
				// Update record on cancellation
				if record != nil {
					record.Status = DeliveryStatusFailed
					record.LastError = ctx.Err().Error()
					if err := d.store.UpdateRecord(ctx, record); err != nil {
						slog.Warn("failed to update delivery record", "error", err, "jobID", request.JobID)
					}
				}
				return ctx.Err()
			case <-time.After(delay):
				// Exponential backoff with jitter cap
				delay *= 2
				if delay > d.maxDelay {
					delay = d.maxDelay
				}
			}
		}
	}

	slog.Error("webhook delivery exhausted all retries",
		"jobID", request.JobID,
		"eventType", request.EventType,
		"attempts", d.maxRetries,
		"lastError", lastErr,
		"url", url,
	)

	// Update record on final failure
	if record != nil {
		record.Status = DeliveryStatusFailed
		record.LastError = lastErr.Error()
		record.Attempts = d.maxRetries
		if err := d.store.UpdateRecord(ctx, record); err != nil {
			slog.Warn("failed to update delivery record", "error", err, "jobID", request.JobID)
		}
	}

	return fmt.Errorf("exhausted retries: %w", lastErr)
}

// attemptDelivery makes a single HTTP POST attempt.
// Returns the error (if any) and the HTTP response code.
func (d *Dispatcher) attemptDelivery(ctx context.Context, client *http.Client, url string, request deliveryRequest, secret string) (error, int) {
	reqCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(request.Body))
	if err != nil {
		return fmt.Errorf("create request: %w", err), 0
	}

	req.Header.Set("Content-Type", request.ContentType)
	req.Header.Set("User-Agent", "SpartanScraper-Webhook/1.0")
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	if secret != "" {
		sig := d.signPayload(request.Body, secret)
		req.Header.Set("X-Webhook-Signature", sig)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err), 0
	}
	defer resp.Body.Close()

	// Consume response body to enable connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	// Consider 2xx status codes as success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode), resp.StatusCode
	}

	return nil, resp.StatusCode
}

// signPayload creates HMAC-SHA256 signature for payload.
func (d *Dispatcher) signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
