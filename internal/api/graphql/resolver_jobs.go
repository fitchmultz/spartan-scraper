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
	"fmt"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
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

	var kindVal model.Kind
	switch k := input["kind"].(type) {
	case model.Kind:
		kindVal = k
	case string:
		// Convert GraphQL enum string to model.Kind
		switch k {
		case "SCRAPE":
			kindVal = model.KindScrape
		case "CRAWL":
			kindVal = model.KindCrawl
		case "RESEARCH":
			kindVal = model.KindResearch
		default:
			return nil, apperrors.Validation("invalid kind: " + k)
		}
	default:
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

	// Set ResultPath before creating job
	job.ResultPath = filepath.Join(r.Manager.DataDir, "jobs", job.ID, "results.jsonl")

	// Validate inputs based on job kind
	if err := validateJobInput(kindVal, job.Params); err != nil {
		return nil, err
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

// validateJobInput validates job parameters based on job kind.
// Mirrors the validation logic in internal/jobs/spec.go JobSpec.Validate().
func validateJobInput(kind model.Kind, params map[string]interface{}) error {
	switch kind {
	case model.KindScrape:
		url, _ := params["url"].(string)
		if err := validate.ValidateURL(url); err != nil {
			return err
		}
		timeout := extractInt(params["timeout"])
		if err := validate.ValidateTimeout(timeout); err != nil {
			return err
		}
	case model.KindCrawl:
		url, _ := params["url"].(string)
		if err := validate.ValidateURL(url); err != nil {
			return err
		}
		maxDepth := extractInt(params["maxDepth"])
		if err := validate.ValidateMaxDepth(maxDepth); err != nil {
			return err
		}
		maxPages := extractInt(params["maxPages"])
		if err := validate.ValidateMaxPages(maxPages); err != nil {
			return err
		}
		timeout := extractInt(params["timeout"])
		if err := validate.ValidateTimeout(timeout); err != nil {
			return err
		}
		// sitemapOnly requires sitemapURL
		sitemapOnly, _ := params["sitemapOnly"].(bool)
		sitemapURL, _ := params["sitemapURL"].(string)
		if sitemapOnly && sitemapURL == "" {
			return apperrors.Validation("sitemapOnly requires sitemapURL to be set")
		}
		if sitemapURL != "" {
			if err := validate.ValidateURL(sitemapURL); err != nil {
				return fmt.Errorf("invalid sitemapURL: %w", err)
			}
		}
	case model.KindResearch:
		query, _ := params["query"].(string)
		if query == "" {
			return apperrors.Validation("query is required for research jobs")
		}
		// URLs validation - need to handle []interface{} from GraphQL
		if urlsRaw, ok := params["urls"].([]interface{}); ok {
			urls := make([]string, 0, len(urlsRaw))
			for i, u := range urlsRaw {
				s, ok := u.(string)
				if !ok {
					return apperrors.Validation(fmt.Sprintf("urls[%d] is not a string", i))
				}
				urls = append(urls, s)
			}
			if err := validate.ValidateURLs(urls); err != nil {
				return err
			}
		} else if params["urls"] != nil {
			return apperrors.Validation("urls must be an array")
		}
		maxDepth := extractInt(params["maxDepth"])
		if err := validate.ValidateMaxDepth(maxDepth); err != nil {
			return err
		}
		maxPages := extractInt(params["maxPages"])
		if err := validate.ValidateMaxPages(maxPages); err != nil {
			return err
		}
		timeout := extractInt(params["timeout"])
		if err := validate.ValidateTimeout(timeout); err != nil {
			return err
		}
	default:
		return apperrors.Validation(fmt.Sprintf("unknown job kind: %s", kind))
	}
	return nil
}

// extractInt extracts an int value from interface{}, handling float64 (GraphQL numbers).
func extractInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
