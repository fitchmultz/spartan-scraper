// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL resolvers for job-related queries and mutations.
// It handles fetching, listing, creating, canceling, and deleting jobs.
//
// This file does NOT handle:
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
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
)

// ResolveJob resolves a single job by ID.
func (r *Resolver) ResolveJob(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("job id is required")
	}

	job, err := r.Store.Get(p.Context, id)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// ResolveJobs resolves a paginated list of jobs.
func (r *Resolver) ResolveJobs(p graphql.ResolveParams) (interface{}, error) {
	// Parse pagination args
	first, _ := p.Args["first"].(int)
	after, _ := p.Args["after"].(string)
	last, _ := p.Args["last"].(int)
	before, _ := p.Args["before"].(string)

	// Parse filter
	var filter *JobFilter
	if filterArg, ok := p.Args["filter"].(map[string]interface{}); ok {
		filter = parseJobFilter(filterArg)
	}

	// Default limit
	limit := 20
	if first > 0 {
		limit = first
	} else if last > 0 {
		limit = last
	}
	if limit > 1000 {
		limit = 1000
	}

	// Get total count
	totalCount, err := r.Store.CountJobs(p.Context, "")
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to count jobs", err)
	}

	// Calculate offset from cursor
	offset := 0
	if after != "" {
		offset = decodeCursor(after) + 1
	} else if before != "" {
		offset = decodeCursor(before) - limit
		if offset < 0 {
			offset = 0
		}
	}

	// Fetch jobs
	var jobList []model.Job
	if filter != nil && filter.Status != "" {
		opts := store.ListByStatusOptions{Limit: limit, Offset: offset}
		jobList, err = r.Store.ListByStatus(p.Context, filter.Status, opts)
	} else {
		opts := store.ListOptions{Limit: limit, Offset: offset}
		jobList, err = r.Store.ListOpts(p.Context, opts)
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list jobs", err)
	}

	// Build edges
	edges := make([]map[string]interface{}, len(jobList))
	for i, job := range jobList {
		edges[i] = map[string]interface{}{
			"node":   job,
			"cursor": encodeCursor(offset + i),
		}
	}

	// Build page info
	hasNextPage := len(jobList) == limit && offset+limit < totalCount
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

// ResolveCreateJob creates a new job.
func (r *Resolver) ResolveCreateJob(p graphql.ResolveParams) (interface{}, error) {
	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, apperrors.Validation("input is required")
	}

	kindVal, ok := input["kind"].(model.Kind)
	if !ok {
		return nil, apperrors.Validation("kind is required")
	}

	job := model.Job{
		ID:        uuid.New().String(),
		Kind:      kindVal,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    make(map[string]interface{}),
	}

	if params, ok := input["params"].(map[string]interface{}); ok {
		job.Params = params
	}

	if dependsOn, ok := input["dependsOn"].([]interface{}); ok {
		job.DependsOn = make([]string, len(dependsOn))
		for i, dep := range dependsOn {
			job.DependsOn[i] = dep.(string)
		}
	}

	if chainID, ok := input["chainId"].(string); ok {
		job.ChainID = chainID
	}

	if err := r.Store.Create(p.Context, job); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create job", err)
	}

	if err := r.Manager.Enqueue(job); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to enqueue job", err)
	}

	return job, nil
}

// ResolveCancelJob cancels a job.
func (r *Resolver) ResolveCancelJob(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("job id is required")
	}

	if err := r.Manager.CancelJob(p.Context, id); err != nil {
		return nil, err
	}

	return r.Store.Get(p.Context, id)
}

// ResolveDeleteJob deletes a job.
func (r *Resolver) ResolveDeleteJob(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, apperrors.Validation("job id is required")
	}

	force, _ := p.Args["force"].(bool)

	if force {
		if err := r.Store.DeleteWithArtifacts(p.Context, id); err != nil {
			return false, err
		}
	} else {
		if err := r.Store.Delete(p.Context, id); err != nil {
			return false, err
		}
	}

	return true, nil
}
