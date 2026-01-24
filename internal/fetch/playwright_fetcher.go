package fetch

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

type PlaywrightFetcher struct {
	mu          sync.RWMutex
	pw          *playwright.Playwright
	browser     playwright.Browser
	initialized bool
	headless    bool
}

func (f *PlaywrightFetcher) Fetch(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("Playwright fetch start", "url", req.URL)

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
			slog.Debug("retrying Playwright fetch", "url", req.URL, "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", req.URL)
			_ = req.Limiter.Wait(ctx, req.URL)
		}

		result, err := f.fetchOnce(ctx, req, prof, navTimeout)
		if err == nil {
			slog.Debug("Playwright fetch success", "url", req.URL)
			return result, nil
		}

		slog.Warn("Playwright fetch failed", "url", req.URL, "error", err, "attempt", attempt)

		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		delay := backoff(baseDelay, attempt)
		slog.Debug("backing off before retry", "url", req.URL, "delay", delay)
		time.Sleep(delay)
	}

	slog.Error("Playwright fetch max retries exceeded", "url", req.URL)
	return Result{}, errors.New("max retries exceeded")
}

func (f *PlaywrightFetcher) ensureInitialized(ctx context.Context, headless bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// If already initialized with same headless setting, we're good
	if f.initialized && f.headless == headless {
		return nil
	}

	// Clean up existing if headless setting changed
	if f.initialized && f.headless != headless {
		slog.Debug("headless mode changed, cleaning up existing browser", "old", f.headless, "new", headless)
		if err := f.cleanup(); err != nil {
			slog.Warn("cleanup during headless switch failed, continuing", "error", err)
		}
	}

	// Initialize Playwright
	slog.Debug("initializing Playwright")
	pw, err := playwright.Run()
	if err != nil {
		slog.Error("failed to run Playwright", "error", err)
		return err
	}

	// Initialize Browser
	slog.Debug("launching Chromium browser", "headless", headless)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
	})
	if err != nil {
		slog.Error("failed to launch browser", "error", err)
		pw.Stop()
		return err
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

	slog.Debug("creating browser context", "url", req.URL)
	browserCtx, err := browser.NewContext(ctxOptions)
	if err != nil {
		slog.Error("failed to create browser context", "error", err)
		return Result{}, err
	}
	defer func() {
		_ = browserCtx.Close()
	}()

	if len(prof.Block.URLPatterns) > 0 || len(prof.Block.ResourceTypes) > 0 {
		slog.Debug("setting up resource blocking", "url", req.URL, "patterns", prof.Block.URLPatterns, "types", prof.Block.ResourceTypes)
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
				slog.Debug("adding cookies to context", "url", req.URL, "count", len(cookies))
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
		slog.Info("performing headless login", "url", req.URL, "loginURL", req.Auth.LoginURL)
		if _, err = page.Goto(req.Auth.LoginURL, playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
			slog.Error("login page navigation failed", "url", req.URL, "loginURL", req.Auth.LoginURL, "error", err)
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
			slog.Warn("wait for load state after login timed out", "url", req.URL)
		}
		slog.Info("login complete", "url", req.URL)
	}

	if len(req.PreNavJS) > 0 {
		slog.Debug("running pre-navigation JS", "url", req.URL, "count", len(req.PreNavJS))
		if _, err = page.Goto("about:blank", playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
			return Result{}, err
		}
		for _, script := range req.PreNavJS {
			if strings.TrimSpace(script) == "" {
				continue
			}
			if _, err := page.Evaluate(script); err != nil {
				slog.Error("pre-navigation JS failed", "url", req.URL, "error", err)
				return Result{}, err
			}
		}
	}

	slog.Debug("navigating to target", "url", req.URL)
	resp, err := page.Goto(req.URL, playwright.PageGotoOptions{Timeout: &navTimeoutFloat, WaitUntil: waitUntil})
	if err != nil {
		slog.Error("navigation failed", "url", req.URL, "error", err)
		return Result{}, err
	}

	statusCode := 200
	if resp != nil {
		statusCode = resp.Status()
	}
	slog.Debug("navigation complete", "url", req.URL, "status", statusCode)

	slog.Debug("waiting for page to be ready", "url", req.URL, "mode", prof.Wait.Mode)
	if err := f.performWait(page, prof.Wait, timeoutFloat); err != nil {
		slog.Warn("wait strategy failed or timed out", "url", req.URL, "mode", prof.Wait.Mode, "error", err)
		// Fall through to capture whatever we have
	}

	if prof.Wait.ExtraSleepMs > 0 {
		slog.Debug("extra sleep", "url", req.URL, "ms", prof.Wait.ExtraSleepMs)
		time.Sleep(time.Duration(prof.Wait.ExtraSleepMs) * time.Millisecond)
	}

	for _, selector := range req.WaitSelectors {
		if strings.TrimSpace(selector) == "" {
			continue
		}
		slog.Debug("waiting for selector", "url", req.URL, "selector", selector)
		if _, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{State: playwright.WaitForSelectorStateVisible, Timeout: &timeoutFloat}); err != nil {
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
			if _, err := page.Evaluate(script); err != nil {
				slog.Error("post-navigation JS failed", "url", req.URL, "error", err)
				return Result{}, err
			}
		}
	}

	slog.Debug("capturing page content", "url", req.URL)
	content, err := page.Content()
	if err != nil {
		slog.Error("failed to capture content", "url", req.URL, "error", err)
		return Result{}, err
	}

	return Result{
		URL:          req.URL,
		Status:       statusCode,
		HTML:         content,
		FetchedAt:    time.Now(),
		Engine:       RenderEnginePlaywright,
		ETag:         "", // Headless browsers don't easily expose response headers without complex interception.
		LastModified: "",
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
