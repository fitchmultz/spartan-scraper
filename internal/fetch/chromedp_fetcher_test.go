// Package fetch provides tests for the Chromedp fetcher.
// Tests cover network tracking, response tracking, URL matching, and context cancellation handling.
// Does NOT test actual browser rendering or full fetch integration (skipped in short mode).
package fetch

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
)

func TestNetworkTracker(t *testing.T) {
	t.Run("single request", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 100 * time.Millisecond,
			done:          make(chan struct{}),
		}

		tracker.onEvent(&network.EventRequestWillBeSent{})
		assert.Equal(t, int32(1), atomic.LoadInt32(&tracker.inflight))

		tracker.onEvent(&network.EventLoadingFinished{})
		assert.Equal(t, int32(0), atomic.LoadInt32(&tracker.inflight))

		time.Sleep(150 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			// Success
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}
	})

	t.Run("multiple concurrent requests", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 100 * time.Millisecond,
			done:          make(chan struct{}),
		}

		tracker.onEvent(&network.EventRequestWillBeSent{})
		tracker.onEvent(&network.EventRequestWillBeSent{})
		assert.Equal(t, int32(2), atomic.LoadInt32(&tracker.inflight))

		tracker.onEvent(&network.EventLoadingFinished{})
		assert.Equal(t, int32(1), atomic.LoadInt32(&tracker.inflight))

		time.Sleep(50 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			t.Fatal("should not be idle yet")
		default:
			// Expected
		}

		tracker.onEvent(&network.EventLoadingFinished{})
		time.Sleep(150 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			// Success
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}
	})

	t.Run("new request during quiet window resets timer", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 50 * time.Millisecond,
			done:          make(chan struct{}),
		}

		tracker.onEvent(&network.EventRequestWillBeSent{})
		tracker.onEvent(&network.EventLoadingFinished{})
		time.Sleep(25 * time.Millisecond)

		tracker.onEvent(&network.EventRequestWillBeSent{})
		tracker.onEvent(&network.EventLoadingFinished{})

		start := time.Now()
		time.Sleep(75 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			elapsed := time.Since(start)
			assert.True(t, elapsed >= 50*time.Millisecond, "should wait full quiet period")
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}
	})

	t.Run("failed requests decrement inflight", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 100 * time.Millisecond,
			done:          make(chan struct{}),
		}

		tracker.onEvent(&network.EventRequestWillBeSent{})
		assert.Equal(t, int32(1), atomic.LoadInt32(&tracker.inflight))

		tracker.onEvent(&network.EventLoadingFailed{})
		assert.Equal(t, int32(0), atomic.LoadInt32(&tracker.inflight))

		time.Sleep(150 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			// Success
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}
	})

	t.Run("rapid burst of requests", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 50 * time.Millisecond,
			done:          make(chan struct{}),
		}

		for i := 0; i < 10; i++ {
			tracker.onEvent(&network.EventRequestWillBeSent{})
		}
		assert.Equal(t, int32(10), atomic.LoadInt32(&tracker.inflight))

		for i := 0; i < 10; i++ {
			tracker.onEvent(&network.EventLoadingFinished{})
		}
		assert.Equal(t, int32(0), atomic.LoadInt32(&tracker.inflight))

		time.Sleep(75 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
			// Success
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}
	})

	t.Run("close only happens once", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 50 * time.Millisecond,
			done:          make(chan struct{}),
		}

		tracker.onEvent(&network.EventRequestWillBeSent{})
		tracker.onEvent(&network.EventLoadingFinished{})
		time.Sleep(100 * time.Millisecond)

		tracker.checkIdle()

		tracker.checkIdle()

		select {
		case <-tracker.done:
		case <-time.After(50 * time.Millisecond):
			t.Fatal("channel should be closed")
		}
	})

	t.Run("concurrent events from multiple goroutines", func(t *testing.T) {
		tracker := &networkTracker{
			quietDuration: 100 * time.Millisecond,
			done:          make(chan struct{}),
		}

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tracker.onEvent(&network.EventRequestWillBeSent{})
				time.Sleep(time.Duration(i%50) * time.Millisecond)
				if i%2 == 0 {
					tracker.onEvent(&network.EventLoadingFinished{})
				} else {
					tracker.onEvent(&network.EventLoadingFailed{})
				}
			}()
		}
		wg.Wait()
		time.Sleep(150 * time.Millisecond)
		tracker.checkIdle()

		select {
		case <-tracker.done:
		case <-time.After(50 * time.Millisecond):
			t.Fatal("timeout waiting for idle detection")
		}

		assert.Equal(t, int32(0), atomic.LoadInt32(&tracker.inflight))
	})
}

