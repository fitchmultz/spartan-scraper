package fetch

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type PlaywrightFetcher struct{}

func (f *PlaywrightFetcher) Fetch(req Request, prof RenderProfile) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

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
		if req.Limiter != nil {
			_ = req.Limiter.Wait(context.Background(), req.URL)
		}

		result, err := f.fetchOnce(req, prof, navTimeout)
		if err == nil {
			return result, nil
		}
		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		time.Sleep(backoff(baseDelay, attempt))
	}

	return Result{}, errors.New("max retries exceeded")
}

func (f *PlaywrightFetcher) fetchOnce(req Request, prof RenderProfile, navTimeout time.Duration) (Result, error) {
	pw, err := playwright.Run()
	if err != nil {
		return Result{}, err
	}
	defer func() {
		_ = pw.Stop()
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(req.Headless),
	})
	if err != nil {
		return Result{}, err
	}
	defer func() {
		_ = browser.Close()
	}()

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

	ctx, err := browser.NewContext(ctxOptions)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		_ = ctx.Close()
	}()

	if len(prof.Block.URLPatterns) > 0 || len(prof.Block.ResourceTypes) > 0 {
		for _, pattern := range prof.Block.URLPatterns {
			_ = ctx.Route(pattern, func(route playwright.Route) {
				_ = route.Abort("blockedbyclient")
			})
		}
		if len(prof.Block.ResourceTypes) > 0 {
			_ = ctx.Route("**/*", func(route playwright.Route) {
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
				_ = ctx.AddCookies(cookies)
			}
		}
	}

	page, err := ctx.NewPage()
	if err != nil {
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
		if req.Auth.LoginUserSelector == "" || req.Auth.LoginPassSelector == "" || req.Auth.LoginSubmitSelector == "" {
			return Result{}, errors.New("login selectors are required for headless login")
		}
		if _, err = page.Goto(req.Auth.LoginURL, playwright.PageGotoOptions{Timeout: &timeoutFloat, WaitUntil: waitUntil}); err != nil {
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
			return Result{}, err
		}
	}

	resp, err := page.Goto(req.URL, playwright.PageGotoOptions{Timeout: &navTimeoutFloat, WaitUntil: waitUntil})
	if err != nil {
		return Result{}, err
	}

	statusCode := 200
	if resp != nil {
		statusCode = resp.Status()
	}

	if err := f.performWait(page, prof.Wait, timeoutFloat); err != nil {
		return Result{}, err
	}

	if prof.Wait.ExtraSleepMs > 0 {
		time.Sleep(time.Duration(prof.Wait.ExtraSleepMs) * time.Millisecond)
	}

	content, err := page.Content()
	if err != nil {
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
	}
	return false
}
