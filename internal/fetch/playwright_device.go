// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file provides device emulation for Playwright fetcher.
// It handles viewport configuration, mobile device simulation, and device profile
// resolution from requests and render profiles.
package fetch

import (
	"github.com/playwright-community/playwright-go"
)

// resolveDevice determines which device emulation to use.
// Priority: req.Device > prof.Device > req.Screenshot.Device > prof.Screenshot.Device > none
func (f *PlaywrightFetcher) resolveDevice(req Request, prof RenderProfile) *DeviceEmulation {
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

// applyDeviceEmulation applies device emulation settings to the Playwright context options.
func (f *PlaywrightFetcher) applyDeviceEmulation(opts *playwright.BrowserNewContextOptions, device *DeviceEmulation) {
	if device == nil {
		return
	}

	// Apply orientation if needed (swaps dimensions for landscape on mobile/tablet)
	effectiveDevice := device
	if device.Orientation == OrientationLandscape && device.Category != DeviceCategoryDesktop {
		effectiveDevice = device.ApplyOrientation(OrientationLandscape)
	}

	// Set viewport
	viewport := playwright.Size{
		Width:  effectiveDevice.ViewportWidth,
		Height: effectiveDevice.ViewportHeight,
	}
	opts.Viewport = &viewport

	// Set device scale factor
	if effectiveDevice.DeviceScaleFactor > 0 {
		opts.DeviceScaleFactor = &effectiveDevice.DeviceScaleFactor
	}

	// Set mobile flag
	opts.IsMobile = &effectiveDevice.IsMobile

	// Set touch support
	opts.HasTouch = &effectiveDevice.HasTouch

	// Set user agent
	if effectiveDevice.UserAgent != "" {
		opts.UserAgent = &effectiveDevice.UserAgent
	}
}
