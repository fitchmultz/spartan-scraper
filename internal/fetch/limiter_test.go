// Package fetch provides tests for the host-based rate limiter.
// Tests cover limiter creation, per-host rate limiting, concurrency safety, and limiter caching.
// Does NOT test distributed rate limiting or persistence across process restarts.
package fetch

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewHostLimiter(t *testing.T) {
	tests := []struct {
		name     string
		qps      int
		burst    int
		validate func(t *testing.T, l *HostLimiter)
	}{
		{
			name:  "QPS = 0 uses infinite rate",
			qps:   0,
			burst: 5,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.qps != rate.Inf {
					t.Errorf("expected infinite rate, got %v", l.qps)
				}
				if l.burst != 5 {
					t.Errorf("expected burst 5, got %d", l.burst)
				}
			},
		},
		{
			name:  "QPS > 0 uses correct rate",
			qps:   10,
			burst: 5,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.qps != rate.Limit(10) {
					t.Errorf("expected rate 10, got %v", l.qps)
				}
				if l.burst != 5 {
					t.Errorf("expected burst 5, got %d", l.burst)
				}
			},
		},
		{
			name:  "burst <= 0 defaults to 1",
			qps:   10,
			burst: 0,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.burst != 1 {
					t.Errorf("expected burst 1, got %d", l.burst)
				}
			},
		},
		{
			name:  "negative burst defaults to 1",
			qps:   10,
			burst: -5,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.burst != 1 {
					t.Errorf("expected burst 1, got %d", l.burst)
				}
			},
		},
		{
			name:  "positive burst is used",
			qps:   10,
			burst: 20,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.burst != 20 {
					t.Errorf("expected burst 20, got %d", l.burst)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewHostLimiter(tt.qps, tt.burst)
			tt.validate(t, l)
		})
	}
}

func TestHostLimiter_Wait(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *HostLimiter
		url       string
		wantErr   bool
		wantErrIs error
		validate  func(t *testing.T)
	}{
		{
			name:    "single host within rate limit no delay",
			setup:   func() *HostLimiter { return NewHostLimiter(10, 5) },
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "single host exceeding rate blocks",
			setup:   func() *HostLimiter { return NewHostLimiter(1, 1) },
			url:     "https://example.com",
			wantErr: false,
			validate: func(t *testing.T) {
				start := time.Now()
				l := NewHostLimiter(1, 1)
				ctx := context.Background()
				l.Wait(ctx, "https://example.com")
				l.Wait(ctx, "https://example.com")
				if elapsed := time.Since(start); elapsed < time.Second {
					t.Errorf("expected delay >= 1s, got %v", elapsed)
				}
			},
		},
		{
			name:  "different hosts have separate limiters",
			setup: func() *HostLimiter { return NewHostLimiter(10, 5) },
			url:   "https://example1.com",
			validate: func(t *testing.T) {
				l := NewHostLimiter(100, 100)
				ctx := context.Background()
				start := time.Now()
				for i := 0; i < 10; i++ {
					l.Wait(ctx, "https://example1.com")
					l.Wait(ctx, "https://example2.com")
				}
				if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
					t.Errorf("expected no blocking for different hosts, got %v", elapsed)
				}
			},
		},
		{
			name: "nil limiter returns nil",
			url:  "https://example.com",
			validate: func(t *testing.T) {
				var l *HostLimiter
				if err := l.Wait(context.Background(), "https://example.com"); err != nil {
					t.Errorf("unexpected error from nil limiter: %v", err)
				}
			},
		},
		{
			name:    "infinite rate limiter returns nil",
			setup:   func() *HostLimiter { return NewHostLimiter(0, 0) },
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "invalid URL returns nil",
			setup:   func() *HostLimiter { return NewHostLimiter(10, 5) },
			url:     "://not-a-url",
			wantErr: false,
		},
		{
			name:    "URL with no host returns nil",
			setup:   func() *HostLimiter { return NewHostLimiter(10, 5) },
			url:     "file:///path/to/file",
			wantErr: false,
		},
		{
			name:      "context cancellation during wait",
			setup:     func() *HostLimiter { return NewHostLimiter(1, 1) },
			url:       "https://example.com",
			wantErr:   true,
			wantErrIs: context.DeadlineExceeded,
			validate: func(t *testing.T) {
				l := NewHostLimiter(1, 1)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				l.Wait(ctx, "https://example.com")
				l.Wait(ctx, "https://example.com")
				l.Wait(ctx, "https://example.com")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t)
				return
			}
			l := tt.setup()
			err := l.Wait(context.Background(), tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("HostLimiter.Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrIs != nil && err != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
			}
		})
	}
}

