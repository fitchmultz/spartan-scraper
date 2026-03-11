// Package artifacts manages canonical job artifact metadata.
//
// Purpose:
// - Write a stable per-job manifest for operator inspection and debugging.
//
// Responsibilities:
// - Enumerate files under a job artifact directory.
// - Compute content hashes and sizes for recorded artifacts.
// - Persist manifest.json beside job outputs.
//
// Scope:
// - Local artifact manifests only.
//
// Usage:
// - Called by jobs runtime after a job reaches a terminal state.
//
// Invariants/Assumptions:
// - The manifest lives at DATA_DIR/jobs/<job-id>/manifest.json.
// - Only files within the job directory are enumerated.
// - Missing artifact files are skipped rather than causing manifest writes to fail.
package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type FileRecord struct {
	Role         model.ArtifactRole `json:"role"`
	Path         string             `json:"path"`
	Size         int64              `json:"size"`
	SHA256       string             `json:"sha256"`
	LastModified time.Time          `json:"lastModified"`
}

type ExportRecord struct {
	Destination string    `json:"destination"`
	Format      string    `json:"format"`
	Path        string    `json:"path,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Manifest struct {
	JobID          string         `json:"jobId"`
	Kind           model.Kind     `json:"kind"`
	Status         model.Status   `json:"status"`
	SpecVersion    int            `json:"specVersion"`
	SpecHash       string         `json:"specHash"`
	SelectedEngine string         `json:"selectedEngine,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	StartedAt      *time.Time     `json:"startedAt,omitempty"`
	FinishedAt     *time.Time     `json:"finishedAt,omitempty"`
	DurationMs     int64          `json:"durationMs,omitempty"`
	Files          []FileRecord   `json:"files"`
	Exports        []ExportRecord `json:"exports,omitempty"`
}

func WriteManifest(job model.Job) error {
	jobDir := filepath.Dir(job.ResultPath)
	if err := fsutil.MkdirAllSecure(jobDir); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create job artifact directory", err)
	}

	specHash, err := hashJSON(job.Spec)
	if err != nil {
		return err
	}
	files, err := enumerateFiles(jobDir)
	if err != nil {
		return err
	}

	manifest := Manifest{
		JobID:          job.ID,
		Kind:           job.Kind,
		Status:         job.Status,
		SpecVersion:    job.SpecVersion,
		SpecHash:       specHash,
		SelectedEngine: job.SelectedEngine,
		CreatedAt:      job.CreatedAt,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
		Files:          files,
		Exports:        []ExportRecord{},
	}
	if job.StartedAt != nil && job.FinishedAt != nil {
		manifest.DurationMs = job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
	}

	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal artifact manifest", err)
	}

	manifestPath := filepath.Join(jobDir, "manifest.json")
	file, err := fsutil.CreateSecure(manifestPath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create artifact manifest", err)
	}
	defer file.Close()

	if _, err := file.Write(append(raw, '\n')); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write artifact manifest", err)
	}
	return nil
}

func enumerateFiles(jobDir string) ([]FileRecord, error) {
	var records []FileRecord
	err := filepath.Walk(jobDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "manifest.json" {
			return nil
		}
		role := model.ArtifactRoleResults
		if filepath.Ext(path) != ".jsonl" {
			role = model.ArtifactRoleExport
		}
		relPath, err := filepath.Rel(jobDir, path)
		if err != nil {
			return err
		}
		hash, err := hashFile(path)
		if err != nil {
			return err
		}
		records = append(records, FileRecord{
			Role:         role,
			Path:         relPath,
			Size:         info.Size(),
			SHA256:       hash,
			LastModified: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to enumerate job artifacts", err)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, nil
}

func hashJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to hash job spec", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to open artifact file", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to hash artifact file", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
