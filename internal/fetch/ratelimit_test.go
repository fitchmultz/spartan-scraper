// Package fetch provides fetch functionality for Spartan Scraper.
//
// Purpose:
// - Verify ratelimit test behavior for package fetch.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `fetch` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package fetch

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestParseRateLimitHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		want     RateLimitInfo
		wantZero bool
	}{
		{
			name:   "RFC 9440 standard format",
			header: "limit=100, remaining=50, reset=60",
			want: RateLimitInfo{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Now().Add(60 * time.Second),
			},
		},
		{
			name:   "RFC 9440 with spaces",
			header: "limit=1000, remaining=999, reset=3600",
			want: RateLimitInfo{
				Limit:     1000,
				Remaining: 999,
				Reset:     time.Now().Add(3600 * time.Second),
			},
		},
		{
			name:   "Only limit",
			header: "limit=100",
			want: RateLimitInfo{
				Limit: 100,
			},
		},
		{
			name:   "Only remaining",
			header: "remaining=50",
			want: RateLimitInfo{
				Remaining: 50,
			},
		},
		{
			name:   "Unix timestamp reset (GitHub style)",
			header: "limit=5000, remaining=4999, reset=" + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
			want: RateLimitInfo{
				Limit:     5000,
				Remaining: 4999,
				Reset:     time.Now().Add(time.Hour),
			},
		},
		{
			name:     "Empty header",
			header:   "",
			wantZero: true,
		},
		{
			name:   "Zero limit (should be ignored)",
			header: "limit=0, remaining=50",
			want: RateLimitInfo{
				Remaining: 50,
			},
		},
		{
			name:   "Negative remaining",
			header: "limit=100, remaining=-1",
			want: RateLimitInfo{
				Limit:     100,
				Remaining: -1,
			},
		},
		{
			name:   "Case insensitivity",
			header: "LIMIT=100, REMAINING=50, RESET=60",
			want: RateLimitInfo{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Now().Add(60 * time.Second),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRateLimitHeader(tt.header)
			if err != nil {
				t.Fatalf("ParseRateLimitHeader() error = %v", err)
			}

			if tt.wantZero {
				if got.Limit != 0 || got.Remaining != 0 || !got.Reset.IsZero() {
					t.Errorf("ParseRateLimitHeader() = %+v, want zero values", got)
				}
				return
			}

			if got.Limit != tt.want.Limit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.want.Limit)
			}
			if got.Remaining != tt.want.Remaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tt.want.Remaining)
			}

			// Compare reset times with tolerance for delta seconds
			if !tt.want.Reset.IsZero() {
				diff := got.Reset.Sub(tt.want.Reset)
				if diff < 0 {
					diff = -diff
				}
				if diff > 2*time.Second {
					t.Errorf("Reset = %v, want approximately %v (diff: %v)", got.Reset, tt.want.Reset, diff)
				}
			} else if !got.Reset.IsZero() {
				t.Errorf("Reset = %v, want zero", got.Reset)
			}
		})
	}
}

func TestParseXRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    RateLimitInfo
	}{
		{
			name: "GitHub style headers",
			headers: http.Header{
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Remaining": []string{"4999"},
				"X-Ratelimit-Reset":     []string{strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)},
			},
			want: RateLimitInfo{
				Limit:     5000,
				Remaining: 4999,
				Reset:     time.Now().Add(time.Hour),
			},
		},
		{
			name: "Lowercase variant",
			headers: http.Header{
				"X-Ratelimit-Limit":     []string{"100"},
				"X-Ratelimit-Remaining": []string{"50"},
				"X-Ratelimit-Reset":     []string{"60"},
			},
			want: RateLimitInfo{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Now().Add(60 * time.Second),
			},
		},
		{
			name: "X-Rate-Limit variant",
			headers: http.Header{
				"X-Rate-Limit-Limit":     []string{"1000"},
				"X-Rate-Limit-Remaining": []string{"500"},
				"X-Rate-Limit-Reset":     []string{"3600"},
			},
			want: RateLimitInfo{
				Limit:     1000,
				Remaining: 500,
				Reset:     time.Now().Add(3600 * time.Second),
			},
		},
		{
			name: "Only limit header",
			headers: http.Header{
				"X-Ratelimit-Limit": []string{"100"},
			},
			want: RateLimitInfo{
				Limit: 100,
			},
		},
		{
			name:    "No headers",
			headers: http.Header{},
			want:    RateLimitInfo{},
		},
		{
			name: "HTTP date reset",
			headers: func() http.Header {
				// Use a fixed reference time to avoid timing issues
				refTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)
				return http.Header{
					"X-Ratelimit-Reset": []string{refTime.Add(30 * time.Minute).Format(http.TimeFormat)},
				}
			}(),
			want: RateLimitInfo{
				// HTTP dates are parsed as UTC
				Reset: time.Date(2026, 1, 30, 12, 30, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseXRateLimitHeaders(tt.headers)
			if err != nil {
				t.Fatalf("ParseXRateLimitHeaders() error = %v", err)
			}

			if got.Limit != tt.want.Limit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.want.Limit)
			}
			if got.Remaining != tt.want.Remaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tt.want.Remaining)
			}

			if !tt.want.Reset.IsZero() {
				diff := got.Reset.Sub(tt.want.Reset)
				if diff < 0 {
					diff = -diff
				}
				if diff > 2*time.Second {
					t.Errorf("Reset = %v, want approximately %v (diff: %v)", got.Reset, tt.want.Reset, diff)
				}
			} else if !got.Reset.IsZero() {
				t.Errorf("Reset = %v, want zero", got.Reset)
			}
		})
	}
}