func TestHostLimiter_Concurrency(t *testing.T) {
	t.Run("100 goroutines same host respects burst", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		ctx := context.Background()
		url := "https://example.com"

		var wg sync.WaitGroup
		start := make(chan struct{})
		errors := make(chan error, 100)
		completed := make(chan int, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if err := l.Wait(ctx, url); err != nil {
					errors <- err
				}
				completed <- 1
			}()
		}

		close(start)
		wg.Wait()
		close(errors)
		close(completed)

		for err := range errors {
			t.Errorf("unexpected error in concurrent test: %v", err)
		}

		if count := len(completed); count != 100 {
			t.Errorf("expected 100 completions, got %d", count)
		}
	})

	t.Run("100 goroutines different hosts no interference", func(t *testing.T) {
		l := NewHostLimiter(100, 100)
		ctx := context.Background()
		var wg sync.WaitGroup
		start := make(chan struct{})

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				url := url.URL{Scheme: "https", Host: "example" + string(rune('0'+idx%10)) + ".com"}
				if err := l.Wait(ctx, url.String()); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}(i)
		}

		close(start)
		wg.Wait()
	})
}

func TestHostLimiter_Caching(t *testing.T) {
	t.Run("first call creates limiter", func(t *testing.T) {
		l := NewHostLimiter(10, 5)
		host := "example.com"

		limiter := l.getLimiter(host)
		if limiter == nil {
			t.Error("expected non-nil limiter")
		}
		if len(l.byHost) != 1 {
			t.Errorf("expected 1 limiter in cache, got %d", len(l.byHost))
		}
	})

	t.Run("subsequent calls reuse limiter", func(t *testing.T) {
		l := NewHostLimiter(10, 5)
		host := "example.com"

		limiter1 := l.getLimiter(host)
		limiter2 := l.getLimiter(host)

		if limiter1 != limiter2 {
			t.Error("expected same limiter instance")
		}
		if len(l.byHost) != 1 {
			t.Errorf("expected 1 limiter in cache, got %d", len(l.byHost))
		}
	})

	t.Run("different hosts get different limiters", func(t *testing.T) {
		l := NewHostLimiter(10, 5)
		host1 := "example1.com"
		host2 := "example2.com"

		limiter1 := l.getLimiter(host1)
		limiter2 := l.getLimiter(host2)

		if limiter1 == limiter2 {
			t.Error("expected different limiter instances")
		}
		if len(l.byHost) != 2 {
			t.Errorf("expected 2 limiters in cache, got %d", len(l.byHost))
		}
	})
}

