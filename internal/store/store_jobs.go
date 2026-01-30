// Package store provides persistent storage for jobs and crawl states using SQLite.
//
// This file is responsible for:
// - Job CRUD operations (Create, Get, UpdateStatus, Delete)
// - Job listing with pagination and status filtering
// - Job artifact deletion with path traversal protection
// - Result path updates for completed jobs
//
// This file does NOT handle:
// - Job execution (jobs package handles this)
// - Crawl state operations (store_crawl_states.go handles this)
// - Store initialization (store_init.go handles this)
//
// Invariants:
// - Uses prepared statements for all database operations
// - Job params are marshaled to JSON for storage
// - Timestamps are stored as RFC3339Nano strings
// - DeleteWithArtifacts validates paths to prevent traversal attacks
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// Create stores a new job in the database.
// Job params are marshaled to JSON for storage.
func (s *Store) Create(ctx context.Context, job model.Job) error {
	params, err := json.Marshal(job.Params)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal job params", err)
	}

	// Marshal depends_on as JSON array
	var dependsOnJSON string
	if len(job.DependsOn) > 0 {
		dependsOnBytes, err := json.Marshal(job.DependsOn)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to marshal depends_on", err)
		}
		dependsOnJSON = string(dependsOnBytes)
	}

	// Set default dependency status
	depStatus := job.DependencyStatus
	if depStatus == "" {
		if len(job.DependsOn) > 0 {
			depStatus = model.DependencyStatusPending
		} else {
			depStatus = model.DependencyStatusReady
		}
	}

	_, err = s.insertJobStmt.ExecContext(
		ctx,
		job.ID,
		job.Kind,
		job.Status,
		job.CreatedAt.Format(time.RFC3339Nano),
		job.UpdatedAt.Format(time.RFC3339Nano),
		string(params),
		job.ResultPath,
		job.Error,
		dependsOnJSON,
		string(depStatus),
		job.ChainID,
	)
	return err
}

// UpdateStatus changes a job's status and error message.
// Updates the updated_at timestamp automatically.
func (s *Store) UpdateStatus(ctx context.Context, id string, status model.Status, errMsg string) error {
	_, err := s.updateJobStatusStmt.ExecContext(
		ctx,
		status,
		time.Now().Format(time.RFC3339Nano),
		errMsg,
		id,
	)
	return err
}

// Get retrieves a job by ID.
// Returns apperrors.NotFound if the job does not exist.
func (s *Store) Get(ctx context.Context, id string) (model.Job, error) {
	row := s.getJobStmt.QueryRowContext(ctx, id)
	var job model.Job
	var createdAt, updatedAt string
	var params string
	var dependsOnJSON string
	var depStatusStr string

	if err := row.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error, &dependsOnJSON, &depStatusStr, &job.ChainID); err != nil {
		if err == sql.ErrNoRows || err.Error() == "sql: no rows in result set" {
			return model.Job{}, apperrors.NotFound("job not found")
		}
		return model.Job{}, err
	}

	var err error
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", err)
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", err)
	}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
			return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
		}
	}
	if dependsOnJSON != "" {
		if err := json.Unmarshal([]byte(dependsOnJSON), &job.DependsOn); err != nil {
			return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal depends_on", err)
		}
	}
	if depStatusStr != "" {
		job.DependencyStatus = model.DependencyStatus(depStatusStr)
	}

	return job, nil
}

// List returns all jobs using default options.
func (s *Store) List(ctx context.Context) ([]model.Job, error) {
	return s.ListOpts(ctx, ListOptions{})
}

// ListOpts returns jobs with pagination options.
// Results are ordered by created_at DESC.
func (s *Store) ListOpts(ctx context.Context, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx, `select id, kind, status, created_at, updated_at, params, result_path, error from jobs order by created_at desc limit ? offset ?`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.Job{}
	for rows.Next() {
		var job model.Job
		var createdAt, updatedAt string
		var params string
		if err := rows.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
			return nil, err
		}
		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", parseErr)
		}
		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
			}
		}
		results = append(results, job)
	}
	return results, rows.Err()
}

