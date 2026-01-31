// Package captcha provides CAPTCHA detection and solving service integration.
//
// This file contains unit tests for the captcha package.
package captcha

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewDetector tests the detector creation.
func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if d.config.MinConfidence != 0.7 {
		t.Errorf("expected default MinConfidence 0.7, got %f", d.config.MinConfidence)
	}
}

// TestNewDetectorWithConfig tests creating detector with custom config.
func TestNewDetectorWithConfig(t *testing.T) {
	config := DetectionConfig{MinConfidence: 0.9}
	d := NewDetectorWithConfig(config)
	if d == nil {
		t.Fatal("NewDetectorWithConfig returned nil")
	}
	if d.config.MinConfidence != 0.9 {
		t.Errorf("expected MinConfidence 0.9, got %f", d.config.MinConfidence)
	}
}

// TestDetectReCAPTCHAV2 tests detection of reCAPTCHA v2.
func TestDetectReCAPTCHAV2(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="g-recaptcha" data-sitekey="6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"></div>
<script src="https://www.google.com/recaptcha/api.js"></script>
</body>
</html>`

	d := NewDetector()
	detection, err := d.Detect(html, "https://example.com")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if detection == nil {
		t.Fatal("Expected CAPTCHA detection, got nil")
	}
	if detection.Type != CaptchaTypeReCAPTCHAV2 {
		t.Errorf("expected type %s, got %s", CaptchaTypeReCAPTCHAV2, detection.Type)
	}
	if detection.SiteKey != "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI" {
		t.Errorf("expected site key, got %s", detection.SiteKey)
	}
}

// TestDetectHCaptcha tests detection of hCaptcha.
func TestDetectHCaptcha(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="h-captcha" data-sitekey="10000000-ffff-ffff-ffff-000000000001"></div>
<script src="https://js.hcaptcha.com/1/api.js"></script>
</body>
</html>`

	d := NewDetector()
	detection, err := d.Detect(html, "https://example.com")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if detection == nil {
		t.Fatal("Expected CAPTCHA detection, got nil")
	}
	if detection.Type != CaptchaTypeHCaptcha {
		t.Errorf("expected type %s, got %s", CaptchaTypeHCaptcha, detection.Type)
	}
}

// TestDetectTurnstile tests detection of Cloudflare Turnstile.
func TestDetectTurnstile(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="cf-turnstile" data-sitekey="1x00000000000000000000AA"></div>
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>
</body>
</html>`

	d := NewDetector()
	detection, err := d.Detect(html, "https://example.com")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if detection == nil {
		t.Fatal("Expected CAPTCHA detection, got nil")
	}
	if detection.Type != CaptchaTypeTurnstile {
		t.Errorf("expected type %s, got %s", CaptchaTypeTurnstile, detection.Type)
	}
}

// TestDetectImageCaptcha tests detection of image CAPTCHA.
func TestDetectImageCaptcha(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<img src="/captcha.php" alt="Security Code">
<input type="text" name="captcha_code" id="captcha_input">
<label>Enter the security code:</label>
</body>
</html>`

	d := NewDetector()
	detection, err := d.Detect(html, "https://example.com")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if detection == nil {
		t.Fatal("Expected CAPTCHA detection, got nil")
	}
	if detection.Type != CaptchaTypeImage {
		t.Errorf("expected type %s, got %s", CaptchaTypeImage, detection.Type)
	}
}

