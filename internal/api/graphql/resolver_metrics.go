// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements the GraphQL resolver for metrics queries.
// It returns current system metrics snapshot based on store counts.
//
// This file does NOT handle:
// - Job operations (see resolver_jobs.go)
// - Chain operations (see resolver_chains.go)
// - Batch operations (see resolver_batches.go)
// - Real-time metrics collection (that happens in the jobs manager)
//
// Invariants:
// - Returns zero values for metrics that can't be calculated from store
// - Never fails - returns partial data on error
package graphql

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/graphql-go/graphql"
)

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
