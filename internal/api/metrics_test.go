// Package api provides unit tests for metrics collection and reporting.
// Tests cover RingBuffer, MetricsCollector, request recording, and snapshot generation.
// Does NOT test HTTP endpoint handlers for metrics (covered in other test files).
package api

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRingBuffer(t *testing.T) {
	t.Run("creates buffer with correct capacity", func(t *testing.T) {
		buf := NewRingBuffer[int](10)
		if buf.capacity != 10 {
			t.Errorf("expected capacity 10, got %d", buf.capacity)
		}
		if buf.Size() != 0 {
			t.Errorf("expected initial size 0, got %d", buf.Size())
		}
	})
}

func TestRingBuffer_Push(t *testing.T) {
	t.Run("pushes items in order", func(t *testing.T) {
		buf := NewRingBuffer[int](3)
		buf.Push(1)
		buf.Push(2)
		buf.Push(3)

		items := buf.GetAll()
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
		if items[0] != 1 || items[1] != 2 || items[2] != 3 {
			t.Errorf("expected [1, 2, 3], got %v", items)
		}
	})

	t.Run("overwrites old items when full", func(t *testing.T) {
		buf := NewRingBuffer[int](3)
		buf.Push(1)
		buf.Push(2)
		buf.Push(3)
		buf.Push(4) // Should overwrite 1

		items := buf.GetAll()
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
		if items[0] != 2 || items[1] != 3 || items[2] != 4 {
			t.Errorf("expected [2, 3, 4], got %v", items)
		}
	})
}

func TestRingBuffer_GetAll(t *testing.T) {
	t.Run("returns nil when empty", func(t *testing.T) {
		buf := NewRingBuffer[int](10)
		items := buf.GetAll()
		if items != nil {
			t.Errorf("expected nil, got %v", items)
		}
	})

	t.Run("returns items in correct order after wrap", func(t *testing.T) {
		buf := NewRingBuffer[int](3)
		buf.Push(1)
		buf.Push(2)
		buf.Push(3)
		buf.Push(4)
		buf.Push(5)

		items := buf.GetAll()
		expected := []int{3, 4, 5}
		if len(items) != len(expected) {
			t.Errorf("expected %v, got %v", expected, items)
		}
		for i, v := range expected {
			if items[i] != v {
				t.Errorf("expected items[%d] = %d, got %d", i, v, items[i])
			}
		}
	})
}

func TestNewMetricsCollector(t *testing.T) {
	t.Run("creates collector with default values", func(t *testing.T) {
		mc := NewMetricsCollector()
		if mc == nil {
			t.Fatal("expected non-nil collector")
		}
		if mc.requestMetrics == nil {
			t.Error("expected request metrics buffer to be initialized")
		}
		if mc.jobDurations == nil {
			t.Error("expected job durations buffer to be initialized")
		}
		if mc.retention != DefaultMetricsRetention {
			t.Errorf("expected retention %v, got %v", DefaultMetricsRetention, mc.retention)
		}
	})
}

func TestMetricsCollector_SetDefaultRateLimit(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetDefaultRateLimit(20, 10)

	if mc.defaultQPS != 20 {
		t.Errorf("expected QPS 20, got %f", mc.defaultQPS)
	}
	if mc.defaultBurst != 10 {
		t.Errorf("expected burst 10, got %d", mc.defaultBurst)
	}
}

func TestMetricsCollector_RegisterHostLimiter(t *testing.T) {
	mc := NewMetricsCollector()
	limiter := rate.NewLimiter(10, 5)

	mc.RegisterHostLimiter("example.com", limiter, 10, 5)

	mc.hostLimitersMu.RLock()
	defer mc.hostLimitersMu.RUnlock()

	if _, ok := mc.hostLimiters["example.com"]; !ok {
		t.Error("expected host to be registered")
	}
}