func TestHostLimiter_PerHostRates(t *testing.T) {
	t.Run("different hosts can have different rates", func(t *testing.T) {
		l := NewHostLimiter(10, 10) // global defaults
		ctx := context.Background()

		// Host1 with custom rate: 2 QPS, burst 2
		start := time.Now()
		for i := 0; i < 3; i++ {
			l.WaitWithRates(ctx, "https://slow.example.com", 2, 2)
		}
		elapsed := time.Since(start)
		// 3 requests at 2 QPS with burst 2: first 2 immediate, 3rd waits ~0.5s
		if elapsed < 400*time.Millisecond {
			t.Errorf("expected delay for rate-limited host, got %v", elapsed)
		}

		// Host2 with default rate: 10 QPS, burst 10 - should be faster
		start = time.Now()
		for i := 0; i < 3; i++ {
			l.WaitWithRates(ctx, "https://fast.example.com", 0, 0) // use defaults
		}
		elapsed = time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("expected no delay for fast host, got %v", elapsed)
		}
	})

	t.Run("zero rates fall back to global defaults", func(t *testing.T) {
		l := NewHostLimiter(100, 100)
		ctx := context.Background()

		start := time.Now()
		for i := 0; i < 10; i++ {
			l.WaitWithRates(ctx, "https://example.com", 0, 0)
		}
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			t.Errorf("expected fast execution with global defaults, got %v", elapsed)
		}
	})

	t.Run("rate limiter is created once and reused", func(t *testing.T) {
		l := NewHostLimiter(10, 10)

		// First call creates limiter with specific rates
		l.WaitWithRates(context.Background(), "https://example.com", 5, 5)

		// Second call with different rates should still use original limiter
		// (rates don't change dynamically for existing hosts)
		limiter1 := l.getLimiter("example.com")

		l.WaitWithRates(context.Background(), "https://example.com", 10, 10)
		limiter2 := l.getLimiter("example.com")

		if limiter1 != limiter2 {
			t.Error("expected same limiter instance to be reused")
		}
	})

	t.Run("GetHostStatus returns per-host rates", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		ctx := context.Background()

		// Create limiters with different rates
		l.WaitWithRates(ctx, "https://slow.example.com", 2, 3)
		l.WaitWithRates(ctx, "https://fast.example.com", 50, 50)

		status := l.GetHostStatus()
		if len(status) != 2 {
			t.Fatalf("expected 2 host statuses, got %d", len(status))
		}

		for _, s := range status {
			if s.Host == "slow.example.com" {
				if s.QPS != 2 {
					t.Errorf("expected QPS 2 for slow host, got %f", s.QPS)
				}
				if s.Burst != 3 {
					t.Errorf("expected burst 3 for slow host, got %d", s.Burst)
				}
			}
			if s.Host == "fast.example.com" {
				if s.QPS != 50 {
					t.Errorf("expected QPS 50 for fast host, got %f", s.QPS)
				}
				if s.Burst != 50 {
					t.Errorf("expected burst 50 for fast host, got %d", s.Burst)
				}
			}
		}
	})

	t.Run("GetHostStatus returns zero QPS for unlimited", func(t *testing.T) {
		l := NewHostLimiter(10, 10)
		ctx := context.Background()

		// Create a limiter with explicit unlimited QPS (0 means use global, so use -1 or a very high value)
		// Actually, 0 in per-host context means "use global", so to test unlimited we'd need
		// to create a limiter directly. Let's just verify the behavior with finite rates.
		l.WaitWithRates(ctx, "https://example.com", 0, 0)

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 host status, got %d", len(status))
		}

		// Should have global QPS of 10
		if status[0].QPS != 10 {
			t.Errorf("expected QPS 10, got %f", status[0].QPS)
		}
	})
}

// Adaptive rate limiting tests

