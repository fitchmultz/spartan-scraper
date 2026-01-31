// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file contains tests for resolver helper functions.
package graphql

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestCursorEncoding tests cursor encoding and decoding.
func TestCursorEncoding(t *testing.T) {
	tests := []struct {
		offset int
	}{
		{offset: 0},
		{offset: 10},
		{offset: 100},
		{offset: 999999},
	}

	for _, tt := range tests {
		t.Run("offset_"+string(rune(tt.offset)), func(t *testing.T) {
			encoded := encodeCursor(tt.offset)
			decoded := decodeCursor(encoded)
			assert.Equal(t, tt.offset, decoded)
		})
	}
}

// TestCursorDecodingInvalid tests decoding invalid cursors.
func TestCursorDecodingInvalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "empty string",
			cursor: "",
		},
		{
			name:   "invalid base64",
			cursor: "!!!",
		},
		{
			name:   "wrong format",
			cursor: "bm90LWN1cnNvcjoxMA==", // "not-cursor:10"
		},
		{
			name:   "non-numeric offset",
			cursor: "Y3Vyc29yOmFiYw==", // "cursor:abc"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeCursor(tt.cursor)
			assert.Equal(t, 0, result)
		})
	}
}

// TestWithResolverContext tests adding resolver context.
func TestWithResolverContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithResolverContext(ctx, nil, nil)

	rctx, ok := ctx.Value(resolverContextKey).(*resolverContext)
	assert.True(t, ok)
	assert.NotNil(t, rctx)
}

// TestParseJobFilter tests parsing job filter arguments.
func TestParseJobFilter(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		expected *JobFilter
	}{
		{
			name:     "empty filter",
			args:     map[string]interface{}{},
			expected: &JobFilter{},
		},
		{
			name: "with status",
			args: map[string]interface{}{
				"status": model.StatusRunning,
			},
			expected: &JobFilter{
				Status: model.StatusRunning,
			},
		},
		{
			name: "with kind",
			args: map[string]interface{}{
				"kind": model.KindScrape,
			},
			expected: &JobFilter{
				Kind: model.KindScrape,
			},
		},
		{
			name: "with chain ID",
			args: map[string]interface{}{
				"chainId": "chain-123",
			},
			expected: &JobFilter{
				ChainID: "chain-123",
			},
		},
		{
			name: "with batch ID",
			args: map[string]interface{}{
				"batchId": "batch-456",
			},
			expected: &JobFilter{
				BatchID: "batch-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJobFilter(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}
