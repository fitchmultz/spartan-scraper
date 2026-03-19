// Package api provides HTTP handlers for persisted watch-check inspection routes.
//
// Purpose:
// - Expose watch history list, detail, and artifact-download endpoints for operator inspection.
//
// Responsibilities:
// - Paginate persisted watch check records.
// - Return canonical watch inspection envelopes with narratives and actions.
// - Serve check-scoped artifact snapshots without exposing host-local filesystem paths.
//
// Scope:
// - `/v1/watch/{id}/history`, `/v1/watch/{id}/history/{checkId}`, and nested history artifact routes only.
//
// Usage:
// - Mounted through the watch detail wrapper in Server.Routes.
//
// Invariants/Assumptions:
// - Watch history is scoped to an existing watch.
// - History artifact downloads resolve from persisted check snapshots, not the mutable latest-artifact slots.
package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func (s *Server) handleWatchHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	watchID, ok := s.requireExistingWatch(w, r)
	if !ok {
		return
	}
	params, err := parsePageParams(r, 10, 100)
	if err != nil {
		writeError(w, r, err)
		return
	}

	records, total, err := watch.NewWatchHistoryStore(s.cfg.DataDir).GetByWatch(watchID, params.Limit, params.Offset)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, BuildWatchCheckHistoryResponse(records, total, params.Limit, params.Offset))
}

func (s *Server) handleWatchHistoryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	watchID, ok := s.requireExistingWatch(w, r)
	if !ok {
		return
	}
	checkID := extractWatchHistoryCheckID(r.URL.Path)
	if strings.TrimSpace(checkID) == "" {
		writeError(w, r, apperrors.Validation("check id is required"))
		return
	}

	record, err := watch.NewWatchHistoryStore(s.cfg.DataDir).GetByID(watchID, checkID)
	if err != nil {
		writeError(w, r, apperrors.NotFound("watch check not found"))
		return
	}
	writeJSON(w, WatchCheckInspectionResponse{Check: BuildWatchCheckInspection(*record)})
}

func (s *Server) handleWatchHistoryArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	watchID, ok := s.requireExistingWatch(w, r)
	if !ok {
		return
	}
	checkID := extractWatchHistoryCheckID(r.URL.Path)
	if strings.TrimSpace(checkID) == "" {
		writeError(w, r, apperrors.Validation("check id is required"))
		return
	}
	artifactKind, ok := watch.ParseArtifactKind(extractWatchHistoryArtifactKind(r.URL.Path))
	if !ok {
		writeError(w, r, apperrors.Validation("invalid watch artifact kind"))
		return
	}

	historyStore := watch.NewWatchHistoryStore(s.cfg.DataDir)
	record, err := historyStore.GetByID(watchID, checkID)
	if err != nil {
		writeError(w, r, apperrors.NotFound("watch check not found"))
		return
	}
	if !recordHasArtifact(*record, artifactKind) {
		writeError(w, r, apperrors.NotFound("watch artifact not found"))
		return
	}

	artifact, err := historyStore.ResolveArtifact(watchID, checkID, artifactKind)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, r, apperrors.NotFound("watch artifact not found"))
			return
		}
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to load watch artifact", err))
		return
	}

	w.Header().Set("Content-Type", artifact.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, artifact.Filename))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, artifact.Path)
}

func (s *Server) requireExistingWatch(w http.ResponseWriter, r *http.Request) (string, bool) {
	watchID, err := requireResourceID(r, "watch", "watch id")
	if err != nil {
		writeError(w, r, err)
		return "", false
	}
	storage := watch.NewFileStorage(s.cfg.DataDir)
	if _, ok := getStoredResource(w, r, func() (*watch.Watch, error) {
		return storage.Get(watchID)
	}, watch.IsNotFoundError, "watch"); !ok {
		return "", false
	}
	return watchID, true
}

func extractWatchHistoryCheckID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "history" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func extractWatchHistoryArtifactKind(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	marker := "/artifacts/"
	idx := strings.LastIndex(trimmed, marker)
	if idx == -1 {
		return ""
	}
	return trimmed[idx+len(marker):]
}

func recordHasArtifact(record watch.WatchCheckRecord, kind watch.ArtifactKind) bool {
	for _, artifact := range record.Artifacts {
		if artifact.Kind == kind {
			return true
		}
	}
	return false
}
