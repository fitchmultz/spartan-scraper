package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
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
	if err := row.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
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
func (s *Store) DeleteWithArtifacts(ctx context.Context, id string) error {
	if err := s.Delete(ctx, id); err != nil {
		return err
	}

	jobDir := filepath.Join(s.dataDir, "jobs", id)
	if err := os.RemoveAll(jobDir); err != nil {
		return err
	}

	return nil
}

// UpdateResultPath updates the result_path field for a job.
func (s *Store) UpdateResultPath(ctx context.Context, id string, resultPath string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET result_path = ? WHERE id = ?", resultPath, id)
	return err
}
