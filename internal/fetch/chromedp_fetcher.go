package fetch

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type ChromedpFetcher struct {
	networkTracker *networkTracker
}

type networkTracker struct {
	inflight      int32         // Number of active network requests (atomic)
	mu            sync.Mutex    // Protects idleSince, done, and closed
	idleSince     time.Time     // When inflight first reached 0
	quietDuration time.Duration // How long to wait at 0 inflight before declaring idle
	done          chan struct{} // Closed when network idle is confirmed
	closed        int32         // 0 = open, 1 = closed (atomic for double-close protection)
	firstSeen     int32         // 0 = not seen, 1 = seen (atomic)
}

func (f *ChromedpFetcher) Fetch(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("Chromedp fetch start", "url", req.URL)

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	// Determine timeouts
	renderTimeout := req.Timeout
	if prof.Timeouts.MaxRenderMs > 0 {
		renderLimit := time.Duration(prof.Timeouts.MaxRenderMs) * time.Millisecond
		if renderLimit < renderTimeout {
			renderTimeout = renderLimit
		}
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying Chromedp fetch", "url", req.URL, "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", req.URL)
			_ = req.Limiter.Wait(ctx, req.URL)
		}

		res, err := f.doFetch(ctx, req, prof, renderTimeout)
		if err == nil {
			slog.Debug("Chromedp fetch success", "url", req.URL)
			return res, nil
		}

		slog.Warn("Chromedp fetch failed", "url", req.URL, "error", err, "attempt", attempt)

		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		delay := backoff(baseDelay, attempt)
		slog.Debug("backing off before retry", "url", req.URL, "delay", delay)
		time.Sleep(delay)
	}

	slog.Error("Chromedp fetch max retries exceeded", "url", req.URL)
	return Result{}, errors.New("max retries exceeded")
}