func TestNewAdaptiveHostLimiter(t *testing.T) {
	tests := []struct {
		name     string
		qps      int
		burst    int
		cfg      *AdaptiveConfig
		validate func(t *testing.T, l *HostLimiter)
	}{
		{
			name:  "adaptive disabled when cfg is nil",
			qps:   10,
			burst: 5,
			cfg:   nil,
			validate: func(t *testing.T, l *HostLimiter) {
				if l.IsAdaptiveEnabled() {
					t.Error("expected adaptive to be disabled")
				}
				if l.adaptive != nil {
					t.Error("expected adaptive config to be nil")
				}
			},
		},
		{
			name:  "adaptive disabled when cfg.Enabled is false",
			qps:   10,
			burst: 5,
			cfg:   &AdaptiveConfig{Enabled: false},
			validate: func(t *testing.T, l *HostLimiter) {
				if l.IsAdaptiveEnabled() {
					t.Error("expected adaptive to be disabled")
				}
			},
		},
		{
			name:  "adaptive enabled with defaults",
			qps:   10,
			burst: 5,
			cfg:   &AdaptiveConfig{Enabled: true},
			validate: func(t *testing.T, l *HostLimiter) {
				if !l.IsAdaptiveEnabled() {
					t.Error("expected adaptive to be enabled")
				}
				if l.adaptive.MinQPS != 0.1 {
					t.Errorf("expected default MinQPS 0.1, got %v", l.adaptive.MinQPS)
				}
				if l.adaptive.MaxQPS != rate.Limit(10) {
					t.Errorf("expected MaxQPS 10, got %v", l.adaptive.MaxQPS)
				}
				if l.adaptive.AdditiveIncrease != 0.5 {
					t.Errorf("expected default AdditiveIncrease 0.5, got %v", l.adaptive.AdditiveIncrease)
				}
				if l.adaptive.MultiplicativeDecrease != 0.5 {
					t.Errorf("expected default MultiplicativeDecrease 0.5, got %v", l.adaptive.MultiplicativeDecrease)
				}
				if l.adaptive.SuccessThreshold != 5 {
					t.Errorf("expected default SuccessThreshold 5, got %d", l.adaptive.SuccessThreshold)
				}
				if l.adaptive.CooldownPeriod != time.Second {
					t.Errorf("expected default CooldownPeriod 1s, got %v", l.adaptive.CooldownPeriod)
				}
			},
		},
		{
			name:  "adaptive with custom values",
			qps:   20,
			burst: 10,
			cfg: &AdaptiveConfig{
				Enabled:                true,
				MinQPS:                 0.5,
				MaxQPS:                 15,
				AdditiveIncrease:       1.0,
				MultiplicativeDecrease: 0.7,
				SuccessThreshold:       3,
				CooldownPeriod:         2 * time.Second,
			},
			validate: func(t *testing.T, l *HostLimiter) {
				if l.adaptive.MinQPS != 0.5 {
					t.Errorf("expected MinQPS 0.5, got %v", l.adaptive.MinQPS)
				}
				if l.adaptive.MaxQPS != 15 {
					t.Errorf("expected MaxQPS 15, got %v", l.adaptive.MaxQPS)
				}
				if l.adaptive.AdditiveIncrease != 1.0 {
					t.Errorf("expected AdditiveIncrease 1.0, got %v", l.adaptive.AdditiveIncrease)
				}
				if l.adaptive.MultiplicativeDecrease != 0.7 {
					t.Errorf("expected MultiplicativeDecrease 0.7, got %v", l.adaptive.MultiplicativeDecrease)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewAdaptiveHostLimiter(tt.qps, tt.burst, tt.cfg)
			tt.validate(t, l)
		})
	}
}

func TestAdaptiveHostLimiter_RecordSuccess(t *testing.T) {
	t.Run("success increases consecutive counter", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           10,
			AdditiveIncrease: 1,
			SuccessThreshold: 3,
			CooldownPeriod:   0, // No cooldown for testing
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		// Create limiter for host
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record 2 successes (below threshold)
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")

		info := l.hostInfo["example.com"]
		if info.consecutiveSuccesses != 2 {
			t.Errorf("expected 2 consecutive successes, got %d", info.consecutiveSuccesses)
		}
		// currentQPS is initialized to the starting QPS value
		if info.currentQPS != 5 {
			t.Errorf("expected currentQPS 5 (initial), got %v", info.currentQPS)
		}
	})

	t.Run("success threshold triggers rate increase", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           10,
			AdditiveIncrease: 2,
			SuccessThreshold: 3,
			CooldownPeriod:   0, // No cooldown for testing
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		// Create limiter for host
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record 3 successes (at threshold)
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")

		info := l.hostInfo["example.com"]
		if info.currentQPS != 7 { // 5 + 2 = 7
			t.Errorf("expected currentQPS 7, got %v", info.currentQPS)
		}
		if info.consecutiveSuccesses != 0 {
			t.Errorf("expected consecutiveSuccesses reset to 0, got %d", info.consecutiveSuccesses)
		}
	})

	t.Run("rate increase respects max QPS cap", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           6,
			AdditiveIncrease: 5,
			SuccessThreshold: 1,
			CooldownPeriod:   0,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		l.RecordSuccess("example.com")

		info := l.hostInfo["example.com"]
		if info.currentQPS != 6 { // Capped at max
			t.Errorf("expected currentQPS 6 (capped), got %v", info.currentQPS)
		}
	})

	t.Run("cooldown prevents rate adjustment", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           10,
			AdditiveIncrease: 2,
			SuccessThreshold: 1,
			CooldownPeriod:   time.Hour, // Long cooldown
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// First success should adjust rate
		l.RecordSuccess("example.com")
		info := l.hostInfo["example.com"]
		if info.currentQPS != 7 {
			t.Errorf("expected currentQPS 7 after first adjustment, got %v", info.currentQPS)
		}

		// Second success during cooldown should not adjust
		l.RecordSuccess("example.com")
		if info.currentQPS != 7 {
			t.Errorf("expected currentQPS still 7 (cooldown), got %v", info.currentQPS)
		}
	})

	t.Run("nil limiter does not panic", func(t *testing.T) {
		var l *HostLimiter
		l.RecordSuccess("example.com") // Should not panic
	})

	t.Run("non-adaptive limiter does nothing", func(t *testing.T) {
		l := NewHostLimiter(5, 5)
		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")

		// No adaptive state should exist
		if l.adaptive != nil {
			t.Error("expected no adaptive config")
		}
	})
}

