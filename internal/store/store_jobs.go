// Package store provides persistent storage for jobs and crawl states using SQLite.
//
// Purpose:
// - Store and retrieve canonical typed-job records from SQLite.
//
// Responsibilities:
// - Job CRUD operations and status transitions.
// - Job listing with pagination and dependency-aware scans.
// - Artifact path updates and secure deletion.
//
// Scope:
// - Jobs only. Crawl states, chains, batches, and analytics live elsewhere.
//
// Usage:
// - Used by API, jobs runtime, retention, scheduler, and MCP layers.
//
// Invariants/Assumptions:
// - Persisted jobs store spec_version + spec_json, not params.
// - Timestamps are stored as RFC3339Nano strings.
// - Result path updates are validated against the local data dir.
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

func marshalSpec(job model.Job) (string, error) {
	if job.SpecVersion == 0 {
		job.SpecVersion = model.JobSpecVersion1
	}
	raw, err := model.MarshalJobSpec(job.Spec)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func encodeDependsOn(job model.Job) (string, model.DependencyStatus, error) {
	var dependsOnJSON string
	if len(job.DependsOn) > 0 {
		dependsOnBytes, err := json.Marshal(job.DependsOn)
		if err != nil {
			return "", "", apperrors.Wrap(apperrors.KindInternal, "failed to marshal depends_on", err)
		}
		dependsOnJSON = string(dependsOnBytes)
	}

	depStatus := job.DependencyStatus
	if depStatus == "" {
		if len(job.DependsOn) > 0 {
			depStatus = model.DependencyStatusPending
		} else {
			depStatus = model.DependencyStatusReady
		}
	}
	return dependsOnJSON, depStatus, nil
}

// Create stores a new job in the database.
func (s *Store) Create(ctx context.Context, job model.Job) error {
	if job.SpecVersion == 0 {
		job.SpecVersion = model.JobSpecVersion1
	}
	specJSON, err := marshalSpec(job)
	if err != nil {
		return err
	}
	dependsOnJSON, depStatus, err := encodeDependsOn(job)
	if err != nil {
		return err
	}

	var startedAt any
	if job.StartedAt != nil {
		startedAt = job.StartedAt.Format(time.RFC3339Nano)
	}
	var finishedAt any
	if job.FinishedAt != nil {
		finishedAt = job.FinishedAt.Format(time.RFC3339Nano)
	}

	_, err = s.insertJobStmt.ExecContext(
		ctx,
		job.ID,
		job.Kind,
		job.Status,
		job.CreatedAt.Format(time.RFC3339Nano),
		job.UpdatedAt.Format(time.RFC3339Nano),
		job.SpecVersion,
		specJSON,
		job.ResultPath,
		job.Error,
		dependsOnJSON,
		string(depStatus),
		job.ChainID,
		startedAt,
		finishedAt,
		job.SelectedEngine,
	)
	return err
}

// UpdateStatus changes a job's status and error message.
func (s *Store) UpdateStatus(ctx context.Context, id string, status model.Status, errMsg string) error {
	now := time.Now()
	var startedAt any
	var finishedAt any
	if status == model.StatusRunning {
		startedAt = now.Format(time.RFC3339Nano)
	}
	if status.IsTerminal() {
		finishedAt = now.Format(time.RFC3339Nano)
	}
	_, err := s.updateJobStatusStmt.ExecContext(
		ctx,
		status,
		now.Format(time.RFC3339Nano),
		errMsg,
		startedAt,
		finishedAt,
		id,
	)
	return err
}

func (s *Store) scanJob(rowScanner interface{ Scan(dest ...any) error }, withDeps bool) (model.Job, error) {
	var job model.Job
	var createdAt, updatedAt string
	var specJSON string
	var dependsOnJSON sql.NullString
	var depStatusStr sql.NullString
	var startedAt sql.NullString
	var finishedAt sql.NullString

	dest := []any{
		&job.ID,
		&job.Kind,
		&job.Status,
		&createdAt,
		&updatedAt,
		&job.SpecVersion,
		&specJSON,
		&job.ResultPath,
		&job.Error,
	}
	if withDeps {
		dest = append(dest, &dependsOnJSON, &depStatusStr, &job.ChainID, &startedAt, &finishedAt, &job.SelectedEngine)
	} else {
		dest = append(dest, &startedAt, &finishedAt, &job.SelectedEngine)
	}

	if err := rowScanner.Scan(dest...); err != nil {
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
	if startedAt.Valid && startedAt.String != "" {
		parsed, parseErr := time.Parse(time.RFC3339Nano, startedAt.String)
		if parseErr != nil {
			return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job started_at", parseErr)
		}
		job.StartedAt = &parsed
	}
	if finishedAt.Valid && finishedAt.String != "" {
		parsed, parseErr := time.Parse(time.RFC3339Nano, finishedAt.String)
		if parseErr != nil {
			return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job finished_at", parseErr)
		}
		job.FinishedAt = &parsed
	}
	if specJSON != "" {
		job.Spec, err = model.DecodeJobSpec(job.Kind, job.SpecVersion, []byte(specJSON))
		if err != nil {
			var generic map[string]interface{}
			if genericErr := json.Unmarshal([]byte(specJSON), &generic); genericErr == nil {
				job.Spec = generic
			} else {
				return model.Job{}, err
			}
		}
	}
	if withDeps {
		if dependsOnJSON.Valid && dependsOnJSON.String != "" {
			if err := json.Unmarshal([]byte(dependsOnJSON.String), &job.DependsOn); err != nil {
				return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal depends_on", err)
			}
		}
		if depStatusStr.Valid && depStatusStr.String != "" {
			job.DependencyStatus = model.DependencyStatus(depStatusStr.String)
		}
	}
	return job, nil
}

// Get retrieves a job by ID.
func (s *Store) Get(ctx context.Context, id string) (model.Job, error) {
	row := s.getJobStmt.QueryRowContext(ctx, id)
	job, err := s.scanJob(row, true)
	if err != nil {
		return model.Job{}, wrapScanError(err, "job not found", "failed to get job")
	}
	return job, nil
}

func (s *Store) listJobs(ctx context.Context, query string, args ...any) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Job
	for rows.Next() {
		job, err := s.scanJob(rows, false)
		if err != nil {
			return nil, err
		}
		results = append(results, job)
	}
	return results, rows.Err()
}

// List returns all jobs using default options.
func (s *Store) List(ctx context.Context) ([]model.Job, error) {
	return s.ListOpts(ctx, ListOptions{})
}

// ListOpts returns jobs with pagination options.
func (s *Store) ListOpts(ctx context.Context, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	return s.listJobs(ctx, `select id, kind, status, created_at, updated_at, spec_version, spec_json, result_path, error, started_at, finished_at, selected_engine from jobs order by created_at desc limit ? offset ?`, opts.Limit, opts.Offset)
}

// ListByStatus returns all jobs with the given status.
func (s *Store) ListByStatus(ctx context.Context, status model.Status, opts ListByStatusOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	return s.listJobs(ctx, `select id, kind, status, created_at, updated_at, spec_version, spec_json, result_path, error, started_at, finished_at, selected_engine from jobs where status = ? order by created_at desc limit ? offset ?`, status, opts.Limit, opts.Offset)
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
func (s *Store) DeleteWithArtifacts(ctx context.Context, id string) error {
	_, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Delete(ctx, id); err != nil {
		return err
	}

	jobDir := filepath.Join(s.dataDir, "jobs", id)
	cleanPath := filepath.Clean(jobDir)
	baseDir := filepath.Clean(filepath.Join(s.dataDir, "jobs"))
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) && cleanPath != baseDir {
		return apperrors.Permission("invalid job id: path traversal detected")
	}
	return os.RemoveAll(cleanPath)
}

// UpdateResultPath updates the result_path field for a job.
func (s *Store) UpdateResultPath(ctx context.Context, id string, resultPath string) error {
	if err := model.ValidateResultPath(id, resultPath, s.dataDir); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET result_path = ? WHERE id = ?", resultPath, id)
	return err
}

// UpdateSelectedEngine updates the selected execution engine for a job.
func (s *Store) UpdateSelectedEngine(ctx context.Context, id, selectedEngine string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET selected_engine = ? WHERE id = ?", selectedEngine, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update selected engine", err)
	}
	return nil
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
func (s *Store) GetDependentJobs(ctx context.Context, jobID string) ([]model.Job, error) {
	rows, err := s.stmtGetDependentJobs.QueryContext(ctx, jobID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query dependent jobs", err)
	}
	defer rows.Close()
	return s.scanJobsWithDependencies(rows)
}

// GetJobsByChain returns all jobs belonging to a chain.
func (s *Store) GetJobsByChain(ctx context.Context, chainID string) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, status, created_at, updated_at, spec_version, spec_json, result_path, error, depends_on, dependency_status, chain_id, started_at, finished_at, selected_engine
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
		job, err := s.scanJob(rows, true)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan job row", err)
		}
		results = append(results, job)
	}
	return results, rows.Err()
}
