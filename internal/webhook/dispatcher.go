// Package webhook provides webhook dispatch functionality for job notifications.
// It handles HTTP delivery with HMAC-SHA256 signatures and exponential backoff retry.
//
// The dispatcher is designed to be non-blocking - webhook deliveries happen
// asynchronously in goroutines to avoid delaying job status updates.
//
// Security considerations:
//   - Webhook secrets are passed per-dispatch (from job params), not stored in the dispatcher
//   - HMAC-SHA256 signatures are generated when a secret is provided
//   - Timeouts prevent hanging connections from blocking the system
//   - Retries use exponential backoff to avoid overwhelming receiving endpoints
//
// This package does NOT:
//   - Store webhook delivery state persistently
//   - Guarantee exactly-once delivery (at-least-once is attempted)
//   - Validate webhook URLs against SSRF attacks (callers should validate)
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// EventType represents the type of webhook event.
type EventType string

const (
	EventJobCreated   EventType = "job.created"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
)

// Payload represents the webhook notification body.
type Payload struct {
	EventID     string     `json:"eventId"`
	EventType   EventType  `json:"eventType"`
	Timestamp   time.Time  `json:"timestamp"`
	JobID       string     `json:"jobId"`
	JobKind     string     `json:"jobKind"`
	Status      string     `json:"status"`
	PrevStatus  string     `json:"prevStatus,omitempty"`
	Error       string     `json:"error,omitempty"`
	ResultPath  string     `json:"resultPath,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
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
}

// Dispatcher manages webhook delivery with retry logic.
type Dispatcher struct {
	client     *http.Client
	secret     string
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	timeout    time.Duration
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

	return &Dispatcher{
		client:     &http.Client{Timeout: timeout},
		secret:     cfg.Secret,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
		timeout:    timeout,
	}
}

// Dispatch sends a webhook notification asynchronously.
// It executes in a goroutine and returns immediately.
// The secret parameter overrides the dispatcher's default secret if non-empty.
func (d *Dispatcher) Dispatch(ctx context.Context, url string, payload Payload, secret string) {
	go d.dispatchWithRetry(ctx, url, payload, secret)
}

// dispatchWithRetry attempts delivery with exponential backoff.
// This method should be called in a goroutine as it blocks during retries.
func (d *Dispatcher) dispatchWithRetry(ctx context.Context, url string, payload Payload, secret string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal webhook payload", "error", err, "jobID", payload.JobID)
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Use per-dispatch secret if provided, otherwise fall back to dispatcher secret
	useSecret := secret
	if useSecret == "" {
		useSecret = d.secret
	}

	var lastErr error
	delay := d.baseDelay

	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		err := d.attemptDelivery(ctx, url, body, useSecret)
		if err == nil {
			slog.Debug("webhook delivered successfully",
				"jobID", payload.JobID,
				"eventType", payload.EventType,
				"attempt", attempt,
				"url", url,
			)
			return nil
		}

		lastErr = err
		slog.Warn("webhook delivery failed",
			"jobID", payload.JobID,
			"eventType", payload.EventType,
			"attempt", attempt,
			"maxRetries", d.maxRetries,
			"error", err,
			"url", url,
		)

		if attempt < d.maxRetries {
			select {
			case <-ctx.Done():
				slog.Debug("webhook delivery canceled", "jobID", payload.JobID, "attempt", attempt)
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
		"jobID", payload.JobID,
		"eventType", payload.EventType,
		"attempts", d.maxRetries,
		"lastError", lastErr,
		"url", url,
	)
	return fmt.Errorf("exhausted retries: %w", lastErr)
}

// attemptDelivery makes a single HTTP POST attempt.
func (d *Dispatcher) attemptDelivery(ctx context.Context, url string, body []byte, secret string) error {
	reqCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SpartanScraper-Webhook/1.0")

	if secret != "" {
		sig := d.signPayload(body, secret)
		req.Header.Set("X-Webhook-Signature", sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Consume response body to enable connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	// Consider 2xx status codes as success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// signPayload creates HMAC-SHA256 signature for payload.
func (d *Dispatcher) signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// ShouldSendEvent checks if the given event type matches the configured events.
// Supported events: "completed", "failed", "canceled", "started", "all".
// An empty configuredEvents slice defaults to ["completed"].
func ShouldSendEvent(eventType EventType, status string, configuredEvents []string) bool {
	if len(configuredEvents) == 0 {
		// Default: only send on terminal states (completed)
		return eventType == EventJobCompleted
	}

	for _, e := range configuredEvents {
		switch e {
		case "all":
			return true
		case "started":
			if eventType == EventJobStarted {
				return true
			}
		case "created":
			if eventType == EventJobCreated {
				return true
			}
		case "completed":
			if eventType == EventJobCompleted {
				return true
			}
		case "failed":
			if eventType == EventJobCompleted && status == "failed" {
				return true
			}
		case "canceled":
			if eventType == EventJobCompleted && status == "canceled" {
				return true
			}
		case "succeeded":
			if eventType == EventJobCompleted && status == "succeeded" {
				return true
			}
		}
	}

	return false
}
