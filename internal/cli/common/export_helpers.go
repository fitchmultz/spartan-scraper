// Package common provides common functionality for Spartan Scraper.
//
// Purpose:
// - Implement export helpers support for package common.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `common` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func LoadJobResultBytes(cfg config.Config, jobID string) (model.Kind, []byte, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return "", nil, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()
	job, err := st.Get(context.Background(), jobID)
	if err != nil {
		return "", nil, fmt.Errorf("load job: %w", err)
	}
	if strings.TrimSpace(job.ResultPath) == "" {
		return "", nil, fmt.Errorf("job %s has no result file", jobID)
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return "", nil, fmt.Errorf("read job result file: %w", err)
	}
	return job.Kind, data, nil
}

func ReadExportShapeFile(path string) (exporter.ShapeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return exporter.ShapeConfig{}, fmt.Errorf("read shape file: %w", err)
	}
	var shape exporter.ShapeConfig
	if err := json.Unmarshal(data, &shape); err != nil {
		return exporter.ShapeConfig{}, fmt.Errorf("decode shape file: %w", err)
	}
	return exporter.NormalizeShapeConfig(shape), nil
}

func ReadTransformConfigFile(path string) (exporter.TransformConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return exporter.TransformConfig{}, fmt.Errorf("read transform file: %w", err)
	}
	var transform exporter.TransformConfig
	if err := json.Unmarshal(data, &transform); err != nil {
		return exporter.TransformConfig{}, fmt.Errorf("decode transform file: %w", err)
	}
	return exporter.NormalizeTransformConfig(transform), nil
}
