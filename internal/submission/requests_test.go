// Package submission verifies operator-facing request validation and conversion.
//
// Purpose:
// - Prove shared request conversion rejects invalid webhook configuration before job creation.
//
// Responsibilities:
// - Exercise strict raw-request conversion for scrape, crawl, and research payloads.
// - Verify webhook URL syntax errors are surfaced consistently at create time.
//
// Scope:
// - Submission request validation only.
//
// Usage:
// - Run with `go test ./internal/submission`.
//
// Invariants/Assumptions:
// - ResolveAuth is disabled so tests focus on request validation rather than auth profile lookup.
package submission

import (
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestJobSpecFromRawRequestRejectsInvalidWebhookURL(t *testing.T) {
	defaults := Defaults{DefaultTimeoutSeconds: 30, DefaultUsePlaywright: false, ResolveAuth: false}
	tests := []struct {
		name    string
		kind    model.Kind
		body    string
		wantErr string
	}{
		{
			name:    "scrape invalid webhook scheme",
			kind:    model.KindScrape,
			body:    "{\"url\":\"https://example.com\",\"webhook\":{\"url\":\"ftp://example.com/webhook\"}}",
			wantErr: "webhook URL must use http or https scheme",
		},
		{
			name:    "crawl invalid webhook scheme",
			kind:    model.KindCrawl,
			body:    "{\"url\":\"https://example.com\",\"maxDepth\":1,\"maxPages\":10,\"webhook\":{\"url\":\"ftp://example.com/webhook\"}}",
			wantErr: "webhook URL must use http or https scheme",
		},
		{
			name:    "research invalid webhook scheme",
			kind:    model.KindResearch,
			body:    "{\"query\":\"pricing\",\"urls\":[\"https://example.com\"],\"maxDepth\":1,\"maxPages\":10,\"webhook\":{\"url\":\"ftp://example.com/webhook\"}}",
			wantErr: "webhook URL must use http or https scheme",
		},
		{
			name:    "present webhook object requires url",
			kind:    model.KindScrape,
			body:    "{\"url\":\"https://example.com\",\"webhook\":{}}",
			wantErr: "webhook URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := JobSpecFromRawRequest(config.Config{}, defaults, tt.kind, []byte(tt.body))
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
