// Package api provides watch artifact response shaping and download handlers.
//
// Purpose:
// - Convert internal watch artifact metadata into stable public response shapes.
//
// Responsibilities:
// - Build watch-check artifact descriptors with download URLs.
// - Serve persisted watch artifacts through explicit download endpoints.
// - Hide host-local filesystem paths from public watch-check responses.
//
// Scope:
// - `/v1/watch/{id}/artifacts/{artifactKind}` and watch-check response shaping.
//
// Usage:
//   - Called by the watch handlers after manual checks and by the nested watch
//     artifact route.
//
// Invariants/Assumptions:
// - Public responses expose download URLs, never host-local paths.
// - Artifact downloads are resolved only from deterministic watch artifact files.
// - Artifact kinds are limited to the canonical watch artifact enum.
package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

// WatchArtifactResponse describes a public watch artifact download.
type WatchArtifactResponse struct {
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ByteSize    int64  `json:"byteSize,omitempty"`
	DownloadURL string `json:"downloadUrl"`
}

func toWatchArtifactResponse(watchID string, artifact watch.Artifact) WatchArtifactResponse {
	return toWatchArtifactResponseForCheck(watchID, "", artifact)
}

func toWatchArtifactResponseForCheck(watchID string, checkID string, artifact watch.Artifact) WatchArtifactResponse {
	downloadURL := watchArtifactDownloadURL(watchID, artifact.Kind)
	if strings.TrimSpace(checkID) != "" {
		downloadURL = watchHistoryArtifactDownloadURL(watchID, checkID, artifact.Kind)
	}
	return WatchArtifactResponse{
		Kind:        string(artifact.Kind),
		Filename:    artifact.Filename,
		ContentType: artifact.ContentType,
		ByteSize:    artifact.ByteSize,
		DownloadURL: downloadURL,
	}
}

func watchArtifactDownloadURL(watchID string, kind watch.ArtifactKind) string {
	return fmt.Sprintf("/v1/watch/%s/artifacts/%s", watchID, kind)
}

func watchHistoryArtifactDownloadURL(watchID string, checkID string, kind watch.ArtifactKind) string {
	return fmt.Sprintf("/v1/watch/%s/history/%s/artifacts/%s", watchID, checkID, kind)
}

func extractWatchArtifactKind(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	marker := "/artifacts/"
	idx := strings.LastIndex(trimmed, marker)
	if idx == -1 {
		return ""
	}
	return trimmed[idx+len(marker):]
}

func (s *Server) handleWatchArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	watchID, err := requireResourceID(r, "watch", "watch id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	artifactKind, ok := watch.ParseArtifactKind(extractWatchArtifactKind(r.URL.Path))
	if !ok {
		writeError(w, r, apperrors.Validation("invalid watch artifact kind"))
		return
	}

	storage := watch.NewFileStorage(s.cfg.DataDir)
	if _, ok := getStoredResource(w, r, func() (*watch.Watch, error) {
		return storage.Get(watchID)
	}, watch.IsNotFoundError, "watch"); !ok {
		return
	}

	artifact, err := watch.NewArtifactStore(s.cfg.DataDir).Resolve(watchID, artifactKind)
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
