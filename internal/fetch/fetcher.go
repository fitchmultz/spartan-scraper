// Package fetch provides browser/tooling availability checks and fetcher construction helpers.
//
// Purpose:
// - Centralize fetcher factories plus browser and Playwright prerequisite detection.
//
// Responsibilities:
// - Create adaptive fetchers with optional metrics and proxy-pool wiring.
// - Detect Chrome/Chromium availability across supported host platforms.
// - Cache Playwright readiness checks while allowing explicit refresh probes.
//
// Scope:
// - Shared fetcher setup and availability probing only; concrete fetching lives in sibling files.
//
// Usage:
// - Called by runtime initialization, health endpoints, and diagnostic helpers.
//
// Invariants/Assumptions:
// - Availability checks must never launch long-running browser sessions.
// - Fresh diagnostic probes may invalidate cached Playwright readiness state.
package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/playwright-community/playwright-go"
)

var (
	lookPath      = exec.LookPath
	osStat        = os.Stat
	osGetenv      = os.Getenv
	playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
		return playwright.Run(options...)
	}
	currentOS      = runtime.GOOS
	playwrightMu   sync.Mutex
	playwrightOnce = &sync.Once{}
	playwrightErr  error
)

type Fetcher interface {
	Fetch(ctx context.Context, req Request) (Result, error)
}

// MetricsCallback is the function signature for metrics collection callbacks.
type MetricsCallback func(duration time.Duration, success bool, fetcherType, url string)

// FetcherWithMetrics is a fetcher that supports metrics callbacks.
type FetcherWithMetrics interface {
	Fetcher
	SetMetricsCallback(cb MetricsCallback)
}

func NewFetcher(dataDir string) Fetcher {
	return NewAdaptiveFetcher(dataDir)
}

// NewFetcherWithMetrics creates a new fetcher with metrics callback support.
func NewFetcherWithMetrics(dataDir string, callback MetricsCallback) FetcherWithMetrics {
	af := NewAdaptiveFetcher(dataDir)
	if callback != nil {
		af.SetMetricsCallback(callback)
	}
	return af
}

// NewFetcherWithProxyPool creates a new fetcher with proxy pool support.
func NewFetcherWithProxyPool(dataDir string, pool *ProxyPool) Fetcher {
	af := NewAdaptiveFetcher(dataDir)
	if pool != nil {
		af.SetProxyPool(pool)
	}
	return af
}

// NewFetcherWithMetricsAndProxyPool creates a new fetcher with both metrics and proxy pool support.
func NewFetcherWithMetricsAndProxyPool(dataDir string, callback MetricsCallback, pool *ProxyPool) FetcherWithMetrics {
	af := NewAdaptiveFetcher(dataDir)
	if callback != nil {
		af.SetMetricsCallback(callback)
	}
	if pool != nil {
		af.SetProxyPool(pool)
	}
	return af
}

var (
	ErrChromeNotFound     = apperrors.ErrChromeNotFound
	ErrPlaywrightNotReady = apperrors.ErrPlaywrightNotReady
)

// FindChrome resolves the Chrome/Chromium binary path for diagnostics and runtime checks.
func FindChrome() (string, error) {
	return findChrome()
}

// CheckBrowserAvailability checks if the required browser binaries are available.
func CheckBrowserAvailability(usePlaywright bool) error {
	if usePlaywright {
		return checkPlaywrightAvailability()
	}
	return checkChromeAvailability()
}

// CheckBrowserAvailabilityFresh forces a new availability probe.
func CheckBrowserAvailabilityFresh(usePlaywright bool) error {
	if !usePlaywright {
		return checkChromeAvailability()
	}

	playwrightMu.Lock()
	playwrightOnce = &sync.Once{}
	playwrightErr = nil
	playwrightMu.Unlock()

	return checkPlaywrightAvailability()
}

func checkChromeAvailability() error {
	_, err := findChrome()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "chrome/chromium not found on PATH", ErrChromeNotFound)
	}
	return nil
}

func checkPlaywrightAvailability() error {
	playwrightMu.Lock()
	once := playwrightOnce
	playwrightMu.Unlock()

	once.Do(func() {
		err := performPlaywrightAvailabilityCheck()
		playwrightMu.Lock()
		playwrightErr = err
		playwrightMu.Unlock()
	})

	playwrightMu.Lock()
	defer playwrightMu.Unlock()
	return playwrightErr
}

func performPlaywrightAvailabilityCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type result struct {
		pw  *playwright.Playwright
		err error
	}
	resultChan := make(chan result, 1)

	go func() {
		pw, err := playwrightRun()
		resultChan <- result{pw: pw, err: err}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: timeout while checking playwright availability", ErrPlaywrightNotReady)
	case res := <-resultChan:
		if res.err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "playwright drivers not installed or not found", ErrPlaywrightNotReady)
		}
		if res.pw != nil {
			if err := res.pw.Stop(); err != nil {
				slog.Debug("playwright.Stop() failed during availability check", "error", err)
			}
		}
		return nil
	}
}

func findChrome() (string, error) {
	binaries := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"chrome",
	}

	for _, bin := range binaries {
		if path, err := lookPath(bin); err == nil {
			return path, nil
		}
	}

	switch currentOS {
	case "darwin":
		macPaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, path := range macPaths {
			if _, err := osStat(path); err == nil {
				return path, nil
			}
		}
	case "windows":
		winPaths := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			osGetenv("USERPROFILE") + `\AppData\Local\Google\Chrome\Application\chrome.exe`,
		}
		for _, path := range winPaths {
			if _, err := osStat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", ErrChromeNotFound
}