func TestMetricsCollector_RecordRequest(t *testing.T) {
	mc := NewMetricsCollector()

	t.Run("records request metric", func(t *testing.T) {
		mc.RecordRequest(100*time.Millisecond, true, "http", "https://example.com/path")

		if mc.totalRequests != 1 {
			t.Errorf("expected totalRequests 1, got %d", mc.totalRequests)
		}
		if mc.successCount != 1 {
			t.Errorf("expected successCount 1, got %d", mc.successCount)
		}
	})

	t.Run("records failure metric", func(t *testing.T) {
		mc.RecordRequest(50*time.Millisecond, false, "chromedp", "https://example.com/fail")

		if mc.totalRequests != 2 {
			t.Errorf("expected totalRequests 2, got %d", mc.totalRequests)
		}
		if mc.failureCount != 1 {
			t.Errorf("expected failureCount 1, got %d", mc.failureCount)
		}
	})

	t.Run("updates fetcher usage counters", func(t *testing.T) {
		mc.RecordRequest(100*time.Millisecond, true, "playwright", "https://example.com/pw")

		if mc.fetcherUsage.Playwright != 1 {
			t.Errorf("expected playwright usage 1, got %d", mc.fetcherUsage.Playwright)
		}
	})
}

func TestMetricsCollector_StartRequest(t *testing.T) {
	mc := NewMetricsCollector()

	mc.StartRequest()
	if mc.activeRequests != 1 {
		t.Errorf("expected activeRequests 1, got %d", mc.activeRequests)
	}

	mc.StartRequest()
	if mc.activeRequests != 2 {
		t.Errorf("expected activeRequests 2, got %d", mc.activeRequests)
	}
}

func TestMetricsCollector_RecordJobDuration(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordJobDuration(5 * time.Second)
	mc.RecordJobDuration(10 * time.Second)

	if mc.jobDurations.Size() != 2 {
		t.Errorf("expected 2 durations recorded, got %d", mc.jobDurations.Size())
	}
}

func TestMetricsCollector_GetSnapshot(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetDefaultRateLimit(10, 5)

	// Register a host limiter
	limiter := rate.NewLimiter(10, 5)
	mc.RegisterHostLimiter("example.com", limiter, 10, 5)

	// Record some requests
	mc.RecordRequest(100*time.Millisecond, true, "http", "https://example.com/1")
	mc.RecordRequest(200*time.Millisecond, true, "http", "https://example.com/2")
	mc.RecordRequest(50*time.Millisecond, false, "chromedp", "https://example.com/3")

	// Record job durations
	mc.RecordJobDuration(5 * time.Second)
	mc.RecordJobDuration(10 * time.Second)

	snapshot := mc.GetSnapshot()

	t.Run("returns correct request counts", func(t *testing.T) {
		if snapshot.TotalRequests != 3 {
			t.Errorf("expected TotalRequests 3, got %d", snapshot.TotalRequests)
		}
		if snapshot.SuccessRate != 66.66666666666667 {
			t.Errorf("expected SuccessRate ~66.67, got %f", snapshot.SuccessRate)
		}
	})

	t.Run("returns correct fetcher usage", func(t *testing.T) {
		if snapshot.FetcherUsage.HTTP != 2 {
			t.Errorf("expected HTTP usage 2, got %d", snapshot.FetcherUsage.HTTP)
		}
		if snapshot.FetcherUsage.Chromedp != 1 {
			t.Errorf("expected Chromedp usage 1, got %d", snapshot.FetcherUsage.Chromedp)
		}
	})

	t.Run("returns correct job metrics", func(t *testing.T) {
		expectedAvg := 7500.0 // (5000 + 10000) / 2 = 7500ms
		if snapshot.AvgJobDuration != expectedAvg {
			t.Errorf("expected AvgJobDuration %f, got %f", expectedAvg, snapshot.AvgJobDuration)
		}
	})

	t.Run("returns rate limit status", func(t *testing.T) {
		if len(snapshot.RateLimitStatus) != 1 {
			t.Errorf("expected 1 rate limit status, got %d", len(snapshot.RateLimitStatus))
		}
		if snapshot.RateLimitStatus[0].Host != "example.com" {
			t.Errorf("expected host example.com, got %s", snapshot.RateLimitStatus[0].Host)
		}
	})

	t.Run("returns timestamp", func(t *testing.T) {
		if snapshot.Timestamp == 0 {
			t.Error("expected non-zero timestamp")
		}
	})
}

