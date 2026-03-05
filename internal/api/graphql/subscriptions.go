// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements GraphQL subscriptions for real-time updates.
// It leverages the existing WebSocket hub infrastructure for event distribution.
//
// This file does NOT handle:
// - Schema definition (see schema.go)
// - Query/mutation resolvers (see resolvers.go)
// - Custom scalar serialization (see scalars.go)
//
// Invariants:
// - Subscriptions are filtered by jobId if specified
// - Events are delivered via the existing Hub pattern
// - Context cancellation stops the subscription
package graphql

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/graphql-go/graphql"
)

// SubscriptionManager handles GraphQL subscription resolution.
type SubscriptionManager struct {
	jobEvents chan jobs.JobEvent
	manager   *jobs.Manager
}

// NewSubscriptionManager creates a new subscription manager.
func NewSubscriptionManager(manager *jobs.Manager) *SubscriptionManager {
	sm := &SubscriptionManager{
		jobEvents: make(chan jobs.JobEvent, 256),
		manager:   manager,
	}

	// Subscribe to job manager events
	manager.SubscribeToEvents(sm.jobEvents)

	return sm
}

// Stop unsubscribes from job events.
func (sm *SubscriptionManager) Stop() {
	sm.manager.UnsubscribeFromEvents(sm.jobEvents)
	close(sm.jobEvents)
}

// SubscribeJobStatusChanged subscribes to job status changes.
func (sm *SubscriptionManager) SubscribeJobStatusChanged(p graphql.ResolveParams) (interface{}, error) {
	// Return a channel that will receive events
	jobID, hasJobID := p.Args["jobId"].(string)

	eventCh := make(chan interface{})

	go func() {
		defer close(eventCh)

		for event := range sm.jobEvents {
			// Filter by job ID if specified
			if hasJobID && event.Job.ID != jobID {
				continue
			}

			// Only send status change events
			if event.Type == jobs.JobEventStatus || event.Type == jobs.JobEventCompleted {
				select {
				case eventCh <- event.Job:
				case <-p.Context.Done():
					return
				}
			}
		}
	}()

	return eventCh, nil
}

// ResolveJobStatusChanged resolves the job for status change subscription.
func (sm *SubscriptionManager) ResolveJobStatusChanged(p graphql.ResolveParams) (interface{}, error) {
	return p.Source, nil
}

// SubscribeJobCompleted subscribes to job completions.
func (sm *SubscriptionManager) SubscribeJobCompleted(p graphql.ResolveParams) (interface{}, error) {
	jobID, hasJobID := p.Args["jobId"].(string)

	eventCh := make(chan interface{})

	go func() {
		defer close(eventCh)

		for event := range sm.jobEvents {
			// Filter by job ID if specified
			if hasJobID && event.Job.ID != jobID {
				continue
			}

			// Only send completed events
			if event.Type == jobs.JobEventCompleted {
				select {
				case eventCh <- event.Job:
				case <-p.Context.Done():
					return
				}
			}
		}
	}()

	return eventCh, nil
}

// ResolveJobCompleted resolves the job for completion subscription.
func (sm *SubscriptionManager) ResolveJobCompleted(p graphql.ResolveParams) (interface{}, error) {
	return p.Source, nil
}

// SubscribeMetricsUpdated subscribes to metrics updates.
func (sm *SubscriptionManager) SubscribeMetricsUpdated(p graphql.ResolveParams) (interface{}, error) {
	// For metrics, we'll create a ticker-based subscription
	// In a real implementation, this would subscribe to the metrics collector
	metricsCh := make(chan interface{})

	go func() {
		defer close(metricsCh)

		// Simple ticker for demo - in production this would use proper metrics events
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send current metrics snapshot
				metrics := map[string]interface{}{
					"requestsPerSec":  0.0,
					"successRate":     100.0,
					"avgResponseTime": 0.0,
					"activeRequests":  0,
					"totalRequests":   0,
					"jobThroughput":   0.0,
					"avgJobDuration":  0.0,
					"timestamp":       time.Now(),
				}
				select {
				case metricsCh <- metrics:
				case <-p.Context.Done():
					return
				}
			case <-p.Context.Done():
				return
			}
		}
	}()

	return metricsCh, nil
}

// ResolveMetricsUpdated resolves the metrics for subscription.
func (sm *SubscriptionManager) ResolveMetricsUpdated(p graphql.ResolveParams) (interface{}, error) {
	return p.Source, nil
}
