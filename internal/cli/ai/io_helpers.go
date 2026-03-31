// Purpose:
// - Centralize shared input-resolution, parsing, and JSON output helpers for AI CLI commands.
//
// Responsibilities:
// - Load optional local artifacts, infer result metadata, normalize small CLI inputs, and serialize command results.
//
// Scope:
// - Shared helper logic used across `spartan ai *` command handlers only.
//
// Usage:
// - Imported implicitly within the `ai` package by the feature-scoped command files.
//
// Invariants/Assumptions:
// - Helper error messages remain actionable for operators using the CLI directly.
// - Input loaders do not silently fall back across mutually exclusive sources.
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	commoncli "github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func resolveHTMLInput(raw string, path string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" && strings.TrimSpace(path) != "" {
		return "", fmt.Errorf("--html and --html-file are mutually exclusive")
	}
	if strings.TrimSpace(path) == "" {
		return trimmed, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read html file: %w", err)
	}
	return string(data), nil
}
func resolveTemplateInput(cfg config.Config, name string, path string) (extract.Template, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedPath := strings.TrimSpace(path)
	if trimmedName == "" && trimmedPath == "" {
		return extract.Template{}, fmt.Errorf("--template-name or --template-file is required")
	}
	if trimmedName != "" && trimmedPath != "" {
		return extract.Template{}, fmt.Errorf("--template-name and --template-file are mutually exclusive")
	}
	if trimmedPath != "" {
		data, err := os.ReadFile(trimmedPath)
		if err != nil {
			return extract.Template{}, fmt.Errorf("read template file: %w", err)
		}
		var template extract.Template
		if err := json.Unmarshal(data, &template); err != nil {
			return extract.Template{}, fmt.Errorf("decode template file: %w", err)
		}
		return template, nil
	}
	registry, err := extract.LoadTemplateRegistry(cfg.DataDir)
	if err != nil {
		return extract.Template{}, fmt.Errorf("load template registry: %w", err)
	}
	return registry.GetTemplate(trimmedName)
}
func resolveRenderProfileInput(cfg config.Config, name string, path string) (fetch.RenderProfile, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedPath := strings.TrimSpace(path)
	if trimmedName == "" && trimmedPath == "" {
		return fetch.RenderProfile{}, fmt.Errorf("--profile-name or --profile-file is required")
	}
	if trimmedName != "" && trimmedPath != "" {
		return fetch.RenderProfile{}, fmt.Errorf("--profile-name and --profile-file are mutually exclusive")
	}
	if trimmedPath != "" {
		data, err := os.ReadFile(trimmedPath)
		if err != nil {
			return fetch.RenderProfile{}, fmt.Errorf("read render profile file: %w", err)
		}
		var profile fetch.RenderProfile
		if err := json.Unmarshal(data, &profile); err != nil {
			return fetch.RenderProfile{}, fmt.Errorf("decode render profile file: %w", err)
		}
		return profile, nil
	}
	profile, found, err := fetch.GetRenderProfile(cfg.DataDir, trimmedName)
	if err != nil {
		return fetch.RenderProfile{}, fmt.Errorf("load render profiles: %w", err)
	}
	if !found {
		return fetch.RenderProfile{}, fmt.Errorf("render profile not found: %s", trimmedName)
	}
	return profile, nil
}
func resolvePipelineJSScriptInput(cfg config.Config, name string, path string) (pipeline.JSTargetScript, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedPath := strings.TrimSpace(path)
	if trimmedName == "" && trimmedPath == "" {
		return pipeline.JSTargetScript{}, fmt.Errorf("--script-name or --script-file is required")
	}
	if trimmedName != "" && trimmedPath != "" {
		return pipeline.JSTargetScript{}, fmt.Errorf("--script-name and --script-file are mutually exclusive")
	}
	if trimmedPath != "" {
		data, err := os.ReadFile(trimmedPath)
		if err != nil {
			return pipeline.JSTargetScript{}, fmt.Errorf("read pipeline JS file: %w", err)
		}
		var script pipeline.JSTargetScript
		if err := json.Unmarshal(data, &script); err != nil {
			return pipeline.JSTargetScript{}, fmt.Errorf("decode pipeline JS file: %w", err)
		}
		return script, nil
	}
	script, found, err := pipeline.GetJSScript(cfg.DataDir, trimmedName)
	if err != nil {
		return pipeline.JSTargetScript{}, fmt.Errorf("load pipeline JS registry: %w", err)
	}
	if !found {
		return pipeline.JSTargetScript{}, fmt.Errorf("pipeline JS script not found: %s", trimmedName)
	}
	return script, nil
}
func resolveResearchResultInput(cfg config.Config, jobID string, resultFile string) (research.Result, error) {
	trimmedJobID := strings.TrimSpace(jobID)
	trimmedResultFile := strings.TrimSpace(resultFile)
	if trimmedJobID == "" && trimmedResultFile == "" {
		return research.Result{}, fmt.Errorf("--job-id or --result-file is required")
	}
	if trimmedJobID != "" && trimmedResultFile != "" {
		return research.Result{}, fmt.Errorf("--job-id and --result-file are mutually exclusive")
	}
	if trimmedResultFile != "" {
		data, err := os.ReadFile(trimmedResultFile)
		if err != nil {
			return research.Result{}, fmt.Errorf("read result file: %w", err)
		}
		return parseResearchResultBytes(data)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return research.Result{}, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	job, err := st.Get(context.Background(), trimmedJobID)
	if err != nil {
		return research.Result{}, fmt.Errorf("load job: %w", err)
	}
	if job.Kind != model.KindResearch {
		return research.Result{}, fmt.Errorf("job %s is not a research job", trimmedJobID)
	}
	if strings.TrimSpace(job.ResultPath) == "" {
		return research.Result{}, fmt.Errorf("job %s has no result file", trimmedJobID)
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return research.Result{}, fmt.Errorf("read job result file: %w", err)
	}
	return parseResearchResultBytes(data)
}
func resolveExportShapeInput(cfg config.Config, jobID string, resultFile string, kindHint string, format string, scheduleID string, shapeFile string) (model.Kind, []byte, string, exporter.ShapeConfig, error) {
	if scheduleID != "" && shapeFile != "" {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("--schedule-id and --shape-file are mutually exclusive")
	}
	resolvedFormat := strings.TrimSpace(format)
	currentShape := exporter.ShapeConfig{}
	if scheduleID != "" {
		store := scheduler.NewExportStorage(cfg.DataDir)
		schedule, err := store.Get(scheduleID)
		if err != nil {
			return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("load export schedule: %w", err)
		}
		if resolvedFormat == "" {
			resolvedFormat = strings.TrimSpace(schedule.Export.Format)
		}
		currentShape = schedule.Export.Shape
	}
	if shapeFile != "" {
		shape, err := commoncli.ReadExportShapeFile(shapeFile)
		if err != nil {
			return "", nil, "", exporter.ShapeConfig{}, err
		}
		currentShape = shape
	}
	if resolvedFormat == "" {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("--format is required unless --schedule-id supplies one")
	}
	if !exporter.SupportsShapeFormat(resolvedFormat) {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("--format must be one of: md, csv, xlsx")
	}
	if jobID == "" && resultFile == "" {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("--job-id or --result-file is required")
	}
	if jobID != "" && resultFile != "" {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("--job-id and --result-file are mutually exclusive")
	}
	if jobID != "" {
		kind, data, err := commoncli.LoadJobResultBytes(cfg, jobID)
		if err != nil {
			return "", nil, "", exporter.ShapeConfig{}, err
		}
		return kind, data, resolvedFormat, currentShape, nil
	}
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return "", nil, "", exporter.ShapeConfig{}, fmt.Errorf("read result file: %w", err)
	}
	kind, err := inferJobKindFromResultBytes(data, kindHint)
	if err != nil {
		return "", nil, "", exporter.ShapeConfig{}, err
	}
	return kind, data, resolvedFormat, currentShape, nil
}
func resolveTransformRequestInput(cfg config.Config, jobID string, resultFile string, scheduleID string, transformFile string, preferredLanguage string, expression string) ([]byte, exporter.TransformConfig, string, error) {
	trimmedJobID := strings.TrimSpace(jobID)
	trimmedResultFile := strings.TrimSpace(resultFile)
	if trimmedJobID == "" && trimmedResultFile == "" {
		return nil, exporter.TransformConfig{}, "", fmt.Errorf("--job-id or --result-file is required")
	}
	if trimmedJobID != "" && trimmedResultFile != "" {
		return nil, exporter.TransformConfig{}, "", fmt.Errorf("--job-id and --result-file are mutually exclusive")
	}
	if strings.TrimSpace(scheduleID) != "" && strings.TrimSpace(transformFile) != "" {
		return nil, exporter.TransformConfig{}, "", fmt.Errorf("--schedule-id and --transform-file are mutually exclusive")
	}

	var raw []byte
	if trimmedResultFile != "" {
		data, err := os.ReadFile(trimmedResultFile)
		if err != nil {
			return nil, exporter.TransformConfig{}, "", fmt.Errorf("read result file: %w", err)
		}
		raw = data
	} else {
		_, data, err := commoncli.LoadJobResultBytes(cfg, trimmedJobID)
		if err != nil {
			return nil, exporter.TransformConfig{}, "", err
		}
		raw = data
	}

	current := exporter.TransformConfig{
		Expression: strings.TrimSpace(expression),
		Language:   strings.TrimSpace(preferredLanguage),
	}
	if current.Expression == "" {
		switch {
		case strings.TrimSpace(scheduleID) != "":
			store := scheduler.NewExportStorage(cfg.DataDir)
			schedule, err := store.Get(strings.TrimSpace(scheduleID))
			if err != nil {
				return nil, exporter.TransformConfig{}, "", fmt.Errorf("load export schedule: %w", err)
			}
			current = schedule.Export.Transform
		case strings.TrimSpace(transformFile) != "":
			transform, err := commoncli.ReadTransformConfigFile(strings.TrimSpace(transformFile))
			if err != nil {
				return nil, exporter.TransformConfig{}, "", err
			}
			current = transform
		}
	}
	if strings.TrimSpace(preferredLanguage) != "" {
		current.Language = strings.TrimSpace(preferredLanguage)
	}
	current = exporter.NormalizeTransformConfig(current)
	resolvedPreferred := strings.TrimSpace(preferredLanguage)
	if resolvedPreferred == "" {
		resolvedPreferred = current.Language
	}
	return raw, current, resolvedPreferred, nil
}
func inferJobKindFromResultBytes(data []byte, kindHint string) (model.Kind, error) {
	if trimmedHint := strings.TrimSpace(kindHint); trimmedHint != "" {
		kind := model.Kind(trimmedHint)
		switch kind {
		case model.KindScrape, model.KindCrawl, model.KindResearch:
			return kind, nil
		default:
			return "", fmt.Errorf("--kind must be scrape, crawl, or research")
		}
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("result file is empty")
	}
	if trimmed[0] == '{' {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &probe); err != nil {
			return "", fmt.Errorf("decode result file: %w", err)
		}
		if _, ok := probe["query"]; ok {
			return model.KindResearch, nil
		}
		if _, ok := probe["evidence"]; ok {
			return model.KindResearch, nil
		}
		if _, ok := probe["normalized"]; ok {
			return model.KindScrape, nil
		}
		return model.KindScrape, nil
	}
	if trimmed[0] == '[' {
		return model.KindCrawl, nil
	}
	lines := strings.Split(string(trimmed), "\n")
	jsonLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		jsonLines++
	}
	if jsonLines > 1 {
		return model.KindCrawl, nil
	}
	return "", fmt.Errorf("unable to infer job kind from result file; pass --kind")
}
func parseResearchResultBytes(data []byte) (research.Result, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return research.Result{}, fmt.Errorf("result file is empty")
	}

	switch trimmed[0] {
	case '{':
		var result research.Result
		if err := json.Unmarshal(trimmed, &result); err != nil {
			return research.Result{}, fmt.Errorf("decode research result object: %w", err)
		}
		return result, nil
	case '[':
		var results []research.Result
		if err := json.Unmarshal(trimmed, &results); err != nil {
			return research.Result{}, fmt.Errorf("decode research result array: %w", err)
		}
		if len(results) != 1 {
			return research.Result{}, fmt.Errorf("research result array must contain exactly 1 item")
		}
		return results[0], nil
	default:
		lines := strings.Split(string(trimmed), "\n")
		results := make([]research.Result, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var result research.Result
			if err := json.Unmarshal([]byte(line), &result); err != nil {
				return research.Result{}, fmt.Errorf("decode research result JSONL line: %w", err)
			}
			results = append(results, result)
		}
		if len(results) != 1 {
			return research.Result{}, fmt.Errorf("research result JSONL must contain exactly 1 item")
		}
		return results[0], nil
	}
}
func parseJSONObject(raw string, emptyMessage string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New(emptyMessage)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("JSON object must not be empty")
	}
	return decoded, nil
}
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
func loadAIImageFiles(paths []string) ([]extract.AIImageInput, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	images := make([]extract.AIImageInput, 0, len(paths))
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read image file %q: %w", path, err)
		}
		mimeType := detectAIImageMimeType(path, data)
		images = append(images, extract.AIImageInput{
			Data:     base64.StdEncoding.EncodeToString(data),
			MimeType: mimeType,
		})
	}
	if len(images) == 0 {
		return nil, nil
	}
	return images, nil
}
func detectAIImageMimeType(path string, data []byte) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	}
	if detected := http.DetectContentType(data); strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "application/octet-stream"
}
func writeJSONResult(v interface{}, outPath string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if strings.TrimSpace(outPath) == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	_, err = fsutil.WritePrivateFileWithinRoot(cwd, filepath.Clean(strings.TrimSpace(outPath)), data)
	return err
}
