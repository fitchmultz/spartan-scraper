// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
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

type responseTracker struct {
	mu        sync.Mutex
	captured  bool
	status    int64
	targetURL string
}

func (f *ChromedpFetcher) Fetch(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("Chromedp fetch start", "url", apperrors.SanitizeURL(req.URL))

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
			slog.Debug("retrying Chromedp fetch", "url", apperrors.SanitizeURL(req.URL), "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", apperrors.SanitizeURL(req.URL))
			if err := req.Limiter.Wait(ctx, req.URL); err != nil {
				return Result{}, err
			}
		}

		res, err := f.doFetch(ctx, req, prof, renderTimeout)
		if err == nil {
			slog.Debug("Chromedp fetch success", "url", apperrors.SanitizeURL(req.URL))
			return res, nil
		}

		slog.Warn("Chromedp fetch failed", "url", apperrors.SanitizeURL(req.URL), "error", err, "attempt", attempt)

		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		delay := backoff(baseDelay, attempt)
		slog.Debug("backing off before retry", "url", apperrors.SanitizeURL(req.URL), "delay", delay)
		time.Sleep(delay)
	}

	slog.Error("Chromedp fetch max retries exceeded", "url", apperrors.SanitizeURL(req.URL))
	return Result{}, errors.New("max retries exceeded")
}

