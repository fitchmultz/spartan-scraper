// Package fetch provides tests for circuit breaker integration with HostLimiter.
// Tests cover circuit breaker construction, request blocking, result recording,
// and rate limit header-based adjustments.
// Does NOT test standalone circuit breaker (see circuit_breaker_test.go).
package fetch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewHostLimiterWithCircuitBreaker(t *testing.T) {
	t.Run("circuit breaker disabled by default", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		if l.IsCircuitBreakerEnabled() {
			t.Error("expected circuit breaker to be disabled by default")
		}
	})

	t.Run("circuit breaker enabled", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 3,
			ResetTimeout:     30 * time.Second,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(10, 10, cb)

		if !l.IsCircuitBreakerEnabled() {
			t.Error("expected circuit breaker to be enabled")
		}
		if l.GetCircuitBreaker() != cb {
			t.Error("expected circuit breaker to be the same instance")
		}
	})

	t.Run("nil circuit breaker does not panic", func(t *testing.T) {
		l := NewHostLimiterWithCircuitBreaker(10, 10, nil)
		if l.IsCircuitBreakerEnabled() {
			t.Error("expected circuit breaker to be disabled with nil")
		}
	})

	t.Run("adaptive with circuit breaker", func(t *testing.T) {
		adaptiveCfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  10,
		}
		cbCfg := CircuitBreakerConfig{Enabled: true}
		cb := NewCircuitBreaker(cbCfg)
		l := NewAdaptiveHostLimiterWithCircuitBreaker(10, 10, adaptiveCfg, cb)

		if !l.IsAdaptiveEnabled() {
			t.Error("expected adaptive to be enabled")
		}
		if !l.IsCircuitBreakerEnabled() {
			t.Error("expected circuit breaker to be enabled")
		}
	})
}

func TestHostLimiter_CircuitBreakerBlocksRequests(t *testing.T) {
	t.Run("circuit breaker blocks when open", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 2,
			SuccessThreshold: 1,
			ResetTimeout:     1 * time.Hour, // Long timeout
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(100, 100, cb)

		ctx := context.Background()

		// First request should succeed
		err := l.Wait(ctx, "https://example.com")
		if err != nil {
			t.Errorf("expected first request to succeed, got error: %v", err)
		}

		// Record failures to open circuit
		l.RecordResult("example.com", errors.New("connection refused"), 0)
		l.RecordResult("example.com", errors.New("connection refused"), 0)

		// Next request should be blocked
		err = l.Wait(ctx, "https://example.com")
		if err == nil {
			t.Error("expected request to be blocked when circuit is open")
		}
		if !errors.Is(err, ErrCircuitBreakerOpen) {
			t.Errorf("expected ErrCircuitBreakerOpen, got: %v", err)
		}
	})

	t.Run("circuit breaker allows different hosts", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 2,
			ResetTimeout:     1 * time.Hour,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(100, 100, cb)

		ctx := context.Background()

		// Open circuit for host1
		l.Wait(ctx, "https://host1.example.com")
		l.RecordResult("host1.example.com", errors.New("error"), 0)
		l.RecordResult("host1.example.com", errors.New("error"), 0)

		// host1 should be blocked
		err := l.Wait(ctx, "https://host1.example.com")
		if err == nil {
			t.Error("expected host1 to be blocked")
		}

		// host2 should still work
		err = l.Wait(ctx, "https://host2.example.com")
		if err != nil {
			t.Errorf("expected host2 to be allowed, got error: %v", err)
		}
	})
}

func TestHostLimiter_RecordResult(t *testing.T) {
	t.Run("record success updates circuit breaker", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 3,
			SuccessThreshold: 2,
			ResetTimeout:     50 * time.Millisecond,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(100, 100, cb)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record failures to open circuit
		l.RecordResult("example.com", errors.New("error"), 0)
		l.RecordResult("example.com", errors.New("error"), 0)
		l.RecordResult("example.com", errors.New("error"), 0)

		// Circuit should be open
		if cb.GetState("example.com") != StateOpen {
			t.Error("expected circuit to be open")
		}

		// Wait for reset timeout
		time.Sleep(75 * time.Millisecond)

		// Record successes to close circuit
		l.RecordResult("example.com", nil, 200)
		l.RecordResult("example.com", nil, 200)

		// Circuit should be closed
		if cb.GetState("example.com") != StateClosed {
			t.Errorf("expected circuit to be closed, got %v", cb.GetState("example.com"))
		}
	})

	t.Run("record 5xx failure updates circuit breaker", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 2,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(100, 100, cb)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record 5xx failures
		l.RecordResult("example.com", nil, 500)
		l.RecordResult("example.com", nil, 503)

		if cb.GetState("example.com") != StateOpen {
			t.Errorf("expected circuit to be open after 2 5xx errors, got %v", cb.GetState("example.com"))
		}
	})

	t.Run("record 4xx does not update circuit breaker", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 2,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(100, 100, cb)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record 4xx responses (not failures)
		l.RecordResult("example.com", nil, 404)
		l.RecordResult("example.com", nil, 403)

		if cb.GetState("example.com") != StateClosed {
			t.Errorf("expected circuit to still be closed after 4xx, got %v", cb.GetState("example.com"))
		}
	})

	t.Run("nil limiter record result does not panic", func(t *testing.T) {
		var l *HostLimiter
		l.RecordResult("example.com", errors.New("error"), 0) // Should not panic
	})
}