func TestChromedpFetcher_Fetch_NetworkIdle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocatorOpts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	req := Request{
		URL:      "https://example.com",
		Timeout:  30 * time.Second,
		Headless: true,
	}

	prof := RenderProfile{
		Wait: RenderWaitPolicy{
			Mode:               RenderWaitModeNetworkIdle,
			NetworkIdleQuietMs: 1000,
		},
	}

	fetcher := &ChromedpFetcher{}
	result, err := fetcher.Fetch(taskCtx, req, prof)

	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assert.NotEmpty(t, result.HTML)
	assert.Equal(t, 200, result.Status)
	assert.Equal(t, RenderEngineChromedp, result.Engine)
	assert.NotZero(t, result.FetchedAt)
}

// TestChromedpFetch_ContextCancellationDuringLimiterWait verifies that context
// cancellation is properly propagated when waiting for the rate limiter.
//
// This test documents the fix for RQ-0022: the Chromedp fetcher checks the
// error return from req.Limiter.Wait and returns immediately on cancellation
// instead of continuing to allocate Chrome browser contexts and perform the fetch.
func TestChromedpFetch_ContextCancellationDuringLimiterWait(t *testing.T) {
	limiter := NewHostLimiter(1, 1)

	// Consume the burst token so that the next Fetch call will block in Wait.
	// We call Wait directly on the limiter to avoid needing a real Chrome binary.
	ctx := context.Background()
	_ = limiter.Wait(ctx, "http://example.com")

	// Now create a cancelled context for the fetch request
	// This request will need to wait for the rate limiter, but the context
	// is already cancelled, so Wait should return immediately with context.Canceled
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	fetcher := &ChromedpFetcher{}
	req := Request{
		URL:     "http://example.com",
		Timeout: 5 * time.Second,
		Limiter: limiter,
	}

	result, err := fetcher.Fetch(cancelledCtx, req, RenderProfile{})

	// Assert: should return context.Canceled error
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Assert: result should be empty (zero value)
	if result.URL != "" || result.Status != 0 || result.HTML != "" {
		t.Errorf("expected empty Result, got %+v", result)
	}
}

