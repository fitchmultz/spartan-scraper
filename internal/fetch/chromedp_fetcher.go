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
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type ChromedpFetcher struct {
	networkTracker *networkTracker
	proxyPool      *ProxyPool
}

// SetProxyPool sets the proxy pool for this fetcher.
func (f *ChromedpFetcher) SetProxyPool(pool *ProxyPool) {
	f.proxyPool = pool
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
		if err := SleepWithContext(ctx, delay); err != nil {
			return Result{}, err
		}
	}

	slog.Error("Chromedp fetch max retries exceeded", "url", apperrors.SanitizeURL(req.URL))
	return Result{}, errors.New("max retries exceeded")
}

func (f *ChromedpFetcher) doFetch(parentCtx context.Context, req Request, prof RenderProfile, timeout time.Duration) (Result, error) {
	slog.Debug("starting Chromedp allocator", "url", apperrors.SanitizeURL(req.URL), "timeout", timeout)
	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocatorOpts = appendChromedpCIFlags(allocatorOpts)
	if req.UserAgent != "" {
		allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
	}

	// Track selected proxy for metrics
	var selectedProxy *ProxyEntry

	// If proxy pool is configured and no explicit proxy, select from pool
	if f.proxyPool != nil && (req.Auth.Proxy == nil || req.Auth.Proxy.URL == "") {
		hints := ProxySelectionHints{}
		if req.Auth.ProxyHints != nil {
			hints = *req.Auth.ProxyHints
		}

		proxy, err := f.proxyPool.Select(hints)
		if err != nil {
			slog.Warn("failed to select proxy from pool", "url", apperrors.SanitizeURL(req.URL), "error", err)
		} else {
			selectedProxy = &proxy
			proxyConfig := proxy.ToProxyConfig()
			req.Auth.Proxy = &proxyConfig
			slog.Debug("selected proxy from pool", "url", apperrors.SanitizeURL(req.URL), "proxy_id", proxy.ID)
		}
	}

	// Add proxy configuration if provided (either explicit or from pool)
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

		// Extract and save cookies if session ID is provided
		if req.SessionID != "" && req.DataDir != "" {
			if err := f.extractAndSaveSession(ctx, req); err != nil {
				slog.Warn("failed to save session cookies", "sessionID", req.SessionID, "error", err)
			}
		}
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

	// Set up network interceptor if configured
	var interceptor *networkInterceptor
	if req.NetworkIntercept != nil && req.NetworkIntercept.Enabled {
		slog.Debug("setting up network interception", "url", apperrors.SanitizeURL(req.URL))
		config := *req.NetworkIntercept
		if config.MaxBodySize == 0 {
			config.MaxBodySize = 1024 * 1024 // 1MB default
		}
		if config.MaxEntries == 0 {
			config.MaxEntries = 1000 // Default max entries
		}
		interceptor = newNetworkInterceptor(config)
		chromedp.ListenTarget(ctx, func(ev any) {
			interceptor.onEvent(ev)
		})
	}

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

	// Collect intercepted data if enabled
	var interceptedData []InterceptedEntry
	if interceptor != nil {
		interceptor.stop()
		interceptedData = interceptor.getEntries()
		slog.Debug("network interception complete", "url", apperrors.SanitizeURL(req.URL), "entriesCaptured", len(interceptedData))
	}

	// Record proxy pool metrics
	if selectedProxy != nil && f.proxyPool != nil {
		if status >= 200 && status < 400 {
			f.proxyPool.RecordSuccess(selectedProxy.ID, 0)
		} else if status >= 500 {
			f.proxyPool.RecordFailure(selectedProxy.ID, fmt.Errorf("HTTP %d", status))
		}
	}

	return Result{
		URL:             req.URL,
		Status:          status,
		HTML:            html,
		FetchedAt:       time.Now(),
		Engine:          RenderEngineChromedp,
		ETag:            "", // Headless browsers don't easily expose response headers without complex interception
		LastModified:    "",
		ScreenshotPath:  screenshotPath,
		InterceptedData: interceptedData,
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
		if err := SleepWithContext(ctx, pollInterval); err != nil {
			return err
		}
	}
	return nil
}

func appendChromedpCIFlags(opts []chromedp.ExecAllocatorOption) []chromedp.ExecAllocatorOption {
	if !shouldUseChromedpCISandboxBypass() {
		return opts
	}

	return append(
		opts,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
	)
}

func shouldUseChromedpCISandboxBypass() bool {
	return os.Getenv("CI") != ""
}

func isAbortErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "net::ERR_ABORTED")
}
