// Package exporter provides JSONL parsing utilities.
//
// These utilities parse JSONL (one JSON object per line) from readers and bytes.
// Functions include:
// - parseSingle/parseSingleReader: Parse a single JSON object
// - parseLines/parseLinesReader: Parse multiple JSON objects
// - safe: Provide default values for empty strings
//
// This file does NOT handle export logic or format-specific transformations.
package exporter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// ensureSeekable ensures that the provided reader is an io.ReadSeeker.
// If it is already a seeker, it is returned as is with a no-op cleanup.
// Otherwise, it is buffered to a temporary file which is returned as the seeker.
func ensureSeekable(r io.Reader) (io.ReadSeeker, func(), error) {
	if rs, ok := r.(io.ReadSeeker); ok {
		return rs, func() {}, nil
	}

	tempFile, err := os.CreateTemp("", "spartan-export-*")
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "failed to create temp file for export", err)
	}

	cleanup := func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}

	if _, err := io.Copy(tempFile, r); err != nil {
		cleanup()
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "failed to buffer export data to temp file", err)
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "failed to seek temp file", err)
	}

	return tempFile, cleanup, nil
}

// scanReader scans JSON objects from the reader and calls the provided function for each object.
func scanReader[T any](r io.Reader, fn func(T) error) error {
	scanner := bufio.NewScanner(r)
	// Match buffer size (10MB max line)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return err
		}
		if err := fn(item); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// parseSingle parses a single JSON object from raw bytes.
// Returns the parsed object or an error if no valid JSON is found.
func parseSingle[T any](raw []byte) (T, error) {
	var out T
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			return out, err
		}
		return out, nil
	}
	return out, errors.New("no content")
}

// parseLines parses multiple JSON objects (one per line) from raw bytes.
// Returns a slice of parsed objects or an error if parsing fails.
func parseLines[T any](raw []byte) ([]T, error) {
	items := make([]T, 0)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// parseSingleReader parses a single JSON object from an io.Reader.
// Returns the parsed object or an error if no valid JSON is found.
func parseSingleReader[T any](r io.Reader) (T, error) {
	var out T
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			return out, err
		}
		return out, nil
	}
	return out, errors.New("no content")
}

// parseLinesReader parses multiple JSON objects (one per line) from an io.Reader.
// Returns a slice of parsed objects or an error if parsing fails.
func parseLinesReader[T any](r io.Reader) ([]T, error) {
	items := make([]T, 0)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// safe returns the value if it's non-empty after trimming, otherwise returns the fallback.
// Used to provide default values for optional fields in formatted output.
func safe(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
