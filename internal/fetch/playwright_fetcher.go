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
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type PlaywrightFetcher struct {
	mu          sync.RWMutex
	pw          *playwright.Playwright
	browser     playwright.Browser
	initialized bool
	headless    bool
	proxyPool   *ProxyPool
}

// SetProxyPool sets the proxy pool for this fetcher.
func (f *PlaywrightFetcher) SetProxyPool(pool *ProxyPool) {
	f.proxyPool = pool
}

func (f *PlaywrightFetcher) Fetch(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("Playwright fetch start", "url", apperrors.SanitizeURL(req.URL))

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 800 * time.Millisecond
	}

	navTimeout := req.Timeout
	if prof.Timeouts.NavigationMs > 0 {
		navLimit := time.Duration(prof.Timeouts.NavigationMs) * time.Millisecond
		if navLimit < navTimeout {
			navTimeout = navLimit
		}
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying Playwright fetch", "url", apperrors.SanitizeURL(req.URL), "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", apperrors.SanitizeURL(req.URL))
			if err := req.Limiter.Wait(ctx, req.URL); err != nil {
				return Result{}, err
			}
		}

		result, err := f.fetchOnce(ctx, req, prof, navTimeout)
		if err == nil {
			slog.Debug("Playwright fetch success", "url", apperrors.SanitizeURL(req.URL))
			return result, nil
		}

		slog.Warn("Playwright fetch failed", "url", apperrors.SanitizeURL(req.URL), "error", err, "attempt", attempt)

		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		delay := backoff(baseDelay, attempt)
		slog.Debug("backing off before retry", "url", apperrors.SanitizeURL(req.URL), "delay", delay)
		time.Sleep(delay)
	}

	slog.Error("Playwright fetch max retries exceeded", "url", apperrors.SanitizeURL(req.URL))
	return Result{}, errors.New("max retries exceeded")
}

func (f *PlaywrightFetcher) ensureInitialized(ctx context.Context, headless bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// If already initialized with same headless setting, perform health check
	if f.initialized && f.headless == headless {
		if f.browser != nil && f.browser.IsConnected() {
			return nil
		}
		slog.Warn("Playwright browser disconnected or crashed, attempting recovery")
		if err := f.cleanup(); err != nil {
			slog.Warn("cleanup during browser recovery failed", "error", err)
		}
	}

	// Clean up existing if headless setting changed
	if f.initialized && f.headless != headless {
		slog.Debug("headless mode changed, cleaning up existing browser", "old", f.headless, "new", headless)
		if err := f.cleanup(); err != nil {
			slog.Warn("cleanup during headless switch failed, continuing", "error", err)
		}
	}

	// Setup initialization context with timeout if not already present
	initCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		initCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Helper to check context cancellation during init
	checkCtx := func() error {
		select {
		case <-initCtx.Done():
			return initCtx.Err()
		default:
			return nil
		}
	}

	if err := checkCtx(); err != nil {
		return err
	}

	// Initialize Playwright
	slog.Debug("initializing Playwright")
	pw, err := playwright.Run()
	if err != nil {
		return apperrors.Wrap(
			apperrors.KindInternal,
			"failed to initialize Playwright: ensure Playwright is installed with 'go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps' (see README.md for details)",
			err,
		)
	}

	if err := checkCtx(); err != nil {
		pw.Stop()
		return err
	}

	// Initialize Browser
	slog.Debug("launching Chromium browser", "headless", headless)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
	})
	if err != nil {
		pw.Stop()
		return apperrors.Wrap(
			apperrors.KindInternal,
			"failed to launch Playwright browser: ensure Playwright is installed with 'go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps' (see README.md for details)",
			err,
		)
	}

	f.pw = pw
	f.browser = browser
	f.initialized = true
	f.headless = headless
	return nil
}

func (f *PlaywrightFetcher) cleanup() error {
	var errs []error

	if f.browser != nil {
		if err := f.browser.Close(); err != nil {
			errs = append(errs, err)
		}
		f.browser = nil
	}

	if f.pw != nil {
		if err := f.pw.Stop(); err != nil {
			errs = append(errs, err)
		}
		f.pw = nil
	}

	f.initialized = false
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (f *PlaywrightFetcher) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cleanup()
}

