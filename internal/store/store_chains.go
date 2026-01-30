// Package store provides persistent storage for job chains using SQLite.
//
// This file is responsible for:
// - JobChain CRUD operations (CreateChain, GetChain, UpdateChain, DeleteChain)
// - Chain lookup by name
// - Listing all chains
//
// This file does NOT handle:
// - Chain execution or job management (jobs package handles this)
// - Chain validation (model package handles this)
//
// Invariants:
// - Chain names must be unique
// - Chain definitions are stored as JSON
// - All timestamps are stored as RFC3339Nano strings
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// CreateChain creates a new job chain record.
func (s *Store) CreateChain(ctx context.Context, chain model.JobChain) error {
	definitionJSON, err := json.Marshal(chain.Definition)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal chain definition", err)
	}

	_, err = s.stmtCreateChain.ExecContext(
		ctx,
		chain.ID,
		chain.Name,
		chain.Description,
		string(definitionJSON),
		chain.CreatedAt.Format(time.RFC3339Nano),
		chain.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert chain", err)
	}

	return nil
}

// GetChain retrieves a chain by ID.
// Returns apperrors.NotFound if the chain does not exist.
func (s *Store) GetChain(ctx context.Context, id string) (model.JobChain, error) {
	row := s.stmtGetChain.QueryRowContext(ctx, id)

	var chain model.JobChain
	var createdAt, updatedAt string
	var definitionJSON string

	err := row.Scan(&chain.ID, &chain.Name, &chain.Description, &definitionJSON, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.JobChain{}, apperrors.NotFound("chain not found")
		}
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to get chain", err)
	}

	chain.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain created_at", err)
	}

	chain.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain updated_at", err)
	}

	if definitionJSON != "" {
		if err := json.Unmarshal([]byte(definitionJSON), &chain.Definition); err != nil {
			return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal chain definition", err)
		}
	}

	return chain, nil
}

// GetChainByName retrieves a chain by name.
// Returns apperrors.NotFound if the chain does not exist.
func (s *Store) GetChainByName(ctx context.Context, name string) (model.JobChain, error) {
	row := s.stmtGetChainByName.QueryRowContext(ctx, name)

	var chain model.JobChain
	var createdAt, updatedAt string
	var definitionJSON string

	err := row.Scan(&chain.ID, &chain.Name, &chain.Description, &definitionJSON, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.JobChain{}, apperrors.NotFound("chain not found")
		}
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to get chain by name", err)
	}

	chain.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain created_at", err)
	}

	chain.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain updated_at", err)
	}

	if definitionJSON != "" {
		if err := json.Unmarshal([]byte(definitionJSON), &chain.Definition); err != nil {
			return model.JobChain{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal chain definition", err)
		}
	}

	return chain, nil
}

// UpdateChain updates an existing chain record.
func (s *Store) UpdateChain(ctx context.Context, chain model.JobChain) error {
	definitionJSON, err := json.Marshal(chain.Definition)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal chain definition", err)
	}

	_, err = s.stmtUpdateChain.ExecContext(
		ctx,
		chain.Name,
		chain.Description,
		string(definitionJSON),
		time.Now().Format(time.RFC3339Nano),
		chain.ID,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update chain", err)
	}

	return nil
}

// DeleteChain removes a chain by ID.
func (s *Store) DeleteChain(ctx context.Context, id string) error {
	_, err := s.stmtDeleteChain.ExecContext(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete chain", err)
	}
	return nil
}

// ListChains returns all chains ordered by creation date (newest first).
func (s *Store) ListChains(ctx context.Context) ([]model.JobChain, error) {
	rows, err := s.stmtListChains.QueryContext(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list chains", err)
	}
	defer rows.Close()

	var chains []model.JobChain
	for rows.Next() {
		var chain model.JobChain
		var createdAt, updatedAt string
		var definitionJSON string

		if err := rows.Scan(&chain.ID, &chain.Name, &chain.Description, &definitionJSON, &createdAt, &updatedAt); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan chain row", err)
		}

		var parseErr error
		chain.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain created_at", parseErr)
		}

		chain.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse chain updated_at", parseErr)
		}

		if definitionJSON != "" {
			if err := json.Unmarshal([]byte(definitionJSON), &chain.Definition); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal chain definition", err)
			}
		}

		chains = append(chains, chain)
	}

	return chains, rows.Err()
}
