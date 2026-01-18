package fetch

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type ChromedpFetcher struct{}

func (f *ChromedpFetcher) Fetch(req Request, prof RenderProfile) (Result, error) {
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

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
		if req.Limiter != nil {
			_ = req.Limiter.Wait(context.Background(), req.URL)
		}

		res, err := f.doFetch(req, prof, renderTimeout)
		if err == nil {
			return res, nil
		}

		if attempt >= retries || !shouldRetry(err, 0) {
			return Result{}, err
		}
		time.Sleep(backoff(baseDelay, attempt))
	}

	return Result{}, errors.New("max retries exceeded")
}

func (f *ChromedpFetcher) doFetch(req Request, prof RenderProfile, timeout time.Duration) (Result, error) {
	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if req.UserAgent != "" {
		allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocatorOpts...)
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
		actions = append(actions, network.SetBlockedURLs(blockedPatterns))
	}

	// Run initial setup
	if err := chromedp.Run(ctx, actions...); err != nil {
		return Result{}, err
	}

	// Login flow if configured
	currentURL := ""
	if req.Auth.LoginURL != "" {
		err := f.performLogin(ctx, req.Auth)
		if err != nil {
			return Result{}, err
		}
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return Result{}, err
		}
	}

	// Navigate to target
	if currentURL == "" || currentURL == req.Auth.LoginURL {
		if err := chromedp.Run(ctx, chromedp.Navigate(req.URL)); err != nil {
			if !isAbortErr(err) {
				return Result{}, err
			}
		}
	}

	// Wait strategies
	waitErr := f.performWait(ctx, prof.Wait)
	if waitErr != nil && !strings.Contains(waitErr.Error(), "timeout") {
		// Log error but might try to capture HTML anyway?
		// For now, fail on wait error unless it's just a timeout and we want partial results.
		// Strict strictness: fail.
		return Result{}, waitErr
	}

	// Extra sleep if requested
	if prof.Wait.ExtraSleepMs > 0 {
		_ = chromedp.Run(ctx, chromedp.Sleep(time.Duration(prof.Wait.ExtraSleepMs)*time.Millisecond))
	}

	var html string
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
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
		// Not natively supported as a single call in basic chromedp without listeners.
		// Fallback to a fixed sleep or check simple signals.
		// Implementing robust network idle in chromedp requires event listeners which is complex here.
		// We will simulate with a fixed sleep for now + DOM ready.
		// TODO: Implement event listener for network idle.
		return chromedp.Run(ctx, chromedp.Sleep(2*time.Second))
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