// TestDetectNoCaptcha tests that no CAPTCHA is detected in clean HTML.
func TestDetectNoCaptcha(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
<p>This is a normal page without CAPTCHA.</p>
</body>
</html>`

	d := NewDetector()
	detection, err := d.Detect(html, "https://example.com")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if detection != nil {
		t.Errorf("expected no detection, got %v", detection)
	}
}

// TestCaptchaDetectionIsServiceBased tests the IsServiceBased method.
func TestCaptchaDetectionIsServiceBased(t *testing.T) {
	tests := []struct {
		captchaType CaptchaType
		expected    bool
	}{
		{CaptchaTypeReCAPTCHAV2, true},
		{CaptchaTypeReCAPTCHAV3, true},
		{CaptchaTypeHCaptcha, true},
		{CaptchaTypeTurnstile, true},
		{CaptchaTypeImage, false},
		{CaptchaTypeUnknown, false},
	}

	for _, tt := range tests {
		d := &CaptchaDetection{Type: tt.captchaType}
		if got := d.IsServiceBased(); got != tt.expected {
			t.Errorf("IsServiceBased() for %s = %v, want %v", tt.captchaType, got, tt.expected)
		}
	}
}

// TestDefaultCaptchaConfig tests the default configuration.
func TestDefaultCaptchaConfig(t *testing.T) {
	cfg := DefaultCaptchaConfig()

	if cfg.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if cfg.AutoSolve {
		t.Error("expected AutoSolve to be false by default")
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.RetryDelay != 5*time.Second {
		t.Errorf("expected RetryDelay 5s, got %v", cfg.RetryDelay)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected Timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.PollingInterval != 5*time.Second {
		t.Errorf("expected PollingInterval 5s, got %v", cfg.PollingInterval)
	}
	if cfg.MinConfidence != 0.7 {
		t.Errorf("expected MinConfidence 0.7, got %f", cfg.MinConfidence)
	}
}

// TestCaptchaConfigValidate tests configuration validation.
func TestCaptchaConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  CaptchaConfig
		wantErr bool
	}{
		{
			name:    "disabled config is valid",
			config:  CaptchaConfig{Enabled: false},
			wantErr: false,
		},
		{
			name: "enabled without auto-solve is valid",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: false,
			},
			wantErr: false,
		},
		{
			name: "auto-solve without service is invalid",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				APIKey:    "test-key",
			},
			wantErr: true,
		},
		{
			name: "auto-solve without api key is invalid",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "2captcha",
			},
			wantErr: true,
		},
		{
			name: "valid 2captcha config",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "2captcha",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
		{
			name: "valid anticaptcha config",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "anticaptcha",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
		{
			name: "invalid service",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "invalid",
				APIKey:    "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewCaptchaMetrics tests metrics creation.
func TestNewCaptchaMetrics(t *testing.T) {
	m := NewCaptchaMetrics()
	if m == nil {
		t.Fatal("NewCaptchaMetrics returned nil")
	}
	if m.ByType == nil {
		t.Error("ByType map not initialized")
	}
}

// TestCaptchaMetricsRecordDetection tests recording detections.
func TestCaptchaMetricsRecordDetection(t *testing.T) {
	m := NewCaptchaMetrics()

	m.RecordDetection(CaptchaTypeReCAPTCHAV2)
	if m.DetectedCount != 1 {
		t.Errorf("expected DetectedCount 1, got %d", m.DetectedCount)
	}
	if m.ByType[CaptchaTypeReCAPTCHAV2] != 1 {
		t.Errorf("expected ByType[recaptcha_v2] 1, got %d", m.ByType[CaptchaTypeReCAPTCHAV2])
	}

	m.RecordDetection(CaptchaTypeHCaptcha)
	if m.DetectedCount != 2 {
		t.Errorf("expected DetectedCount 2, got %d", m.DetectedCount)
	}
}

// TestCaptchaMetricsRecordSolve tests recording solves.
func TestCaptchaMetricsRecordSolve(t *testing.T) {
	m := NewCaptchaMetrics()

	m.RecordSolve(10 * time.Second)
	if m.SolvedCount != 1 {
		t.Errorf("expected SolvedCount 1, got %d", m.SolvedCount)
	}
	if m.AvgSolveTime != 10*time.Second {
		t.Errorf("expected AvgSolveTime 10s, got %v", m.AvgSolveTime)
	}

	m.RecordSolve(20 * time.Second)
	if m.SolvedCount != 2 {
		t.Errorf("expected SolvedCount 2, got %d", m.SolvedCount)
	}
	// Average of 10s and 20s = 15s
	if m.AvgSolveTime != 15*time.Second {
		t.Errorf("expected AvgSolveTime 15s, got %v", m.AvgSolveTime)
	}
}

// TestIsRetryableError tests the retry logic.
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"timeout error", ErrCaptchaTimeout, true},
		{"service error", ErrServiceError, true},
		{"unsolvable error", ErrCaptchaUnsolvable, false},
		{"invalid key error", ErrInvalidAPIKey, false},
		{"insufficient balance", ErrInsufficientBalance, false},
		{"generic error", errors.New("something went wrong"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.want {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestSolverFactory tests the solver factory.
func TestSolverFactory(t *testing.T) {
	tests := []struct {
		name    string
		config  CaptchaConfig
		wantErr bool
	}{
		{
			name: "disabled",
			config: CaptchaConfig{
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "enabled but no auto-solve",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: false,
			},
			wantErr: true,
		},
		{
			name: "2captcha",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "2captcha",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
		{
			name: "anticaptcha",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "anticaptcha",
				APIKey:    "test-key",
			},
			wantErr: false,
		},
		{
			name: "unknown service",
			config: CaptchaConfig{
				Enabled:   true,
				AutoSolve: true,
				Service:   "unknown",
				APIKey:    "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solver, err := SolverFactory(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SolverFactory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && solver == nil {
				t.Error("expected solver, got nil")
			}
		})
	}
}

// TestBaseSolverValidateDetection tests detection validation.
func TestBaseSolverValidateDetection(t *testing.T) {
	s := NewBaseSolver(CaptchaConfig{MinConfidence: 0.7})

	tests := []struct {
		name       string
		detection  CaptchaDetection
		wantErr    bool
		errContain string
	}{
		{
			name: "valid detection",
			detection: CaptchaDetection{
				Type:    CaptchaTypeReCAPTCHAV2,
				Score:   0.8,
				SiteKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "low confidence",
			detection: CaptchaDetection{
				Type:    CaptchaTypeReCAPTCHAV2,
				Score:   0.5,
				SiteKey: "test-key",
			},
			wantErr:    true,
			errContain: "confidence",
		},
		{
			name: "missing site key",
			detection: CaptchaDetection{
				Type:  CaptchaTypeReCAPTCHAV2,
				Score: 0.8,
			},
			wantErr:    true,
			errContain: "site key",
		},
		{
			name: "image captcha no site key needed",
			detection: CaptchaDetection{
				Type:  CaptchaTypeImage,
				Score: 0.8,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ValidateDetection(tt.detection)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDetection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTwoCaptchaSolverName tests the solver name.
func TestTwoCaptchaSolverName(t *testing.T) {
	s := NewTwoCaptchaSolver(CaptchaConfig{})
	if s.Name() != "2captcha" {
		t.Errorf("expected name '2captcha', got %s", s.Name())
	}
}

// TestAntiCaptchaSolverName tests the solver name.
func TestAntiCaptchaSolverName(t *testing.T) {
	s := NewAntiCaptchaSolver(CaptchaConfig{})
	if s.Name() != "anticaptcha" {
		t.Errorf("expected name 'anticaptcha', got %s", s.Name())
	}
}

// TestContainsCaptchaKeyword tests the helper function.
func TestContainsCaptchaKeyword(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"captcha", true},
		{"CAPTCHA", true},
		{"Captcha", true},
		{"verify", true},
		{"challenge", true},
		{"security code", true},
		{"normal text", false},
		{"hello world", false},
	}

	for _, tt := range tests {
		if got := containsCaptchaKeyword(tt.input); got != tt.want {
			t.Errorf("containsCaptchaKeyword(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestExtractSiteKey tests the site key extraction.
func TestExtractSiteKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`sitekey: "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"`, "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"},
		{`data-sitekey="6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"`, "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"},
		{"no site key here", ""},
	}

	for _, tt := range tests {
		if got := extractSiteKey(tt.input); got != tt.want {
			t.Errorf("extractSiteKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestSafeError tests error sanitization.
func TestSafeError(t *testing.T) {
	err := errors.New("error with key: abc123")
	safe := SafeError(err)
	if safe == nil {
		t.Fatal("SafeError returned nil")
	}
	// The safe error should not panic and should return an error
	_ = safe.Error()
}

// TestSafeErrorNil tests that nil errors are handled.
func TestSafeErrorNil(t *testing.T) {
	result := SafeError(nil)
	if result != nil {
		t.Error("SafeError(nil) should return nil")
	}
}

// MockSolver is a mock implementation for testing.
type MockSolver struct {
	solveFunc func(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error)
	balance   float64
}

func (m *MockSolver) Solve(ctx context.Context, detection CaptchaDetection, pageURL string) (string, error) {
	if m.solveFunc != nil {
		return m.solveFunc(ctx, detection, pageURL)
	}
	return "mock-solution", nil
}

func (m *MockSolver) GetBalance(ctx context.Context) (float64, error) {
	return m.balance, nil
}

func (m *MockSolver) Name() string {
	return "mock"
}

// TestBaseSolverSolveWithRetry tests the retry logic.
func TestBaseSolverSolveWithRetry(t *testing.T) {
	config := CaptchaConfig{
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		Timeout:         5 * time.Second,
		PollingInterval: 100 * time.Millisecond,
	}
	s := NewBaseSolver(config)

	// Test successful solve on first attempt
	solution, err := s.SolveWithRetry(
		context.Background(),
		CaptchaDetection{Type: CaptchaTypeReCAPTCHAV2, Score: 0.9},
		func(ctx context.Context) (string, error) {
			return "task-123", nil
		},
		func(ctx context.Context, taskID string) (string, bool, error) {
			return "solution", true, nil
		},
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if solution != "solution" {
		t.Errorf("expected solution 'solution', got %s", solution)
	}
}

// TestBaseSolverSolveWithRetryTimeout tests timeout handling.
func TestBaseSolverSolveWithRetryTimeout(t *testing.T) {
	config := CaptchaConfig{
		MaxRetries:      1,
		RetryDelay:      50 * time.Millisecond,
		Timeout:         100 * time.Millisecond,
		PollingInterval: 50 * time.Millisecond,
	}
	s := NewBaseSolver(config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := s.SolveWithRetry(
		ctx,
		CaptchaDetection{Type: CaptchaTypeReCAPTCHAV2, Score: 0.9},
		func(ctx context.Context) (string, error) {
			return "task-123", nil
		},
		func(ctx context.Context, taskID string) (string, bool, error) {
			// Never return ready
			return "", false, nil
		},
	)
	if err == nil {
		t.Error("expected timeout error")
	}
}

// TestBaseSolverCalculateBackoff tests backoff calculation.
func TestBaseSolverCalculateBackoff(t *testing.T) {
	config := CaptchaConfig{RetryDelay: 1 * time.Second}
	s := NewBaseSolver(config)

	tests := []struct {
		attempt int
		max     time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{10, 60 * time.Second}, // capped at 60s
	}

	for _, tt := range tests {
		backoff := s.calculateBackoff(tt.attempt)
		if backoff > tt.max {
			t.Errorf("attempt %d: backoff %v > max %v", tt.attempt, backoff, tt.max)
		}
	}
}
