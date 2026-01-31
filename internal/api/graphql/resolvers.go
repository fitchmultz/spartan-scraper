// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL resolvers for queries, mutations, and field resolution.
// It handles data fetching from stores and job manager.
//
// This file does NOT handle:
// - Schema definition (see schema.go)
// - Custom scalar serialization (see scalars.go)
// - Subscription handling (see subscriptions.go)
//
// Invariants:
// - All resolvers use apperrors for error handling
// - Context is passed through for cancellation
// - Pagination uses cursor-based approach
package graphql

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
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

// ResolveMetrics resolves the current metrics snapshot.
func (r *Resolver) ResolveMetrics(p graphql.ResolveParams) (interface{}, error) {
	// Get metrics from the manager's metrics callback
	// Since we don't have direct access to metrics collector here,
	// we'll return basic stats from the store

	queued, err := r.Store.CountJobs(p.Context, model.StatusQueued)
	if err != nil {
		queued = 0
	}
	running, err := r.Store.CountJobs(p.Context, model.StatusRunning)
	if err != nil {
		running = 0
	}
	succeeded, err := r.Store.CountJobs(p.Context, model.StatusSucceeded)
	if err != nil {
		succeeded = 0
	}
	failed, err := r.Store.CountJobs(p.Context, model.StatusFailed)
	if err != nil {
		failed = 0
	}

	total := queued + running + succeeded + failed

	return map[string]interface{}{
		"requestsPerSec":  0.0,
		"successRate":     0.0,
		"avgResponseTime": 0.0,
		"activeRequests":  running,
		"totalRequests":   total,
		"jobThroughput":   0.0,
		"avgJobDuration":  0.0,
		"timestamp":       time.Now(),
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

// Field resolvers for relationships

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

// Helper types and functions

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

// resolverContextKey is the key for resolver context values.
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
