// Package jobs provides chain management and dependency resolution for the job manager.
//
// This file is responsible for:
// - Creating and managing job chains
// - Submitting/instantiating chains into jobs
// - Dependency resolution when jobs complete
// - Failure propagation to dependent jobs
//
// This file does NOT handle:
// - Individual job execution (see job_run.go)
// - Chain persistence (see store package)
//
// Invariants:
// - Chain definitions are validated before creation
// - Jobs with dependencies are not enqueued until dependencies are satisfied
// - Failed jobs propagate failure to all dependent jobs
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// CreateChain creates a new job chain definition.
// Validates the chain definition before creating.
func (m *Manager) CreateChain(ctx context.Context, name, description string, definition model.ChainDefinition) (*model.JobChain, error) {
	// Validate the chain definition
	if err := model.ValidateChainDefinition(definition); err != nil {
		return nil, apperrors.Validation(err.Error())
	}

	chain := &model.JobChain{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Definition:  definition,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.store.CreateChain(ctx, *chain); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create chain", err)
	}

	slog.Info("created chain", "chainID", chain.ID, "name", name, "nodes", len(definition.Nodes))
	return chain, nil
}

// GetChain retrieves a chain by ID.
func (m *Manager) GetChain(ctx context.Context, chainID string) (*model.JobChain, error) {
	chain, err := m.store.GetChain(ctx, chainID)
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// GetChainByName retrieves a chain by name.
func (m *Manager) GetChainByName(ctx context.Context, name string) (*model.JobChain, error) {
	chain, err := m.store.GetChainByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// ListChains returns all chain definitions.
func (m *Manager) ListChains(ctx context.Context) ([]model.JobChain, error) {
	return m.store.ListChains(ctx)
}

// DeleteChain removes a chain definition.
// Only chains with no active jobs can be deleted.
func (m *Manager) DeleteChain(ctx context.Context, chainID string) error {
	// Check if there are any jobs referencing this chain
	jobs, err := m.store.GetJobsByChain(ctx, chainID)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to check chain jobs", err)
	}

	if len(jobs) > 0 {
		return apperrors.Validation("cannot delete chain with existing jobs")
	}

	if err := m.store.DeleteChain(ctx, chainID); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete chain", err)
	}

	slog.Info("deleted chain", "chainID", chainID)
	return nil
}

// SubmitChain instantiates jobs from a chain definition.
// Returns the list of created jobs.
// Overrides can be provided to modify parameters for specific nodes.
func (m *Manager) SubmitChain(ctx context.Context, chainID string, overrides map[string]json.RawMessage) ([]model.Job, error) {
	chain, err := m.store.GetChain(ctx, chainID)
	if err != nil {
		return nil, err
	}

	// Create a map to track job IDs by node ID
	nodeJobIDs := make(map[string]string)
	jobs := make([]model.Job, 0, len(chain.Definition.Nodes))

	// First pass: create all jobs
	for _, node := range chain.Definition.Nodes {
		jobID := uuid.NewString()
		nodeJobIDs[node.ID] = jobID

		// Parse typed spec and apply overrides
		rawSpec := node.Spec
		if len(overrides[node.ID]) > 0 {
			rawSpec = overrides[node.ID]
		}
		spec, err := model.DecodeJobSpec(node.Kind, model.JobSpecVersion1, rawSpec)
		if err != nil {
			return nil, apperrors.Validation(fmt.Sprintf("invalid spec for node %s: %v", node.ID, err))
		}
		// Build depends_on list from edges
		var dependsOn []string
		for _, edge := range chain.Definition.Edges {
			if edge.To == node.ID {
				// This will be filled in second pass after all jobs are created
				dependsOn = append(dependsOn, edge.From)
			}
		}

		// Determine initial dependency status
		depStatus := model.DependencyStatusReady
		if len(dependsOn) > 0 {
			depStatus = model.DependencyStatusPending
		}

		job := model.Job{
			ID:               jobID,
			Kind:             node.Kind,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			SpecVersion:      model.JobSpecVersion1,
			Spec:             spec,
			ResultPath:       filepath.Join(m.DataDir, "jobs", jobID, "results.jsonl"),
			DependsOn:        make([]string, 0), // Will be filled after all jobs created
			DependencyStatus: depStatus,
			ChainID:          chainID,
		}
		jobs = append(jobs, job)
	}

	// Second pass: update depends_on with actual job IDs and persist
	for i, job := range jobs {
		node := chain.Definition.Nodes[i]

		// Convert node dependencies to job ID dependencies
		for _, edge := range chain.Definition.Edges {
			if edge.To == node.ID {
				if depJobID, ok := nodeJobIDs[edge.From]; ok {
					job.DependsOn = append(job.DependsOn, depJobID)
				}
			}
		}

		if err := m.store.Create(ctx, job); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create chain job", err)
		}

		// Enqueue jobs that are ready immediately
		if job.DependencyStatus == model.DependencyStatusReady {
			if err := m.Enqueue(job); err != nil {
				slog.Warn("failed to enqueue chain job", "jobID", job.ID, "error", err)
			}
		}
	}

	slog.Info("submitted chain", "chainID", chainID, "jobsCreated", len(jobs))
	return jobs, nil
}

