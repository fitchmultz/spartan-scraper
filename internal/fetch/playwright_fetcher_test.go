package fetch

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPlaywrightFetcher_SingletonBehavior verifies that the same Playwright
// and browser instances are reused across multiple fetch operations.
func TestPlaywrightFetcher_SingletonBehavior(t *testing.T) {
	f := &PlaywrightFetcher{}

	// First fetch - should initialize
	f.mu.Lock()
	firstInit := !f.initialized
	f.mu.Unlock()

	if !firstInit {
		t.Error("expected fetcher to be uninitialized before first fetch")
	}

	// Simulate initialization
	err := f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Skipf("Skipping test: Playwright not available: %v", err)
		return
	}

	f.mu.RLock()
	firstPW := f.pw
	firstBrowser := f.browser
	firstHeadless := f.headless
	wasInitialized := f.initialized
	f.mu.RUnlock()

	if !wasInitialized {
		t.Error("expected fetcher to be initialized after ensureInitialized")
	}
	if firstPW == nil {
		t.Error("expected playwright instance to be non-nil")
	}
	if firstBrowser == nil {
		t.Error("expected browser instance to be non-nil")
	}
	if !firstHeadless {
		t.Error("expected headless to be true")
	}

	// Second ensureInitialized with same headless setting - should reuse
	err = f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Fatalf("second ensureInitialized failed: %v", err)
	}

	f.mu.RLock()
	secondPW := f.pw
	secondBrowser := f.browser
	f.mu.RUnlock()

	if secondPW != firstPW {
		t.Error("expected same playwright instance to be reused")
	}
	if secondBrowser != firstBrowser {
		t.Error("expected same browser instance to be reused")
	}

	// Cleanup
	_ = f.Close()
}

// TestPlaywrightFetcher_HeadlessModeSwitch verifies that switching headless
// mode triggers cleanup and reinitialization.
func TestPlaywrightFetcher_HeadlessModeSwitch(t *testing.T) {
	f := &PlaywrightFetcher{}

	// Initialize with headless=true
	err := f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Skipf("Skipping test: Playwright not available: %v", err)
		return
	}

	f.mu.RLock()
	firstBrowser := f.browser
	f.mu.RUnlock()

	// Switch to headless=false
	err = f.ensureInitialized(context.Background(), false)
	if err != nil {
		t.Fatalf("ensureInitialized with headless=false failed: %v", err)
	}

	f.mu.RLock()
	secondBrowser := f.browser
	newHeadless := f.headless
	f.mu.RUnlock()

	if secondBrowser == firstBrowser {
		t.Error("expected new browser instance after headless mode switch")
	}
	if newHeadless {
		t.Error("expected headless to be false")
	}

	// Cleanup
	_ = f.Close()
}

// TestPlaywrightFetcher_ConcurrentSafety verifies that concurrent fetches
// are handled safely without races.
func TestPlaywrightFetcher_ConcurrentSafety(t *testing.T) {
	f := &PlaywrightFetcher{}

	ctx := context.Background()
	const numGoroutines = 10

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(headless bool) {
			defer wg.Done()
			err := f.ensureInitialized(ctx, headless)
			if err != nil {
				errChan <- err
			}
		}(i%2 == 0) // Mix of headless and non-headless
	}

	wg.Wait()
	close(errChan)

	// Check for initialization errors (but skip if Playwright not available)
	errorCount := 0
	for err := range errChan {
		if err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Skipf("Skipping test: Playwright not available")
		return
	}

	// Verify that the fetcher is in a consistent state
	f.mu.RLock()
	initialized := f.initialized
	hasPW := f.pw != nil
	hasBrowser := f.browser != nil
	f.mu.RUnlock()

	if !initialized {
		t.Error("expected fetcher to be initialized after concurrent calls")
	}
	if !hasPW {
		t.Error("expected playwright instance to be non-nil")
	}
	if !hasBrowser {
		t.Error("expected browser instance to be non-nil")
	}

	// Cleanup
	_ = f.Close()
}

