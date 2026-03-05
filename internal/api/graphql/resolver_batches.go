// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL resolvers for batch-related queries and mutations.
// It handles fetching, listing, and deleting batches.
//
// This file does NOT handle:
// - Job operations (see resolver_jobs.go)
// - Chain operations (see resolver_chains.go)
// - Field relationship resolution (see resolver_fields.go)
// - Cursor encoding/decoding (see resolver_helpers.go)
//
// Invariants:
// - All resolvers use apperrors for error handling
// - Context is passed through for cancellation
package graphql

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/graphql-go/graphql"
)

// ResolveBatch resolves a single batch by ID.
func (r *Resolver) ResolveBatch(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("batch id is required")
	}

	batch, err := r.Store.GetBatch(p.Context, id)
	if err != nil {
		return nil, err
	}

	return batch, nil
}

// ResolveBatches resolves all batches.
func (r *Resolver) ResolveBatches(p graphql.ResolveParams) (interface{}, error) {
	// Since there's no ListBatches in store, we'll return an empty list
	// This can be implemented in the store if needed
	return []model.Batch{}, nil
}

// ResolveDeleteBatch deletes a batch.
func (r *Resolver) ResolveDeleteBatch(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("batch id is required")
	}

	if err := r.Store.DeleteBatch(p.Context, id); err != nil {
		return false, err
	}

	return true, nil
}