func TestParseRateLimitPolicyHeader(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantLimit  int
		wantWindow time.Duration
	}{
		{
			name:       "Standard format with window",
			header:     "100;w=60",
			wantLimit:  100,
			wantWindow: 60 * time.Second,
		},
		{
			name:      "Limit only",
			header:    "1000",
			wantLimit: 1000,
		},
		{
			name:       "With spaces",
			header:     "500 ; w=3600",
			wantLimit:  500,
			wantWindow: 3600 * time.Second,
		},
		{
			name:   "Empty header",
			header: "",
		},
		{
			name:       "Invalid limit",
			header:     "invalid;w=60",
			wantWindow: 60 * time.Second, // Window can be parsed even if limit is invalid
		},
		{
			name:      "Invalid window",
			header:    "100;w=invalid",
			wantLimit: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotWindow := ParseRateLimitPolicyHeader(tt.header)
			if gotLimit != tt.wantLimit {
				t.Errorf("ParseRateLimitPolicyHeader() limit = %d, want %d", gotLimit, tt.wantLimit)
			}
			if gotWindow != tt.wantWindow {
				t.Errorf("ParseRateLimitPolicyHeader() window = %v, want %v", gotWindow, tt.wantWindow)
			}
		})
	}
}

func TestExtractRateLimitInfo(t *testing.T) {
	tests := []struct {
		name      string
		headers   http.Header
		want      RateLimitInfo
		wantFound bool
	}{
		{
			name: "RFC 9440 RateLimit header only",
			headers: http.Header{
				"Ratelimit": []string{"limit=100, remaining=50, reset=60"},
			},
			want: RateLimitInfo{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Now().Add(60 * time.Second),
			},
			wantFound: true,
		},
		{
			name: "X-RateLimit headers only",
			headers: http.Header{
				"X-Ratelimit-Limit":     []string{"1000"},
				"X-Ratelimit-Remaining": []string{"500"},
				"X-Ratelimit-Reset":     []string{"3600"},
			},
			want: RateLimitInfo{
				Limit:     1000,
				Remaining: 500,
				Reset:     time.Now().Add(3600 * time.Second),
			},
			wantFound: true,
		},
		{
			name: "RateLimit-Policy header",
			headers: http.Header{
				"Ratelimit-Policy": []string{"100;w=60"},
			},
			want: RateLimitInfo{
				Limit:  100,
				Window: 60 * time.Second,
			},
			wantFound: true,
		},
		{
			name:      "No rate limit headers",
			headers:   http.Header{},
			want:      RateLimitInfo{},
			wantFound: false,
		},
		{
			name: "RFC 9440 preferred over X-RateLimit",
			headers: http.Header{
				"Ratelimit":             []string{"limit=100, remaining=50"},
				"X-Ratelimit-Limit":     []string{"200"},
				"X-Ratelimit-Remaining": []string{"100"},
			},
			want: RateLimitInfo{
				Limit:     100,
				Remaining: 50,
			},
			wantFound: true,
		},
		{
			name: "X-RateLimit fills gaps in RFC 9440",
			headers: http.Header{
				"Ratelimit":         []string{"limit=100"},
				"X-Ratelimit-Reset": []string{"3600"},
			},
			want: RateLimitInfo{
				Limit: 100,
				Reset: time.Now().Add(3600 * time.Second),
			},
			wantFound: true,
		},
		{
			name: "X-RateLimit-Policy variant",
			headers: http.Header{
				"X-Ratelimit-Policy": []string{"500;w=3600"},
			},
			want: RateLimitInfo{
				Limit:  500,
				Window: 3600 * time.Second,
			},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := ExtractRateLimitInfo(tt.headers)
			if found != tt.wantFound {
				t.Errorf("ExtractRateLimitInfo() found = %v, want %v", found, tt.wantFound)
			}

			if got.Limit != tt.want.Limit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.want.Limit)
			}
			if got.Remaining != tt.want.Remaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tt.want.Remaining)
			}
			if got.Window != tt.want.Window {
				t.Errorf("Window = %v, want %v", got.Window, tt.want.Window)
			}

			if !tt.want.Reset.IsZero() {
				diff := got.Reset.Sub(tt.want.Reset)
				if diff < 0 {
					diff = -diff
				}
				if diff > 2*time.Second {
					t.Errorf("Reset = %v, want approximately %v (diff: %v)", got.Reset, tt.want.Reset, diff)
				}
			} else if !got.Reset.IsZero() {
				t.Errorf("Reset = %v, want zero", got.Reset)
			}
		})
	}
}

