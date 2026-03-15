// Package batch provides CLI commands for batch job operations.
//
// This file contains API client functions for batch operations.
//
// Responsibilities:
// - Submit batch jobs via HTTP API
// - Get batch status via HTTP API
// - Cancel batches via HTTP API
//
// Does NOT handle:
// - Direct execution (when server is not running)
// - CLI command parsing
// - File parsing
package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func submitBatchScrapeViaAPI(ctx context.Context, port string, req BatchScrapeRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/scrape", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchCrawlViaAPI(ctx context.Context, port string, req BatchCrawlRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/crawl", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchResearchViaAPI(ctx context.Context, port string, req BatchResearchRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/research", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchViaAPI(ctx context.Context, url string, req interface{}) (*BatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result BatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func getBatchStatusViaAPI(ctx context.Context, port, batchID string, includeJobs bool) (*BatchStatusResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/%s", port, batchID)
	if includeJobs {
		url += "?include_jobs=true"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("batch %s not found", batchID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result BatchStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func cancelBatchViaAPI(ctx context.Context, port, batchID string) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/%s", port, batchID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("batch %s not found", batchID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result BatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
