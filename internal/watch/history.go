// Package watch persists operator-facing watch check history.
//
// Purpose:
// - Record canonical watch check outcomes and per-check artifact snapshots shared by API, Web, CLI, and MCP inspection flows.
//
// Responsibilities:
// - Persist watch check records with stable IDs and derived outcome statuses.
// - Snapshot per-check artifacts so historical inspections do not drift as the latest watch artifacts rotate.
// - Provide paginated lookup, single-record retrieval, artifact resolution, and watch-level cleanup helpers.
//
// Scope:
// - Watch check history persistence and artifact snapshot storage only; check execution lives in watch.go.
//
// Usage:
// - Used by Watcher.Check, watch history API handlers, CLI history commands, and MCP watch inspection tools.
//
// Invariants/Assumptions:
// - History file path is DATA_DIR/watch_history.json.
// - Per-check artifacts live under DATA_DIR/watch_history_artifacts/<watch-id>/<check-id>/.
// - Records are stored newest-first when paginated but persist append-only on disk.
package watch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

// CheckStatus describes the high-level outcome of a persisted watch check.
type CheckStatus string

const (
	CheckStatusBaseline  CheckStatus = "baseline"
	CheckStatusChanged   CheckStatus = "changed"
	CheckStatusUnchanged CheckStatus = "unchanged"
	CheckStatusFailed    CheckStatus = "failed"
)

// WatchCheckRecord stores one persisted watch check outcome.
type WatchCheckRecord struct {
	ID                 string      `json:"id"`
	WatchID            string      `json:"watchId"`
	URL                string      `json:"url"`
	CheckedAt          time.Time   `json:"checkedAt"`
	Status             CheckStatus `json:"status"`
	Changed            bool        `json:"changed"`
	Baseline           bool        `json:"baseline,omitempty"`
	PreviousHash       string      `json:"previousHash,omitempty"`
	CurrentHash        string      `json:"currentHash,omitempty"`
	DiffText           string      `json:"diffText,omitempty"`
	DiffHTML           string      `json:"diffHtml,omitempty"`
	Error              string      `json:"error,omitempty"`
	Selector           string      `json:"selector,omitempty"`
	Artifacts          []Artifact  `json:"artifacts,omitempty"`
	VisualHash         string      `json:"visualHash,omitempty"`
	PreviousVisualHash string      `json:"previousVisualHash,omitempty"`
	VisualChanged      bool        `json:"visualChanged"`
	VisualSimilarity   float64     `json:"visualSimilarity,omitempty"`
	TriggeredJobs      []string    `json:"triggeredJobs,omitempty"`
}

// WatchHistoryStore manages watch history persistence.
type WatchHistoryStore struct {
	dataDir string
	mu      sync.RWMutex
}

type watchHistoryEnvelope struct {
	Checks []WatchCheckRecord `json:"checks"`
}

// NewWatchHistoryStore creates a new watch history store rooted at DATA_DIR.
func NewWatchHistoryStore(dataDir string) *WatchHistoryStore {
	return &WatchHistoryStore{dataDir: dataDir}
}

// Record persists one watch check result and snapshots any attached artifacts.
func (s *WatchHistoryStore) Record(result WatchCheckResult) (*WatchCheckRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}

	record := RecordFromCheckResult(&result)
	record.ID = uuid.NewString()
	if record.CheckedAt.IsZero() {
		record.CheckedAt = time.Now()
	}

	artifacts, artifactErr := s.persistArtifactsUnsafe(record.WatchID, record.ID, result.Artifacts)
	record.Artifacts = artifacts

	history.Checks = append(history.Checks, record)
	if err := s.saveUnsafe(history); err != nil {
		return nil, err
	}
	return &record, artifactErr
}

// GetByWatch returns paginated history for one watch sorted newest-first.
func (s *WatchHistoryStore) GetByWatch(watchID string, limit, offset int) ([]WatchCheckRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, 0, err
	}

	records := make([]WatchCheckRecord, 0)
	for _, record := range history.Checks {
		if record.WatchID == watchID {
			records = append(records, record)
		}
	}
	return paginateWatchCheckRecords(records, limit, offset), len(records), nil
}

// GetByID returns one persisted watch check scoped to its watch.
func (s *WatchHistoryStore) GetByID(watchID string, checkID string) (*WatchCheckRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}

	for i := range history.Checks {
		if history.Checks[i].WatchID == watchID && history.Checks[i].ID == checkID {
			record := history.Checks[i]
			return &record, nil
		}
	}
	return nil, errors.New("watch check record not found")
}

// ResolveArtifact returns metadata and the persisted path for a check-scoped artifact.
func (s *WatchHistoryStore) ResolveArtifact(watchID string, checkID string, kind ArtifactKind) (Artifact, error) {
	path := s.historyArtifactPath(watchID, checkID, kind)
	info, err := os.Stat(path)
	if err != nil {
		return Artifact{}, err
	}
	contentType, err := detectContentType(path)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:        kind,
		Filename:    historyArtifactFilename(watchID, checkID, kind, contentType),
		ContentType: contentType,
		ByteSize:    info.Size(),
		Path:        path,
	}, nil
}

