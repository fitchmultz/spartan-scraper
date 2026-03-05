// Package fetch provides tests for retry logic and backoff strategies.
// Tests cover retry eligibility for errors and status codes, exponential backoff, jitter, and circuit breaker patterns.
package fetch

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestClampRetry(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{
			name:  "negative count returns 0",
			count: -5,
			want:  0,
		},
		{
			name:  "zero returns 0",
			count: 0,
			want:  0,
		},
		{
			name:  "small count returns as is",
			count: 3,
			want:  3,
		},
		{
			name:  "exactly 10 returns 10",
			count: 10,
			want:  10,
		},
		{
			name:  "15 is clamped to 10",
			count: 15,
			want:  10,
		},
		{
			name:  "100 is clamped to 10",
			count: 100,
			want:  10,
		},
		{
			name:  "just above limit",
			count: 11,
			want:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampRetry(tt.count); got != tt.want {
				t.Errorf("clampRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		want     time.Duration
		validate func(t *testing.T, got time.Duration)
	}{
		{
			name: "nil response returns 0",
			resp: nil,
			want: 0,
		},
		{
			name: "empty header returns 0",
			resp: &http.Response{Header: http.Header{}},
			want: 0,
		},
		{
			name: "seconds format 60",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"60"}},
			},
			want: 60 * time.Second,
		},
		{
			name: "seconds format 120",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"120"}},
			},
			want: 120 * time.Second,
		},
		{
			name: "future date returns positive duration",
			resp: func() *http.Response {
				futureTime := time.Now().Add(5 * time.Minute).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
				return &http.Response{
					Header: http.Header{"Retry-After": []string{futureTime}},
				}
			}(),
			validate: func(t *testing.T, got time.Duration) {
				if got < 4*time.Minute || got > 6*time.Minute {
					t.Errorf("readRetryAfter() = %v, want approximately 5m0s", got)
				}
			},
		},
		{
			name: "past date returns <= 0",
			resp: func() *http.Response {
				pastTime := time.Now().Add(-5 * time.Minute).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
				return &http.Response{
					Header: http.Header{"Retry-After": []string{pastTime}},
				}
			}(),
			validate: func(t *testing.T, got time.Duration) {
				if got > 0 {
					t.Errorf("readRetryAfter() = %v, want <= 0s", got)
				}
			},
		},
		{
			name: "invalid format returns 0",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"invalid"}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readRetryAfter(tt.resp)
			if tt.validate != nil {
				tt.validate(t, got)
			} else if got != tt.want {
				t.Errorf("readRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLockedRand_ConcurrentAccess(t *testing.T) {
	// Ensure the random source is safe for concurrent use
	var wg sync.WaitGroup
	numGoroutines := 10
	numIterations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				_ = randSource.Float64()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no panic
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for concurrent random access")
	}
}
