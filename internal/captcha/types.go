// Package captcha provides CAPTCHA detection and solving service integration.
//
// This package is responsible for:
//   - Detecting CAPTCHA challenges in HTML and headless-rendered pages
//   - Integrating with solving services (2captcha, anti-captcha)
//   - Managing CAPTCHA solving retry logic with exponential backoff
//   - Providing metrics for CAPTCHA detection and solving
//
// It does NOT handle:
//   - Browser automation for solving (delegated to headless fetchers)
//   - CAPTCHA solving via machine learning (only service-based solving)
//   - Rate limiting or request throttling (handled by fetch package)
//
// Detection heuristics:
//   - reCAPTCHA v2: g-recaptcha class, data-sitekey attribute
//   - reCAPTCHA v3: grecaptcha.execute() calls, data-action attribute
//   - hCaptcha: h-captcha class, data-sitekey attribute
//   - Turnstile: cf-turnstile class
//   - Image CAPTCHA: <img> with specific URL patterns, input fields with "captcha" in name/id
//
// Invariants:
//   - Detection confidence is always 0.0-1.0
//   - Solving service API keys are never logged
//   - Timeouts are always respected via context cancellation
package captcha

import (
	"context"
	"errors"
	"time"
)

// CaptchaType represents the type of CAPTCHA detected.
type CaptchaType string

const (
	// CaptchaTypeReCAPTCHAV2 is Google's reCAPTCHA v2 ("I'm not a robot").
	CaptchaTypeReCAPTCHAV2 CaptchaType = "recaptcha_v2"
	// CaptchaTypeReCAPTCHAV3 is Google's reCAPTCHA v3 (invisible scoring).
	CaptchaTypeReCAPTCHAV3 CaptchaType = "recaptcha_v3"
	// CaptchaTypeHCaptcha is hCaptcha from Intuition Machines.
	CaptchaTypeHCaptcha CaptchaType = "hcaptcha"
	// CaptchaTypeTurnstile is Cloudflare Turnstile.
	CaptchaTypeTurnstile CaptchaType = "turnstile"
	// CaptchaTypeImage is a generic image-based CAPTCHA.
	CaptchaTypeImage CaptchaType = "image"
	// CaptchaTypeUnknown is an unrecognized CAPTCHA type.
	CaptchaTypeUnknown CaptchaType = "unknown"
)

// DetectionReason represents why a CAPTCHA was detected.
type DetectionReason string

const (
	// ReasonClassMatch indicates CAPTCHA detected via CSS class.
	ReasonClassMatch DetectionReason = "class_match"
	// ReasonAttributeMatch indicates CAPTCHA detected via HTML attribute.
	ReasonAttributeMatch DetectionReason = "attribute_match"
	// ReasonScriptMatch indicates CAPTCHA detected via script reference.
	ReasonScriptMatch DetectionReason = "script_match"
	// ReasonIFrameMatch indicates CAPTCHA detected via iframe src.
	ReasonIFrameMatch DetectionReason = "iframe_match"
	// ReasonInputPattern indicates CAPTCHA detected via input field pattern.
	ReasonInputPattern DetectionReason = "input_pattern"
)

// CaptchaDetection represents a detected CAPTCHA challenge.
type CaptchaDetection struct {
	Type       CaptchaType       `json:"type"`       // Type of CAPTCHA detected
	Selector   string            `json:"selector"`   // CSS selector for the CAPTCHA element
	SiteKey    string            `json:"siteKey"`    // Site key for service-based CAPTCHAs
	Action     string            `json:"action"`     // Action parameter (reCAPTCHA v3)
	Score      float64           `json:"score"`      // Detection confidence (0.0-1.0)
	Reasons    []DetectionReason `json:"reasons"`    // Why this was detected
	HTML       string            `json:"html"`       // Snippet of relevant HTML
	PageURL    string            `json:"pageURL"`    // URL where CAPTCHA was detected
	DetectedAt time.Time         `json:"detectedAt"` // When detection occurred
}

// IsServiceBased returns true if the CAPTCHA type requires a solving service.
func (d *CaptchaDetection) IsServiceBased() bool {
	switch d.Type {
	case CaptchaTypeReCAPTCHAV2, CaptchaTypeReCAPTCHAV3,
		CaptchaTypeHCaptcha, CaptchaTypeTurnstile:
		return true
	default:
		return false
	}
}

// CaptchaSolver defines the interface for CAPTCHA solving services.
type CaptchaSolver interface {
	// Solve submits the CAPTCHA to the solving service and returns the token/answer.
	// The token can be used to bypass the CAPTCHA in the browser.
	Solve(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error)

	// GetBalance returns the current account balance in USD.
	GetBalance(ctx context.Context) (float64, error)

	// Name returns the solver service name.
	Name() string
}

