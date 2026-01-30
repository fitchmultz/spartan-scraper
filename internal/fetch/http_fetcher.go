// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"golang.org/x/net/proxy"
)

// HTTPFetcher implements content fetching using the standard library http.Client.
// Provides retry logic, rate limiting, authentication, conditional requests,
// and response size limits. See fetcher.go for the Fetcher interface definition.
type HTTPFetcher struct {
	proxyPool *ProxyPool
}

// SetProxyPool sets the proxy pool for this fetcher.
func (f *HTTPFetcher) SetProxyPool(pool *ProxyPool) {
	f.proxyPool = pool
}

// isSuccessStatus returns true for 2xx and 3xx status codes (excluding 304 which is handled separately)
func isSuccessStatus(status int) bool {
	return status >= 200 && status < 400
}

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

	// Load session cookies if SessionID is provided
	if req.SessionID != "" && req.DataDir != "" {
		if err := applySessionToJar(jar, req.SessionID, req.DataDir, req.URL); err != nil {
			slog.Warn("failed to apply session cookies", "sessionID", req.SessionID, "error", err)
		} else {
			slog.Debug("applied session cookies", "sessionID", req.SessionID)
		}
	}

	// Configure transport with proxy support
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // Default: use env vars
	}

	// Track selected proxy for metrics
	var selectedProxy *ProxyEntry

	// If proxy pool is configured and no explicit proxy, select from pool
	if f.proxyPool != nil && (req.Auth.Proxy == nil || req.Auth.Proxy.URL == "") {
		hints := ProxySelectionHints{}
		if req.Auth.ProxyHints != nil {
			hints = *req.Auth.ProxyHints
		}

		proxy, err := f.proxyPool.Select(hints)
		if err != nil {
			slog.Warn("failed to select proxy from pool", "url", apperrors.SanitizeURL(req.URL), "error", err)
		} else {
			selectedProxy = &proxy
			proxyConfig := proxy.ToProxyConfig()
			req.Auth.Proxy = &proxyConfig
			slog.Debug("selected proxy from pool", "url", apperrors.SanitizeURL(req.URL), "proxy_id", proxy.ID)
		}
	}

	// If proxy explicitly configured (or selected from pool), apply it
	if req.Auth.Proxy != nil && req.Auth.Proxy.URL != "" {
		proxyURL, err := url.Parse(req.Auth.Proxy.URL)
		if err != nil {
			if selectedProxy != nil {
				f.proxyPool.RecordFailure(selectedProxy.ID, err)
			}
			return Result{}, fmt.Errorf("invalid proxy URL: %w", err)
		}

		// Handle SOCKS5 proxies
		if strings.HasPrefix(strings.ToLower(req.Auth.Proxy.URL), "socks5://") {
			dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				if selectedProxy != nil {
					f.proxyPool.RecordFailure(selectedProxy.ID, err)
				}
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

	// Parse host once for circuit breaker and result tracking
	parsedURL, _ := url.Parse(req.URL)
	host := ""
	if parsedURL != nil {
		host = parsedURL.Host
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
			// Record proxy failure on final attempt
			if selectedProxy != nil && f.proxyPool != nil && (attempt >= retries || !shouldRetry(err, 0)) {
				f.proxyPool.RecordFailure(selectedProxy.ID, err)
			}
			// Record failure for circuit breaker
			if req.Limiter != nil && host != "" {
				req.Limiter.RecordResult(host, err, 0)
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
			// Record failure for circuit breaker
			if req.Limiter != nil && host != "" {
				req.Limiter.RecordResult(host, readErr, resp.StatusCode)
			}
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

		// Report 429 to adaptive rate limiter if enabled (regardless of whether we'll retry)
		if resp.StatusCode == http.StatusTooManyRequests && req.Limiter != nil && req.Limiter.IsAdaptiveEnabled() {
			if host != "" {
				delay := readRetryAfter(resp)
				if delay <= 0 {
					delay = backoff(baseDelay, attempt)
				}
				req.Limiter.RecordRateLimit(host, delay)
			}
		}

		if shouldRetry(nil, resp.StatusCode) && attempt < retries {
			// Record failure for circuit breaker (status code triggered retry)
			if req.Limiter != nil && host != "" {
				req.Limiter.RecordResult(host, nil, resp.StatusCode)
			}
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

		slog.Debug("HTTP fetch complete", "url", apperrors.SanitizeURL(req.URL), "status", resp.StatusCode)

		// Record success or failure for circuit breaker
		if req.Limiter != nil && host != "" {
			if isSuccessStatus(resp.StatusCode) {
				req.Limiter.RecordResult(host, nil, resp.StatusCode)
				// Also report success to adaptive rate limiter
				if req.Limiter.IsAdaptiveEnabled() {
					req.Limiter.RecordSuccess(host)
				}
			} else if resp.StatusCode >= 500 {
				// Server error - record as failure
				req.Limiter.RecordResult(host, nil, resp.StatusCode)
			}
		}

		// Record proxy pool metrics
		if selectedProxy != nil && f.proxyPool != nil {
			if isSuccessStatus(resp.StatusCode) {
				f.proxyPool.RecordSuccess(selectedProxy.ID, 0) // Latency not measured for HTTP
			} else if resp.StatusCode >= 500 {
				f.proxyPool.RecordFailure(selectedProxy.ID, fmt.Errorf("HTTP %d", resp.StatusCode))
			}
		}

		result := Result{
			URL:       req.URL,
			Status:    resp.StatusCode,
			HTML:      string(body),
			FetchedAt: time.Now(),
			Engine:    RenderEngineHTTP,
		}

		// Extract rate limit information from response headers
		if rlInfo, ok := ExtractRateLimitInfo(resp.Header); ok {
			result.RateLimit = &rlInfo
			// Update adaptive rate limiter with server-provided limit if available
			if req.Limiter != nil && host != "" && req.Limiter.IsAdaptiveEnabled() {
				req.Limiter.UpdateRateLimitInfo(host, rlInfo)
			}
		}

		return result, nil
	}

	slog.Error("HTTP fetch max retries exceeded", "url", apperrors.SanitizeURL(req.URL))
	return Result{}, errors.New("max retries exceeded")
}

// sessionCookie represents a cookie stored in a session.
type sessionCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"httpOnly,omitempty"`
}

// session represents a persisted cookie session for a domain.
type session struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Domain  string          `json:"domain"`
	Cookies []sessionCookie `json:"cookies"`
}

// applySessionToJar loads session cookies from disk and adds them to the cookie jar.
func applySessionToJar(jar http.CookieJar, sessionID, dataDir, targetURL string) error {
	sessions, err := loadSessions(dataDir)
	if err != nil {
		return err
	}

	var sess *session
	for i := range sessions {
		if sessions[i].ID == sessionID {
			sess = &sessions[i]
			break
		}
	}
	if sess == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	httpCookies := make([]*http.Cookie, 0, len(sess.Cookies))
	for _, c := range sess.Cookies {
		httpCookies = append(httpCookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		})
	}

	jar.SetCookies(parsed, httpCookies)
	return nil
}

// loadSessions loads all sessions from the sessions.json file.
func loadSessions(dataDir string) ([]session, error) {
	if dataDir == "" {
		dataDir = ".data"
	}
	path := filepath.Join(dataDir, "sessions.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []session{}, nil
		}
		return nil, err
	}

	var sessions []session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}
