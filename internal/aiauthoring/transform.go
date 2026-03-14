package aiauthoring

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

const (
	transformSampleRecordLimit = 5
	transformSampleValueLimit  = 3
	transformPreviewLimit      = 5
	transformFieldDepthLimit   = 6
)

type TransformRequest struct {
	JobKind           model.Kind
	RawResult         []byte
	CurrentTransform  exporter.TransformConfig
	PreferredLanguage string
	Instructions      string
}

type TransformInputStats struct {
	SampleRecordCount        int  `json:"sampleRecordCount"`
	FieldPathCount           int  `json:"fieldPathCount"`
	CurrentTransformProvided bool `json:"currentTransformProvided"`
}

type TransformResult struct {
	Issues      []string                 `json:"issues,omitempty"`
	InputStats  TransformInputStats      `json:"inputStats"`
	Transform   exporter.TransformConfig `json:"transform"`
	Preview     []any                    `json:"preview,omitempty"`
	Explanation string                   `json:"explanation,omitempty"`
	RouteID     string                   `json:"route_id,omitempty"`
	Provider    string                   `json:"provider,omitempty"`
	Model       string                   `json:"model,omitempty"`
}

type transformInputDiagnostics struct {
	Issues        []string
	SampleRecords []map[string]any
	SampleFields  []piai.TransformSampleField
}

type sampleSet struct {
	values []string
	seen   map[string]struct{}
}

func (s *Service) GenerateTransform(ctx context.Context, req TransformRequest) (TransformResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return TransformResult{}, err
	}
	if err := validateTransformInput(req); err != nil {
		return TransformResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	diagnostics, err := analyzeTransformInput(req.RawResult, req.CurrentTransform)
	if err != nil {
		return TransformResult{}, err
	}
	aiReq := piai.GenerateTransformRequest{
		JobKind:           string(req.JobKind),
		SampleRecords:     cloneSampleRecords(diagnostics.SampleRecords),
		SampleFields:      cloneTransformSampleFields(diagnostics.SampleFields),
		CurrentTransform:  bridgeTransformConfig(req.CurrentTransform),
		PreferredLanguage: strings.TrimSpace(req.PreferredLanguage),
		Instructions:      strings.TrimSpace(req.Instructions),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GenerateTransform(ctx, aiReq)
		if err != nil {
			return TransformResult{}, apperrors.Wrap(apperrors.KindInternal, "AI transform generation failed", err)
		}
		candidate := exporter.NormalizeTransformConfig(transformConfig(aiResult.Transform))
		preview, issues := validateTransformCandidate(candidate, diagnostics.SampleRecords, strings.TrimSpace(req.PreferredLanguage))
		if len(issues) == 0 {
			return TransformResult{
				Issues: dedupeIssues(diagnostics.Issues),
				InputStats: TransformInputStats{
					SampleRecordCount:        len(diagnostics.SampleRecords),
					FieldPathCount:           len(diagnostics.SampleFields),
					CurrentTransformProvided: exporter.HasMeaningfulTransform(req.CurrentTransform),
				},
				Transform:   candidate,
				Preview:     limitPreview(preview, transformPreviewLimit),
				Explanation: strings.TrimSpace(aiResult.Explanation),
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}
		if attempt == 1 {
			return TransformResult{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = joinFeedback(aiReq.Feedback, strings.Join(issues, "; "))
	}

	return TransformResult{}, apperrors.Internal("AI transform generation failed")
}

func validateTransformInput(req TransformRequest) error {
	if len(bytes.TrimSpace(req.RawResult)) == 0 {
		return apperrors.Validation("raw result is required")
	}
	preferredLanguage := strings.TrimSpace(req.PreferredLanguage)
	if preferredLanguage != "" && preferredLanguage != "jmespath" && preferredLanguage != "jsonata" {
		return apperrors.Validation("preferred language must be 'jmespath' or 'jsonata'")
	}
	return nil
}

func analyzeTransformInput(raw []byte, current exporter.TransformConfig) (transformInputDiagnostics, error) {
	records, err := decodeTransformSampleRecords(raw, transformSampleRecordLimit)
	if err != nil {
		return transformInputDiagnostics{}, err
	}
	if len(records) == 0 {
		return transformInputDiagnostics{}, apperrors.Validation("result contains no records")
	}

	issues := []string{}
	if exporter.HasMeaningfulTransform(current) {
		if err := exporter.ValidateTransformConfig(current); err != nil {
			issues = append(issues, "current transform is invalid: "+apperrors.SafeMessage(err))
		} else if _, err := exporter.ApplyTransformConfig(recordsAsAny(records), current); err != nil {
			issues = append(issues, "current transform failed against the sample records: "+err.Error())
		}
	}

	return transformInputDiagnostics{
		Issues:        dedupeIssues(issues),
		SampleRecords: records,
		SampleFields:  collectTransformSampleFields(records),
	}, nil
}

func validateTransformCandidate(config exporter.TransformConfig, records []map[string]any, preferredLanguage string) ([]any, []string) {
	issues := []string{}
	if err := exporter.ValidateTransformConfig(config); err != nil {
		issues = append(issues, apperrors.SafeMessage(err))
		return nil, dedupeIssues(issues)
	}
	if preferredLanguage != "" && config.Language != preferredLanguage {
		issues = append(issues, fmt.Sprintf("transform.language must be %s", preferredLanguage))
	}
	preview, err := exporter.ApplyTransformConfig(recordsAsAny(records), config)
	if err != nil {
		issues = append(issues, "transform did not run against the sample records: "+err.Error())
	}
	return preview, dedupeIssues(issues)
}

func decodeTransformSampleRecords(raw []byte, limit int) ([]map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var items []map[string]any
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "decode result array", err)
		}
		if len(items) > limit {
			items = items[:limit]
		}
		return items, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	records := make([]map[string]any, 0, limit)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "decode result record", err)
		}
		records = append(records, item)
		if len(records) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "scan result records", err)
	}
	return records, nil
}

