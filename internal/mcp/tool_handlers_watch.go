// Package mcp implements watch-oriented MCP tool handlers.
//
// Purpose:
// - Keep watch CRUD and check operations separated from unrelated MCP domains.
//
// Responsibilities:
// - Validate watch identifiers and decode create/update payloads.
// - Reuse shared watch helper logic for defaults and normalization.
// - Return canonical watch and watch-check inspection responses.
//
// Scope:
// - MCP watch handlers only.
//
// Usage:
// - Registered through watchToolRegistry in tool_registry.go.
//
// Invariants/Assumptions:
// - Watch limits cap at 1000 for list operations.
// - Not-found watch storage errors are converted into apperrors.NotFound.
// - Manual checks return persisted inspection payloads, not transient internal structs.
package mcp

import (
	"context"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func (s *Server) handleWatchListTool(_ context.Context, params callParams) (interface{}, error) {
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	watches, total, err := watch.NewFileStorage(s.cfg.DataDir).ListPage(limit, offset)
	if err != nil {
		return nil, err
	}
	return api.BuildWatchListResponse(watches, total, limit, offset), nil
}

func (s *Server) handleWatchGetTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	watchItem, err := watch.NewFileStorage(s.cfg.DataDir).Get(id)
	if err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	return api.BuildWatchResponse(*watchItem), nil
}

func (s *Server) handleWatchCreateTool(_ context.Context, params callParams) (interface{}, error) {
	var args watchCreateArgs
	if err := decodeToolArguments(params.Arguments, &args); err != nil {
		return nil, err
	}
	watchItem, err := s.buildWatchCreate(args)
	if err != nil {
		return nil, err
	}
	created, err := watch.NewFileStorage(s.cfg.DataDir).Add(watchItem)
	if err != nil {
		return nil, err
	}
	return api.BuildWatchResponse(*created), nil
}

func (s *Server) handleWatchUpdateTool(_ context.Context, params callParams) (interface{}, error) {
	var args watchUpdateArgs
	if err := decodeToolArguments(params.Arguments, &args); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	watchStore := watch.NewFileStorage(s.cfg.DataDir)
	existing, err := watchStore.Get(id)
	if err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	if err := s.applyWatchUpdate(existing, args); err != nil {
		return nil, err
	}
	if err := watchStore.Update(existing); err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	return api.BuildWatchResponse(*existing), nil
}

func (s *Server) handleWatchDeleteTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	if err := watch.NewFileStorage(s.cfg.DataDir).Delete(id); err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	return map[string]interface{}{"deleted": true, "id": id}, nil
}

func (s *Server) handleWatchCheckTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	watchStore := watch.NewFileStorage(s.cfg.DataDir)
	watchItem, err := watchStore.Get(id)
	if err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	watcher := watch.NewWatcher(watchStore, s.store, s.cfg.DataDir, nil, &watch.TriggerRuntime{
		Config:  s.cfg,
		Manager: s.manager,
	})
	result, err := watcher.Check(ctx, watchItem)
	if result != nil {
		return api.WatchCheckInspectionResponse{Check: api.BuildWatchCheckInspection(watch.RecordFromCheckResult(result))}, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, apperrors.Internal("watch check returned no result")
}

func (s *Server) handleWatchCheckHistoryTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	if _, err := watch.NewFileStorage(s.cfg.DataDir).Get(id); err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 10)
	offset := paramdecode.Decode[int](params.Arguments, "offset")
	if offset < 0 {
		offset = 0
	}
	records, total, err := watch.NewWatchHistoryStore(s.cfg.DataDir).GetByWatch(id, limit, offset)
	if err != nil {
		return nil, err
	}
	return api.BuildWatchCheckHistoryResponse(records, total, limit, offset), nil
}

func (s *Server) handleWatchCheckGetTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	checkID := strings.TrimSpace(paramdecode.String(params.Arguments, "checkId"))
	if checkID == "" {
		return nil, apperrors.Validation("checkId is required")
	}
	if _, err := watch.NewFileStorage(s.cfg.DataDir).Get(id); err != nil {
		if watch.IsNotFoundError(err) {
			return nil, apperrors.NotFound("watch not found")
		}
		return nil, err
	}
	record, err := watch.NewWatchHistoryStore(s.cfg.DataDir).GetByID(id, checkID)
	if err != nil {
		return nil, apperrors.NotFound("watch check not found")
	}
	return api.WatchCheckInspectionResponse{Check: api.BuildWatchCheckInspection(*record)}, nil
}
