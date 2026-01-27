package fetch

import (
	"context"
	"errors"
	"net/url"
	"sync"
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