func (f *ChromedpFetcher) doFetch(parentCtx context.Context, req Request, prof RenderProfile, timeout time.Duration) (Result, error) {
	slog.Debug("starting Chromedp allocator", "url", apperrors.SanitizeURL(req.URL), "timeout", timeout)
	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if req.UserAgent != "" {
		allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
	}

	// Add proxy configuration if provided
	if req.Auth.Proxy != nil && req.Auth.Proxy.URL != "" {
		// For authenticated proxies, embed credentials in the URL
		proxyURL := req.Auth.Proxy.URL
		if req.Auth.Proxy.Username != "" {
			parsedURL, err := url.Parse(req.Auth.Proxy.URL)
			if err == nil {
				// Reconstruct URL with credentials
				userInfo := url.UserPassword(req.Auth.Proxy.Username, req.Auth.Proxy.Password)
				parsedURL.User = userInfo
				proxyURL = parsedURL.String()
			}
		}
		allocatorOpts = append(allocatorOpts, chromedp.ProxyServer(proxyURL))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, allocatorOpts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	// Apply device emulation if specified
	device := f.resolveDevice(req, prof)
	if device != nil {
		slog.Debug("applying device emulation", "url", apperrors.SanitizeURL(req.URL), "device", device.Name)
		if err := f.applyDeviceEmulation(ctx, device); err != nil {
			slog.Warn("failed to apply device emulation", "url", apperrors.SanitizeURL(req.URL), "error", err)
		}
	}

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
		slog.Debug("blocking resources", "url", apperrors.SanitizeURL(req.URL), "patterns", blockedPatterns)
		actions = append(actions, network.SetBlockedURLs(blockedPatterns))
	}

	// Run initial setup
	if err := chromedp.Run(ctx, actions...); err != nil {
		slog.Error("Chromedp setup failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}

	// Login flow if configured
	currentURL := ""
	if req.Auth.LoginURL != "" {
		slog.Info("performing headless login", "url", apperrors.SanitizeURL(req.URL), "loginURL", apperrors.SanitizeURL(req.Auth.LoginURL))
		err := f.performLogin(ctx, req.Auth)
		if err != nil {
			slog.Error("headless login failed", "url", apperrors.SanitizeURL(req.URL), "loginURL", apperrors.SanitizeURL(req.Auth.LoginURL), "error", err)
			return Result{}, err
		}
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return Result{}, err
		}
		slog.Info("login complete", "url", apperrors.SanitizeURL(req.URL), "currentURL", apperrors.SanitizeURL(currentURL))
	}

	if len(req.PreNavJS) > 0 {
		slog.Debug("running pre-navigation JS", "url", apperrors.SanitizeURL(req.URL), "count", len(req.PreNavJS))
		if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
			return Result{}, err
		}
		for _, script := range req.PreNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
				slog.Error("pre-navigation JS failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
				return Result{}, err
			}
		}
	}

	// Set up response tracker to capture HTTP status
	respTracker := &responseTracker{
		targetURL: req.URL,
	}
	chromedp.ListenTarget(ctx, respTracker.onEvent)

	// Navigate to target
	if currentURL == "" || currentURL == req.Auth.LoginURL {
		slog.Debug("navigating to target", "url", apperrors.SanitizeURL(req.URL))
		if err := chromedp.Run(ctx, chromedp.Navigate(req.URL)); err != nil {
			if !isAbortErr(err) {
				slog.Error("navigation failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
				return Result{}, err
			}
			slog.Warn("navigation aborted (ignored)", "url", apperrors.SanitizeURL(req.URL), "error", err)
		}
	}

	// Wait strategies
	slog.Debug("waiting for page to be ready", "url", apperrors.SanitizeURL(req.URL), "mode", prof.Wait.Mode)
	waitErr := f.performWait(ctx, prof.Wait)
	if waitErr != nil && !strings.Contains(waitErr.Error(), "timeout") {
		// Log error but might try to capture HTML anyway?
		// For now, fail on wait error unless it's just a timeout and we want partial results.
		// Strict strictness: fail.
		slog.Error("wait strategy failed", "url", apperrors.SanitizeURL(req.URL), "mode", prof.Wait.Mode, "error", waitErr)
		return Result{}, waitErr
	}
	if waitErr != nil && strings.Contains(waitErr.Error(), "timeout") {
		slog.Warn("wait strategy timed out (continuing)", "url", apperrors.SanitizeURL(req.URL), "mode", prof.Wait.Mode)
	}

	// Extra sleep if requested
	if prof.Wait.ExtraSleepMs > 0 {
		slog.Debug("extra sleep", "url", apperrors.SanitizeURL(req.URL), "ms", prof.Wait.ExtraSleepMs)
		_ = chromedp.Run(ctx, chromedp.Sleep(time.Duration(prof.Wait.ExtraSleepMs)*time.Millisecond))
	}

	for _, selector := range req.WaitSelectors {
		if strings.TrimSpace(selector) == "" {
			continue
		}
		slog.Debug("waiting for selector", "url", apperrors.SanitizeURL(req.URL), "selector", selector)
		if err := chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery)); err != nil {
			slog.Error("wait for selector failed", "url", apperrors.SanitizeURL(req.URL), "selector", selector, "error", err)
			return Result{}, err
		}
	}

	if len(req.PostNavJS) > 0 {
		slog.Debug("running post-navigation JS", "url", apperrors.SanitizeURL(req.URL), "count", len(req.PostNavJS))
		for _, script := range req.PostNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
				slog.Error("post-navigation JS failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
				return Result{}, err
			}
		}
	}

	var html string
	slog.Debug("capturing outer HTML", "url", apperrors.SanitizeURL(req.URL))
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
		slog.Error("failed to capture HTML", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}

	// Get the captured status, defaulting to 200 if not captured (for backward compatibility)
	capturedStatus := respTracker.getStatus()
	status := int(capturedStatus)
	if status == 0 {
		status = 200
		slog.Warn("status not captured from network events, using default 200", "url", apperrors.SanitizeURL(req.URL))
	} else {
		slog.Debug("using captured status", "url", apperrors.SanitizeURL(req.URL), "status", status)
	}

	// Capture screenshot if requested
	var screenshotPath string
	if req.Screenshot != nil && req.Screenshot.Enabled {
		path, err := f.captureScreenshot(ctx, req, req.DataDir)
		if err != nil {
			// Log warning but don't fail the fetch if screenshot fails
			slog.Warn("screenshot capture failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
		} else {
			screenshotPath = path
			slog.Debug("screenshot captured", "url", apperrors.SanitizeURL(req.URL), "path", screenshotPath)
		}
	}

	return Result{
		URL:            req.URL,
		Status:         status,
		HTML:           html,
		FetchedAt:      time.Now(),
		Engine:         RenderEngineChromedp,
		ETag:           "", // Headless browsers don't easily expose response headers without complex interception
		LastModified:   "",
		ScreenshotPath: screenshotPath,
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

	timer := time.NewTimer(time.Duration(quietMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-tracker.done:
		duration := time.Since(start)
		slog.Debug("network idle detected", "duration", duration.Milliseconds())
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
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

// onEvent captures the first document response status.
// Note: For redirect chains (e.g., 302 -> 200), this captures the redirect status (302).
// This is intentional as it represents the HTTP-level navigation result.
func (rt *responseTracker) onEvent(ev any) {
	rt.mu.Lock()
	// Double-check: if already captured, release lock and return
	if rt.captured {
		rt.mu.Unlock()
		return
	}

	evResp, ok := ev.(*network.EventResponseReceived)
	if !ok {
		rt.mu.Unlock()
		return
	}

	// Capture the status of the main document request
	if evResp.Type == network.ResourceTypeDocument {
		// Check if the URL matches our target (allowing for redirects)
		// We require that one URL is a prefix of the other AND they share
		// the same scheme and netloc (host:port) to avoid false matches.
		respURL := evResp.Response.URL
		if rt.urlsMatch(respURL, rt.targetURL) {
			rt.status = evResp.Response.Status
			rt.captured = true
			rt.mu.Unlock()
			slog.Debug("captured response status", "url", apperrors.SanitizeURL(respURL), "target", apperrors.SanitizeURL(rt.targetURL), "status", rt.status)
			return
		}
		slog.Debug("document response URL does not match target", "respURL", apperrors.SanitizeURL(respURL), "targetURL", apperrors.SanitizeURL(rt.targetURL))
	}
	rt.mu.Unlock()
}

func (rt *responseTracker) getStatus() int64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.captured {
		return rt.status
	}
	return 0
}

// urlsMatch checks if two URLs match for the purpose of identifying the main navigation.
// It returns true if:
// 1. The URLs are exactly equal, OR
// 2. One is a prefix of the other AND they share the same scheme and host
//
// This allows for URL redirects (e.g., https://example.com -> https://example.com/welcome)
// while preventing false matches (e.g., https://example.com/a matching https://other.com/a).
//
// Examples:
//   - https://example.com matches https://example.com/path (same host, prefix)
//   - https://example.com:8080 matches https://example.com:8080/api (same host+port)
//   - https://example.com does NOT match https://other.com (different host)
//   - https://example.com does NOT match http://example.com (different scheme)
//   - https://example.com/api does NOT match https://example.com/app (no prefix relationship)
func (rt *responseTracker) urlsMatch(a, b string) bool {
	if a == b {
		return true
	}

	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}

	// Must match scheme and host (host includes port for non-standard ports)
	if ua.Scheme != ub.Scheme || ua.Host != ub.Host {
		return false
	}

	// One path must be a prefix of the other (allows for redirects)
	// Normalize paths to ensure /path and /path/ match consistently
	aPath := strings.TrimSuffix(ua.Path, "/")
	bPath := strings.TrimSuffix(ub.Path, "/")
	if aPath == "" {
		aPath = "/"
	}
	if bPath == "" {
		bPath = "/"
	}

	return strings.HasPrefix(aPath, bPath) || strings.HasPrefix(bPath, aPath)
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

// captureScreenshot captures a screenshot of the current page using chromedp.
// It returns the path to the saved screenshot file.
func (f *ChromedpFetcher) captureScreenshot(ctx context.Context, req Request, dataDir string) (string, error) {
	if req.Screenshot == nil || !req.Screenshot.Enabled {
		return "", nil
	}

	// Generate screenshot filename
	screenshotDir := filepath.Join(dataDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create screenshots directory: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	ext := "png"
	if req.Screenshot.Format == ScreenshotFormatJPEG {
		ext = "jpg"
	}
	filename := fmt.Sprintf("chromedp_%d_%d.%s", timestamp, req.Screenshot.Quality, ext)
	path := filepath.Join(screenshotDir, filename)

	// Set viewport if custom dimensions provided and no device emulation is active
	device := f.resolveDevice(req, RenderProfile{})
	if device == nil && req.Screenshot.Width > 0 && req.Screenshot.Height > 0 {
		width := int64(req.Screenshot.Width)
		height := int64(req.Screenshot.Height)
		if err := chromedp.Run(ctx, chromedp.EmulateViewport(width, height)); err != nil {
			return "", fmt.Errorf("failed to set viewport: %w", err)
		}
	}

	var buf []byte
	if req.Screenshot.FullPage {
		if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, req.Screenshot.Quality)); err != nil {
			return "", fmt.Errorf("failed to capture full page screenshot: %w", err)
		}
	} else {
		if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
			return "", fmt.Errorf("failed to capture screenshot: %w", err)
		}
	}

	// Write screenshot to file
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return "", fmt.Errorf("failed to write screenshot file: %w", err)
	}

	return path, nil
}