// CaptchaConfig holds configuration for CAPTCHA handling.
type CaptchaConfig struct {
	// Enabled enables CAPTCHA detection.
	Enabled bool `json:"enabled,omitempty"`

	// AutoSolve automatically solve detected CAPTCHAs.
	AutoSolve bool `json:"autoSolve,omitempty"`

	// Service is the solving service to use ("2captcha" or "anticaptcha").
	Service string `json:"service,omitempty"`

	// APIKey is the API key for the solving service.
	// This is sensitive and should be stored securely.
	APIKey string `json:"apiKey,omitempty"`

	// CustomEndpoint is an optional custom service endpoint.
	CustomEndpoint string `json:"customEndpoint,omitempty"`

	// MaxRetries is the maximum solving attempts (default: 3).
	MaxRetries int `json:"maxRetries,omitempty"`

	// RetryDelay is the delay between retries (default: 5s).
	RetryDelay time.Duration `json:"retryDelay,omitempty"`

	// Timeout is the maximum time to wait for solution (default: 120s).
	Timeout time.Duration `json:"timeout,omitempty"`

	// PollingInterval is the poll interval for solution (default: 5s).
	PollingInterval time.Duration `json:"pollingInterval,omitempty"`

	// MinConfidence is the minimum detection confidence to trigger solving (default: 0.7).
	MinConfidence float64 `json:"minConfidence,omitempty"`
}

// DefaultCaptchaConfig returns a default configuration with sensible values.
func DefaultCaptchaConfig() CaptchaConfig {
	return CaptchaConfig{
		Enabled:         false,
		AutoSolve:       false,
		Service:         "",
		MaxRetries:      3,
		RetryDelay:      5 * time.Second,
		Timeout:         120 * time.Second,
		PollingInterval: 5 * time.Second,
		MinConfidence:   0.7,
	}
}

// Validate checks if the configuration is valid.
func (c *CaptchaConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.AutoSolve {
		if c.Service == "" {
			return errors.New("captcha service is required when auto-solve is enabled")
		}
		if c.APIKey == "" {
			return errors.New("captcha API key is required when auto-solve is enabled")
		}
		switch c.Service {
		case "2captcha", "anticaptcha":
			// Valid
		default:
			return errors.New("captcha service must be '2captcha' or 'anticaptcha'")
		}
	}

	if c.MaxRetries < 1 {
		c.MaxRetries = 3
	}
	if c.RetryDelay < 1*time.Second {
		c.RetryDelay = 5 * time.Second
	}
	if c.Timeout < 10*time.Second {
		c.Timeout = 120 * time.Second
	}
	if c.PollingInterval < 1*time.Second {
		c.PollingInterval = 5 * time.Second
	}
	if c.MinConfidence <= 0 || c.MinConfidence > 1 {
		c.MinConfidence = 0.7
	}

	return nil
}

// CaptchaMetrics tracks CAPTCHA-related metrics.
type CaptchaMetrics struct {
	DetectedCount  int                 `json:"detectedCount"`  // Total CAPTCHAs detected
	SolvedCount    int                 `json:"solvedCount"`    // Successfully solved
	FailedCount    int                 `json:"failedCount"`    // Failed to solve
	SkippedCount   int                 `json:"skippedCount"`   // Skipped (not solved)
	TotalSolveTime time.Duration       `json:"totalSolveTime"` // Cumulative solve time
	AvgSolveTime   time.Duration       `json:"avgSolveTime"`   // Average solve time
	ByType         map[CaptchaType]int `json:"byType"`         // Count by CAPTCHA type
}

// NewCaptchaMetrics creates a new metrics tracker.
func NewCaptchaMetrics() *CaptchaMetrics {
	return &CaptchaMetrics{
		ByType: make(map[CaptchaType]int),
	}
}

// RecordDetection records a CAPTCHA detection.
func (m *CaptchaMetrics) RecordDetection(captchaType CaptchaType) {
	m.DetectedCount++
	m.ByType[captchaType]++
}

// RecordSolve records a successful solve.
func (m *CaptchaMetrics) RecordSolve(duration time.Duration) {
	m.SolvedCount++
	m.TotalSolveTime += duration
	if m.SolvedCount > 0 {
		m.AvgSolveTime = m.TotalSolveTime / time.Duration(m.SolvedCount)
	}
}

// RecordFailure records a failed solve attempt.
func (m *CaptchaMetrics) RecordFailure() {
	m.FailedCount++
}

// RecordSkip records a skipped CAPTCHA.
func (m *CaptchaMetrics) RecordSkip() {
	m.SkippedCount++
}

// DetectionConfig holds configuration for the CAPTCHA detector.
type DetectionConfig struct {
	MinConfidence float64 // Minimum confidence threshold (default: 0.7)
}

// DefaultDetectionConfig returns default detection configuration.
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{
		MinConfidence: 0.7,
	}
}

// CAPTCHA detection errors.
var (
	// ErrCaptchaDetected is returned when a CAPTCHA is detected but not solved.
	ErrCaptchaDetected = errors.New("CAPTCHA detected")

	// ErrCaptchaTimeout is returned when solving times out.
	ErrCaptchaTimeout = errors.New("CAPTCHA solving timeout")

	// ErrCaptchaUnsolvable is returned when the service reports CAPTCHA as unsolvable.
	ErrCaptchaUnsolvable = errors.New("CAPTCHA unsolvable")

	// ErrInsufficientBalance is returned when the account has insufficient balance.
	ErrInsufficientBalance = errors.New("insufficient solving service balance")

	// ErrServiceError is returned for generic service errors.
	ErrServiceError = errors.New("CAPTCHA service error")

	// ErrInvalidAPIKey is returned when the API key is invalid.
	ErrInvalidAPIKey = errors.New("invalid CAPTCHA service API key")

	// ErrNoCaptchaFound is returned when no CAPTCHA is detected.
	ErrNoCaptchaFound = errors.New("no CAPTCHA found")
)
