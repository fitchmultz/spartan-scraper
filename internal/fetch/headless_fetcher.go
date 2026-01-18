package fetch

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type HeadlessFetcher struct{}

func (f *HeadlessFetcher) Fetch(req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if req.Limiter != nil {
			_ = req.Limiter.Wait(context.Background(), req.URL)
		}

		allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
		if req.UserAgent != "" {
			allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
		}
		allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts...)
		ctx, cancelCtx := chromedp.NewContext(allocCtx)
		ctx, cancelTimeout := context.WithTimeout(ctx, req.Timeout)

		actions := []chromedp.Action{network.Enable()}
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

		err := chromedp.Run(ctx, actions...)
		if err == nil && req.Auth.LoginURL != "" {
			if req.Auth.LoginUserSelector == "" || req.Auth.LoginPassSelector == "" || req.Auth.LoginSubmitSelector == "" {
				cancelTimeout()
				cancelCtx()
				cancel()
				return Result{}, errors.New("login selectors are required for headless login")
			}
			err = chromedp.Run(ctx,
				chromedp.Navigate(req.Auth.LoginURL),
				chromedp.WaitVisible(req.Auth.LoginUserSelector),
				chromedp.SendKeys(req.Auth.LoginUserSelector, req.Auth.LoginUser),
				chromedp.SendKeys(req.Auth.LoginPassSelector, req.Auth.LoginPass),
				chromedp.Click(req.Auth.LoginSubmitSelector),
			)
		}

		var html string
		if err == nil {
			err = chromedp.Run(ctx,
				chromedp.Navigate(req.URL),
				chromedp.OuterHTML("html", &html, chromedp.ByQuery),
			)
		}

		cancelTimeout()
		cancelCtx()
		cancel()

		if err != nil {
			if attempt >= retries || !shouldRetry(err, 0) {
				return Result{}, err
			}
			time.Sleep(backoff(baseDelay, attempt))
			continue
		}

		return Result{
			URL:       req.URL,
			Status:    200,
			HTML:      html,
			FetchedAt: time.Now(),
		}, nil
	}

	return Result{}, errors.New("max retries exceeded")
}