func TestHostLimiter_GetHostStatus_WithCircuitBreaker(t *testing.T) {
	t.Run("status includes circuit breaker fields", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 3,
		}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(10, 10, cb)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")
		l.RecordResult("example.com", errors.New("error"), 0)

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 status, got %d", len(status))
		}

		s := status[0]
		if s.CircuitBreakerState == "" {
			t.Error("expected CircuitBreakerState to be set")
		}
		if s.CircuitBreakerFailures != 1 {
			t.Errorf("expected CircuitBreakerFailures = 1, got %d", s.CircuitBreakerFailures)
		}
		if s.CircuitBreakerLastFail.IsZero() {
			t.Error("expected CircuitBreakerLastFail to be set")
		}
	})

	t.Run("status without circuit breaker has empty CB fields", func(t *testing.T) {
		l := NewHostLimiter(10, 10)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 status, got %d", len(status))
		}

		s := status[0]
		if s.CircuitBreakerState != "" {
			t.Errorf("expected CircuitBreakerState to be empty, got %s", s.CircuitBreakerState)
		}
	})
}

func TestHostLimiter_IsCircuitBreakerEnabled(t *testing.T) {
	t.Run("nil limiter returns false", func(t *testing.T) {
		var l *HostLimiter
		if l.IsCircuitBreakerEnabled() {
			t.Error("expected IsCircuitBreakerEnabled to return false for nil limiter")
		}
	})

	t.Run("disabled circuit breaker returns false", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{Enabled: false}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(10, 10, cb)

		if l.IsCircuitBreakerEnabled() {
			t.Error("expected IsCircuitBreakerEnabled to return false when CB disabled")
		}
	})

	t.Run("enabled circuit breaker returns true", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{Enabled: true}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(10, 10, cb)

		if !l.IsCircuitBreakerEnabled() {
			t.Error("expected IsCircuitBreakerEnabled to return true when CB enabled")
		}
	})
}

func TestHostLimiter_GetCircuitBreaker(t *testing.T) {
	t.Run("nil limiter returns nil", func(t *testing.T) {
		var l *HostLimiter
		if l.GetCircuitBreaker() != nil {
			t.Error("expected GetCircuitBreaker to return nil for nil limiter")
		}
	})

	t.Run("returns circuit breaker instance", func(t *testing.T) {
		cbCfg := CircuitBreakerConfig{Enabled: true}
		cb := NewCircuitBreaker(cbCfg)
		l := NewHostLimiterWithCircuitBreaker(10, 10, cb)

		if l.GetCircuitBreaker() != cb {
			t.Error("expected GetCircuitBreaker to return the same instance")
		}
	})

	t.Run("no circuit breaker returns nil", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		if l.GetCircuitBreaker() != nil {
			t.Error("expected GetCircuitBreaker to return nil when no CB")
		}
	})
}

