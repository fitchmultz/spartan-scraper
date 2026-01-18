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

func (f *PlaywrightFetcher) Fetch(req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 800 * time.Millisecond
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if req.Limiter != nil {
			_ = req.Limiter.Wait(context.Background(), req.URL)
		}

		result, err := f.fetchOnce(req)
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

func (f *PlaywrightFetcher) fetchOnce(req Request) (Result, error) {
	pw, err := playwright.Run()
	if err != nil {
		return Result{}, err
	}
	defer func() {
		_ = pw.Stop()
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
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

	timeoutMs := float64(req.Timeout.Milliseconds())
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	page.SetDefaultTimeout(timeoutMs)
	page.SetDefaultNavigationTimeout(timeoutMs)

	if req.Auth.LoginURL != "" {
		if req.Auth.LoginUserSelector == "" || req.Auth.LoginPassSelector == "" || req.Auth.LoginSubmitSelector == "" {
			return Result{}, errors.New("login selectors are required for headless login")
		}
		if _, err = page.Goto(req.Auth.LoginURL, playwright.PageGotoOptions{Timeout: &timeoutMs, WaitUntil: playwright.WaitUntilStateNetworkidle}); err != nil {
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
		if err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle, Timeout: &timeoutMs}); err != nil {
			return Result{}, err
		}
	}

	if _, err = page.Goto(req.URL, playwright.PageGotoOptions{Timeout: &timeoutMs, WaitUntil: playwright.WaitUntilStateNetworkidle}); err != nil {
		return Result{}, err
	}
	content, err := page.Content()
	if err != nil {
		return Result{}, err
	}

	return Result{
		URL:       req.URL,
		Status:    200,
		HTML:      content,
		FetchedAt: time.Now(),
	}, nil
}