// ListByStatus returns all jobs with the given status, ordered by created_at DESC.
// If no options are provided, it uses safe defaults (limit 100, offset 0).
func (s *Store) ListByStatus(ctx context.Context, status model.Status, opts ListByStatusOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`select id, kind, status, created_at, updated_at, params, result_path, error
		 from jobs where status = ? order by created_at desc limit ? offset ?`, status, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.Job{}
	for rows.Next() {
		var job model.Job
		var createdAt, updatedAt string
		var params string
		if err := rows.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
			return nil, err
		}
		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", parseErr)
		}
		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
			}
		}
		results = append(results, job)
	}
	return results, rows.Err()
}

// CountJobs returns the total number of jobs, optionally filtered by status.
func (s *Store) CountJobs(ctx context.Context, status model.Status) (int, error) {
	var query string
	var args []interface{}
	if status != "" {
		query = "select count(*) from jobs where status = ?"
		args = append(args, status)
	} else {
		query = "select count(*) from jobs"
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// Delete permanently removes a job from the store.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
	return err
}

// DeleteWithArtifacts permanently removes a job from store and deletes its result file and directory.
// This is used for force delete operations.
// The function verifies the job exists before deletion and ensures the delete path stays
// within the data directory to prevent path traversal attacks.
func (s *Store) DeleteWithArtifacts(ctx context.Context, id string) error {
	// First verify job exists before any deletion
	_, err := s.Get(ctx, id)
	if err != nil {
		return err // Returns NotFound if job doesn't exist
	}

	if err := s.Delete(ctx, id); err != nil {
		return err
	}

	// Secure path construction with traversal check
	jobDir := filepath.Join(s.dataDir, "jobs", id)
	cleanPath := filepath.Clean(jobDir)
	baseDir := filepath.Clean(filepath.Join(s.dataDir, "jobs"))

	// Ensure the cleaned path is within the jobs directory
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) && cleanPath != baseDir {
		return apperrors.Permission("invalid job id: path traversal detected")
	}

	if err := os.RemoveAll(cleanPath); err != nil {
		return err
	}

	return nil
}

// UpdateResultPath updates the result_path field for a job.
func (s *Store) UpdateResultPath(ctx context.Context, id string, resultPath string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET result_path = ? WHERE id = ?", resultPath, id)
	return err
}

// GetJobsByDependencyStatus returns jobs with the specified dependency status.
func (s *Store) GetJobsByDependencyStatus(ctx context.Context, status model.DependencyStatus) ([]model.Job, error) {
	rows, err := s.stmtGetJobsByDependencyStatus.QueryContext(ctx, string(status))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query jobs by dependency status", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// UpdateDependencyStatus updates only the dependency status field for a job.
func (s *Store) UpdateDependencyStatus(ctx context.Context, jobID string, status model.DependencyStatus) error {
	_, err := s.stmtUpdateDependencyStatus.ExecContext(ctx, string(status), jobID)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update dependency status", err)
	}
	return nil
}

// GetDependentJobs returns jobs that depend on the given job ID.
// Uses a LIKE query to find jobs where depends_on contains the job ID.
func (s *Store) GetDependentJobs(ctx context.Context, jobID string) ([]model.Job, error) {
	// Use JSON array pattern matching: %"jobID"%
	pattern := `%"` + jobID + `"%`
	rows, err := s.stmtGetDependentJobs.QueryContext(ctx, pattern)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query dependent jobs", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// GetJobsByChain returns all jobs belonging to a chain.
func (s *Store) GetJobsByChain(ctx context.Context, chainID string) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id
		 FROM jobs WHERE chain_id = ? ORDER BY created_at ASC`,
		chainID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query jobs by chain", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// scanJobsWithDependencies scans job rows including dependency fields.
func (s *Store) scanJobsWithDependencies(rows *sql.Rows) ([]model.Job, error) {
	var results []model.Job
	for rows.Next() {
		var job model.Job
		var createdAt, updatedAt string
		var params string
		var dependsOnJSON string
		var depStatusStr string

		if err := rows.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error, &dependsOnJSON, &depStatusStr, &job.ChainID); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan job row", err)
		}

		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", parseErr)
		}

		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
			}
		}
		if dependsOnJSON != "" {
			if err := json.Unmarshal([]byte(dependsOnJSON), &job.DependsOn); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal depends_on", err)
			}
		}
		if depStatusStr != "" {
			job.DependencyStatus = model.DependencyStatus(depStatusStr)
		}

		results = append(results, job)
	}
	return results, rows.Err()
}