func collectTransformSampleFields(records []map[string]any) []piai.TransformSampleField {
	paths := map[string]*sampleSet{}
	for _, record := range records {
		collectTransformFieldValues(record, "", 0, paths)
	}
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]piai.TransformSampleField, 0, len(keys))
	for _, key := range keys {
		entry := paths[key]
		field := piai.TransformSampleField{Path: key}
		if len(entry.values) > 0 {
			field.SampleValues = append([]string(nil), entry.values...)
		}
		out = append(out, field)
	}
	return out
}

func collectTransformFieldValues(value any, prefix string, depth int, paths map[string]*sampleSet) {
	if depth >= transformFieldDepthLimit {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			collectTransformFieldValues(typed[key], next, depth+1, paths)
		}
	case []any:
		arrayPath := prefix + "[]"
		if prefix != "" {
			addTransformSampleValue(paths, arrayPath, fmt.Sprintf("%d item(s)", len(typed)))
		}
		for idx, item := range typed {
			if idx >= transformSampleValueLimit {
				break
			}
			collectTransformFieldValues(item, arrayPath, depth+1, paths)
		}
	default:
		if prefix != "" {
			addTransformSampleValue(paths, prefix, stringifyTransformSampleValue(typed))
		}
	}
}

func addTransformSampleValue(paths map[string]*sampleSet, path string, value string) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return
	}
	trimmedValue := strings.TrimSpace(value)
	entry, ok := paths[trimmedPath]
	if !ok {
		entry = &sampleSet{seen: map[string]struct{}{}}
		paths[trimmedPath] = entry
	}
	if trimmedValue == "" {
		return
	}
	if _, exists := entry.seen[trimmedValue]; exists {
		return
	}
	entry.seen[trimmedValue] = struct{}{}
	if len(entry.values) < transformSampleValueLimit {
		entry.values = append(entry.values, trimmedValue)
	}
}

func stringifyTransformSampleValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return "null"
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func recordsAsAny(records []map[string]any) []any {
	out := make([]any, 0, len(records))
	for _, record := range records {
		out = append(out, record)
	}
	return out
}

func cloneSampleRecords(records []map[string]any) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	data, err := json.Marshal(records)
	if err != nil {
		return nil
	}
	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func cloneTransformSampleFields(fields []piai.TransformSampleField) []piai.TransformSampleField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]piai.TransformSampleField, 0, len(fields))
	for _, field := range fields {
		out = append(out, piai.TransformSampleField{
			Path:         strings.TrimSpace(field.Path),
			SampleValues: append([]string(nil), field.SampleValues...),
		})
	}
	return out
}

func bridgeTransformConfig(config exporter.TransformConfig) piai.BridgeTransformConfig {
	config = exporter.NormalizeTransformConfig(config)
	return piai.BridgeTransformConfig{
		Expression: config.Expression,
		Language:   config.Language,
	}
}

func transformConfig(config piai.BridgeTransformConfig) exporter.TransformConfig {
	return exporter.NormalizeTransformConfig(exporter.TransformConfig{
		Expression: config.Expression,
		Language:   config.Language,
	})
}

func limitPreview(values []any, limit int) []any {
	if len(values) == 0 {
		return nil
	}
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