func TestMetricsCollector_GetRateLimitStatus(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetDefaultRateLimit(10, 5)

	// Register host limiters
	limiter1 := rate.NewLimiter(10, 5)
	limiter2 := rate.NewLimiter(20, 10)
	mc.RegisterHostLimiter("example.com", limiter1, 10, 5)
	mc.RegisterHostLimiter("test.org", limiter2, 20, 10)

	status := mc.GetRateLimitStatus()

	if len(status) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(status))
	}

	// Check that hosts are present
	hosts := make(map[string]bool)
	for _, s := range status {
		hosts[s.Host] = true
		if s.QPS != 10 && s.QPS != 20 {
			t.Errorf("unexpected QPS %f for host %s", s.QPS, s.Host)
		}
	}

	if !hosts["example.com"] {
		t.Error("expected example.com in status")
	}
	if !hosts["test.org"] {
		t.Error("expected test.org in status")
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector()

	// Add some data
	mc.RecordRequest(100*time.Millisecond, true, "http", "https://example.com/1")
	mc.RecordJobDuration(5 * time.Second)
	mc.RegisterHostLimiter("example.com", rate.NewLimiter(10, 5), 10, 5)

	// Reset
	mc.Reset()

	t.Run("clears request metrics", func(t *testing.T) {
		if mc.totalRequests != 0 {
			t.Errorf("expected totalRequests 0, got %d", mc.totalRequests)
		}
		if mc.successCount != 0 {
			t.Errorf("expected successCount 0, got %d", mc.successCount)
		}
	})

	t.Run("clears host limiters", func(t *testing.T) {
		mc.hostLimitersMu.RLock()
		defer mc.hostLimitersMu.RUnlock()
		if len(mc.hostLimiters) != 0 {
			t.Errorf("expected 0 host limiters, got %d", len(mc.hostLimiters))
		}
	})

	t.Run("clears job durations", func(t *testing.T) {
		if mc.jobDurations.Size() != 0 {
			t.Errorf("expected 0 job durations, got %d", mc.jobDurations.Size())
		}
	})
}

func TestMetricsCollector_RequestMetric_URLSanitization(t *testing.T) {
	mc := NewMetricsCollector()

	testCases := []struct {
		url      string
		expected string
	}{
		{"https://example.com/path?query=1", "example.com"},
		{"https://api.test.org/v1/users", "api.test.org"},
		{"http://localhost:8080/debug", "localhost:8080"},
		{"invalid-url", ""},
	}

	for _, tc := range testCases {
		mc.RecordRequest(100*time.Millisecond, true, "http", tc.url)

		metrics := mc.requestMetrics.GetAll()
		lastMetric := metrics[len(metrics)-1]

		if lastMetric.Host != tc.expected {
			t.Errorf("URL %s: expected host %q, got %q", tc.url, tc.expected, lastMetric.Host)
		}
	}
}

func BenchmarkMetricsCollector_RecordRequest(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	for b.Loop() {
		mc.RecordRequest(100*time.Millisecond, true, "http", "https://example.com/test")
	}
}

func BenchmarkMetricsCollector_GetSnapshot(b *testing.B) {
	mc := NewMetricsCollector()

	// Pre-populate with data
	for range 1000 {
		mc.RecordRequest(100*time.Millisecond, true, "http", "https://example.com/test")
	}

	b.ResetTimer()
	for b.Loop() {
		_ = mc.GetSnapshot()
	}
}