func (f *PlaywrightFetcher) fetchOnce(ctx context.Context, req Request, prof RenderProfile, navTimeout time.Duration) (Result, error) {
	// Ensure initialized (lazily creates pw + browser if needed)
	if err := f.ensureInitialized(ctx, req.Headless); err != nil {
		return Result{}, err
	}

	f.mu.RLock()
	browser := f.browser
	f.mu.RUnlock()

	if browser == nil {
		return Result{}, errors.New("browser not initialized")
	}

	ctxOptions := playwright.BrowserNewContextOptions{}
	if req.UserAgent != "" {
		ctxOptions.UserAgent = playwright.String(req.UserAgent)
	}
	if req.Auth.Basic != "" {
		parts := strings.SplitN(req.Auth.Basic, ":", 2)
		if len(parts) == 2 {
			ctxOptions.HttpCredentials = &playwright.HttpCredentials{Username: parts[0], Password: parts[1]}
		}
	}
	if len(req.Auth.Headers) > 0 {
		ctxOptions.ExtraHttpHeaders = map[string]string{}
		for k, v := range req.Auth.Headers {
			ctxOptions.ExtraHttpHeaders[k] = v
		}
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
		proxyOpts := playwright.Proxy{
			Server: req.Auth.Proxy.URL,
		}
		if req.Auth.Proxy.Username != "" {
			proxyOpts.Username = &req.Auth.Proxy.Username
			proxyOpts.Password = &req.Auth.Proxy.Password
		}
		ctxOptions.Proxy = &proxyOpts
	}

	// Apply device emulation if specified
	device := f.resolveDevice(req, prof)
	if device != nil {
		slog.Debug("applying device emulation", "url", apperrors.SanitizeURL(req.URL), "device", device.Name)
		f.applyDeviceEmulation(&ctxOptions, device)
	}

	slog.Debug("creating browser context", "url", apperrors.SanitizeURL(req.URL))
	browserCtx, err := browser.NewContext(ctxOptions)
	if err != nil {
		slog.Error("failed to create browser context", "error", err)
		return Result{}, err
	}
	defer func() {
		_ = browserCtx.Close()
	}()

	if len(prof.Block.URLPatterns) > 0 || len(prof.Block.ResourceTypes) > 0 {
		slog.Debug("setting up resource blocking", "url", apperrors.SanitizeURL(req.URL), "patterns", prof.Block.URLPatterns, "types", prof.Block.ResourceTypes)
		for _, pattern := range prof.Block.URLPatterns {
			_ = browserCtx.Route(pattern, func(route playwright.Route) {
				_ = route.Abort("blockedbyclient")
			})
		}
		if len(prof.Block.ResourceTypes) > 0 {
			_ = browserCtx.Route("**/*", func(route playwright.Route) {
				req := route.Request()
				resType := req.ResourceType()
				for _, blockType := range prof.Block.ResourceTypes {
					if isBlockedType(resType, blockType) {
						_ = route.Abort("blockedbyclient")
						return
					}
				}
				_ = route.Continue()
			})
		}
	}

	if len(req.Auth.Cookies) > 0 {
		u, parseErr := url.Parse(req.URL)
		if parseErr == nil {
			cookies := make([]playwright.OptionalCookie, 0, len(req.Auth.Cookies))
			for _, cookie := range req.Auth.Cookies {
				parts := strings.SplitN(cookie, "=", 2)
				if len(parts) != 2 {
					continue
				}
				cookieURL := u.String()
				cookies = append(cookies, playwright.OptionalCookie{
					Name:  parts[0],
					Value: parts[1],
					URL:   &cookieURL,
				})
			}
			if len(cookies) > 0 {
				slog.Debug("adding cookies to context", "url", apperrors.SanitizeURL(req.URL), "count", len(cookies))
				_ = browserCtx.AddCookies(cookies)
			}
		}
	}

	page, err := browserCtx.NewPage()
	if err != nil {
		slog.Error("failed to create new page", "error", err)
		return Result{}, err
	}
	defer func() {
		_ = page.Close()
	}()

	// Set up network interceptor if configured
	var interceptor *playwrightInterceptor
	if req.NetworkIntercept != nil && req.NetworkIntercept.Enabled {
		slog.Debug("setting up network interception", "url", apperrors.SanitizeURL(req.URL))
		config := *req.NetworkIntercept
		if config.MaxBodySize == 0 {
			config.MaxBodySize = 1024 * 1024 // 1MB default
		}
		if config.MaxEntries == 0 {
			config.MaxEntries = 1000 // Default max entries
		}
		interceptor = newPlaywrightInterceptor(config)

		// Set up route handler for interception
		_ = page.Route("**/*", func(route playwright.Route) {
			interceptor.handleRoute(route)
		})

		// Set up response listener
		page.On("response", func(resp playwright.Response) {
			interceptor.onResponse(resp)
		})

		// Set up request failed listener
		page.On("requestfailed", func(req playwright.Request) {
			interceptor.onRequestFailed(req)
		})
	}

	timeoutFloat := float64(req.Timeout.Milliseconds())
	navTimeoutFloat := float64(navTimeout.Milliseconds())
	if timeoutFloat <= 0 {
		timeoutFloat = 30000
		navTimeoutFloat = 30000
	}
	page.SetDefaultTimeout(timeoutFloat)
	page.SetDefaultNavigationTimeout(navTimeoutFloat)

	waitUntil := playwright.WaitUntilStateCommit

	if req.Auth.LoginURL != "" {
		slog.Info("performing headless login", "url", apperrors.SanitizeURL(req.URL), "loginURL", apperrors.SanitizeURL(req.Auth.LoginURL))

		// Auto-detect form fields if enabled
		userSelector := req.Auth.LoginUserSelector
		passSelector := req.Auth.LoginPassSelector
		submitSelector := req.Auth.LoginSubmitSelector

		if req.Auth.LoginAutoDetect {
			detected, err := f.detectLoginForm(page, req.Auth.LoginURL, timeoutFloat)
			if err != nil {
				return Result{}, fmt.Errorf("failed to detect login form: %w", err)
			}
			if detected == nil {
				return Result{}, errors.New("could not detect login form on page")
			}

			userSelector = detected.UserField.Selector
			passSelector = detected.PassField.Selector
			submitSelector = detected.SubmitField.Selector

			slog.Info("login form auto-detected",
				"userSelector", userSelector,
				"passSelector", passSelector,
				"submitSelector", submitSelector,
				"confidence", detected.Score)
		}

		// Validate selectors are present
		if userSelector == "" || passSelector == "" || submitSelector == "" {
			return Result{}, errors.New("login selectors are required (provide manually or use auto-detect)")
		}

		if _, err = page.Goto(req.Auth.LoginURL, playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
			slog.Error("login page navigation failed", "url", apperrors.SanitizeURL(req.URL), "loginURL", apperrors.SanitizeURL(req.Auth.LoginURL), "error", err)
			return Result{}, err
		}
		if err = page.Fill(userSelector, req.Auth.LoginUser); err != nil {
			return Result{}, err
		}
		if err = page.Fill(passSelector, req.Auth.LoginPass); err != nil {
			return Result{}, err
		}
		if err = page.Click(submitSelector); err != nil {
			return Result{}, err
		}
		if err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded, Timeout: &timeoutFloat}); err != nil {
			slog.Warn("wait for load state after login timed out", "url", apperrors.SanitizeURL(req.URL))
		}
		slog.Info("login complete", "url", apperrors.SanitizeURL(req.URL))

		// Extract and save cookies if session ID is provided
		if req.SessionID != "" && req.DataDir != "" {
			if err := f.extractAndSaveSession(browserCtx, req); err != nil {
				slog.Warn("failed to save session cookies", "sessionID", req.SessionID, "error", err)
			}
		}
	}

	if len(req.PreNavJS) > 0 {
		slog.Debug("running pre-navigation JS", "url", apperrors.SanitizeURL(req.URL), "count", len(req.PreNavJS))
		if _, err = page.Goto("about:blank", playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
			return Result{}, err
		}
		for _, script := range req.PreNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if _, err := page.Evaluate(script); err != nil {
				slog.Error("pre-navigation JS failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
				return Result{}, err
			}
		}
	}

	slog.Debug("navigating to target", "url", apperrors.SanitizeURL(req.URL))
	resp, err := page.Goto(req.URL, playwright.PageGotoOptions{Timeout: &navTimeoutFloat, WaitUntil: waitUntil})
	if err != nil {
		slog.Error("navigation failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}

	statusCode := 200
	if resp != nil {
		statusCode = resp.Status()
	}
	slog.Debug("navigation complete", "url", apperrors.SanitizeURL(req.URL), "status", statusCode)

	slog.Debug("waiting for page to be ready", "url", apperrors.SanitizeURL(req.URL), "mode", prof.Wait.Mode)
	if err := f.performWait(page, prof.Wait, timeoutFloat); err != nil {
		slog.Warn("wait strategy failed or timed out", "url", apperrors.SanitizeURL(req.URL), "mode", prof.Wait.Mode, "error", err)
		// Fall through to capture whatever we have
	}

	if prof.Wait.ExtraSleepMs > 0 {
		slog.Debug("extra sleep", "url", apperrors.SanitizeURL(req.URL), "ms", prof.Wait.ExtraSleepMs)
		time.Sleep(time.Duration(prof.Wait.ExtraSleepMs) * time.Millisecond)
	}

	for _, selector := range req.WaitSelectors {
		if strings.TrimSpace(selector) == "" {
			continue
		}
		slog.Debug("waiting for selector", "url", apperrors.SanitizeURL(req.URL), "selector", selector)
		if _, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{State: playwright.WaitForSelectorStateVisible, Timeout: &timeoutFloat}); err != nil {
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
			if _, err := page.Evaluate(script); err != nil {
				slog.Error("post-navigation JS failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
				return Result{}, err
			}
		}
	}

	slog.Debug("capturing page content", "url", apperrors.SanitizeURL(req.URL))
	content, err := page.Content()
	if err != nil {
		slog.Error("failed to capture content", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}

	// Capture screenshot if requested
	var screenshotPath string
	if req.Screenshot != nil && req.Screenshot.Enabled {
		path, err := f.captureScreenshot(page, req, prof, req.DataDir)
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
		interceptedData = interceptor.getEntries()
		slog.Debug("network interception complete", "url", apperrors.SanitizeURL(req.URL), "entriesCaptured", len(interceptedData))
	}

	// Record proxy pool metrics
	if selectedProxy != nil && f.proxyPool != nil {
		if statusCode >= 200 && statusCode < 400 {
			f.proxyPool.RecordSuccess(selectedProxy.ID, 0)
		} else if statusCode >= 500 {
			f.proxyPool.RecordFailure(selectedProxy.ID, fmt.Errorf("HTTP %d", statusCode))
		}
	}

	return Result{
		URL:             req.URL,
		Status:          statusCode,
		HTML:            content,
		FetchedAt:       time.Now(),
		Engine:          RenderEnginePlaywright,
		ETag:            "", // Headless browsers don't easily expose response headers without complex interception.
		LastModified:    "",
		ScreenshotPath:  screenshotPath,
		InterceptedData: interceptedData,
	}, nil
}

func (f *PlaywrightFetcher) performWait(page playwright.Page, policy RenderWaitPolicy, timeout float64) error {
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return err
	}

	switch policy.Mode {
	case RenderWaitModeNetworkIdle:
		return page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle, Timeout: &timeout})
	case RenderWaitModeSelector:
		if policy.Selector != "" {
			_, err := page.WaitForSelector(policy.Selector, playwright.PageWaitForSelectorOptions{State: playwright.WaitForSelectorStateVisible, Timeout: &timeout})
			return err
		}
	case RenderWaitModeStability:
		return f.waitForStability(page, policy)
	}
	return nil
}

