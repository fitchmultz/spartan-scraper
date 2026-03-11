// Package model defines shared domain types for jobs, crawling, and state tracking.
//
// Purpose:
// - Define the canonical persisted Job model and related enums for the 1.0 core.
//
// Responsibilities:
// - Hold stable job identity, lifecycle, dependency, and persisted spec metadata.
// - Validate artifact result paths against the local data directory.
//
// Scope:
// - Domain model types only. Persistence and execution live in other packages.
//
// Usage:
// - Used across API, scheduler, MCP, store, and jobs runtime code.
//
// Invariants/Assumptions:
// - Jobs persist typed versioned specs, not generic params bags.
// - Result paths always live under DATA_DIR/jobs/<job-id>/.
// - Unsupported multi-user/workspace fields are not part of the stable 1.0 model.
package model

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type Kind string

type Status string

// DependencyStatus represents the state of a job's dependencies.
type DependencyStatus string

const (
	KindScrape   Kind = "scrape"
	KindCrawl    Kind = "crawl"
	KindResearch Kind = "research"

	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"

	DependencyStatusPending DependencyStatus = "pending"
	DependencyStatusReady   DependencyStatus = "ready"
	DependencyStatusFailed  DependencyStatus = "failed"
)

var validStatuses = map[Status]bool{
	StatusQueued:    true,
	StatusRunning:   true,
	StatusSucceeded: true,
	StatusFailed:    true,
	StatusCanceled:  true,
}

var validDependencyStatuses = map[DependencyStatus]bool{
	DependencyStatusPending: true,
	DependencyStatusReady:   true,
	DependencyStatusFailed:  true,
}

// ArtifactRole identifies the purpose of an artifact in a job manifest.
type ArtifactRole string

const (
	ArtifactRoleResults  ArtifactRole = "results"
	ArtifactRoleManifest ArtifactRole = "manifest"
	ArtifactRoleExport   ArtifactRole = "export"
)

// IsValid returns true if the dependency status is a recognized value.
func (s DependencyStatus) IsValid() bool {
	return validDependencyStatuses[s]
}

func (s Status) IsTerminal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCanceled
}

func (s Status) IsValid() bool {
	return validStatuses[s]
}

func ValidStatuses() []Status {
	return []Status{StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCanceled}
}

// Job represents a single persisted scrape, crawl, or research task.
type Job struct {
	ID               string           `json:"id"`
	Kind             Kind             `json:"kind"`
	Status           Status           `json:"status"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
	StartedAt        *time.Time       `json:"startedAt,omitempty"`
	FinishedAt       *time.Time       `json:"finishedAt,omitempty"`
	SpecVersion      int              `json:"specVersion"`
	Spec             any              `json:"spec,omitempty"`
	ResultPath       string           `json:"resultPath,omitempty"`
	Error            string           `json:"error"`
	DependsOn        []string         `json:"dependsOn,omitempty"`
	DependencyStatus DependencyStatus `json:"dependencyStatus,omitempty"`
	ChainID          string           `json:"chainId,omitempty"`
	SelectedEngine   string           `json:"selectedEngine,omitempty"`
}

// ExtractWebhookConfig extracts webhook configuration from the job's typed spec.
func (j Job) ExtractWebhookConfig() *WebhookSpec {
	return ExtractWebhookSpec(j.Spec)
}

// SpecMap returns the job spec as a generic map for diagnostics and tests.
func (j Job) SpecMap() map[string]interface{} {
	if j.Spec == nil {
		return nil
	}
	raw, err := json.Marshal(j.Spec)
	if err != nil {
		return nil
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil
	}
	if execRaw, ok := generic["execution"]; ok {
		if execMap, ok := execRaw.(map[string]interface{}); ok {
			for key, value := range execMap {
				if _, exists := generic[key]; !exists {
					generic[key] = value
				}
			}
		}
	}
	return generic
}

// ValidateResultPath validates that a job's result path is within the allowed directory.
func ValidateResultPath(jobID, resultPath, dataDir string) error {
	if resultPath == "" {
		return nil
	}

	absPath, err := filepath.Abs(resultPath)
	if err != nil {
		return apperrors.Validation("invalid result path")
	}

	baseDir := filepath.Join(dataDir, "jobs", jobID)
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return apperrors.Internal("failed to resolve base directory")
	}

	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
		return apperrors.Validation("result path outside allowed directory")
	}

	return nil
}
