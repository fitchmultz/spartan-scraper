// Sample Spartan Scraper Plugin
//
// This is a sample WASM plugin that demonstrates how to extend Spartan Scraper
// with custom pipeline hooks. This plugin adds a custom header to all requests.
//
// Building:
//   tinygo build -o plugin.wasm -target wasm .
//
// The plugin must export functions matching the hook names it wants to handle:
//   - pre_fetch(input_ptr, input_len) -> output_ptr_with_len
//   - post_fetch(input_ptr, input_len) -> output_ptr_with_len
//   - pre_extract(input_ptr, input_len) -> output_ptr_with_len
//   - post_extract(input_ptr, input_len) -> output_ptr_with_len
//   - pre_output(input_ptr, input_len) -> output_ptr_with_len
//   - post_output(input_ptr, input_len) -> output_ptr_with_len
//
// Input/output is JSON-encoded. Memory management uses exported malloc/free.

//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
)

// FetchInput represents the input to the pre_fetch hook
type FetchInput struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Timeout   int               `json:"timeout_ms,omitempty"`
	Headless  bool              `json:"headless,omitempty"`
}

// FetchOutput represents the output from the pre_fetch hook
type FetchOutput struct {
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Timeout   int               `json:"timeout_ms,omitempty"`
	Headless  *bool             `json:"headless,omitempty"`
	Skip      bool              `json:"skip,omitempty"`
}

// pre_fetch is called before each fetch request
//
//export pre_fetch
func pre_fetch(inputPtr int32, inputLen int32) int64 {
	// Read input from memory using host function
	inputBytes := readMemory(inputPtr, inputLen)

	var input FetchInput
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return 0 // Return 0 on error
	}

	// Add custom header
	if input.Headers == nil {
		input.Headers = make(map[string]string)
	}
	input.Headers["X-Sample-Plugin"] = "enabled"

	// Build output
	output := FetchOutput{
		URL:       input.URL,
		Headers:   input.Headers,
		UserAgent: input.UserAgent,
		Timeout:   input.Timeout,
		Headless:  &input.Headless,
	}

	// Serialize output
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return 0
	}

	// Write output to memory and return encoded pointer
	return writeMemory(outputBytes)
}

// Host function imports (provided by the host runtime)
// These are defined by the WASM runtime and linked at load time

//go:wasmimport spartan log
func hostLog(ptr int32, len int32)

//go:wasmimport spartan malloc
func hostMalloc(size int32) int32

//go:wasmimport spartan free
func hostFree(ptr int32)

// Helper functions

func readMemory(ptr int32, length int32) []byte {
	// In actual WASM, this reads from linear memory at the given pointer
	// This implementation depends on the specific WASM runtime
	return nil
}

func writeMemory(data []byte) int64 {
	// Allocate memory via host
	ptr := hostMalloc(int32(len(data)))
	if ptr == 0 {
		return 0
	}

	// Write data to memory
	// In actual WASM, this writes to linear memory at the allocated pointer

	// Return encoded pointer: high 32 bits = ptr, low 32 bits = len
	return (int64(ptr) << 32) | int64(len(data))
}

func main() {
	// Required for building as WASM, but not used
	// The actual entry points are the exported functions
}
