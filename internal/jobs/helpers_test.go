// Package jobs provides unit tests for helper functions.
// This file contains tests for decodeBytes and other parameter decoding utilities.
//
// Test coverage:
// - decodeBytes: nil, raw []byte, base64-encoded strings, regular strings,
//                []interface{} (JSON arrays), and invalid base64
//
// Invariants:
// - All decode functions must handle nil input gracefully
// - Base64 decoding must correctly handle JSON round-tripped []byte values
// - Regular strings must remain unchanged when not valid base64

package jobs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []byte
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "raw []byte returns same bytes",
			input:    []byte("hello world"),
			expected: []byte("hello world"),
		},
		{
			name:     "empty []byte returns empty",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "base64-encoded string returns decoded bytes",
			input:    base64.StdEncoding.EncodeToString([]byte("hello world")),
			expected: []byte("hello world"),
		},
		{
			name:     "base64-encoded binary data",
			input:    base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0xFF}),
			expected: []byte{0x00, 0x01, 0x02, 0xFF},
		},
		{
			name:     "regular string returns raw bytes (backward compatibility)",
			input:    "hello world",
			expected: []byte("hello world"),
		},
		{
			name:     "empty string returns empty bytes",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "invalid base64 returns raw bytes",
			input:    "!!!not-valid-base64!!!",
			expected: []byte("!!!not-valid-base64!!!"),
		},
		{
			name:     "string with spaces treated as raw bytes (not valid base64)",
			input:    "hello world test",
			expected: []byte("hello world test"),
		},
		{
			name:     "[]interface{} of float64 converts to bytes",
			input:    []interface{}{104.0, 101.0, 108.0, 108.0, 111.0}, // "hello"
			expected: []byte("hello"),
		},
		{
			name:     "empty []interface{} returns empty",
			input:    []interface{}{},
			expected: []byte{},
		},
		{
			name:     "[]interface{} with non-float64 values filtered",
			input:    []interface{}{104.0, "ignored", 105.0}, // "hi"
			expected: []byte("hi"),
		},
		{
			name:     "JSON round-trip simulation: base64-encoded JSON body",
			input:    base64.StdEncoding.EncodeToString([]byte(`{"key": "value"}`)),
			expected: []byte(`{"key": "value"}`),
		},
		{
			name:     "Unicode content in base64",
			input:    base64.StdEncoding.EncodeToString([]byte("Hello, 世界! 🌍")),
			expected: []byte("Hello, 世界! 🌍"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeBytes(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("decodeBytes(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDecodeBytes_JSONRoundTrip verifies that bytes survive JSON marshal/unmarshal cycle
func TestDecodeBytes_JSONRoundTrip(t *testing.T) {
	original := []byte(`{"query": "test", "payload": [1, 2, 3]}`)

	// Simulate what happens during job persistence:
	// 1. Job created with []byte body
	// 2. json.Marshal base64-encodes the []byte
	// 3. Stored in database as JSON string
	// 4. json.Unmarshal into map[string]interface{} gives us a string

	// Step 2: Marshal the params
	params := map[string]interface{}{"body": original}
	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	// Step 4: Unmarshal back into map (simulating retrieval from store)
	var retrieved map[string]interface{}
	if err := json.Unmarshal(jsonData, &retrieved); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	// The body should now be a base64-encoded string
	bodyValue, ok := retrieved["body"].(string)
	if !ok {
		t.Fatalf("expected body to be string after unmarshal, got %T", retrieved["body"])
	}

	// Verify it's base64 encoded (this is what happens during storage)
	if _, err := base64.StdEncoding.DecodeString(bodyValue); err != nil {
		t.Logf("Note: body is not valid base64 (raw string), treating as raw bytes")
	}

	// Now decode using our function
	decoded := decodeBytes(retrieved["body"])

	if !bytes.Equal(decoded, original) {
		t.Errorf("JSON round-trip failed: got %q, want %q", decoded, original)
	}
}
