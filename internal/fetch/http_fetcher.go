package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type HTTPFetcher struct{}

func (f *HTTPFetcher) Fetch(req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 300 * time.Millisecond
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if req.Limiter != nil {
			_ = req.Limiter.Wait(context.Background(), req.URL)
		}

		jar, _ := cookiejar.New(nil)
		client := &http.Client{
			Timeout: req.Timeout,
			Jar:     jar,
		}

		httpReq, err := http.NewRequest(http.MethodGet, req.URL, nil)
		if err != nil {
			return Result{}, err
		}

		if req.UserAgent != "" {
			httpReq.Header.Set("User-Agent", req.UserAgent)
		}
		if req.IfNoneMatch != "" {
			httpReq.Header.Set("If-None-Match", req.IfNoneMatch)
		}
		if req.IfModifiedSince != "" {
			httpReq.Header.Set("If-Modified-Since", req.IfModifiedSince)
		}
		for k, v := range req.Auth.Headers {
			httpReq.Header.Set(k, v)
		}
		for _, cookie := range req.Auth.Cookies {
			parts := strings.SplitN(cookie, "=", 2)
			if len(parts) == 2 {
				httpReq.AddCookie(&http.Cookie{Name: parts[0], Value: parts[1]})
			}
		}
		if req.Auth.Basic != "" {
			parts := strings.SplitN(req.Auth.Basic, ":", 2)
			if len(parts) == 2 {
				httpReq.SetBasicAuth(parts[0], parts[1])
			}
		}

		resp, err := client.Do(httpReq)
		if err != nil || resp == nil {
			if attempt >= retries || !shouldRetry(err, 0) {
				return Result{}, err
			}
			time.Sleep(backoff(baseDelay, attempt))
			continue
		}

		if resp.StatusCode == http.StatusNotModified {
			_ = resp.Body.Close()
			return Result{
				URL:          req.URL,
				Status:       resp.StatusCode,
				HTML:         "",
				FetchedAt:    time.Now(),
				Engine:       RenderEngineHTTP,
				ETag:         resp.Header.Get("ETag"),
				LastModified: resp.Header.Get("Last-Modified"),
			}, nil
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			if attempt >= retries || !shouldRetry(readErr, resp.StatusCode) {
				return Result{}, readErr
			}
			time.Sleep(backoff(baseDelay, attempt))
			continue
		}

		if shouldRetry(nil, resp.StatusCode) && attempt < retries {
			delay := readRetryAfter(resp)
			if delay <= 0 {
				delay = backoff(baseDelay, attempt)
			}
			time.Sleep(delay)
			continue
		}

		return Result{
			URL:       req.URL,
			Status:    resp.StatusCode,
			HTML:      string(body),
			FetchedAt: time.Now(),
			Engine:    RenderEngineHTTP,
		}, nil
	}

	return Result{}, errors.New("max retries exceeded")
}
