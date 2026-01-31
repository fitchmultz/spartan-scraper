// Package captcha provides CAPTCHA detection and solving service integration.
//
// This file implements the base solver functionality with retry logic and
// exponential backoff for CAPTCHA solving services.
//
// It does NOT implement specific service APIs (those are in twocaptcha.go and
// anticaptcha.go).
package captcha

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// BaseSolver provides common solver functionality for CAPTCHA solving services.
type BaseSolver struct {
	config CaptchaConfig
}

// NewBaseSolver creates a base solver with the given config.
func NewBaseSolver(config CaptchaConfig) *BaseSolver {
	return &BaseSolver{
		config: config,
	}
}

// SolveWithRetry attempts to solve a CAPTCHA with exponential backoff.
// The submitFunc should submit the CAPTCHA to the service and return a task ID.
// The checkFunc should poll for the solution using the task ID.
func (s *BaseSolver) SolveWithRetry(
	ctx context.Context,
	detection CaptchaDetection,
	submitFunc func(context.Context) (string, error),
	checkFunc func(context.Context, string) (string, bool, error),
) (string, error) {
	start := time.Now()

	for attempt := 0; attempt < s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := s.calculateBackoff(attempt)
			slog.Debug("captcha solve retry",
				"attempt", attempt+1,
				"maxRetries", s.config.MaxRetries,
				"delay", delay,
			)

			select {
			case <-ctx.Done():
				return "", apperrors.Wrap(apperrors.KindValidation, "CAPTCHA solving cancelled", ctx.Err())
			case <-time.After(delay):
				// Continue to next attempt
			}
		}

		// Submit the CAPTCHA
		taskID, err := submitFunc(ctx)
		if err != nil {
			slog.Warn("captcha submission failed",
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}

		// Poll for solution
		solution, err := s.pollForSolution(ctx, taskID, checkFunc)
		if err != nil {
			if errors.Is(err, ErrCaptchaUnsolvable) {
				// Don't retry unsolvable CAPTCHAs
				return "", err
			}
			slog.Warn("captcha polling failed",
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}

		slog.Debug("captcha solved successfully",
			"attempt", attempt+1,
			"duration", time.Since(start),
		)
		return solution, nil
	}

	return "", apperrors.Wrap(apperrors.KindValidation, ErrCaptchaTimeout.Error(), fmt.Errorf("max retries exceeded"))
}

// pollForSolution polls the service for a solution until one is available
// or the timeout is reached.
func (s *BaseSolver) pollForSolution(
	ctx context.Context,
	taskID string,
	checkFunc func(context.Context, string) (string, bool, error),
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	ticker := time.NewTicker(s.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ErrCaptchaTimeout

		case <-ticker.C:
			solution, ready, err := checkFunc(ctx, taskID)
			if err != nil {
				return "", err
			}
			if ready {
				return solution, nil
			}
		}
	}
}

// calculateBackoff calculates the delay for the given retry attempt
// using exponential backoff with a cap.
func (s *BaseSolver) calculateBackoff(attempt int) time.Duration {
	// Exponential: base * 2^attempt, capped at 60 seconds
	base := float64(s.config.RetryDelay)
	backoff := base * math.Pow(2, float64(attempt))
	maxBackoff := 60.0 * float64(time.Second)

	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return time.Duration(backoff)
}

// ValidateDetection checks if a detection has the required fields for solving.
func (s *BaseSolver) ValidateDetection(detection CaptchaDetection) error {
	if detection.Score < s.config.MinConfidence {
		return apperrors.Validation(fmt.Sprintf("detection confidence %.2f below threshold %.2f",
			detection.Score, s.config.MinConfidence))
	}

	if detection.IsServiceBased() && detection.SiteKey == "" {
		return apperrors.Validation("site key required for service-based CAPTCHA")
	}

	return nil
}

// SolverFactory creates a CaptchaSolver based on the configuration.
func SolverFactory(config CaptchaConfig) (CaptchaSolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if !config.Enabled || !config.AutoSolve {
		return nil, apperrors.Validation("CAPTCHA solving not enabled")
	}

	switch config.Service {
	case "2captcha":
		return NewTwoCaptchaSolver(config), nil
	case "anticaptcha":
		return NewAntiCaptchaSolver(config), nil
	default:
		return nil, apperrors.Validation(fmt.Sprintf("unknown CAPTCHA service: %s", config.Service))
	}
}

// IsRetryableError determines if an error should trigger a retry.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Don't retry these errors
	if errors.Is(err, ErrCaptchaUnsolvable) {
		return false
	}
	if errors.Is(err, ErrInvalidAPIKey) {
		return false
	}
	if errors.Is(err, ErrInsufficientBalance) {
		return false
	}

	// Retry timeout errors (might succeed on retry)
	if errors.Is(err, ErrCaptchaTimeout) {
		return true
	}

	// Retry service errors
	if errors.Is(err, ErrServiceError) {
		return true
	}

	return true
}

// SafeError returns an error safe for logging (no API keys).
// It redacts potential API keys from error messages.
func SafeError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	// Redact potential API keys (32+ alphanumeric characters)
	// This is a simple heuristic to avoid logging sensitive data
	re := regexp.MustCompile(`[a-zA-Z0-9]{32,}`)
	msg = re.ReplaceAllString(msg, "[REDACTED]")

	return fmt.Errorf("%s", msg)
}
