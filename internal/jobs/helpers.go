// Package jobs provides helper functions for parameter decoding and type conversion.
// This file contains utility functions that decode job parameters from map[string]interface{}
// storage into strongly-typed Go structs used by job execution.
//
// Responsibilities:
// - Decoding AuthOptions from stored parameters
// - Decoding ExtractOptions from stored parameters
// - Decoding PipelineOptions from stored parameters
// - Type-safe conversions with fallback values (toInt, toBool, toStringSlice)
//
// This file does NOT:
// - Perform validation beyond type checking
// - Handle business logic or parameter constraints
//
// Invariants:
// - All decode functions return empty structs on failure (never panic)
// - Type conversion helpers return explicit fallback values for invalid inputs
// - Nested maps (headers, cookies, query) are flattened safely

package jobs

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func (m *Manager) DefaultTimeoutSeconds() int {
	return int(m.requestTimeout.Seconds())
}

func (m *Manager) DefaultUsePlaywright() bool {
	return m.usePlaywright
}

// GetLimiter returns the host rate limiter for metrics registration.
func (m *Manager) GetLimiter() *fetch.HostLimiter {
	return m.limiter
}

// SetMetricsCallback sets the callback function for recording fetch metrics.
func (m *Manager) SetMetricsCallback(cb func(duration time.Duration, success bool, fetcherType, url string)) {
	m.metricsCallback = cb
}

func reqRetries(v int) int {
	return v
}

func decodeAuth(value interface{}) fetch.AuthOptions {
	return paramdecode.DecodeValue[fetch.AuthOptions](value)
}

func decodeExtract(value interface{}) extract.ExtractOptions {
	return paramdecode.DecodeValue[extract.ExtractOptions](value)
}

func toInt(value interface{}, fallback int) int {
	return paramdecode.PositiveIntValue(value, fallback)
}

func toStringSlice(value interface{}) []string {
	return paramdecode.StringSliceValue(value)
}

func toBool(value interface{}, fallback bool) bool {
	return paramdecode.BoolValue(value, fallback)
}

func decodePipeline(value interface{}) pipeline.Options {
	return paramdecode.DecodeValue[pipeline.Options](value)
}

func decodeScreenshot(value interface{}) *fetch.ScreenshotConfig {
	return paramdecode.DecodeValuePtr[fetch.ScreenshotConfig](value)
}

// decodeBytes decodes a byte slice from interface{}.
// Handles []byte, []interface{} (of uint8), and string types.
// Strings are first attempted to be base64-decoded (for JSON round-tripped bytes),
// and if that fails, treated as raw bytes for backward compatibility.
func decodeBytes(value interface{}) []byte {
	return paramdecode.BytesValue(value)
}