func TestHostLimiter_UpdateRateLimitInfo(t *testing.T) {
	t.Run("nil limiter does not panic", func(t *testing.T) {
		var l *HostLimiter
		info := RateLimitInfo{Limit: 100, Remaining: 50}
		l.UpdateRateLimitInfo("example.com", info) // Should not panic
	})

	t.Run("non-adaptive limiter does nothing", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		info := RateLimitInfo{Limit: 100, Remaining: 50}
		l.UpdateRateLimitInfo("example.com", info)

		// No adaptive state should exist
		if l.adaptive != nil {
			t.Error("expected no adaptive config for non-adaptive limiter")
		}
	})

	t.Run("server limit adjusts current QPS downward", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Server says limit is 10, we should adjust to 80% of that (8)
		info := RateLimitInfo{Limit: 10, Remaining: 5}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		if hostInfo.currentQPS != 8 {
			t.Errorf("expected currentQPS 8 (80%% of 10), got %v", hostInfo.currentQPS)
		}
	})

	t.Run("server limit higher than current is ignored", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(10, 10, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Server says limit is 100, but we're at 10 - should not jump immediately
		info := RateLimitInfo{Limit: 100, Remaining: 90}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		if hostInfo.currentQPS != 10 {
			t.Errorf("expected currentQPS to stay at 10, got %v", hostInfo.currentQPS)
		}
	})

	t.Run("reset time with low remaining sets cooldown", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		resetTime := time.Now().Add(5 * time.Minute)
		// Low remaining (5% = 5/100) should trigger cooldown
		info := RateLimitInfo{Limit: 100, Remaining: 5, Reset: resetTime}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		if hostInfo.cooldownUntil.Before(resetTime.Add(-time.Second)) {
			t.Errorf("expected cooldownUntil to be set to reset time, got %v", hostInfo.cooldownUntil)
		}
	})

	t.Run("low remaining enters cooldown until reset", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		resetTime := time.Now().Add(10 * time.Minute)
		// Only 5% remaining (5 out of 100)
		info := RateLimitInfo{Limit: 100, Remaining: 5, Reset: resetTime}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		if hostInfo.cooldownUntil.Before(resetTime.Add(-time.Second)) {
			t.Errorf("expected cooldown until reset due to low remaining, got %v", hostInfo.cooldownUntil)
		}
	})

	t.Run("high remaining does not enter cooldown", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		resetTime := time.Now().Add(10 * time.Minute)
		// 50% remaining - should not trigger cooldown
		info := RateLimitInfo{Limit: 100, Remaining: 50, Reset: resetTime}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		// Cooldown should not be extended by low remaining
		if !hostInfo.cooldownUntil.IsZero() && hostInfo.cooldownUntil.After(time.Now().Add(time.Second)) {
			t.Errorf("expected no cooldown for high remaining, got %v", hostInfo.cooldownUntil)
		}
	})

	t.Run("respects min QPS floor", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  5,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Server says limit is 1, but min QPS is 5 - should stay at 5
		info := RateLimitInfo{Limit: 1, Remaining: 0}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		if hostInfo.currentQPS != 5 {
			t.Errorf("expected currentQPS to respect min floor of 5, got %v", hostInfo.currentQPS)
		}
	})

	t.Run("respects max QPS ceiling", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  10,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Server says limit is 1000, but max QPS is 10 - should not exceed 10
		// (though we also don't increase significantly from server data alone)
		info := RateLimitInfo{Limit: 1000, Remaining: 500}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		// Should not exceed max QPS
		if hostInfo.currentQPS > 10 {
			t.Errorf("expected currentQPS to respect max ceiling of 10, got %v", hostInfo.currentQPS)
		}
	})

	t.Run("unknown host does nothing", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)

		// Don't create limiter for this host first
		info := RateLimitInfo{Limit: 10, Remaining: 5}
		l.UpdateRateLimitInfo("unknown.example.com", info)

		// Should not create an entry
		if _, ok := l.hostInfo["unknown.example.com"]; ok {
			t.Error("expected no host info for unknown host")
		}
	})

	t.Run("past reset time is ignored", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Reset time in the past
		resetTime := time.Now().Add(-5 * time.Minute)
		info := RateLimitInfo{Limit: 100, Remaining: 0, Reset: resetTime}
		l.UpdateRateLimitInfo("example.com", info)

		hostInfo := l.hostInfo["example.com"]
		// Should not set cooldown for past reset time
		if hostInfo.cooldownUntil.After(time.Now()) {
			t.Error("expected no cooldown for past reset time")
		}
	})

	t.Run("concurrent updates are safe", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  0.1,
			MaxQPS:  100,
		}
		l := NewAdaptiveHostLimiter(50, 50, cfg)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(remaining int) {
				defer wg.Done()
				info := RateLimitInfo{Limit: 100, Remaining: remaining}
				l.UpdateRateLimitInfo("example.com", info)
			}(i)
		}
		wg.Wait()

		// Should not panic and state should be valid
		hostInfo := l.hostInfo["example.com"]
		if hostInfo.currentQPS < cfg.MinQPS || hostInfo.currentQPS > cfg.MaxQPS {
			t.Errorf("QPS %v out of bounds [%v, %v]", hostInfo.currentQPS, cfg.MinQPS, cfg.MaxQPS)
		}
	})
}
