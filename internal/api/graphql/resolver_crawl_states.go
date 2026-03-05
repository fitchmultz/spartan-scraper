// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL resolvers for crawl state-related queries and mutations.
// It handles fetching, listing, and deleting crawl states.
//
// This file does NOT handle:
// - Job operations (see resolver_jobs.go)
// - Chain operations (see resolver_chains.go)
// - Batch operations (see resolver_batches.go)
// - Field relationship resolution (see resolver_fields.go)
// - Cursor encoding/decoding (see resolver_helpers.go)
//
// Invariants:
// - All resolvers use apperrors for error handling
// - Context is passed through for cancellation
// - Pagination uses cursor-based approach from resolver_helpers.go
package graphql

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/graphql-go/graphql"
)

// ResolveCrawlState resolves a crawl state by URL.
func (r *Resolver) ResolveCrawlState(p graphql.ResolveParams) (interface{}, error) {
	url, ok := p.Args["url"].(string)
	if !ok || url == "" {
		return nil, apperrors.Validation("url is required")
	}

	state, err := r.Store.GetCrawlState(p.Context, url)
	if err != nil {
		return nil, err
	}

	if state.URL == "" {
		return nil, nil
	}

	return state, nil
}

// ResolveCrawlStates resolves a paginated list of crawl states.
func (r *Resolver) ResolveCrawlStates(p graphql.ResolveParams) (interface{}, error) {
	// Parse pagination args
	first, _ := p.Args["first"].(int)
	after, _ := p.Args["after"].(string)

	// Default limit
	limit := 20
	if first > 0 {
		limit = first
	}
	if limit > 1000 {
		limit = 1000
	}

	// Calculate offset from cursor
	offset := 0
	if after != "" {
		offset = decodeCursor(after) + 1
	}

	opts := store.ListCrawlStatesOptions{Limit: limit, Offset: offset}
	states, err := r.Store.ListCrawlStates(p.Context, opts)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list crawl states", err)
	}

	totalCount, _ := r.Store.CountCrawlStates(p.Context)

	// Build edges
	edges := make([]map[string]interface{}, len(states))
	for i, state := range states {
		edges[i] = map[string]interface{}{
			"node":   state,
			"cursor": encodeCursor(offset + i),
		}
	}

	// Build page info
	hasNextPage := len(states) == limit && offset+limit < totalCount
	hasPreviousPage := offset > 0

	var startCursor, endCursor string
	if len(edges) > 0 {
		startCursor = edges[0]["cursor"].(string)
		endCursor = edges[len(edges)-1]["cursor"].(string)
	}

	return map[string]interface{}{
		"edges": edges,
		"pageInfo": map[string]interface{}{
			"hasNextPage":     hasNextPage,
			"hasPreviousPage": hasPreviousPage,
			"startCursor":     startCursor,
			"endCursor":       endCursor,
			"totalCount":      totalCount,
		},
	}, nil
}

// ResolveDeleteCrawlState deletes a crawl state.
func (r *Resolver) ResolveDeleteCrawlState(p graphql.ResolveParams) (interface{}, error) {
	url, ok := p.Args["url"].(string)
	if !ok || url == "" {
		return nil, apperrors.Validation("url is required")
	}

	if err := r.Store.DeleteCrawlState(p.Context, url); err != nil {
		return false, err
	}

	return true, nil
}
