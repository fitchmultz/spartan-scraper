// Package common provides common functionality for Spartan Scraper.
//
// Purpose:
// - Implement execution support for package common.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `common` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package common

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func ResolveDevicePreset(name string) (*fetch.DeviceEmulation, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, nil
	}
	device := fetch.GetDevicePreset(trimmed)
	if device == nil {
		return nil, fmt.Errorf("unknown device preset: %s", trimmed)
	}
	return device, nil
}

func BuildScreenshotConfig(enabled bool, fullPage bool, format string, quality int, width int, height int) (*fetch.ScreenshotConfig, error) {
	if !enabled {
		return nil, nil
	}
	config := &fetch.ScreenshotConfig{
		Enabled:  true,
		FullPage: fullPage,
		Format:   fetch.ScreenshotFormat(strings.ToLower(strings.TrimSpace(format))),
		Quality:  quality,
		Width:    width,
		Height:   height,
	}
	switch config.Format {
	case "", fetch.ScreenshotFormatPNG:
		config.Format = fetch.ScreenshotFormatPNG
		config.Quality = 0
	case fetch.ScreenshotFormatJPEG:
		if config.Quality <= 0 {
			config.Quality = 90
		}
		if config.Quality > 100 {
			return nil, fmt.Errorf("invalid screenshot quality: %d", config.Quality)
		}
	default:
		return nil, fmt.Errorf("invalid screenshot format: %s", format)
	}
	return config, nil
}

func BuildNetworkInterceptConfig(enabled bool, urlPatterns []string, resourceTypes []string, captureRequestBody bool, captureResponseBody bool, maxBodySize int, maxEntries int) *fetch.NetworkInterceptConfig {
	if !enabled {
		return nil
	}
	resolvedTypes := make([]fetch.InterceptedResourceType, 0, len(resourceTypes))
	for _, rawType := range resourceTypes {
		if trimmed := strings.TrimSpace(rawType); trimmed != "" {
			resolvedTypes = append(resolvedTypes, fetch.InterceptedResourceType(trimmed))
		}
	}
	return &fetch.NetworkInterceptConfig{
		Enabled:             true,
		URLPatterns:         urlPatterns,
		ResourceTypes:       resolvedTypes,
		CaptureRequestBody:  captureRequestBody,
		CaptureResponseBody: captureResponseBody,
		MaxBodySize:         int64(maxBodySize),
		MaxEntries:          maxEntries,
	}
}