func TestRateLimitInfo_IsRateLimited(t *testing.T) {
	tests := []struct {
		name string
		rl   *RateLimitInfo
		want bool
	}{
		{
			name: "nil receiver",
			rl:   nil,
			want: false,
		},
		{
			name: "remaining positive",
			rl:   &RateLimitInfo{Limit: 100, Remaining: 50},
			want: false,
		},
		{
			name: "remaining zero",
			rl:   &RateLimitInfo{Limit: 100, Remaining: 0},
			want: true,
		},
		{
			name: "remaining negative",
			rl:   &RateLimitInfo{Limit: 100, Remaining: -1},
			want: true,
		},
		{
			name: "no limit set",
			rl:   &RateLimitInfo{Remaining: 0},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rl.IsRateLimited(); got != tt.want {
				t.Errorf("RateLimitInfo.IsRateLimited() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimitInfo_TimeUntilReset(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name string
		rl   *RateLimitInfo
		want time.Duration
	}{
		{
			name: "nil receiver",
			rl:   nil,
			want: 0,
		},
		{
			name: "reset in future",
			rl:   &RateLimitInfo{Reset: future},
			want: time.Until(future),
		},
		{
			name: "reset in past",
			rl:   &RateLimitInfo{Reset: past},
			want: 0,
		},
		{
			name: "no reset set",
			rl:   &RateLimitInfo{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rl.TimeUntilReset()
			// Allow small tolerance for time-based tests
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("RateLimitInfo.TimeUntilReset() = %v, want approximately %v", got, tt.want)
			}
		})
	}
}

func TestRateLimitInfo_UsagePercent(t *testing.T) {
	tests := []struct {
		name string
		rl   *RateLimitInfo
		want float64
	}{
		{
			name: "nil receiver",
			rl:   nil,
			want: -1,
		},
		{
			name: "no limit",
			rl:   &RateLimitInfo{Remaining: 50},
			want: -1,
		},
		{
			name: "50% used",
			rl:   &RateLimitInfo{Limit: 100, Remaining: 50},
			want: 50,
		},
		{
			name: "0% used",
			rl:   &RateLimitInfo{Limit: 100, Remaining: 100},
			want: 0,
		},
		{
			name: "100% used",
			rl:   &RateLimitInfo{Limit: 100, Remaining: 0},
			want: 100,
		},
		{
			name: "negative remaining shows over limit",
			rl:   &RateLimitInfo{Limit: 100, Remaining: -10},
			want: 110, // 100 - (-10) = 110 used, 110/100 * 100 = 110%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rl.UsagePercent()
			// Use tolerance for floating point comparison
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Errorf("RateLimitInfo.UsagePercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseResetValue(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		value string
		check func(time.Time) bool
	}{
		{
			name:  "delta seconds",
			value: "60",
			check: func(got time.Time) bool {
				expected := now.Add(60 * time.Second)
				diff := got.Sub(expected)
				if diff < 0 {
					diff = -diff
				}
				return diff < time.Second
			},
		},
		{
			name:  "unix timestamp",
			value: strconv.FormatInt(now.Add(time.Hour).Unix(), 10),
			check: func(got time.Time) bool {
				expected := now.Add(time.Hour)
				diff := got.Sub(expected)
				if diff < 0 {
					diff = -diff
				}
				return diff < time.Second
			},
		},
		{
			name:  "http date",
			value: "Fri, 30 Jan 2026 12:00:00 GMT",
			check: func(got time.Time) bool {
				expected := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)
				return got.Equal(expected)
			},
		},
		{
			name:  "empty string",
			value: "",
			check: func(got time.Time) bool {
				return got.IsZero()
			},
		},
		{
			name:  "invalid value",
			value: "invalid",
			check: func(got time.Time) bool {
				return got.IsZero()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseResetValue(tt.value)
			if !tt.check(got) {
				t.Errorf("parseResetValue(%q) = %v, check failed", tt.value, got)
			}
		})
	}
}
