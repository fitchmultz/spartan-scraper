// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements field resolvers for GraphQL type relationships.
// It handles resolving connections between types (jobs, chains, batches, crawl states).
//
// This file does NOT handle:
// - Root query/mutation resolution (see resolver_*.go files)
// - Schema definition (see schema.go)
// - Cursor encoding/decoding (see resolver_helpers.go)
//
// Invariants:
// - All field resolvers use resolverContext from context for store access
// - Returns nil gracefully when source type doesn't match
// - Handles missing context gracefully (returns nil)
package graphql

import (
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/graphql-go/graphql"
)

// resolveJobDependsOn resolves the jobs this job depends on.
func resolveJobDependsOn(p graphql.ResolveParams) (interface{}, error) {
	job, ok := p.Source.(model.Job)
	if !ok {
		return nil, nil
	}

	if len(job.DependsOn) == 0 {
		return []model.Job{}, nil
	}

	// We need access to the store to resolve these
	// This is handled by the resolver context
	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	deps := make([]model.Job, 0, len(job.DependsOn))
	for _, depID := range job.DependsOn {
		dep, err := rctx.store.Get(p.Context, depID)
		if err != nil {
			continue
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

// resolveJobDependentJobs resolves jobs that depend on this job.
func resolveJobDependentJobs(p graphql.ResolveParams) (interface{}, error) {
	job, ok := p.Source.(model.Job)
	if !ok {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	deps, err := rctx.store.GetDependentJobs(p.Context, job.ID)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

// resolveJobChain resolves the chain this job belongs to.
func resolveJobChain(p graphql.ResolveParams) (interface{}, error) {
	job, ok := p.Source.(model.Job)
	if !ok || job.ChainID == "" {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	chain, err := rctx.store.GetChain(p.Context, job.ChainID)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// resolveJobBatch resolves the batch this job belongs to.
func resolveJobBatch(p graphql.ResolveParams) (interface{}, error) {
	// Job model doesn't have a direct batch ID, so we need to look it up
	// This would require a store method to find batch by job ID
	return nil, nil
}

// resolveChainJobs resolves jobs belonging to a chain.
func resolveChainJobs(p graphql.ResolveParams) (interface{}, error) {
	chain, ok := p.Source.(model.JobChain)
	if !ok {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	jobs, err := rctx.store.GetJobsByChain(p.Context, chain.ID)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// resolveBatchJobs resolves jobs belonging to a batch.
func resolveBatchJobs(p graphql.ResolveParams) (interface{}, error) {
	batch, ok := p.Source.(model.Batch)
	if !ok {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	opts := store.ListOptions{Limit: batch.JobCount, Offset: 0}
	jobs, err := rctx.store.ListJobsByBatch(p.Context, batch.ID, opts)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// resolveBatchStats resolves statistics for a batch.
func resolveBatchStats(p graphql.ResolveParams) (interface{}, error) {
	batch, ok := p.Source.(model.Batch)
	if !ok {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	stats, err := rctx.store.CountJobsByBatchAndStatus(p.Context, batch.ID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// resolveCrawlStateJob resolves the job associated with a crawl state.
func resolveCrawlStateJob(p graphql.ResolveParams) (interface{}, error) {
	state, ok := p.Source.(model.CrawlState)
	if !ok || state.JobID == "" {
		return nil, nil
	}

	rctx, ok := p.Context.Value(resolverContextKey).(*resolverContext)
	if !ok {
		return nil, nil
	}

	job, err := rctx.store.Get(p.Context, state.JobID)
	if err != nil {
		return nil, err
	}

	return job, nil
}