// resolveDevice determines which device emulation to use.
// Priority: req.Device > prof.Device > req.Screenshot.Device > prof.Screenshot.Device > none
func (f *ChromedpFetcher) resolveDevice(req Request, prof RenderProfile) *DeviceEmulation {
	if req.Device != nil {
		return req.Device
	}
	if prof.Device != nil {
		return prof.Device
	}
	if req.Screenshot != nil && req.Screenshot.Device != nil {
		return req.Screenshot.Device
	}
	if prof.Screenshot.Device != nil {
		return prof.Screenshot.Device
	}
	return nil
}

// applyDeviceEmulation applies device emulation settings to the chromedp context.
func (f *ChromedpFetcher) applyDeviceEmulation(ctx context.Context, device *DeviceEmulation) error {
	if device == nil {
		return nil
	}

	// Set viewport
	width := int64(device.ViewportWidth)
	height := int64(device.ViewportHeight)
	if err := chromedp.Run(ctx, chromedp.EmulateViewport(width, height)); err != nil {
		return fmt.Errorf("failed to set viewport: %w", err)
	}

	// Set device scale factor
	if device.DeviceScaleFactor > 0 {
		if err := chromedp.Run(ctx, emulation.SetDeviceMetricsOverride(
			int64(device.ViewportWidth),
			int64(device.ViewportHeight),
			device.DeviceScaleFactor,
			device.IsMobile,
		)); err != nil {
			return fmt.Errorf("failed to set device metrics: %w", err)
		}
	}

	// Set touch emulation
	if device.HasTouch {
		if err := chromedp.Run(ctx, emulation.SetTouchEmulationEnabled(true)); err != nil {
			return fmt.Errorf("failed to enable touch emulation: %w", err)
		}
	}

	// Set user agent (if not already set via allocator options)
	if device.UserAgent != "" {
		if err := chromedp.Run(ctx, emulation.SetUserAgentOverride(device.UserAgent)); err != nil {
			return fmt.Errorf("failed to set user agent: %w", err)
		}
	}

	return nil
}