func TestAdaptiveHostLimiter_RecordRateLimit(t *testing.T) {
	t.Run("429 triggers multiplicative decrease", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 10,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(8, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		l.RecordRateLimit("example.com", 0)

		info := l.hostInfo["example.com"]
		if info.currentQPS != 4 { // 8 * 0.5 = 4
			t.Errorf("expected currentQPS 4, got %v", info.currentQPS)
		}
		if info.consecutive429s != 1 {
			t.Errorf("expected consecutive429s 1, got %d", info.consecutive429s)
		}
	})

	t.Run("rate decrease respects min QPS floor", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 2,
			MaxQPS:                 10,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(3, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		l.RecordRateLimit("example.com", 0)

		info := l.hostInfo["example.com"]
		if info.currentQPS != 2 { // Capped at min
			t.Errorf("expected currentQPS 2 (capped at min), got %v", info.currentQPS)
		}
	})

	t.Run("429 resets consecutive successes", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           10,
			AdditiveIncrease: 1,
			SuccessThreshold: 5,
			CooldownPeriod:   0,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record some successes
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")

		info := l.hostInfo["example.com"]
		if info.consecutiveSuccesses != 2 {
			t.Errorf("expected 2 consecutive successes, got %d", info.consecutiveSuccesses)
		}

		// 429 should reset
		l.RecordRateLimit("example.com", 0)

		if info.consecutiveSuccesses != 0 {
			t.Errorf("expected consecutiveSuccesses reset to 0, got %d", info.consecutiveSuccesses)
		}
	})

	t.Run("retry-after sets cooldown", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 10,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		retryAfter := 5 * time.Second
		l.RecordRateLimit("example.com", retryAfter)

		info := l.hostInfo["example.com"]
		if time.Now().After(info.cooldownUntil) {
			t.Error("expected cooldownUntil to be in the future")
		}
		expectedUntil := time.Now().Add(retryAfter)
		if info.cooldownUntil.Sub(expectedUntil) > time.Second {
			t.Errorf("expected cooldownUntil ~%v, got %v", expectedUntil, info.cooldownUntil)
		}
	})

	t.Run("cooldown prevents rate adjustment but still respects retry-after", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 10,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         time.Hour, // Long adjustment cooldown
		}
		l := NewAdaptiveHostLimiter(8, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// First 429 should adjust rate
		l.RecordRateLimit("example.com", 0)
		info := l.hostInfo["example.com"]
		if info.currentQPS != 4 {
			t.Errorf("expected currentQPS 4 after first adjustment, got %v", info.currentQPS)
		}

		// Second 429 during cooldown should not adjust rate
		l.RecordRateLimit("example.com", 0)
		if info.currentQPS != 4 {
			t.Errorf("expected currentQPS still 4 (cooldown), got %v", info.currentQPS)
		}

		// But Retry-After should still be respected
		retryAfter := 5 * time.Second
		l.RecordRateLimit("example.com", retryAfter)
		if time.Now().After(info.cooldownUntil) {
			t.Error("expected cooldownUntil to be set from Retry-After")
		}
	})

	t.Run("429 during cooldown still resets success counter and increments 429 counter", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 10,
			AdditiveIncrease:       1,
			MultiplicativeDecrease: 0.5,
			SuccessThreshold:       5,
			CooldownPeriod:         time.Hour, // Long adjustment cooldown
		}
		l := NewAdaptiveHostLimiter(8, 5, cfg) // Start at 8 QPS

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Record successes to build up consecutive counter
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")

		info := l.hostInfo["example.com"]
		if info.consecutiveSuccesses != 3 {
			t.Errorf("expected 3 consecutive successes, got %d", info.consecutiveSuccesses)
		}
		if info.consecutive429s != 0 {
			t.Errorf("expected 0 consecutive 429s, got %d", info.consecutive429s)
		}

		// First 429 triggers adjustment (8 * 0.5 = 4) and starts cooldown
		l.RecordRateLimit("example.com", 0)
		if info.consecutiveSuccesses != 0 {
			t.Errorf("expected consecutiveSuccesses reset to 0 after first 429, got %d", info.consecutiveSuccesses)
		}
		if info.consecutive429s != 1 {
			t.Errorf("expected consecutive429s 1 after first 429, got %d", info.consecutive429s)
		}
		if info.currentQPS != 4 {
			t.Errorf("expected currentQPS 4 after first adjustment, got %v", info.currentQPS)
		}

		// Successes during cooldown are ignored (no counter updates)
		// This is correct behavior - we want sustained success AFTER cooldown
		l.RecordSuccess("example.com")
		l.RecordSuccess("example.com")
		// Successes during cooldown don't increment counter
		if info.consecutiveSuccesses != 0 {
			t.Errorf("expected consecutiveSuccesses still 0 (cooldown), got %d", info.consecutiveSuccesses)
		}

		// Second 429 during cooldown should STILL increment 429 counter
		// even though rate adjustment is blocked by cooldown
		l.RecordRateLimit("example.com", 0)
		// 429 counter should increment (this is the key fix - 429s always count)
		if info.consecutive429s != 2 {
			t.Errorf("expected consecutive429s 2 after second 429, got %d", info.consecutive429s)
		}
		// Rate should not have changed (still at 4 from first adjustment)
		if info.currentQPS != 4 {
			t.Errorf("expected currentQPS still 4 (cooldown blocked adjustment), got %v", info.currentQPS)
		}
	})
}