// TestPlaywrightFetcher_Close verifies that Close properly cleans up resources.
func TestPlaywrightFetcher_Close(t *testing.T) {
	f := &PlaywrightFetcher{}

	// Initialize
	err := f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Skipf("Skipping test: Playwright not available: %v", err)
		return
	}

	// Verify initialized
	f.mu.RLock()
	wasInitialized := f.initialized
	f.mu.RUnlock()

	if !wasInitialized {
		t.Error("expected fetcher to be initialized")
	}

	// Close
	err = f.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify cleanup
	f.mu.RLock()
	isInitialized := f.initialized
	pwNil := f.pw == nil
	browserNil := f.browser == nil
	f.mu.RUnlock()

	if isInitialized {
		t.Error("expected fetcher to be uninitialized after Close")
	}
	if !pwNil {
		t.Error("expected playwright instance to be nil after Close")
	}
	if !browserNil {
		t.Error("expected browser instance to be nil after Close")
	}

	// Calling Close again should be safe
	err = f.Close()
	if err != nil {
		t.Errorf("Close called twice should be safe, got error: %v", err)
	}
}

// TestPlaywrightFetcher_CloseConcurrentWithFetch verifies that Close is safe
// even when concurrent operations might be in progress.
func TestPlaywrightFetcher_CloseConcurrentWithFetch(t *testing.T) {
	f := &PlaywrightFetcher{}

	ctx := context.Background()

	// Initialize first
	err := f.ensureInitialized(ctx, true)
	if err != nil {
		t.Skipf("Skipping test: Playwright not available: %v", err)
		return
	}

	// Close should be safe to call
	err = f.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Start goroutines that try to initialize after close
	// This tests that Close properly cleans up and reinitialization works
	done := make(chan struct{})
	go func() {
		for range 100 {
			_ = f.ensureInitialized(ctx, true)
			time.Sleep(1 * time.Millisecond)
		}
		close(done)
	}()

	// Wait a bit for goroutines to potentially initialize
	time.Sleep(10 * time.Millisecond)

	// Close again - should be safe regardless of goroutine state
	err = f.Close()
	if err != nil {
		t.Errorf("Close called second time failed: %v", err)
	}

	// Wait for goroutines to finish
	<-done

	// Final close to clean up any state
	_ = f.Close()
}

// TestPlaywrightFetcher_NotInitialized verifies behavior before initialization.
func TestPlaywrightFetcher_NotInitialized(t *testing.T) {
	f := &PlaywrightFetcher{}

	f.mu.RLock()
	initialized := f.initialized
	pwNil := f.pw == nil
	browserNil := f.browser == nil
	f.mu.RUnlock()

	if initialized {
		t.Error("expected new fetcher to be uninitialized")
	}
	if !pwNil {
		t.Error("expected playwright instance to be nil initially")
	}
	if !browserNil {
		t.Error("expected browser instance to be nil initially")
	}
}

// TestPlaywrightFetcher_BrowserReuseSequential verifies that the same browser
// instance is used for sequential fetch operations.
func TestPlaywrightFetcher_BrowserReuseSequential(t *testing.T) {
	f := &PlaywrightFetcher{}

	// Initialize
	err := f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Skipf("Skipping test: Playwright not available: %v", err)
		return
	}

	f.mu.RLock()
	firstBrowser := f.browser
	f.mu.RUnlock()

	// Call ensureInitialized again with same settings
	err = f.ensureInitialized(context.Background(), true)
	if err != nil {
		t.Fatalf("second ensureInitialized failed: %v", err)
	}

	f.mu.RLock()
	secondBrowser := f.browser
	f.mu.RUnlock()

	if secondBrowser != firstBrowser {
		t.Error("browser instance changed on re-initialization with same settings")
	}

	// Cleanup
	_ = f.Close()
}
