// Package api implements the REST API server for Spartan Scraper.
//
// This file handles traffic replay functionality for captured network requests.
package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// handleTrafficReplay handles POST /v1/jobs/{id}/replay
func (s *Server) handleTrafficReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Extract job ID from path (path is /v1/jobs/replay/{id})
	id := extractID(r.URL.Path, "replay")
	if id == "" {
		writeError(w, r, apperrors.Validation("job id required"))
		return
	}

	// Parse request body
	var req TrafficReplayRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate required fields
	if req.TargetBaseURL == "" {
		writeError(w, r, apperrors.Validation("targetBaseUrl is required"))
		return
	}

	targetURL, err := url.Parse(req.TargetBaseURL)
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		writeError(w, r, apperrors.Validation("invalid targetBaseUrl"))
		return
	}

	// Get job from store
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := s.validateJobResultPath(job, "job has no result path"); err != nil {
		writeError(w, r, err)
		return
	}

	// Load job results
	entries, err := s.loadInterceptedEntries(job)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if len(entries) == 0 {
		writeError(w, r, apperrors.NotFound("no intercepted traffic found for this job"))
		return
	}

	// Filter entries
	filteredEntries := filterEntries(entries, req.Filter)

	if len(filteredEntries) == 0 {
		writeError(w, r, apperrors.Validation("no requests match the specified filter"))
		return
	}

	// Set default timeout - use config value when request doesn't specify
	timeout := req.Timeout
	if timeout == 0 {
		timeout = s.cfg.ReplayTimeoutSecs
	}

	// Replay requests
	startTime := time.Now()
	results := make([]ReplayResult, 0, len(filteredEntries))
	var comparison *ReplayComparison

	if req.CompareResponses {
		comparison = &ReplayComparison{
			Differences: make([]ResponseDiff, 0),
		}
	}

	successful := 0
	failed := 0

	for _, entry := range filteredEntries {
		result := s.replayRequest(entry, req.TargetBaseURL, req.Modifications, timeout)
		results = append(results, result)

		if result.Error == "" {
			successful++
		} else {
			failed++
		}

		// Compare responses if requested
		if req.CompareResponses && entry.Response != nil && result.Error == "" {
			diff := compareResponse(entry.Response, result.ReplayedResponse)
			if diff != nil {
				comparison.Differences = append(comparison.Differences, *diff)
			} else {
				comparison.Matches++
			}
			comparison.TotalCompared++
		}
	}

	duration := int(time.Since(startTime).Milliseconds())

	if comparison != nil {
		comparison.Mismatches = len(comparison.Differences)
	}

	response := TrafficReplayResponse{
		JobID:         id,
		TotalRequests: len(filteredEntries),
		Successful:    successful,
		Failed:        failed,
		Results:       results,
		Comparison:    comparison,
		Duration:      duration,
	}

	writeJSON(w, response)
}

// loadInterceptedEntries loads intercepted entries from job results.
func (s *Server) loadInterceptedEntries(job model.Job) ([]fetch.InterceptedEntry, error) {
	file, err := s.openJobResultFile(job, "job has no result path")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make([]fetch.InterceptedEntry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result struct {
			InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
		}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		entries = append(entries, result.InterceptedData...)
	}
	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read job results", err)
	}

	return entries, nil
}

