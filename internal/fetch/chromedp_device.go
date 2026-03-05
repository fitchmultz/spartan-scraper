// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file provides device emulation and screenshot capture for chromedp.
// It handles viewport configuration, mobile device simulation, and full-page
// or viewport screenshot generation with configurable formats.
package fetch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// captureScreenshot captures a screenshot of the current page using chromedp.
// It returns the path to the saved screenshot file.
func (f *ChromedpFetcher) captureScreenshot(ctx context.Context, req Request, dataDir string) (string, error) {
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
	if req.Screenshot.Format == ScreenshotFormatJPEG {
		ext = "jpg"
	}
	filename := fmt.Sprintf("chromedp_%d_%d.%s", timestamp, req.Screenshot.Quality, ext)
	path := filepath.Join(screenshotDir, filename)

	// Set viewport if custom dimensions provided and no device emulation is active
	device := f.resolveDevice(req, RenderProfile{})
	if device == nil && req.Screenshot.Width > 0 && req.Screenshot.Height > 0 {
		width := int64(req.Screenshot.Width)
		height := int64(req.Screenshot.Height)
		if err := chromedp.Run(ctx, chromedp.EmulateViewport(width, height)); err != nil {
			return "", fmt.Errorf("failed to set viewport: %w", err)
		}
	}

	var buf []byte
	if req.Screenshot.FullPage {
		if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, req.Screenshot.Quality)); err != nil {
			return "", fmt.Errorf("failed to capture full page screenshot: %w", err)
		}
	} else {
		if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
			return "", fmt.Errorf("failed to capture screenshot: %w", err)
		}
	}

	// Write screenshot to file
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return "", fmt.Errorf("failed to write screenshot file: %w", err)
	}

	return path, nil
}

// resolveDevice determines which device emulation to use.
// Priority: req.Device > prof.Device > req.Screenshot.Device > prof.Screenshot.Device > none
func (f *ChromedpFetcher) resolveDevice(req Request, prof RenderProfile) *DeviceEmulation {
	if req.Device != nil {
		return req.Device
	}
	if prof.Device != nil {
		return prof.Device
	}
	if req.Screenshot != nil && req.Screenshot.Device != nil {
		return req.Screenshot.Device
	}
	if prof.Screenshot.Device != nil {
		return prof.Screenshot.Device
	}
	return nil
}

// applyDeviceEmulation applies device emulation settings to the chromedp context.
func (f *ChromedpFetcher) applyDeviceEmulation(ctx context.Context, device *DeviceEmulation) error {
	if device == nil {
		return nil
	}

	// Apply orientation if needed (swaps dimensions for landscape on mobile/tablet)
	effectiveDevice := device
	if device.Orientation == OrientationLandscape && device.Category != DeviceCategoryDesktop {
		effectiveDevice = device.ApplyOrientation(OrientationLandscape)
	}

	// Set viewport
	width := int64(effectiveDevice.ViewportWidth)
	height := int64(effectiveDevice.ViewportHeight)
	if err := chromedp.Run(ctx, chromedp.EmulateViewport(width, height)); err != nil {
		return fmt.Errorf("failed to set viewport: %w", err)
	}

	// Set device scale factor
	if effectiveDevice.DeviceScaleFactor > 0 {
		if err := chromedp.Run(ctx, emulation.SetDeviceMetricsOverride(
			int64(effectiveDevice.ViewportWidth),
			int64(effectiveDevice.ViewportHeight),
			effectiveDevice.DeviceScaleFactor,
			effectiveDevice.IsMobile,
		)); err != nil {
			return fmt.Errorf("failed to set device metrics: %w", err)
		}
	}

	// Set touch emulation
	if effectiveDevice.HasTouch {
		if err := chromedp.Run(ctx, emulation.SetTouchEmulationEnabled(true)); err != nil {
			return fmt.Errorf("failed to enable touch emulation: %w", err)
		}
	}

	// Set user agent (if not already set via allocator options)
	if effectiveDevice.UserAgent != "" {
		if err := chromedp.Run(ctx, emulation.SetUserAgentOverride(effectiveDevice.UserAgent)); err != nil {
			return fmt.Errorf("failed to set user agent: %w", err)
		}
	}

	return nil
}