func TestResponseTracker(t *testing.T) {
	t.Run("captures status from matching document response", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com",
				Status: 200,
			},
		})

		assert.Equal(t, int64(200), rt.getStatus())
	})

	t.Run("captures status from redirected URL", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com/welcome",
				Status: 302,
			},
		})

		assert.Equal(t, int64(302), rt.getStatus())
	})

	t.Run("captures status when target is prefix of response", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com/path"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com/path/to/page",
				Status: 200,
			},
		})

		assert.Equal(t, int64(200), rt.getStatus())
	})

	t.Run("rejects non-matching host", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://other.com/path",
				Status: 200,
			},
		})

		assert.Equal(t, int64(0), rt.getStatus())
	})

	t.Run("rejects different resource types", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeScript,
			Response: &network.Response{
				URL:    "https://example.com/script.js",
				Status: 200,
			},
		})

		assert.Equal(t, int64(0), rt.getStatus())
	})

	t.Run("rejects false positive prefix with same host", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com/api"}

		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com/app",
				Status: 404,
			},
		})

		// Should NOT match - "app" is not a prefix of "api" nor vice versa
		assert.Equal(t, int64(0), rt.getStatus())
	})

	t.Run("only captures first matching document", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		// First response
		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com",
				Status: 302,
			},
		})

		// Second response (redirected)
		rt.onEvent(&network.EventResponseReceived{
			Type: network.ResourceTypeDocument,
			Response: &network.Response{
				URL:    "https://example.com/home",
				Status: 200,
			},
		})

		// Should have captured the first status
		assert.Equal(t, int64(302), rt.getStatus())
	})

	t.Run("ignores non-ResponseReceived events", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		rt.onEvent(&network.EventRequestWillBeSent{})
		rt.onEvent(&network.EventLoadingFinished{})

		assert.Equal(t, int64(0), rt.getStatus())
	})

	t.Run("thread safe concurrent access", func(t *testing.T) {
		rt := &responseTracker{targetURL: "https://example.com"}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				rt.onEvent(&network.EventResponseReceived{
					Type: network.ResourceTypeDocument,
					Response: &network.Response{
						URL:    "https://example.com",
						Status: int64(200 + idx),
					},
				})
			}(i)
		}
		wg.Wait()

		// Should have captured exactly one status
		status := rt.getStatus()
		assert.True(t, status >= 200 && status < 300, "status should be in range [200, 300)")
	})

	t.Run("urlMatch exact match", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com", "https://example.com"))
	})

	t.Run("urlMatch prefix with same host", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com", "https://example.com/path"))
		assert.True(t, rt.urlsMatch("https://example.com/path", "https://example.com/path/to/page"))
	})

	t.Run("urlMatch different host", func(t *testing.T) {
		rt := &responseTracker{}
		assert.False(t, rt.urlsMatch("https://example.com", "https://other.com"))
		assert.False(t, rt.urlsMatch("https://example.com/path", "https://other.com/path"))
	})

	t.Run("urlMatch different scheme", func(t *testing.T) {
		rt := &responseTracker{}
		assert.False(t, rt.urlsMatch("https://example.com", "http://example.com"))
		assert.False(t, rt.urlsMatch("https://example.com/path", "http://example.com/path"))
	})

	t.Run("urlMatch prevents false positive prefix", func(t *testing.T) {
		rt := &responseTracker{}
		// "app" is a prefix of "apple" but they don't share the same full path structure
		// However, since they both start with /a, this could be a false positive
		// The implementation should handle this by checking the base URL
		assert.True(t, rt.urlsMatch("https://example.com/api", "https://example.com/api/v1"))
		// Different prefix should not match
		assert.False(t, rt.urlsMatch("https://example.com/api", "https://example.com/app"))
	})

	t.Run("urlMatch with ports", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com:8080", "https://example.com:8080/path"))
		assert.True(t, rt.urlsMatch("https://example.com:8080/api", "https://example.com:8080/api/v1"))
		// Different ports should not match
		assert.False(t, rt.urlsMatch("https://example.com:8080", "https://example.com:9090"))
		assert.False(t, rt.urlsMatch("https://example.com:8080/path", "https://example.com:9090/path"))
		// Explicit port vs implicit default port - these don't match because
		// url.Parse preserves explicit ports in the Host field
		assert.False(t, rt.urlsMatch("https://example.com", "https://example.com:443"))
		assert.False(t, rt.urlsMatch("http://example.com", "http://example.com:80"))
	})

	t.Run("urlMatch with query and fragment", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com/path", "https://example.com/path?query=value"))
		assert.True(t, rt.urlsMatch("https://example.com/path", "https://example.com/path#section"))
		assert.True(t, rt.urlsMatch("https://example.com/path", "https://example.com/path?query=value#section"))
	})

	t.Run("urlMatch with trailing slashes", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com/path", "https://example.com/path/"))
		assert.True(t, rt.urlsMatch("https://example.com/path/", "https://example.com/path"))
		assert.True(t, rt.urlsMatch("https://example.com", "https://example.com/"))
	})

	t.Run("urlMatch with empty paths", func(t *testing.T) {
		rt := &responseTracker{}
		assert.True(t, rt.urlsMatch("https://example.com", "https://example.com/"))
		assert.True(t, rt.urlsMatch("https://example.com/", "https://example.com/path"))
	})
}