// filterEntries filters intercepted entries based on criteria.
func filterEntries(entries []fetch.InterceptedEntry, filter *TrafficReplayFilter) []fetch.InterceptedEntry {
	if filter == nil {
		return entries
	}

	var filtered []fetch.InterceptedEntry

	for _, entry := range entries {
		// Check URL patterns
		if len(filter.URLPatterns) > 0 {
			matched := false
			for _, pattern := range filter.URLPatterns {
				if matchURLPattern(entry.Request.URL, pattern) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Check methods
		if len(filter.Methods) > 0 {
			methodMatch := false
			for _, method := range filter.Methods {
				if strings.EqualFold(entry.Request.Method, method) {
					methodMatch = true
					break
				}
			}
			if !methodMatch {
				continue
			}
		}

		// Check resource types
		if len(filter.ResourceTypes) > 0 {
			typeMatch := false
			for _, rt := range filter.ResourceTypes {
				if string(entry.Request.ResourceType) == rt {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				continue
			}
		}

		// Check status codes
		if len(filter.StatusCodes) > 0 && entry.Response != nil {
			statusMatch := false
			for _, code := range filter.StatusCodes {
				if entry.Response.Status == code {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// matchURLPattern checks if a URL matches a glob-style pattern.
// Supports * to match any characters (including /).
func matchURLPattern(urlStr, pattern string) bool {
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}

	// Convert glob pattern to regex
	var result strings.Builder
	result.WriteString("^")

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			result.WriteString(".*")
		case '?', '+', '.', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			result.WriteByte('\\')
			result.WriteByte(c)
		default:
			result.WriteByte(c)
		}
	}

	result.WriteString("$")

	re, err := regexp.Compile(result.String())
	if err != nil {
		return false
	}

	return re.MatchString(urlStr)
}

// replayRequest replays a single intercepted request.
func (s *Server) replayRequest(
	entry fetch.InterceptedEntry,
	targetBaseURL string,
	modifications *TrafficModifications,
	timeout int,
) ReplayResult {
	originalURL := entry.Request.URL

	// Transform URL to target base
	newURL, err := transformURL(originalURL, targetBaseURL)
	if err != nil {
		return ReplayResult{
			OriginalRequest: replayRequestFromIntercepted(entry.Request),
			Error:           fmt.Sprintf("failed to transform URL: %v", err),
		}
	}

	// Build request
	var bodyReader io.Reader
	if entry.Request.Body != "" {
		bodyReader = strings.NewReader(entry.Request.Body)
	}

	req, err := http.NewRequest(entry.Request.Method, newURL, bodyReader)
	if err != nil {
		return ReplayResult{
			OriginalRequest: replayRequestFromIntercepted(entry.Request),
			Error:           fmt.Sprintf("failed to create request: %v", err),
		}
	}

	// Copy original headers
	for key, value := range entry.Request.Headers {
		req.Header.Set(key, value)
	}

	// Apply modifications
	if modifications != nil {
		// Remove headers
		for _, header := range modifications.RemoveHeaders {
			req.Header.Del(header)
		}

		// Add/replace headers
		for key, value := range modifications.Headers {
			req.Header.Set(key, value)
		}
	}

	// Execute request
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return ReplayResult{
			OriginalRequest: replayRequestFromIntercepted(entry.Request),
			ReplayedRequest: ReplayRequestInfo{
				URL:     newURL,
				Method:  entry.Request.Method,
				Headers: headersToMap(req.Header),
				Body:    entry.Request.Body,
			},
			Error:    fmt.Sprintf("request failed: %v", err),
			Duration: duration,
		}
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplayResult{
			OriginalRequest: replayRequestFromIntercepted(entry.Request),
			ReplayedRequest: ReplayRequestInfo{
				URL:     newURL,
				Method:  entry.Request.Method,
				Headers: headersToMap(req.Header),
				Body:    entry.Request.Body,
			},
			Error:    fmt.Sprintf("failed to read response: %v", err),
			Duration: duration,
		}
	}

	return ReplayResult{
		OriginalRequest: replayRequestFromIntercepted(entry.Request),
		ReplayedRequest: ReplayRequestInfo{
			URL:     newURL,
			Method:  entry.Request.Method,
			Headers: headersToMap(req.Header),
			Body:    entry.Request.Body,
		},
		ReplayedResponse: ReplayResponseInfo{
			Status:     resp.StatusCode,
			StatusText: http.StatusText(resp.StatusCode),
			Headers:    headersToMap(resp.Header),
			Body:       string(body),
			BodySize:   len(body),
		},
		Duration: duration,
	}
}

// transformURL transforms an original URL to use a new base URL.
func transformURL(originalURL, targetBaseURL string) (string, error) {
	original, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	target, err := url.Parse(targetBaseURL)
	if err != nil {
		return "", err
	}

	// Build new URL with target base and original path/query
	newURL := *target
	newURL.Path = original.Path
	newURL.RawQuery = original.RawQuery
	newURL.Fragment = original.Fragment

	return newURL.String(), nil
}

// replayRequestFromIntercepted converts an intercepted request to replay format.
func replayRequestFromIntercepted(req fetch.InterceptedRequest) ReplayRequestInfo {
	return ReplayRequestInfo{
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		Body:    req.Body,
	}
}

// headersToMap converts http.Header to a simple map.
func headersToMap(header http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range header {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// compareResponse compares original and replayed responses.
func compareResponse(original *fetch.InterceptedResponse, replayed ReplayResponseInfo) *ResponseDiff {
	hasDiff := false
	diff := ResponseDiff{
		RequestID:   original.RequestID,
		HeaderDiffs: make([]HeaderDiff, 0),
	}

	// Compare status codes
	if original.Status != replayed.Status {
		hasDiff = true
		diff.StatusDiff = &StatusDiff{
			Original: original.Status,
			Replayed: replayed.Status,
		}
	}

	// Compare headers
	for key, origValue := range original.Headers {
		replayValue, exists := replayed.Headers[key]
		if !exists || origValue != replayValue {
			hasDiff = true
			diff.HeaderDiffs = append(diff.HeaderDiffs, HeaderDiff{
				Name:     key,
				Original: origValue,
				Replayed: replayValue,
			})
		}
	}

	// Check for new headers in replay
	for key, replayValue := range replayed.Headers {
		if _, exists := original.Headers[key]; !exists {
			hasDiff = true
			diff.HeaderDiffs = append(diff.HeaderDiffs, HeaderDiff{
				Name:     key,
				Replayed: replayValue,
			})
		}
	}

	// Compare body sizes
	if original.BodySize != int64(replayed.BodySize) {
		hasDiff = true
		diff.BodyDiff = &BodyDiff{
			OriginalSize: int(original.BodySize),
			ReplayedSize: replayed.BodySize,
		}

		// Generate preview of difference
		if len(original.Body) > 0 || len(replayed.Body) > 0 {
			preview := generateBodyDiffPreview(original.Body, replayed.Body)
			diff.BodyDiff.Preview = preview
		}
	}

	if hasDiff {
		return &diff
	}
	return nil
}

// generateBodyDiffPreview generates a preview of body differences.
func generateBodyDiffPreview(original, replayed string) string {
	const maxPreviewLen = 200

	// Simple preview: show first difference
	minLen := len(original)
	if len(replayed) < minLen {
		minLen = len(replayed)
	}

	diffPos := -1
	for i := 0; i < minLen; i++ {
		if original[i] != replayed[i] {
			diffPos = i
			break
		}
	}

	if diffPos == -1 {
		if len(original) != len(replayed) {
			return fmt.Sprintf("Length differs: %d vs %d", len(original), len(replayed))
		}
		return "Bodies are identical"
	}

	// Show context around difference
	start := diffPos - 50
	if start < 0 {
		start = 0
	}
	end := diffPos + 50
	if end > minLen {
		end = minLen
	}

	preview := fmt.Sprintf("Diff at position %d:\nOriginal: ...%s...\nReplayed: ...%s...",
		diffPos,
		truncateString(original[start:end], maxPreviewLen),
		truncateString(replayed[start:end], maxPreviewLen))

	return preview
}

// truncateString truncates a string to max length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