// resolveDependencies is called when a job completes successfully.
// It checks if any pending jobs can now be started.
func (m *Manager) resolveDependencies(ctx context.Context, completedJob model.Job) {
	// Find jobs that depend on this job
	dependents, err := m.store.GetDependentJobs(ctx, completedJob.ID)
	if err != nil {
		slog.Error("failed to get dependent jobs", "jobID", completedJob.ID, "error", err)
		return
	}

	for _, depJob := range dependents {
		// Skip if already resolved
		if depJob.DependencyStatus != model.DependencyStatusPending {
			continue
		}

		// Check if all dependencies are now satisfied
		if m.checkDependencies(ctx, depJob) {
			if err := m.store.UpdateDependencyStatus(ctx, depJob.ID, model.DependencyStatusReady); err != nil {
				slog.Error("failed to update dependency status", "jobID", depJob.ID, "error", err)
				continue
			}

			// Enqueue if job is queued and now ready
			if depJob.Status == model.StatusQueued {
				depJob.DependencyStatus = model.DependencyStatusReady
				if err := m.Enqueue(depJob); err != nil {
					slog.Warn("failed to enqueue ready job", "jobID", depJob.ID, "error", err)
				} else {
					slog.Info("job ready and enqueued", "jobID", depJob.ID, "completedDep", completedJob.ID)
				}
			}
		}
	}
}

// checkDependencies verifies all dependencies are succeeded.
func (m *Manager) checkDependencies(ctx context.Context, job model.Job) bool {
	for _, depID := range job.DependsOn {
		depJob, err := m.store.Get(ctx, depID)
		if err != nil {
			slog.Error("failed to get dependency job", "jobID", job.ID, "depID", depID, "error", err)
			return false
		}
		if depJob.Status != model.StatusSucceeded {
			return false
		}
	}
	return true
}

// propagateFailure marks dependent jobs as failed when a dependency fails.
func (m *Manager) propagateFailure(ctx context.Context, failedJob model.Job) {
	// Find jobs that depend on this job
	dependents, err := m.store.GetDependentJobs(ctx, failedJob.ID)
	if err != nil {
		slog.Error("failed to get dependent jobs for failure propagation", "jobID", failedJob.ID, "error", err)
		return
	}

	for _, depJob := range dependents {
		// Skip if already in terminal state
		if depJob.Status.IsTerminal() {
			continue
		}

		// Mark as failed due to dependency failure
		if err := m.store.UpdateDependencyStatus(ctx, depJob.ID, model.DependencyStatusFailed); err != nil {
			slog.Error("failed to update dependency status", "jobID", depJob.ID, "error", err)
			continue
		}

		if depJob.Status == model.StatusQueued {
			errMsg := fmt.Sprintf("Dependency job %s failed", failedJob.ID)
			if err := m.store.UpdateStatus(ctx, depJob.ID, model.StatusFailed, errMsg); err != nil {
				slog.Error("failed to update job status", "jobID", depJob.ID, "error", err)
				continue
			}

			// Publish event for the failed job
			depJob.Status = model.StatusFailed
			depJob.Error = errMsg
			depJob.DependencyStatus = model.DependencyStatusFailed
			m.publishEvent(JobEvent{
				Type:       JobEventCompleted,
				Job:        depJob,
				PrevStatus: model.StatusQueued,
			})

			slog.Info("job failed due to dependency failure", "jobID", depJob.ID, "failedDep", failedJob.ID)

			// Recursively propagate to dependents of this job
			m.propagateFailure(ctx, depJob)
		}
	}
}
