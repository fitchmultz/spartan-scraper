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
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
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
	if value == nil {
		return fetch.AuthOptions{}
	}
	if auth, ok := value.(fetch.AuthOptions); ok {
		return auth
	}
	data, ok := value.(map[string]interface{})
	if !ok {
		return fetch.AuthOptions{}
	}
	auth := fetch.AuthOptions{}
	if v, ok := data["basic"].(string); ok {
		auth.Basic = v
	}
	if v, ok := data["loginUrl"].(string); ok {
		auth.LoginURL = v
	}
	if v, ok := data["loginUserSelector"].(string); ok {
		auth.LoginUserSelector = v
	}
	if v, ok := data["loginPassSelector"].(string); ok {
		auth.LoginPassSelector = v
	}
	if v, ok := data["loginSubmitSelector"].(string); ok {
		auth.LoginSubmitSelector = v
	}
	if v, ok := data["loginUser"].(string); ok {
		auth.LoginUser = v
	}
	if v, ok := data["loginPass"].(string); ok {
		auth.LoginPass = v
	}
	if headers, ok := data["headers"].(map[string]interface{}); ok {
		m := map[string]string{}
		for k, v := range headers {
			if sv, ok := v.(string); ok {
				m[k] = sv
			}
		}
		auth.Headers = m
	}
	if cookies, ok := data["cookies"].([]interface{}); ok {
		values := make([]string, 0, len(cookies))
		for _, v := range cookies {
			if sv, ok := v.(string); ok {
				values = append(values, sv)
			}
		}
		auth.Cookies = values
	}
	if query, ok := data["query"].(map[string]interface{}); ok {
		m := map[string]string{}
		for k, v := range query {
			if sv, ok := v.(string); ok {
				m[k] = sv
			}
		}
		auth.Query = m
	}
	return auth
}

func decodeExtract(value interface{}) extract.ExtractOptions {
	if value == nil {
		return extract.ExtractOptions{}
	}
	if opts, ok := value.(extract.ExtractOptions); ok {
		return opts
	}
	data, err := json.Marshal(value)
	if err != nil {
		return extract.ExtractOptions{}
	}
	var opts extract.ExtractOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return extract.ExtractOptions{}
	}
	return opts
}

func toInt(value interface{}, fallback int) int {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return fallback
		}
		return v
	case float64:
		if int(v) <= 0 {
			return fallback
		}
		return int(v)
	default:
		return fallback
	}
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	default:
		return nil
	}
}

func toBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return fallback
	}
}

func decodePipeline(value interface{}) pipeline.Options {
	if value == nil {
		return pipeline.Options{}
	}
	if opts, ok := value.(pipeline.Options); ok {
		return opts
	}
	data, err := json.Marshal(value)
	if err != nil {
		return pipeline.Options{}
	}
	var opts pipeline.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return pipeline.Options{}
	}
	return opts
}

func decodeScreenshot(value interface{}) *fetch.ScreenshotConfig {
	if value == nil {
		return nil
	}
	if cfg, ok := value.(*fetch.ScreenshotConfig); ok {
		return cfg
	}
	if cfg, ok := value.(fetch.ScreenshotConfig); ok {
		return &cfg
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var cfg fetch.ScreenshotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

// decodeBytes decodes a byte slice from interface{}.
// Handles []byte, []interface{} (of uint8), and string types.
// Strings are first attempted to be base64-decoded (for JSON round-tripped bytes),
// and if that fails, treated as raw bytes for backward compatibility.
func decodeBytes(value interface{}) []byte {
	if value == nil {
		return nil
	}
	if b, ok := value.([]byte); ok {
		return b
	}
	if s, ok := value.(string); ok {
		// Try base64 decode first (for JSON round-tripped bytes)
		// Go's json.Marshal base64-encodes []byte, which becomes a string after unmarshal
		if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
			return decoded
		}
		// Fall back to raw bytes for actual strings or invalid base64
		return []byte(s)
	}
	// Handle []interface{} case (JSON arrays of numbers)
	if arr, ok := value.([]interface{}); ok {
		result := make([]byte, 0, len(arr))
		for _, v := range arr {
			if n, ok := v.(float64); ok {
				result = append(result, byte(n))
			}
		}
		return result
	}
	return nil
}