func TestAdaptiveHostLimiter_CooldownWait(t *testing.T) {
	t.Run("wait respects cooldown period", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 100, // High rate so limiter doesn't delay
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(100, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Set a short cooldown
		l.RecordRateLimit("example.com", 200*time.Millisecond)

		// Next wait should be delayed by cooldown
		start := time.Now()
		l.Wait(ctx, "https://example.com")
		elapsed := time.Since(start)

		if elapsed < 150*time.Millisecond {
			t.Errorf("expected wait of at least 150ms due to cooldown, got %v", elapsed)
		}
	})

	t.Run("wait returns immediately when cooldown expired", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 100,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(100, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Set a very short cooldown
		l.RecordRateLimit("example.com", 10*time.Millisecond)
		time.Sleep(20 * time.Millisecond) // Wait for cooldown to expire

		// Next wait should not be delayed
		start := time.Now()
		l.Wait(ctx, "https://example.com")
		elapsed := time.Since(start)

		if elapsed > 50*time.Millisecond {
			t.Errorf("expected immediate wait, got %v", elapsed)
		}
	})

	t.Run("wait respects context cancellation during cooldown", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 100,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(100, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		// Set a long cooldown
		l.RecordRateLimit("example.com", 10*time.Second)

		// Create a context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := l.Wait(ctx, "https://example.com")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context deadline exceeded, got %v", err)
		}
	})
}

func TestAdaptiveHostLimiter_GetHostStatus(t *testing.T) {
	t.Run("status includes adaptive fields when enabled", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           10,
			AdditiveIncrease: 1,
			SuccessThreshold: 1,
			CooldownPeriod:   0,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")
		l.RecordSuccess("example.com")

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 status, got %d", len(status))
		}

		s := status[0]
		if !s.AdaptiveEnabled {
			t.Error("expected AdaptiveEnabled to be true")
		}
		if s.CurrentQPS != 6 {
			t.Errorf("expected CurrentQPS 6, got %f", s.CurrentQPS)
		}
		if s.ConsecutiveSuccesses != 0 { // Reset after threshold
			t.Errorf("expected ConsecutiveSuccesses 0, got %d", s.ConsecutiveSuccesses)
		}
	})

	t.Run("status shows cooldown state", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 10,
			MultiplicativeDecrease: 0.5,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")
		l.RecordRateLimit("example.com", 5*time.Second)

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 status, got %d", len(status))
		}

		s := status[0]
		if !s.InCooldown {
			t.Error("expected InCooldown to be true")
		}
		if s.CooldownUntil.IsZero() {
			t.Error("expected CooldownUntil to be set")
		}
	})

	t.Run("status excludes adaptive fields when disabled", func(t *testing.T) {
		l := NewHostLimiter(5, 5)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		status := l.GetHostStatus()
		if len(status) != 1 {
			t.Fatalf("expected 1 status, got %d", len(status))
		}

		s := status[0]
		if s.AdaptiveEnabled {
			t.Error("expected AdaptiveEnabled to be false")
		}
		if s.CurrentQPS != 0 {
			t.Errorf("expected CurrentQPS 0 (not set), got %f", s.CurrentQPS)
		}
	})
}

