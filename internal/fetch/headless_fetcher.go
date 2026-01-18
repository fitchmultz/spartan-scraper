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

	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if req.UserAgent != "" {
		allocatorOpts = append(allocatorOpts, chromedp.UserAgent(req.UserAgent))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, req.Timeout)
	defer cancel()

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
	if err := chromedp.Run(ctx, actions...); err != nil {
		return Result{}, err
	}

	if req.Auth.LoginURL != "" {
		if req.Auth.LoginUserSelector == "" || req.Auth.LoginPassSelector == "" || req.Auth.LoginSubmitSelector == "" {
			return Result{}, errors.New("login selectors are required for headless login")
		}
		err := chromedp.Run(ctx,
			chromedp.Navigate(req.Auth.LoginURL),
			chromedp.WaitVisible(req.Auth.LoginUserSelector),
			chromedp.SendKeys(req.Auth.LoginUserSelector, req.Auth.LoginUser),
			chromedp.SendKeys(req.Auth.LoginPassSelector, req.Auth.LoginPass),
			chromedp.Click(req.Auth.LoginSubmitSelector),
		)
		if err != nil {
			return Result{}, err
		}
	}

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(req.URL),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return Result{}, err
	}

	return Result{
		URL:       req.URL,
		Status:    200,
		HTML:      html,
		FetchedAt: time.Now(),
	}, nil
}
