// Package exporter provides domain models for export results.
//
// These types represent the structure of exported data for different job kinds:
// - ScrapeResult: Single page scrape with extracted fields and metadata
// - CrawlResult: Single page result from a crawl job
// - ResearchResult: Research job output with evidence, clusters, and citations
//
// This file does NOT handle parsing or export logic - it only defines data structures.
package exporter

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// RenderPathTemplate substitutes variables in the path template.
// Supported variables:
//   - {job_id}: Job ID (e.g., "job-abc123")
//   - {timestamp}: Current timestamp in format 20060102_150405
//   - {kind}: Job kind (scrape, crawl, research)
//   - {format}: Export format extension (jsonl, json, md, csv, xlsx)
func RenderPathTemplate(template string, job model.Job, format string) string {
	if template == "" {
		template = "{kind}/{timestamp}.{format}"
	}

	timestamp := time.Now().Format("20060102_150405")

	result := template
	result = replaceAll(result, "{job_id}", job.ID)
	result = replaceAll(result, "{timestamp}", timestamp)
	result = replaceAll(result, "{kind}", string(job.Kind))
	result = replaceAll(result, "{format}", format)

	return result
}

// replaceAll replaces all occurrences of old with new in s.
func replaceAll(s, old, new string) string {
	// Simple string replacement - strings.ReplaceAll would be used
	// but we implement manually to avoid import cycles if needed
	result := s
	for {
		idx := 0
		found := false
		for i := 0; i <= len(result)-len(old); i++ {
			if result[i:i+len(old)] == old {
				idx = i
				found = true
				break
			}
		}
		if !found {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

// ScrapeResult represents a single page scrape result with extracted fields and metadata.
type ScrapeResult struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

// CrawlResult represents a single page result from a crawl job.
type CrawlResult struct {
	URL        string                     `json:"url"`
	Status     int                        `json:"status"`
	Title      string                     `json:"title"`
	Text       string                     `json:"text"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

// ResearchResult represents the output of a research job with evidence, clusters, and citations.
type ResearchResult struct {
	Query      string  `json:"query"`
	Summary    string  `json:"summary"`
	Confidence float64 `json:"confidence"`
	Evidence   []struct {
		URL         string  `json:"url"`
		Title       string  `json:"title"`
		Snippet     string  `json:"snippet"`
		Score       float64 `json:"score"`
		SimHash     uint64  `json:"simhash"`
		ClusterID   string  `json:"clusterId"`
		Confidence  float64 `json:"confidence"`
		CitationURL string  `json:"citationUrl"`
	} `json:"evidence"`
	Clusters []struct {
		ID         string  `json:"id"`
		Label      string  `json:"label"`
		Confidence float64 `json:"confidence"`
		Evidence   []struct {
			URL         string  `json:"url"`
			Title       string  `json:"title"`
			Snippet     string  `json:"snippet"`
			Score       float64 `json:"score"`
			SimHash     uint64  `json:"simhash"`
			ClusterID   string  `json:"clusterId"`
			Confidence  float64 `json:"confidence"`
			CitationURL string  `json:"citationUrl"`
		} `json:"evidence"`
	} `json:"clusters"`
	Citations []struct {
		URL       string `json:"url"`
		Anchor    string `json:"anchor"`
		Canonical string `json:"canonical"`
	} `json:"citations"`
}