func (f *ChromedpFetcher) doFetch(parentCtx context.Context, req Request, prof RenderProfile, timeout time.Duration) (Result, error) {
	slog.Debug("starting Chromedp allocator", "url", req.URL, "timeout", timeout)
	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if req.UserAgent != "" {
		allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, allocatorOpts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	// Configure network interception and blocking
	actions := []chromedp.Action{network.Enable()}

	// Headers and Cookies
	if len(req.Auth.Headers) > 0 {
		headers := network.Headers{}
		for k, v := range req.Auth.Headers {
			headers[k] = v
		}
		actions = append(actions, network.SetExtraHTTPHeaders(headers))
	}
	for _, cookie := range req.Auth.Cookies {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			actions = append(actions, network.SetCookie(parts[0], parts[1]))
		}
	}

	// Resource blocking
	blockedPatterns := []string{}
	for _, pattern := range prof.Block.URLPatterns {
		blockedPatterns = append(blockedPatterns, pattern)
	}

	// Map blocked types to patterns if possible, or use request interception
	// Note: chromedp SetBlockedURLs is powerful but simple. For types, we might need request interception.
	// For simplicity in this version, we map common types to extensions or use simple patterns.
	for _, resType := range prof.Block.ResourceTypes {
		switch resType {
		case BlockedResourceImage:
			blockedPatterns = append(blockedPatterns, "*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.svg", "*.ico")
		case BlockedResourceFont:
			blockedPatterns = append(blockedPatterns, "*.woff", "*.woff2", "*.ttf", "*.otf", "*.eot")
		case BlockedResourceStylesheet:
			blockedPatterns = append(blockedPatterns, "*.css")
		case BlockedResourceMedia:
			blockedPatterns = append(blockedPatterns, "*.mp4", "*.mp3", "*.webm")
		}
	}
	if len(blockedPatterns) > 0 {
		slog.Debug("blocking resources", "url", req.URL, "patterns", blockedPatterns)
		actions = append(actions, network.SetBlockedURLs(blockedPatterns))
	}

	// Run initial setup
	if err := chromedp.Run(ctx, actions...); err != nil {
		slog.Error("Chromedp setup failed", "url", req.URL, "error", err)
		return Result{}, err
	}

	// Login flow if configured
	currentURL := ""
	if req.Auth.LoginURL != "" {
		slog.Info("performing headless login", "url", req.URL, "loginURL", req.Auth.LoginURL)
		err := f.performLogin(ctx, req.Auth)
		if err != nil {
			slog.Error("headless login failed", "url", req.URL, "loginURL", req.Auth.LoginURL, "error", err)
			return Result{}, err
		}
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return Result{}, err
		}
		slog.Info("login complete", "url", req.URL, "currentURL", currentURL)
	}

	if len(req.PreNavJS) > 0 {
		slog.Debug("running pre-navigation JS", "url", req.URL, "count", len(req.PreNavJS))
		if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
			return Result{}, err
		}
		for _, script := range req.PreNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
				slog.Error("pre-navigation JS failed", "url", req.URL, "error", err)
				return Result{}, err
			}
		}
	}

	// Navigate to target
	if currentURL == "" || currentURL == req.Auth.LoginURL {
		slog.Debug("navigating to target", "url", req.URL)
		if err := chromedp.Run(ctx, chromedp.Navigate(req.URL)); err != nil {
			if !isAbortErr(err) {
				slog.Error("navigation failed", "url", req.URL, "error", err)
				return Result{}, err
			}
			slog.Warn("navigation aborted (ignored)", "url", req.URL, "error", err)
		}
	}

	// Wait strategies
	slog.Debug("waiting for page to be ready", "url", req.URL, "mode", prof.Wait.Mode)
	waitErr := f.performWait(ctx, prof.Wait)
	if waitErr != nil && !strings.Contains(waitErr.Error(), "timeout") {
		// Log error but might try to capture HTML anyway?
		// For now, fail on wait error unless it's just a timeout and we want partial results.
		// Strict strictness: fail.
		slog.Error("wait strategy failed", "url", req.URL, "mode", prof.Wait.Mode, "error", waitErr)
		return Result{}, waitErr
	}
	if waitErr != nil && strings.Contains(waitErr.Error(), "timeout") {
		slog.Warn("wait strategy timed out (continuing)", "url", req.URL, "mode", prof.Wait.Mode)
	}

	// Extra sleep if requested
	if prof.Wait.ExtraSleepMs > 0 {
		slog.Debug("extra sleep", "url", req.URL, "ms", prof.Wait.ExtraSleepMs)
		_ = chromedp.Run(ctx, chromedp.Sleep(time.Duration(prof.Wait.ExtraSleepMs)*time.Millisecond))
	}

	for _, selector := range req.WaitSelectors {
		if strings.TrimSpace(selector) == "" {
			continue
		}
		slog.Debug("waiting for selector", "url", req.URL, "selector", selector)
		if err := chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery)); err != nil {
			slog.Error("wait for selector failed", "url", req.URL, "selector", selector, "error", err)
			return Result{}, err
		}
	}

	if len(req.PostNavJS) > 0 {
		slog.Debug("running post-navigation JS", "url", req.URL, "count", len(req.PostNavJS))
		for _, script := range req.PostNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
				slog.Error("post-navigation JS failed", "url", req.URL, "error", err)
				return Result{}, err
			}
		}
	}

	var html string
	slog.Debug("capturing outer HTML", "url", req.URL)
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
		slog.Error("failed to capture HTML", "url", req.URL, "error", err)
		return Result{}, err
	}

	return Result{
		URL:          req.URL,
		Status:       200, // Chromedp doesn't easily give status on navigation without ListenTarget
		HTML:         html,
		FetchedAt:    time.Now(),
		Engine:       RenderEngineChromedp,
		ETag:         "", // Headless browsers don't easily expose response headers without complex interception
		LastModified: "",
	}, nil
}

func (f *ChromedpFetcher) waitForNetworkIdle(ctx context.Context, policy RenderWaitPolicy) error {
	quietMs := policy.NetworkIdleQuietMs
	if quietMs <= 0 {
		quietMs = 500 // Default 500ms quiet window
	}

	tracker := &networkTracker{
		quietDuration: time.Duration(quietMs) * time.Millisecond,
		done:          make(chan struct{}),
	}

	slog.Debug("network idle wait started", "quietMs", quietMs)
	start := time.Now()

	chromedp.ListenTarget(ctx, tracker.onEvent)

	select {
	case <-tracker.done:
		duration := time.Since(start)
		slog.Debug("network idle detected", "duration", duration.Milliseconds())
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(quietMs) * time.Millisecond):
		if atomic.LoadInt32(&tracker.firstSeen) == 0 {
			slog.Debug("no network events received, assuming already idle")
			return nil
		}
	}
	return nil
}

