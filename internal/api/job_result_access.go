// Package api provides shared helpers for job result access across API handlers.
//
// Purpose:
// - Centralize job-result readiness checks, path validation, and file opening.
//
// Responsibilities:
// - Normalize status-based "results unavailable" errors.
// - Validate recorded result paths before handlers read from disk.
// - Provide reusable helpers for opening result files safely.
//
// Scope:
// - API-layer result readers such as export, transform preview, and traffic replay.
//
// Usage:
// - Call requireJobResultFile when a handler needs precise file/empty-file messaging.
// - Call openJobResultFile when a helper needs a validated readable file handle.
//
// Invariants/Assumptions:
// - Result paths must stay within the store data directory.
// - Queued/running/failed/canceled jobs do not have readable results.
package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type jobResultFileMessages struct {
	MissingPath string
	MissingFile string
	EmptyFile   string
}

const maxJobResultLineBytes = 10 * 1024 * 1024

func jobResultsUnavailableError(job model.Job) error {
	switch job.Status {
	case model.StatusQueued:
		return apperrors.Validation("job is queued and has no results yet")
	case model.StatusRunning:
		return apperrors.Validation("job is still running and has no results yet")
	case model.StatusFailed:
		return apperrors.Validation("job failed and produced no results")
	case model.StatusCanceled:
		return apperrors.Validation("job was canceled and produced no results")
	default:
		return nil
	}
}

func (s *Server) validateJobResultPath(job model.Job, missingPathMessage string) error {
	if err := jobResultsUnavailableError(job); err != nil {
		return err
	}
	if job.ResultPath == "" {
		return apperrors.NotFound(missingPathMessage)
	}
	if err := model.ValidateResultPath(job.ID, job.ResultPath, s.store.DataDir()); err != nil {
		return err
	}
	return nil
}

func (s *Server) requireJobResultFile(job model.Job, messages jobResultFileMessages) error {
	if err := s.validateJobResultPath(job, messages.MissingPath); err != nil {
		return err
	}

	info, err := os.Stat(job.ResultPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apperrors.NotFound(messages.MissingFile)
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to stat job results file", err)
	}
	if info.Size() == 0 {
		return apperrors.NotFound(messages.EmptyFile)
	}
	return nil
}

func (s *Server) openJobResultFile(job model.Job, missingPathMessage string) (*os.File, error) {
	if err := s.validateJobResultPath(job, missingPathMessage); err != nil {
		return nil, err
	}

	file, err := os.Open(job.ResultPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperrors.NotFound("job results file is missing")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to open job results file", err)
	}
	return file, nil
}

func decodeJobResultItems(file *os.File, limit int) ([]any, error) {
	results := make([]any, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxJobResultLineBytes)

	for scanner.Scan() {
		if limit > 0 && len(results) >= limit {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var item any
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, apperrors.Wrap(
				apperrors.KindInternal,
				"failed to parse job result",
				err,
			)
		}
		results = append(results, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindInternal,
			"error reading job results file",
			err,
		)
	}

	return results, nil
}
