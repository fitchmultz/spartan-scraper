// Package exporter provides pagination utilities for API endpoints.
//
// These utilities support paginated result export from HTTP endpoints:
// - Limit: Parse and validate limit query parameter (1-1000)
// - Offset: Parse and validate offset query parameter (>= 0)
// - ExportPaginated: Read JSONL and return paginated slice with total count
//
// This file does NOT handle format-specific export logic - only pagination.
package exporter

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Limit parses and validates the limit query parameter from an HTTP request.
// Returns the limit value if valid (1-1000), otherwise returns default of 100.
// Used to control pagination size in results endpoints.
func Limit(r *http.Request) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return 100
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

// Offset parses and validates the offset query parameter from an HTTP request.
// Returns the offset value if valid (>= 0), otherwise returns default of 0.
// Used to control pagination starting position in results endpoints.
func Offset(r *http.Request) int {
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		return 0
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

// ExportPaginated reads JSONL from a reader and returns a paginated slice of items.
// Parameters:
//   - r: Reader containing JSONL data (one JSON object per line)
//   - limit: Maximum number of items to return
//   - offset: Number of items to skip before returning results
//
// Returns:
//   - []T: Slice of parsed items (max of limit items)
//   - int: Total number of items in the source (before pagination)
//   - error: Error if parsing fails or reader encounters an error
//
// Used by API endpoints to implement efficient pagination without loading full dataset into memory.
func ExportPaginated[T any](r io.Reader, limit, offset int) ([]T, int, error) {
	items := make([]T, 0, limit)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	total := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		total++
		if total <= offset {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	return items[:min(len(items), limit)], total, nil
}

// min returns the minimum of two integers.
// Helper function used to enforce pagination limit constraints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