func (t *networkTracker) onEvent(ev any) {
	switch ev := ev.(type) {
	case *network.EventRequestWillBeSent:
		if atomic.LoadInt32(&t.firstSeen) == 0 {
			atomic.StoreInt32(&t.firstSeen, 1)
			atomic.StoreInt32(&t.inflight, 1)
		} else {
			atomic.AddInt32(&t.inflight, 1)
		}
		t.resetIdleSince()
		slog.Debug("request started", "requestId", ev.RequestID, "inflight", atomic.LoadInt32(&t.inflight))

	case *network.EventLoadingFinished:
		if atomic.LoadInt32(&t.firstSeen) == 0 {
			atomic.StoreInt32(&t.firstSeen, 1)
			atomic.StoreInt32(&t.inflight, 0)
		} else {
			newCount := atomic.AddInt32(&t.inflight, -1)
			slog.Debug("request finished", "requestId", ev.RequestID, "inflight", newCount)
			if newCount < 0 {
				slog.Warn("inflight counter went negative", "count", newCount, "requestId", ev.RequestID)
			}
		}
		t.checkIdle()

	case *network.EventLoadingFailed:
		if atomic.LoadInt32(&t.firstSeen) == 0 {
			atomic.StoreInt32(&t.firstSeen, 1)
			atomic.StoreInt32(&t.inflight, 0)
		} else {
			newCount := atomic.AddInt32(&t.inflight, -1)
			slog.Debug("request failed", "requestId", ev.RequestID, "inflight", newCount)
			if newCount < 0 {
				slog.Warn("inflight counter went negative", "count", newCount, "requestId", ev.RequestID)
			}
		}
		t.checkIdle()
	}
}

func (t *networkTracker) resetIdleSince() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.idleSince = time.Time{}
}

func (t *networkTracker) checkIdle() {
	if atomic.LoadInt32(&t.inflight) == 0 {
		t.mu.Lock()
		defer t.mu.Unlock()

		if atomic.LoadInt32(&t.closed) != 0 {
			return
		}

		if t.idleSince.IsZero() {
			t.idleSince = time.Now()
		} else if time.Since(t.idleSince) >= t.quietDuration {
			atomic.StoreInt32(&t.closed, 1)
			close(t.done)
		}
	} else {
		t.resetIdleSince()
	}
}

func (f *ChromedpFetcher) performLogin(ctx context.Context, auth AuthOptions) error {
	if auth.LoginUserSelector == "" || auth.LoginPassSelector == "" || auth.LoginSubmitSelector == "" {
		return errors.New("login selectors are required for headless login")
	}
	return chromedp.Run(ctx,
		chromedp.Navigate(auth.LoginURL),
		chromedp.WaitVisible(auth.LoginUserSelector),
		chromedp.SendKeys(auth.LoginUserSelector, auth.LoginUser),
		chromedp.SendKeys(auth.LoginPassSelector, auth.LoginPass),
		chromedp.Click(auth.LoginSubmitSelector),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
}

func (f *ChromedpFetcher) performWait(ctx context.Context, policy RenderWaitPolicy) error {
	// Always wait for body first
	if err := chromedp.Run(ctx, chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		return err
	}

	switch policy.Mode {
	case RenderWaitModeNetworkIdle:
		return f.waitForNetworkIdle(ctx, policy)
	case RenderWaitModeSelector:
		if policy.Selector != "" {
			return chromedp.Run(ctx, chromedp.WaitVisible(policy.Selector, chromedp.ByQuery))
		}
	case RenderWaitModeStability:
		// Basic stability check: wait loop in Go
		return f.waitForStability(ctx, policy)
	case RenderWaitModeDOMReady:
		// Already waited for body
		return nil
	default:
		// Default behavior
		return nil
	}
	return nil
}

func (f *ChromedpFetcher) waitForStability(ctx context.Context, policy RenderWaitPolicy) error {
	pollInterval := time.Duration(policy.StabilityPollMs) * time.Millisecond
	if pollInterval == 0 {
		pollInterval = 200 * time.Millisecond
	}
	minLen := policy.MinTextLength

	var lastLen int
	stableIterations := 0
	targetIterations := policy.StabilityIterations
	if targetIterations <= 0 {
		targetIterations = 3
	}

	for i := 0; i < 20; i++ { // Max 20 polls to avoid infinite loop
		var text string
		if err := chromedp.Run(ctx, chromedp.Text("body", &text, chromedp.ByQuery)); err != nil {
			return err
		}
		curLen := len(text)

		if curLen >= minLen && curLen == lastLen {
			stableIterations++
		} else {
			stableIterations = 0
		}

		if stableIterations >= targetIterations {
			return nil
		}

		lastLen = curLen
		time.Sleep(pollInterval)
	}
	return nil
}

func isAbortErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "net::ERR_ABORTED")
}
