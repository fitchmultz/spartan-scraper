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
}

// playwrightInterceptor captures network requests and responses for API scraping.
type playwrightInterceptor struct {
	config  NetworkInterceptConfig
	mu      sync.Mutex
	entries []InterceptedEntry
	pending map[string]*InterceptedRequest // URL -> request (for matching)
}

func newPlaywrightInterceptor(config NetworkInterceptConfig) *playwrightInterceptor {
	return &playwrightInterceptor{
		config:  config,
		entries: make([]InterceptedEntry, 0, config.MaxEntries),
		pending: make(map[string]*InterceptedRequest),
	}
}

func (pi *playwrightInterceptor) shouldIntercept(url string, resourceType string) bool {
	// Check resource type
	if len(pi.config.ResourceTypes) > 0 {
		matched := false
		for _, rt := range pi.config.ResourceTypes {
			if string(rt) == resourceType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check URL patterns
	if len(pi.config.URLPatterns) == 0 {
		return true
	}
	for _, pattern := range pi.config.URLPatterns {
		if matchGlob(pattern, url) {
			return true
		}
	}
	return false
}

func (pi *playwrightInterceptor) handleRoute(route playwright.Route) {
	req := route.Request()
	url := req.URL()
	resourceType := req.ResourceType()

	if !pi.shouldIntercept(url, resourceType) {
		_ = route.Continue()
		return
	}

	pi.mu.Lock()
	// Check max entries
	if len(pi.entries) >= pi.config.MaxEntries {
		pi.mu.Unlock()
		slog.Warn("playwright interceptor max entries reached, dropping request", "url", url)
		_ = route.Continue()
		return
	}

	interceptedReq := &InterceptedRequest{
		RequestID:    req.URL(), // Use URL as ID since Playwright doesn't expose request ID
		URL:          url,
		Method:       req.Method(),
		Headers:      make(map[string]string),
		Timestamp:    time.Now(),
		ResourceType: InterceptedResourceType(resourceType),
	}

	// Copy headers
	for k, v := range req.Headers() {
		interceptedReq.Headers[k] = v
	}

	// Capture request body if enabled
	if pi.config.CaptureRequestBody {
		if postData, err := req.PostData(); err == nil && postData != "" {
			body := postData
			if int64(len(body)) > pi.config.MaxBodySize {
				body = body[:pi.config.MaxBodySize]
			}
			interceptedReq.Body = body
			interceptedReq.BodySize = int64(len(postData))
		}
	}

	pi.pending[url] = interceptedReq
	pi.mu.Unlock()

	// Continue the request and capture response
	_ = route.Continue()

	// Note: Response capture happens via page event listeners
}

func (pi *playwrightInterceptor) onResponse(resp playwright.Response) {
	req := resp.Request()
	url := req.URL()

	pi.mu.Lock()
	interceptedReq, exists := pi.pending[url]
	if !exists {
		pi.mu.Unlock()
		return
	}
	delete(pi.pending, url)

	// Check max entries again
	if len(pi.entries) >= pi.config.MaxEntries {
		pi.mu.Unlock()
		return
	}

	interceptedResp := &InterceptedResponse{
		RequestID:  interceptedReq.RequestID,
		Status:     resp.Status(),
		StatusText: resp.StatusText(),
		Headers:    make(map[string]string),
		Timestamp:  time.Now(),
	}

	// Copy headers
	for k, v := range resp.Headers() {
		interceptedResp.Headers[k] = v
	}

	// Capture response body if enabled
	if pi.config.CaptureResponseBody {
		if body, err := resp.Body(); err == nil && len(body) > 0 {
			bodyStr := string(body)
			if int64(len(bodyStr)) > pi.config.MaxBodySize {
				bodyStr = bodyStr[:pi.config.MaxBodySize]
			}
			interceptedResp.Body = bodyStr
			interceptedResp.BodySize = int64(len(bodyStr))
		}
	}

	entry := InterceptedEntry{
		Request:  *interceptedReq,
		Response: interceptedResp,
		Duration: interceptedResp.Timestamp.Sub(interceptedReq.Timestamp),
	}
	pi.entries = append(pi.entries, entry)
	pi.mu.Unlock()
}

func (pi *playwrightInterceptor) onRequestFailed(req playwright.Request) {
	url := req.URL()

	pi.mu.Lock()
	interceptedReq, exists := pi.pending[url]
	if !exists {
		pi.mu.Unlock()
		return
	}
	delete(pi.pending, url)

	entry := InterceptedEntry{
		Request:  *interceptedReq,
		Response: nil,
		Duration: time.Since(interceptedReq.Timestamp),
	}
	pi.entries = append(pi.entries, entry)
	pi.mu.Unlock()
}

func (pi *playwrightInterceptor) getEntries() []InterceptedEntry {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	result := make([]InterceptedEntry, len(pi.entries))
	copy(result, pi.entries)
	return result
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

	// Add proxy configuration if provided
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
		if _, err = page.Goto(req.Auth.LoginURL, playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
			slog.Error("login page navigation failed", "url", apperrors.SanitizeURL(req.URL), "loginURL", apperrors.SanitizeURL(req.Auth.LoginURL), "error", err)
			return Result{}, err
		}
		if err = page.Fill(req.Auth.LoginUserSelector, req.Auth.LoginUser); err != nil {
			return Result{}, err
		}
		if err = page.Fill(req.Auth.LoginPassSelector, req.Auth.LoginPass); err != nil {
			return Result{}, err
		}
		if err = page.Click(req.Auth.LoginSubmitSelector); err != nil {
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

// captureScreenshot captures a screenshot of the current page using Playwright.
// It returns the path to the saved screenshot file.
func (f *PlaywrightFetcher) captureScreenshot(page playwright.Page, req Request, prof RenderProfile, dataDir string) (string, error) {
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
	format := playwright.ScreenshotTypePng
	if req.Screenshot.Format == ScreenshotFormatJPEG {
		ext = "jpg"
		format = playwright.ScreenshotTypeJpeg
	}
	filename := fmt.Sprintf("playwright_%d_%d.%s", timestamp, req.Screenshot.Quality, ext)
	path := filepath.Join(screenshotDir, filename)

	// Set viewport if custom dimensions provided and no device emulation is active
	device := f.resolveDevice(req, prof)
	if device == nil && req.Screenshot.Width > 0 && req.Screenshot.Height > 0 {
		if err := page.SetViewportSize(req.Screenshot.Width, req.Screenshot.Height); err != nil {
			return "", fmt.Errorf("failed to set viewport: %w", err)
		}
	}

	// Build screenshot options
	opts := playwright.PageScreenshotOptions{
		Path:     &path,
		FullPage: &req.Screenshot.FullPage,
		Type:     format,
	}

	// Add quality for JPEG
	if req.Screenshot.Format == ScreenshotFormatJPEG && req.Screenshot.Quality > 0 {
		quality := req.Screenshot.Quality
		opts.Quality = &quality
	}

	// Capture screenshot
	if _, err := page.Screenshot(opts); err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	return path, nil
}

// resolveDevice determines which device emulation to use.
// Priority: req.Device > prof.Device > req.Screenshot.Device > prof.Screenshot.Device > none
func (f *PlaywrightFetcher) resolveDevice(req Request, prof RenderProfile) *DeviceEmulation {
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

// applyDeviceEmulation applies device emulation settings to the Playwright context options.
func (f *PlaywrightFetcher) applyDeviceEmulation(opts *playwright.BrowserNewContextOptions, device *DeviceEmulation) {
	if device == nil {
		return
	}

	// Set viewport
	viewport := playwright.Size{
		Width:  device.ViewportWidth,
		Height: device.ViewportHeight,
	}
	opts.Viewport = &viewport

	// Set device scale factor
	if device.DeviceScaleFactor > 0 {
		opts.DeviceScaleFactor = &device.DeviceScaleFactor
	}

	// Set mobile flag
	opts.IsMobile = &device.IsMobile

	// Set touch support
	opts.HasTouch = &device.HasTouch

	// Set user agent
	if device.UserAgent != "" {
		opts.UserAgent = &device.UserAgent
	}
}

// extractAndSaveSession extracts cookies from the browser context and saves them as a session.
func (f *PlaywrightFetcher) extractAndSaveSession(browserCtx playwright.BrowserContext, req Request) error {
	cookies, err := browserCtx.Cookies()
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Convert playwright.Cookie to sessionCookie
	sessionCookies := make([]sessionCookie, 0, len(cookies))
	for _, c := range cookies {
		sessionCookies = append(sessionCookies, sessionCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		})
	}

	// Save session
	sess := session{
		ID:      req.SessionID,
		Name:    req.SessionID,
		Domain:  extractDomain(req.URL),
		Cookies: sessionCookies,
	}

	if err := saveSession(req.DataDir, sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	slog.Info("saved session cookies", "sessionID", req.SessionID, "domain", sess.Domain, "count", len(sessionCookies))
	return nil
}
