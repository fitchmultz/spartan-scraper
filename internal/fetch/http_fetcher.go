// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"golang.org/x/net/proxy"
)

// HTTPFetcher implements content fetching using the standard library http.Client.
// Provides retry logic, rate limiting, authentication, conditional requests,
// and response size limits. See fetcher.go for the Fetcher interface definition.
type HTTPFetcher struct{}

// sleepWithContext sleeps for the given duration or until the context is cancelled.
// Returns ctx.Err() if cancelled, nil otherwise.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Fetch performs a standard HTTP GET request to retrieve the content of a URL.
// It supports retries, rate limiting, and basic/token authentication.
func (f *HTTPFetcher) Fetch(ctx context.Context, req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	slog.Debug("HTTP fetch start", "url", apperrors.SanitizeURL(req.URL))

	// Apply auth query parameters before making the request
	req.URL = ApplyAuthQuery(req.URL, req.Auth.Query)

	retries := clampRetry(req.MaxRetries)
	baseDelay := req.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 300 * time.Millisecond
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Configure transport with proxy support
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // Default: use env vars
	}

	// If proxy explicitly configured, override
	if req.Auth.Proxy != nil && req.Auth.Proxy.URL != "" {
		proxyURL, err := url.Parse(req.Auth.Proxy.URL)
		if err != nil {
			return Result{}, fmt.Errorf("invalid proxy URL: %w", err)
		}

		// Handle SOCKS5 proxies
		if strings.HasPrefix(strings.ToLower(req.Auth.Proxy.URL), "socks5://") {
			dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				return Result{}, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
			}
			transport.DialContext = dialer.(proxy.ContextDialer).DialContext
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}

		// Set proxy authentication if provided
		if req.Auth.Proxy.Username != "" {
			transport.ProxyConnectHeader = http.Header{
				"Proxy-Authorization": []string{
					"Basic " + base64.StdEncoding.EncodeToString(
						[]byte(req.Auth.Proxy.Username+":"+req.Auth.Proxy.Password),
					),
				},
			}
		}
	}

	client := &http.Client{
		Timeout:   req.Timeout,
		Jar:       jar,
		Transport: transport,
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying HTTP fetch", "url", apperrors.SanitizeURL(req.URL), "attempt", attempt)
		}

		if req.Limiter != nil {
			slog.Debug("waiting for rate limiter", "url", apperrors.SanitizeURL(req.URL))
			if err := req.Limiter.Wait(ctx, req.URL); err != nil {
				return Result{}, err
			}
		}

		// Determine HTTP method (default to GET if not specified)
		method := req.Method
		if method == "" {
			method = http.MethodGet
		}

		// Create request body reader if body is present
		var reqBodyReader io.Reader
		if len(req.Body) > 0 {
			reqBodyReader = bytes.NewReader(req.Body)
		}

		httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, reqBodyReader)
		if err != nil {
			slog.Error("failed to create HTTP request", "url", apperrors.SanitizeURL(req.URL), "error", err)
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

		// Set Content-Type header if body is present and content type is specified
		if len(req.Body) > 0 && req.ContentType != "" {
			httpReq.Header.Set("Content-Type", req.ContentType)
		}

		resp, err := client.Do(httpReq)
		if err != nil || resp == nil {
			slog.Warn("HTTP request failed", "url", apperrors.SanitizeURL(req.URL), "error", err, "attempt", attempt)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if attempt >= retries || !shouldRetry(err, 0) {
				return Result{}, err
			}
			delay := backoff(baseDelay, attempt)
			slog.Debug("backing off before retry", "url", apperrors.SanitizeURL(req.URL), "delay", delay)
			if err := sleepWithContext(ctx, delay); err != nil {
				return Result{}, err
			}
			continue
		}

		if resp.StatusCode == http.StatusNotModified {
			slog.Debug("HTTP 304 Not Modified", "url", apperrors.SanitizeURL(req.URL))
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

		// Enforce max response size limit
		var bodyReader io.Reader = resp.Body
		if req.MaxResponseBytes > 0 {
			// +1 allows us to detect when limit is exceeded
			bodyReader = io.LimitReader(resp.Body, req.MaxResponseBytes+1)
		}
		body, readErr := io.ReadAll(bodyReader)

		// Check if response exceeded the size limit
		if req.MaxResponseBytes > 0 && int64(len(body)) > req.MaxResponseBytes {
			_ = resp.Body.Close()
			return Result{}, fmt.Errorf("response body exceeded maximum size of %d bytes", req.MaxResponseBytes)
		}

		_ = resp.Body.Close()
		if readErr != nil {
			slog.Warn("failed to read HTTP response body", "url", apperrors.SanitizeURL(req.URL), "error", readErr, "attempt", attempt)
			if attempt >= retries || !shouldRetry(readErr, resp.StatusCode) {
				return Result{}, readErr
			}
			delay := backoff(baseDelay, attempt)
			slog.Debug("backing off before retry", "url", apperrors.SanitizeURL(req.URL), "delay", delay)
			if err := sleepWithContext(ctx, delay); err != nil {
				return Result{}, err
			}
			continue
		}

		if shouldRetry(nil, resp.StatusCode) && attempt < retries {
			delay := readRetryAfter(resp)
			if delay <= 0 {
				delay = backoff(baseDelay, attempt)
			}
			slog.Debug("retrying HTTP request based on status code", "url", apperrors.SanitizeURL(req.URL), "status", resp.StatusCode, "attempt", attempt, "delay", delay)
			if err := sleepWithContext(ctx, delay); err != nil {
				return Result{}, err
			}
			continue
		}

		slog.Debug("HTTP fetch success", "url", apperrors.SanitizeURL(req.URL), "status", resp.StatusCode)
		return Result{
			URL:       req.URL,
			Status:    resp.StatusCode,
			HTML:      string(body),
			FetchedAt: time.Now(),
			Engine:    RenderEngineHTTP,
		}, nil
	}

	slog.Error("HTTP fetch max retries exceeded", "url", apperrors.SanitizeURL(req.URL))
	return Result{}, errors.New("max retries exceeded")
}
