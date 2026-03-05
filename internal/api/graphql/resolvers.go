// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements the main Resolver type and constructor.
// Resolver methods are split across domain-specific files:
// - resolver_jobs.go: Job queries and mutations
// - resolver_chains.go: Chain queries and mutations
// - resolver_batches.go: Batch queries and mutations
// - resolver_crawl_states.go: Crawl state queries and mutations
// - resolver_metrics.go: Metrics query
// - resolver_fields.go: Field relationship resolvers
// - resolver_helpers.go: Shared helpers (cursors, filters, context)
//
// This file does NOT handle:
// - Query/mutation resolution (see resolver_*.go files)
// - Field relationship resolution (see resolver_fields.go)
// - Schema definition (see schema.go)
// - Custom scalar serialization (see scalars.go)
//
// Invariants:
// - Resolver is immutable after creation
// - Store and Manager are required dependencies
package graphql

import (
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// Resolver handles GraphQL query and mutation resolution.
type Resolver struct {
	Store   *store.Store
	Manager *jobs.Manager
}

// NewResolver creates a new GraphQL resolver.
func NewResolver(store *store.Store, manager *jobs.Manager) *Resolver {
	return &Resolver{
		Store:   store,
		Manager: manager,
	}
}