// DeleteWatch removes all history and persisted per-check artifacts for one watch.
func (s *WatchHistoryStore) DeleteWatch(watchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	filtered := make([]WatchCheckRecord, 0, len(history.Checks))
	for _, record := range history.Checks {
		if record.WatchID != watchID {
			filtered = append(filtered, record)
		}
	}
	history.Checks = filtered
	if err := s.saveUnsafe(history); err != nil {
		return err
	}
	return os.RemoveAll(s.watchHistoryArtifactsDir(watchID))
}

// RecordFromCheckResult converts one immediate watch check result into its persisted history shape.
func RecordFromCheckResult(result *WatchCheckResult) WatchCheckRecord {
	if result == nil {
		return WatchCheckRecord{}
	}
	return WatchCheckRecord{
		ID:                 strings.TrimSpace(result.CheckID),
		WatchID:            strings.TrimSpace(result.WatchID),
		URL:                strings.TrimSpace(result.URL),
		CheckedAt:          result.CheckedAt,
		Status:             deriveCheckStatus(*result),
		Changed:            result.Changed,
		Baseline:           result.Baseline,
		PreviousHash:       strings.TrimSpace(result.PreviousHash),
		CurrentHash:        strings.TrimSpace(result.CurrentHash),
		DiffText:           result.DiffText,
		DiffHTML:           result.DiffHTML,
		Error:              strings.TrimSpace(result.Error),
		Selector:           strings.TrimSpace(result.Selector),
		Artifacts:          append([]Artifact(nil), result.Artifacts...),
		VisualHash:         strings.TrimSpace(result.VisualHash),
		PreviousVisualHash: strings.TrimSpace(result.PreviousVisualHash),
		VisualChanged:      result.VisualChanged,
		VisualSimilarity:   result.VisualSimilarity,
		TriggeredJobs:      append([]string(nil), result.TriggeredJobs...),
	}
}

func deriveCheckStatus(result WatchCheckResult) CheckStatus {
	switch {
	case result.Baseline:
		return CheckStatusBaseline
	case result.Changed:
		return CheckStatusChanged
	case strings.TrimSpace(result.Error) != "":
		return CheckStatusFailed
	default:
		return CheckStatusUnchanged
	}
}

func paginateWatchCheckRecords(records []WatchCheckRecord, limit, offset int) []WatchCheckRecord {
	sortWatchCheckRecords(records)
	total := len(records)
	if offset >= total {
		return []WatchCheckRecord{}
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return records[offset:end]
}

func sortWatchCheckRecords(records []WatchCheckRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CheckedAt.After(records[j].CheckedAt)
	})
}

func (s *WatchHistoryStore) persistArtifactsUnsafe(watchID string, checkID string, artifacts []Artifact) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}
	if err := fsutil.MkdirAllSecure(s.historyArtifactsDir(watchID, checkID)); err != nil {
		return nil, err
	}

	persisted := make([]Artifact, 0, len(artifacts))
	var firstErr error
	for _, artifact := range artifacts {
		if artifact.Kind == "" || strings.TrimSpace(artifact.Path) == "" {
			continue
		}
		destPath := s.historyArtifactPath(watchID, checkID, artifact.Kind)
		if err := copyFileAtomic(artifact.Path, destPath); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		persisted = append(persisted, Artifact{
			Kind:        artifact.Kind,
			Filename:    historyArtifactFilename(watchID, checkID, artifact.Kind, artifact.ContentType),
			ContentType: artifact.ContentType,
			ByteSize:    artifact.ByteSize,
		})
	}
	return persisted, firstErr
}

func (s *WatchHistoryStore) loadUnsafe() (*watchHistoryEnvelope, error) {
	path := s.historyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &watchHistoryEnvelope{Checks: []WatchCheckRecord{}}, nil
		}
		return nil, err
	}

	var history watchHistoryEnvelope
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	if history.Checks == nil {
		history.Checks = []WatchCheckRecord{}
	}
	return &history, nil
}

func (s *WatchHistoryStore) saveUnsafe(history *watchHistoryEnvelope) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.historyPath(), payload, 0o600)
}

func (s *WatchHistoryStore) historyPath() string {
	base := s.dataDir
	if strings.TrimSpace(base) == "" {
		base = ".data"
	}
	return filepath.Join(base, "watch_history.json")
}

func (s *WatchHistoryStore) watchHistoryArtifactsDir(watchID string) string {
	base := s.dataDir
	if strings.TrimSpace(base) == "" {
		base = ".data"
	}
	return filepath.Join(base, "watch_history_artifacts", watchID)
}

func (s *WatchHistoryStore) historyArtifactsDir(watchID string, checkID string) string {
	return filepath.Join(s.watchHistoryArtifactsDir(watchID), checkID)
}

func (s *WatchHistoryStore) historyArtifactPath(watchID string, checkID string, kind ArtifactKind) string {
	return filepath.Join(s.historyArtifactsDir(watchID, checkID), string(kind))
}

func historyArtifactFilename(watchID string, checkID string, kind ArtifactKind, contentType string) string {
	return "watch-" + watchID + "-" + checkID + "-" + string(kind) + extensionForContentType(contentType)
}
