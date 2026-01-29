// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func shouldRetry(err error, status int) bool {
	if status == 429 {
		return true
	}
	if status >= 500 && status < 600 {
		return true
	}

	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidURLScheme) {
			return false
		}
		if errors.Is(err, apperrors.ErrInvalidURLHost) {
			return false
		}

		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			if dnsErr.IsNotFound {
				return false
			}
			if dnsErr.IsTimeout {
				return true
			}
			return false
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}

		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return true
			}
		}

		if errors.Is(err, net.ErrClosed) {
			return true
		}

		if strings.Contains(err.Error(), "timeout") {
			return true
		}

		return false
	}

	return false
}

func backoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	multiplier := math.Pow(2, float64(attempt))
	return time.Duration(float64(base) * multiplier)
}

func clampRetry(count int) int {
	if count < 0 {
		return 0
	}
	if count > 10 {
		return 10
	}
	return count
}

func readRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	value := resp.Header.Get("Retry-After")
	if value == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil {
		return seconds
	}
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}
	return 0
}
