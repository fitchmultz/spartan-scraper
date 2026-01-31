// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements shared helper types and functions for GraphQL resolvers.
// It provides cursor encoding/decoding, filtering, and context management utilities.
//
// This file does NOT handle:
// - Direct query/mutation resolution (see resolver_*.go files)
// - Field relationship resolution (see resolver_fields.go)
// - Schema definition (see schema.go)
//
// Invariants:
// - All cursor encoding uses base64 with "cursor:offset" format
// - Context values are type-safe via resolverContext struct
// - Filter parsing handles nil/empty inputs gracefully
package graphql

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// JobFilter represents filter options for jobs.
type JobFilter struct {
	Status  model.Status
	Kind    model.Kind
	ChainID string
	BatchID string
}

func parseJobFilter(args map[string]interface{}) *JobFilter {
	filter := &JobFilter{}

	if status, ok := args["status"].(model.Status); ok {
		filter.Status = status
	}
	if kind, ok := args["kind"].(model.Kind); ok {
		filter.Kind = kind
	}
	if chainID, ok := args["chainId"].(string); ok {
		filter.ChainID = chainID
	}
	if batchID, ok := args["batchId"].(string); ok {
		filter.BatchID = batchID
	}

	return filter
}

// encodeCursor encodes an offset into a cursor string.
func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor:%d", offset)))
}

// decodeCursor decodes a cursor string into an offset.
func decodeCursor(cursor string) int {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 2 || parts[0] != "cursor" {
		return 0
	}

	offset, _ := strconv.Atoi(parts[1])
	return offset
}

// contextKey is the key for resolver context values.
type contextKey string

const resolverContextKey contextKey = "resolverContext"

// resolverContext holds dependencies for field resolvers.
type resolverContext struct {
	store   *store.Store
	manager *jobs.Manager
}

// WithResolverContext adds resolver dependencies to context.
func WithResolverContext(ctx context.Context, store *store.Store, manager *jobs.Manager) context.Context {
	return context.WithValue(ctx, resolverContextKey, &resolverContext{
		store:   store,
		manager: manager,
	})
}
