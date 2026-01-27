package fetch

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   bool
	}{
		{
			name:   "network error with ErrClosed retries",
			err:    net.ErrClosed,
			status: 0,
			want:   true,
		},
		{
			name:   "timeout error retries",
			err:    errors.New("connection timeout"),
			status: 0,
			want:   true,
		},
		{
			name:   "other errors retry",
			err:    errors.New("some error"),
			status: 0,
			want:   true,
		},
		{
			name:   "success status does not retry",
			err:    nil,
			status: 200,
			want:   false,
		},
		{
			name:   "403 does not retry",
			err:    nil,
			status: 403,
			want:   false,
		},
		{
			name:   "401 does not retry",
			err:    nil,
			status: 401,
			want:   false,
		},
		{
			name:   "429 rate limit retries",
			err:    nil,
			status: 429,
			want:   true,
		},
		{
			name:   "500 server error retries",
			err:    nil,
			status: 500,
			want:   true,
		},
		{
			name:   "502 bad gateway retries",
			err:    nil,
			status: 502,
			want:   true,
		},
		{
			name:   "503 service unavailable retries",
			err:    nil,
			status: 503,
			want:   true,
		},
		{
			name:   "504 gateway timeout retries",
			err:    nil,
			status: 504,
			want:   true,
		},
		{
			name:   "400 bad request does not retry",
			err:    nil,
			status: 400,
			want:   false,
		},
		{
			name:   "404 not found does not retry",
			err:    nil,
			status: 404,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.err, tt.status); got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		name    string
		base    time.Duration
		attempt int
		want    time.Duration
	}{
		{
			name:    "attempt 0 returns base",
			base:    100 * time.Millisecond,
			attempt: 0,
			want:    100 * time.Millisecond,
		},
		{
			name:    "attempt 1 returns base * 2",
			base:    100 * time.Millisecond,
			attempt: 1,
			want:    200 * time.Millisecond,
		},
		{
			name:    "attempt 2 returns base * 4",
			base:    100 * time.Millisecond,
			attempt: 2,
			want:    400 * time.Millisecond,
		},
		{
			name:    "attempt 3 returns base * 8",
			base:    100 * time.Millisecond,
			attempt: 3,
			want:    800 * time.Millisecond,
		},
		{
			name:    "different base 1s",
			base:    1 * time.Second,
			attempt: 1,
			want:    2 * time.Second,
		},
		{
			name:    "exponential growth continues",
			base:    100 * time.Millisecond,
			attempt: 4,
			want:    1600 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backoff(tt.base, tt.attempt); got != tt.want {
				t.Errorf("backoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
