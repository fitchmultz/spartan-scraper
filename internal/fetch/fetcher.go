// Package fetch provides abstractions and implementations for fetching web content.
// It includes support for standard HTTP requests, headless browser rendering
// (via Chromedp or Playwright), rate limiting, and automatic retries with backoff.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/playwright-community/playwright-go"
)

var (
	lookPath      = exec.LookPath
	osStat        = os.Stat
	osGetenv      = os.Getenv
	playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
		return playwright.Run(options...)
	}
	currentOS = runtime.GOOS
)

type Fetcher interface {
	Fetch(ctx context.Context, req Request) (Result, error)
}

func NewFetcher() Fetcher {
	return NewAdaptiveFetcher()
}

var (
	ErrChromeNotFound     = errors.New("chrome/chromium not found on PATH")
	ErrPlaywrightNotReady = errors.New("playwright drivers not installed or not found")
)

// CheckBrowserAvailability checks if the required browser binaries are available.
func CheckBrowserAvailability(usePlaywright bool) error {
	if usePlaywright {
		return checkPlaywrightAvailability()
	}
	return checkChromeAvailability()
}

func checkChromeAvailability() error {
	_, err := findChrome()
	if err != nil {
		return fmt.Errorf("%w: searched for: google-chrome, google-chrome-stable, chromium, chromium-browser, chrome", ErrChromeNotFound)
	}
	return nil
}

func checkPlaywrightAvailability() error {
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
			return fmt.Errorf("%w: %w", ErrPlaywrightNotReady, res.err)
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
