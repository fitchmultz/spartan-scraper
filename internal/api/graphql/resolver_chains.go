// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL resolvers for chain-related queries and mutations.
// It handles fetching, listing, creating, and deleting job chains.
//
// This file does NOT handle:
// - Job operations (see resolver_jobs.go)
// - Batch operations (see resolver_batches.go)
// - Field relationship resolution (see resolver_fields.go)
// - Cursor encoding/decoding (see resolver_helpers.go)
//
// Invariants:
// - All resolvers use apperrors for error handling
// - Context is passed through for cancellation
// - Chain definitions are validated before creation
package graphql

import (
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
)

// ResolveChain resolves a single chain by ID.
func (r *Resolver) ResolveChain(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("chain id is required")
	}

	chain, err := r.Store.GetChain(p.Context, id)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// ResolveChains resolves all chains.
func (r *Resolver) ResolveChains(p graphql.ResolveParams) (interface{}, error) {
	chains, err := r.Store.ListChains(p.Context)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list chains", err)
	}

	return chains, nil
}

// ResolveCreateChain creates a new chain.
func (r *Resolver) ResolveCreateChain(p graphql.ResolveParams) (interface{}, error) {
	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, apperrors.Validation("input is required")
	}

	name, ok := input["name"].(string)
	if !ok || name == "" {
		return nil, apperrors.Validation("name is required")
	}

	chain := model.JobChain{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if desc, ok := input["description"].(string); ok {
		chain.Description = desc
	}

	if def, ok := input["definition"].(map[string]interface{}); ok {
		defJSON, err := json.Marshal(def)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "invalid definition", err)
		}
		if err := json.Unmarshal(defJSON, &chain.Definition); err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "invalid definition structure", err)
		}
	}

	if err := model.ValidateChainDefinition(chain.Definition); err != nil {
		return nil, err
	}

	if err := r.Store.CreateChain(p.Context, chain); err != nil {
		return nil, err
	}

	return chain, nil
}

// ResolveDeleteChain deletes a chain.
func (r *Resolver) ResolveDeleteChain(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("chain id is required")
	}

	if err := r.Store.DeleteChain(p.Context, id); err != nil {
		return false, err
	}

	return true, nil
}
