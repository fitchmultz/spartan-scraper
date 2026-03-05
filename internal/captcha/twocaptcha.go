// Package captcha provides CAPTCHA detection and solving service integration.
//
// This file implements the 2captcha.com service integration.
//
// API Reference: https://2captcha.com/api-2captcha
//
// It does NOT:
//   - Handle browser automation (delegated to headless fetchers)
//   - Store API keys (passed via configuration)
//   - Retry indefinitely (respects MaxRetries configuration)
package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

const (
	twoCaptchaSubmitEndpoint = "http://2captcha.com/in.php"
	twoCaptchaResultEndpoint = "http://2captcha.com/res.php"
)

// TwoCaptchaSolver implements CaptchaSolver for 2captcha.com.
type TwoCaptchaSolver struct {
	BaseSolver
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewTwoCaptchaSolver creates a new 2captcha solver.
func NewTwoCaptchaSolver(config CaptchaConfig) *TwoCaptchaSolver {
	endpoint := twoCaptchaSubmitEndpoint
	if config.CustomEndpoint != "" {
		endpoint = config.CustomEndpoint
	}

	return &TwoCaptchaSolver{
		BaseSolver: BaseSolver{config: config},
		apiKey:     config.APIKey,
		endpoint:   endpoint,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Solve implements CaptchaSolver.
func (s *TwoCaptchaSolver) Solve(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error) {
	if err := s.ValidateDetection(detection); err != nil {
		return "", err
	}

	return s.SolveWithRetry(
		ctx,
		detection,
		func(ctx context.Context) (string, error) {
			return s.submit(ctx, detection, pageURL)
		},
		func(ctx context.Context, taskID string) (string, bool, error) {
			return s.getResult(ctx, taskID)
		},
	)
}

// GetBalance implements CaptchaSolver.
func (s *TwoCaptchaSolver) GetBalance(ctx context.Context) (float64, error) {
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("action", "getbalance")
	params.Set("json", "1")

	reqURL := fmt.Sprintf("%s?%s", twoCaptchaResultEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to create balance request", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to fetch balance", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to read balance response", err)
	}

	var result struct {
		Status  int     `json:"status"`
		Request float64 `json:"request"`
		Error   string  `json:"error_text,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to parse balance response", err)
	}

	if result.Status != 1 {
		return 0, s.parseError(result.Error)
	}

	return result.Request, nil
}

// Name implements CaptchaSolver.
func (s *TwoCaptchaSolver) Name() string {
	return "2captcha"
}

// submit submits the CAPTCHA to 2captcha and returns the task ID.
func (s *TwoCaptchaSolver) submit(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error) {
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("json", "1")
	params.Set("pageurl", pageURL)

	// Set method based on CAPTCHA type
	switch detection.Type {
	case CaptchaTypeReCAPTCHAV2:
		params.Set("method", "userrecaptcha")
		params.Set("googlekey", detection.SiteKey)
	case CaptchaTypeReCAPTCHAV3:
		params.Set("method", "userrecaptcha")
		params.Set("googlekey", detection.SiteKey)
		params.Set("version", "v3")
		if detection.Action != "" {
			params.Set("action", detection.Action)
		}
	case CaptchaTypeHCaptcha:
		params.Set("method", "hcaptcha")
		params.Set("sitekey", detection.SiteKey)
	case CaptchaTypeTurnstile:
		params.Set("method", "turnstile")
		params.Set("sitekey", detection.SiteKey)
	case CaptchaTypeImage:
		// For image CAPTCHA, we would need the image data
		return "", apperrors.Validation("image CAPTCHA requires image data (not yet implemented)")
	default:
		return "", apperrors.Validation(fmt.Sprintf("unsupported CAPTCHA type: %s", detection.Type))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to create submit request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	slog.Debug("submitting captcha to 2captcha",
		"type", detection.Type,
		"pageURL", pageURL,
	)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to submit captcha", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to read submit response", err)
	}

	var result struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
		Error   string `json:"error_text,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to parse submit response", err)
	}

	if result.Status != 1 {
		return "", s.parseError(result.Error)
	}

	slog.Debug("captcha submitted successfully", "taskID", result.Request)
	return result.Request, nil
}

// getResult polls for the solution.
func (s *TwoCaptchaSolver) getResult(ctx context.Context, taskID string) (string, bool, error) {
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("action", "get")
	params.Set("id", taskID)
	params.Set("json", "1")

	reqURL := fmt.Sprintf("%s?%s", twoCaptchaResultEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", false, apperrors.Wrap(apperrors.KindInternal, "failed to create result request", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", false, apperrors.Wrap(apperrors.KindInternal, "failed to fetch result", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, apperrors.Wrap(apperrors.KindInternal, "failed to read result response", err)
	}

	// Check for "CAPCHA_NOT_READY" (not JSON)
	if strings.Contains(string(body), "CAPCHA_NOT_READY") {
		return "", false, nil
	}

	var result struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
		Error   string `json:"error_text,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, apperrors.Wrap(apperrors.KindInternal, "failed to parse result response", err)
	}

	if result.Status != 1 {
		return "", false, s.parseError(result.Error)
	}

	return result.Request, true, nil
}

// parseError converts 2captcha error codes to Go errors.
func (s *TwoCaptchaSolver) parseError(errorCode string) error {
	switch errorCode {
	case "ERROR_WRONG_USER_KEY", "ERROR_KEY_DOES_NOT_EXIST":
		return ErrInvalidAPIKey
	case "ERROR_ZERO_BALANCE":
		return ErrInsufficientBalance
	case "ERROR_CAPTCHA_UNSOLVABLE":
		return ErrCaptchaUnsolvable
	case "ERROR_NO_SLOT_AVAILABLE":
		return ErrServiceError
	case "ERROR_TOO_BIG_CAPTCHA_FILE":
		return apperrors.Validation("CAPTCHA image too large")
	case "ERROR_IMAGE_TYPE_NOT_SUPPORTED":
		return apperrors.Validation("CAPTCHA image type not supported")
	default:
		if strings.HasPrefix(errorCode, "ERROR_") {
			return fmt.Errorf("%w: %s", ErrServiceError, errorCode)
		}
		return nil
	}
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (s *TwoCaptchaSolver) SetHTTPClient(client *http.Client) {
	s.client = client
}
