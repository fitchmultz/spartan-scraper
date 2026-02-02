// Package batch provides CLI commands for batch job operations.
//
// This file contains parsing logic for batch job inputs.
//
// Responsibilities:
// - Parse batch jobs from CSV/JSON files
// - Parse batch jobs from URL lists
// - Detect file formats automatically
//
// Does NOT handle:
// - API submission
// - Direct execution
// - Status checking
package batch

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func parseBatchJobs(filePath, urlsList, method, body, contentType string) ([]BatchJobRequest, error) {
	if filePath != "" {
		return parseBatchJobsFromFile(filePath)
	}

	if urlsList != "" {
		urls := strings.Split(urlsList, ",")
		jobs := make([]BatchJobRequest, 0, len(urls))
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				m := method
				if m == "" {
					m = "GET"
				}
				jobs = append(jobs, BatchJobRequest{
					URL:         url,
					Method:      m,
					Body:        body,
					ContentType: contentType,
				})
			}
		}
		return jobs, nil
	}

	return nil, nil
}

func parseBatchJobsFromFile(filePath string) ([]BatchJobRequest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try JSON first
	if strings.HasSuffix(filePath, ".json") || looksLikeJSON(data) {
		var jobs []BatchJobRequest
		if err := json.Unmarshal(data, &jobs); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		return jobs, nil
	}

	// Try CSV
	if strings.HasSuffix(filePath, ".csv") {
		return parseBatchJobsFromCSV(data)
	}

	return nil, fmt.Errorf("unsupported file format (use .json or .csv)")
}

func looksLikeJSON(data []byte) bool {
	data = []byte(strings.TrimSpace(string(data)))
	return len(data) > 0 && (data[0] == '[' || data[0] == '{')
}

func parseBatchJobsFromCSV(data []byte) ([]BatchJobRequest, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Detect if first row is headers
	var urlIdx, methodIdx, bodyIdx, contentTypeIdx int = -1, -1, -1, -1
	firstRow := records[0]

	// Check if first row looks like headers
	hasHeaders := false
	for _, col := range firstRow {
		lower := strings.ToLower(strings.TrimSpace(col))
		if lower == "url" || lower == "method" || lower == "body" || lower == "contenttype" || lower == "content-type" {
			hasHeaders = true
			break
		}
	}

	if hasHeaders {
		for i, col := range firstRow {
			switch strings.ToLower(strings.TrimSpace(col)) {
			case "url":
				urlIdx = i
			case "method":
				methodIdx = i
			case "body":
				bodyIdx = i
			case "contenttype", "content-type":
				contentTypeIdx = i
			}
		}
		records = records[1:]
	} else {
		// Default column order: url, method, body, contentType
		urlIdx = 0
		if len(firstRow) > 1 {
			methodIdx = 1
		}
		if len(firstRow) > 2 {
			bodyIdx = 2
		}
		if len(firstRow) > 3 {
			contentTypeIdx = 3
		}
	}

	if urlIdx < 0 {
		return nil, fmt.Errorf("CSV must have a 'url' column")
	}

	jobs := make([]BatchJobRequest, 0, len(records))
	for _, row := range records {
		if len(row) <= urlIdx {
			continue
		}
		url := strings.TrimSpace(row[urlIdx])
		if url == "" {
			continue
		}

		job := BatchJobRequest{URL: url, Method: "GET"}

		if methodIdx >= 0 && methodIdx < len(row) {
			if m := strings.TrimSpace(row[methodIdx]); m != "" {
				job.Method = strings.ToUpper(m)
			}
		}
		if bodyIdx >= 0 && bodyIdx < len(row) {
			job.Body = strings.TrimSpace(row[bodyIdx])
		}
		if contentTypeIdx >= 0 && contentTypeIdx < len(row) {
			job.ContentType = strings.TrimSpace(row[contentTypeIdx])
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}