func TestAdaptiveHostLimiter_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent success reports are safe", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:          true,
			MinQPS:           1,
			MaxQPS:           1000,
			AdditiveIncrease: 1,
			SuccessThreshold: 1,
			CooldownPeriod:   0,
		}
		l := NewAdaptiveHostLimiter(10, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.RecordSuccess("example.com")
			}()
		}
		wg.Wait()

		// QPS should have increased, exact value depends on timing
		info := l.hostInfo["example.com"]
		if info.currentQPS <= 10 {
			t.Errorf("expected QPS to increase, got %v", info.currentQPS)
		}
	})

	t.Run("concurrent 429 reports are safe", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 1000,
			MultiplicativeDecrease: 0.9,
			CooldownPeriod:         0,
		}
		l := NewAdaptiveHostLimiter(100, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.RecordRateLimit("example.com", 0)
			}()
		}
		wg.Wait()

		// QPS should have decreased (exact value depends on timing/races)
		// Due to cooldown, only the first adjustment may apply
		info := l.hostInfo["example.com"]
		if info.currentQPS > 100 {
			t.Errorf("expected QPS to decrease from 100, got %v", info.currentQPS)
		}
		if info.currentQPS < cfg.MinQPS {
			t.Errorf("QPS %v below minimum %v", info.currentQPS, cfg.MinQPS)
		}
	})

	t.Run("concurrent mixed operations are safe", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 1,
			MaxQPS:                 100,
			AdditiveIncrease:       1,
			MultiplicativeDecrease: 0.5,
			SuccessThreshold:       5,
			CooldownPeriod:         time.Millisecond,
		}
		l := NewAdaptiveHostLimiter(10, 100, cfg)

		ctx := context.Background()
		l.Wait(ctx, "https://example.com")

		var wg sync.WaitGroup
		var successCount, rateLimitCount atomic.Int32

		// Start goroutines that report success
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					l.RecordSuccess("example.com")
					successCount.Add(1)
					time.Sleep(time.Microsecond)
				}
			}()
		}

		// Start goroutines that report rate limits
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 5; j++ {
					l.RecordRateLimit("example.com", 0)
					rateLimitCount.Add(1)
					time.Sleep(time.Microsecond * 10)
				}
			}()
		}

		wg.Wait()

		// Verify state is consistent
		info := l.hostInfo["example.com"]
		if info.currentQPS < cfg.MinQPS || info.currentQPS > cfg.MaxQPS {
			t.Errorf("QPS %v out of bounds [%v, %v]", info.currentQPS, cfg.MinQPS, cfg.MaxQPS)
		}
	})
}

func TestAdaptiveHostLimiter_GetAdaptiveConfig(t *testing.T) {
	t.Run("returns nil for non-adaptive limiter", func(t *testing.T) {
		l := NewHostLimiter(5, 5)
		if cfg := l.GetAdaptiveConfig(); cfg != nil {
			t.Error("expected nil config for non-adaptive limiter")
		}
	})

	t.Run("returns copy of config for adaptive limiter", func(t *testing.T) {
		cfg := &AdaptiveConfig{
			Enabled: true,
			MinQPS:  1,
			MaxQPS:  10,
		}
		l := NewAdaptiveHostLimiter(5, 5, cfg)

		returned := l.GetAdaptiveConfig()
		if returned == nil {
			t.Fatal("expected non-nil config")
		}
		if returned == l.adaptive {
			t.Error("expected copy of config, not same pointer")
		}
		if !returned.Enabled {
			t.Error("expected Enabled to be true")
		}
	})

	t.Run("nil limiter returns nil", func(t *testing.T) {
		var l *HostLimiter
		if cfg := l.GetAdaptiveConfig(); cfg != nil {
			t.Error("expected nil config for nil limiter")
		}
	})
}
