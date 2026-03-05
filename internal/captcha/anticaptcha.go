// Package captcha provides CAPTCHA detection and solving service integration.
//
// This file implements the anti-captcha.com service integration.
//
// API Reference: https://anti-captcha.com/apidoc/
//
// It does NOT:
//   - Handle browser automation (delegated to headless fetchers)
//   - Store API keys (passed via configuration)
//   - Retry indefinitely (respects MaxRetries configuration)
package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

const (
	antiCaptchaEndpoint = "https://api.anti-captcha.com/"
)

// AntiCaptchaSolver implements CaptchaSolver for anti-captcha.com.
type AntiCaptchaSolver struct {
	BaseSolver
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewAntiCaptchaSolver creates a new anti-captcha solver.
func NewAntiCaptchaSolver(config CaptchaConfig) *AntiCaptchaSolver {
	endpoint := antiCaptchaEndpoint
	if config.CustomEndpoint != "" {
		endpoint = config.CustomEndpoint
	}

	return &AntiCaptchaSolver{
		BaseSolver: BaseSolver{config: config},
		apiKey:     config.APIKey,
		endpoint:   endpoint,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Solve implements CaptchaSolver.
func (s *AntiCaptchaSolver) Solve(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error) {
	if err := s.ValidateDetection(detection); err != nil {
		return "", err
	}

	return s.SolveWithRetry(
		ctx,
		detection,
		func(ctx context.Context) (string, error) {
			return s.createTask(ctx, detection, pageURL)
		},
		func(ctx context.Context, taskID string) (string, bool, error) {
			return s.getTaskResult(ctx, taskID)
		},
	)
}

// GetBalance implements CaptchaSolver.
func (s *AntiCaptchaSolver) GetBalance(ctx context.Context) (float64, error) {
	reqBody := map[string]interface{}{
		"clientKey": s.apiKey,
	}

	body, err := s.postJSON(ctx, "getBalance", reqBody)
	if err != nil {
		return 0, err
	}

	var result struct {
		ErrorID          int     `json:"errorId"`
		ErrorCode        string  `json:"errorCode,omitempty"`
		ErrorDescription string  `json:"errorDescription,omitempty"`
		Balance          float64 `json:"balance"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to parse balance response", err)
	}

	if result.ErrorID != 0 {
		return 0, s.parseError(result.ErrorCode, result.ErrorDescription)
	}

	return result.Balance, nil
}

// Name implements CaptchaSolver.
func (s *AntiCaptchaSolver) Name() string {
	return "anticaptcha"
}

// createTask creates a new task for solving.
func (s *AntiCaptchaSolver) createTask(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error) {
	task := s.buildTask(detection, pageURL)

	reqBody := map[string]interface{}{
		"clientKey": s.apiKey,
		"task":      task,
	}

	body, err := s.postJSON(ctx, "createTask", reqBody)
	if err != nil {
		return "", err
	}

	var result struct {
		ErrorID          int    `json:"errorId"`
		ErrorCode        string `json:"errorCode,omitempty"`
		ErrorDescription string `json:"errorDescription,omitempty"`
		TaskID           int64  `json:"taskId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to parse createTask response", err)
	}

	if result.ErrorID != 0 {
		return "", s.parseError(result.ErrorCode, result.ErrorDescription)
	}

	taskID := fmt.Sprintf("%d", result.TaskID)
	slog.Debug("captcha task created", "taskID", taskID)
	return taskID, nil
}

// buildTask builds the task payload based on CAPTCHA type.
func (s *AntiCaptchaSolver) buildTask(detection CaptchaDetection, pageURL string) map[string]interface{} {
	switch detection.Type {
	case CaptchaTypeReCAPTCHAV2:
		return map[string]interface{}{
			"type":       "RecaptchaV2TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": detection.SiteKey,
		}
	case CaptchaTypeReCAPTCHAV3:
		task := map[string]interface{}{
			"type":       "RecaptchaV3TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": detection.SiteKey,
			"minScore":   0.3,
		}
		if detection.Action != "" {
			task["pageAction"] = detection.Action
		}
		return task
	case CaptchaTypeHCaptcha:
		return map[string]interface{}{
			"type":       "HCaptchaTaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": detection.SiteKey,
		}
	case CaptchaTypeTurnstile:
		return map[string]interface{}{
			"type":       "TurnstileTaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": detection.SiteKey,
		}
	default:
		// Fallback to a generic task type
		return map[string]interface{}{
			"type":       "RecaptchaV2TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": detection.SiteKey,
		}
	}
}

// getTaskResult polls for the task result.
func (s *AntiCaptchaSolver) getTaskResult(ctx context.Context, taskID string) (string, bool, error) {
	var taskIDInt int64
	fmt.Sscanf(taskID, "%d", &taskIDInt)

	reqBody := map[string]interface{}{
		"clientKey": s.apiKey,
		"taskId":    taskIDInt,
	}

	body, err := s.postJSON(ctx, "getTaskResult", reqBody)
	if err != nil {
		return "", false, err
	}

	var result struct {
		ErrorID          int    `json:"errorId"`
		ErrorCode        string `json:"errorCode,omitempty"`
		ErrorDescription string `json:"errorDescription,omitempty"`
		Status           string `json:"status"`
		Solution         struct {
			GRecaptchaResponse string `json:"gRecaptchaResponse,omitempty"`
			Token              string `json:"token,omitempty"`
			Text               string `json:"text,omitempty"`
		} `json:"solution,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, apperrors.Wrap(apperrors.KindInternal, "failed to parse getTaskResult response", err)
	}

	if result.ErrorID != 0 {
		return "", false, s.parseError(result.ErrorCode, result.ErrorDescription)
	}

	if result.Status == "processing" {
		return "", false, nil
	}

	if result.Status == "ready" {
		// Extract solution based on CAPTCHA type
		solution := result.Solution.GRecaptchaResponse
		if solution == "" {
			solution = result.Solution.Token
		}
		if solution == "" {
			solution = result.Solution.Text
		}
		return solution, true, nil
	}

	return "", false, fmt.Errorf("%w: unknown status %s", ErrServiceError, result.Status)
}

// postJSON makes a POST request with JSON body.
func (s *AntiCaptchaSolver) postJSON(ctx context.Context, method string, payload interface{}) ([]byte, error) {
	url := s.endpoint + method

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read response", err)
	}

	return body, nil
}

// parseError converts anti-captcha error codes to Go errors.
func (s *AntiCaptchaSolver) parseError(errorCode, errorDescription string) error {
	switch errorCode {
	case "ERROR_KEY_DOES_NOT_EXIST", "ERROR_IP_NOT_ALLOWED", "ERROR_IP_BANNED":
		return ErrInvalidAPIKey
	case "ERROR_ZERO_BALANCE":
		return ErrInsufficientBalance
	case "ERROR_CAPTCHA_UNSOLVABLE":
		return ErrCaptchaUnsolvable
	case "ERROR_NO_SLOT_AVAILABLE":
		return ErrServiceError
	case "ERROR_TASK_NOT_FOUND":
		return apperrors.NotFound("task not found")
	case "ERROR_TOO_MANY_REQUESTS":
		return ErrServiceError
	default:
		if errorDescription != "" {
			return fmt.Errorf("%w: %s - %s", ErrServiceError, errorCode, errorDescription)
		}
		return fmt.Errorf("%w: %s", ErrServiceError, errorCode)
	}
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (s *AntiCaptchaSolver) SetHTTPClient(client *http.Client) {
	s.client = client
}