func (f *PlaywrightFetcher) waitForStability(page playwright.Page, policy RenderWaitPolicy) error {
	pollMs := policy.StabilityPollMs
	if pollMs <= 0 {
		pollMs = 200
	}
	minLen := policy.MinTextLength
	target := policy.StabilityIterations
	if target <= 0 {
		target = 3
	}

	lastLen := 0
	stable := 0

	for i := 0; i < 20; i++ {
		s, err := page.InnerText("body")
		if err != nil {
			return err
		}
		curLen := len(s)
		if curLen >= minLen && curLen == lastLen {
			stable++
		} else {
			stable = 0
		}
		if stable >= target {
			return nil
		}
		lastLen = curLen
		time.Sleep(time.Duration(pollMs) * time.Millisecond)
	}
	return nil
}

func isBlockedType(resType string, blockType BlockedResourceType) bool {
	switch blockType {
	case BlockedResourceImage:
		return resType == "image"
	case BlockedResourceMedia:
		return resType == "media"
	case BlockedResourceFont:
		return resType == "font"
	case BlockedResourceStylesheet:
		return resType == "stylesheet"
	case BlockedResourceOther:
		// Block all non-essential resource types (scripts, APIs, websockets, etc.)
		// Includes the literal Playwright 'other' type for miscellaneous requests
		return resType == "script" ||
			resType == "xhr" ||
			resType == "fetch" ||
			resType == "websocket" ||
			resType == "eventsource" ||
			resType == "manifest" ||
			resType == "texttrack" ||
			resType == "other"
	}
	return false
}

// detectLoginForm uses form detection to analyze the login page and find form fields.
func (f *PlaywrightFetcher) detectLoginForm(page playwright.Page, loginURL string, timeoutFloat float64) (*DetectedForm, error) {
	waitUntil := playwright.WaitUntilStateCommit

	// Navigate to login page
	if _, err := page.Goto(loginURL, playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
		return nil, err
	}

	// Wait for body to be ready
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return nil, err
	}

	// Extract page HTML
	html, err := page.Content()
	if err != nil {
		return nil, err
	}

	// Use form detector
	detector := NewFormDetector()
	form, err := detector.DetectLoginForm(html)
	if err != nil {
		return nil, err
	}

	return form, nil
}
