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
