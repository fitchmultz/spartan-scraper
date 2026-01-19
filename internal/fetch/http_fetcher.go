package fetch

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// HTTPFetcher implements content fetching using the standard Go http.Client.
type HTTPFetcher struct{}

// Fetch performs a standard HTTP GET request to retrieve the content of a URL.
// It supports retries, rate limiting, and basic/token authentication.
func (f *HTTPFetcher) Fetch(ctx context.Context, req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("HTTP fetch start", "url", req.URL)

	// Apply auth query parameters before making the request
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 300 * time.Millisecond
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying HTTP fetch", "url", req.URL, "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", req.URL)
			_ = req.Limiter.Wait(ctx, req.URL)
		}

		jar, _ := cookiejar.New(nil)
		client := &http.Client{
			Timeout: req.Timeout,
			Jar:     jar,
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
		if err != nil {
			slog.Error("failed to create HTTP request", "url", req.URL, "error", err)
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
			slog.Warn("HTTP request failed", "url", req.URL, "error", err, "attempt", attempt)
			if attempt >= retries || !shouldRetry(err, 0) {
				return Result{}, err
			}
			delay := backoff(baseDelay, attempt)
			slog.Debug("backing off before retry", "url", req.URL, "delay", delay)
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode == http.StatusNotModified {
			slog.Debug("HTTP 304 Not Modified", "url", req.URL)
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
			slog.Warn("failed to read HTTP response body", "url", req.URL, "error", readErr, "attempt", attempt)
			if attempt >= retries || !shouldRetry(readErr, resp.StatusCode) {
				return Result{}, readErr
			}
			delay := backoff(baseDelay, attempt)
			slog.Debug("backing off before retry", "url", req.URL, "delay", delay)
			time.Sleep(delay)
			continue
		}

		if shouldRetry(nil, resp.StatusCode) && attempt < retries {
			delay := readRetryAfter(resp)
			if delay <= 0 {
				delay = backoff(baseDelay, attempt)
			}
			slog.Info("retrying HTTP request based on status code", "url", req.URL, "status", resp.StatusCode, "attempt", attempt, "delay", delay)
			time.Sleep(delay)
			continue
		}

		slog.Debug("HTTP fetch success", "url", req.URL, "status", resp.StatusCode)
		return Result{
			URL:       req.URL,
			Status:    resp.StatusCode,
			HTML:      string(body),
			FetchedAt: time.Now(),
			Engine:    RenderEngineHTTP,
		}, nil
	}

	slog.Error("HTTP fetch max retries exceeded", "url", req.URL)
	return Result{}, errors.New("max retries exceeded")
}
