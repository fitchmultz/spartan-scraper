package fetch

import (
	"errors"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

func shouldRetry(err error, status int) bool {
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return true
		}
		if strings.Contains(err.Error(), "timeout") {
			return true
		}
		return true
	}
	if status == 429 {
		return true
	}
	if status >= 500 && status < 600 {
		return true
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
