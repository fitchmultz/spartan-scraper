// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file provides screenshot capture for Playwright fetcher.
// It handles viewport configuration, file generation, and full-page or viewport
// screenshot capture with configurable formats (PNG/JPEG).
package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

// captureScreenshot captures a screenshot of the current page using Playwright.
// It returns the path to the saved screenshot file.
func (f *PlaywrightFetcher) captureScreenshot(page playwright.Page, req Request, prof RenderProfile, dataDir string) (string, error) {
	if req.Screenshot == nil || !req.Screenshot.Enabled {
		return "", nil
	}

	// Generate screenshot filename
	screenshotDir := filepath.Join(dataDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create screenshots directory: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	ext := "png"
	format := playwright.ScreenshotTypePng
	if req.Screenshot.Format == ScreenshotFormatJPEG {
		ext = "jpg"
		format = playwright.ScreenshotTypeJpeg
	}
	filename := fmt.Sprintf("playwright_%d_%d.%s", timestamp, req.Screenshot.Quality, ext)
	path := filepath.Join(screenshotDir, filename)

	// Set viewport if custom dimensions provided and no device emulation is active
	device := f.resolveDevice(req, prof)
	if device == nil && req.Screenshot.Width > 0 && req.Screenshot.Height > 0 {
		if err := page.SetViewportSize(req.Screenshot.Width, req.Screenshot.Height); err != nil {
			return "", fmt.Errorf("failed to set viewport: %w", err)
		}
	}

	// Build screenshot options
	opts := playwright.PageScreenshotOptions{
		Path:     &path,
		FullPage: &req.Screenshot.FullPage,
		Type:     format,
	}

	// Add quality for JPEG
	if req.Screenshot.Format == ScreenshotFormatJPEG && req.Screenshot.Quality > 0 {
		quality := req.Screenshot.Quality
		opts.Quality = &quality
	}

	// Capture screenshot
	if _, err := page.Screenshot(opts); err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	return path, nil
}
