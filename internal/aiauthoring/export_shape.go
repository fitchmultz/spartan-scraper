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

type ExportShapeRequest struct {
	JobKind      model.Kind
	Format       string
	RawResult    []byte
	CurrentShape exporter.ShapeConfig
	Instructions string
}

type ExportShapeInputStats struct {
	FieldOptionCount     int `json:"fieldOptionCount"`
	TopLevelFieldCount   int `json:"topLevelFieldCount"`
	NormalizedFieldCount int `json:"normalizedFieldCount"`
	EvidenceFieldCount   int `json:"evidenceFieldCount"`
	SampleRecordCount    int `json:"sampleRecordCount"`
}

type ExportShapeResult struct {
	Issues      []string              `json:"issues,omitempty"`
	InputStats  ExportShapeInputStats `json:"inputStats"`
	Shape       exporter.ShapeConfig  `json:"shape"`
	Explanation string                `json:"explanation,omitempty"`
	RouteID     string                `json:"route_id,omitempty"`
	Provider    string                `json:"provider,omitempty"`
	Model       string                `json:"model,omitempty"`
}

type exportShapeDiagnostics struct {
	Issues             []string
	FieldOptions       []piai.ExportShapeFieldOption
	AllowedTopLevel    map[string]struct{}
	AllowedNormalized  map[string]struct{}
	AllowedEvidence    map[string]struct{}
	AllowedSummary     map[string]struct{}
	AllowedLabels      map[string]struct{}
	TopLevelFieldCount int
	NormalizedCount    int
	EvidenceCount      int
	SampleRecordCount  int
}

func (s *Service) GenerateExportShape(ctx context.Context, req ExportShapeRequest) (ExportShapeResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return ExportShapeResult{}, err
	}
	if err := validateExportShapeInput(req); err != nil {
		return ExportShapeResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	diagnostics, err := analyzeExportShapeInput(req.JobKind, req.RawResult)
	if err != nil {
		return ExportShapeResult{}, err
	}
	aiReq := piai.ExportShapeRequest{
		JobKind:      string(req.JobKind),
		Format:       strings.TrimSpace(req.Format),
		FieldOptions: diagnostics.FieldOptions,
		CurrentShape: bridgeExportShapeConfig(req.CurrentShape),
		Instructions: strings.TrimSpace(req.Instructions),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GenerateExportShape(ctx, aiReq)
		if err != nil {
			return ExportShapeResult{}, apperrors.Wrap(apperrors.KindInternal, "AI export shaping failed", err)
		}
		candidate := exporter.NormalizeShapeConfig(exportShapeConfig(aiResult.Shape))
		issues := validateExportShapeCandidate(req.JobKind, req.Format, candidate, diagnostics)
		if len(issues) == 0 {
			return ExportShapeResult{
				Issues: dedupeIssues(diagnostics.Issues),
				InputStats: ExportShapeInputStats{
					FieldOptionCount:     len(diagnostics.FieldOptions),
					TopLevelFieldCount:   diagnostics.TopLevelFieldCount,
					NormalizedFieldCount: diagnostics.NormalizedCount,
					EvidenceFieldCount:   diagnostics.EvidenceCount,
					SampleRecordCount:    diagnostics.SampleRecordCount,
				},
				Shape:       candidate,
				Explanation: strings.TrimSpace(aiResult.Explanation),
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}
		if attempt == 1 {
			return ExportShapeResult{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = joinFeedback(aiReq.Feedback, strings.Join(issues, "; "))
	}

	return ExportShapeResult{}, apperrors.Internal("AI export shaping failed")
}

func validateExportShapeInput(req ExportShapeRequest) error {
	if req.JobKind == "" {
		return apperrors.Validation("job kind is required")
	}
	if strings.TrimSpace(req.Format) == "" {
		return apperrors.Validation("format is required")
	}
	if !exporter.SupportsShapeFormat(strings.TrimSpace(req.Format)) {
		return apperrors.Validation("export shaping is supported only for md, csv, and xlsx formats")
	}
	if len(bytes.TrimSpace(req.RawResult)) == 0 {
		return apperrors.Validation("raw result is required")
	}
	return nil
}

func analyzeExportShapeInput(kind model.Kind, raw []byte) (exportShapeDiagnostics, error) {
	switch kind {
	case model.KindScrape:
		return analyzeScrapeExportShapeInput(raw)
	case model.KindCrawl:
		return analyzeCrawlExportShapeInput(raw)
	case model.KindResearch:
		return analyzeResearchExportShapeInput(raw)
	default:
		return exportShapeDiagnostics{}, apperrors.Validation(fmt.Sprintf("unsupported job kind for export shaping: %s", kind))
	}
}

func analyzeScrapeExportShapeInput(raw []byte) (exportShapeDiagnostics, error) {
	var item exporter.ScrapeResult
	if err := json.Unmarshal(bytes.TrimSpace(raw), &item); err != nil {
		return exportShapeDiagnostics{}, apperrors.Wrap(apperrors.KindValidation, "decode scrape result", err)
	}
	diagnostics := newExportShapeDiagnostics(1)
	addFieldOption(&diagnostics, "url", "top_level", defaultShapeLabel("url"), item.URL)
	addFieldOption(&diagnostics, "status", "top_level", defaultShapeLabel("status"), fmt.Sprintf("%d", item.Status))
	addFieldOption(&diagnostics, "title", "top_level", defaultShapeLabel("title"), firstNonEmptyString(item.Normalized.Title, item.Title))
	addFieldOption(&diagnostics, "description", "top_level", defaultShapeLabel("description"), firstNonEmptyString(item.Normalized.Description, item.Metadata.Description))
	addFieldOption(&diagnostics, "text", "top_level", defaultShapeLabel("text"), firstNonEmptyString(item.Normalized.Text, item.Text))
	for _, key := range sortedKeys(item.Normalized.Fields) {
		addFieldOption(&diagnostics, "field."+key, "normalized", defaultShapeLabel("field."+key), strings.Join(item.Normalized.Fields[key].Values, "; "))
	}
	return diagnostics, nil
}

func analyzeCrawlExportShapeInput(raw []byte) (exportShapeDiagnostics, error) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	diagnostics := newExportShapeDiagnostics(0)
	normalizedSamples := map[string]string{}
	count := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var item exporter.CrawlResult
		if err := json.Unmarshal(line, &item); err != nil {
			return exportShapeDiagnostics{}, apperrors.Wrap(apperrors.KindValidation, "decode crawl result", err)
		}
		count++
		if count == 1 {
			addFieldOption(&diagnostics, "url", "top_level", defaultShapeLabel("url"), item.URL)
			addFieldOption(&diagnostics, "status", "top_level", defaultShapeLabel("status"), fmt.Sprintf("%d", item.Status))
			addFieldOption(&diagnostics, "title", "top_level", defaultShapeLabel("title"), firstNonEmptyString(item.Normalized.Title, item.Title))
			addFieldOption(&diagnostics, "text", "top_level", defaultShapeLabel("text"), firstNonEmptyString(item.Normalized.Text, item.Text))
		}
		for name, value := range item.Normalized.Fields {
			if _, ok := normalizedSamples[name]; ok {
				continue
			}
			normalizedSamples[name] = strings.Join(value.Values, "; ")
		}
		if count >= 100 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return exportShapeDiagnostics{}, apperrors.Wrap(apperrors.KindInternal, "scan crawl result", err)
	}
	if count == 0 {
		return exportShapeDiagnostics{}, apperrors.Validation("crawl result contains no records")
	}
	diagnostics.SampleRecordCount = count
	for _, key := range sortedMapKeys(normalizedSamples) {
		addFieldOption(&diagnostics, "field."+key, "normalized", defaultShapeLabel("field."+key), normalizedSamples[key])
	}
	return diagnostics, nil
}

func analyzeResearchExportShapeInput(raw []byte) (exportShapeDiagnostics, error) {
	var item exporter.ResearchResult
	if err := json.Unmarshal(bytes.TrimSpace(raw), &item); err != nil {
		return exportShapeDiagnostics{}, apperrors.Wrap(apperrors.KindValidation, "decode research result", err)
	}
	diagnostics := newExportShapeDiagnostics(1)
	addFieldOption(&diagnostics, "query", "top_level", defaultShapeLabel("query"), item.Query)
	addFieldOption(&diagnostics, "summary", "top_level", defaultShapeLabel("summary"), item.Summary)
	addFieldOption(&diagnostics, "confidence", "top_level", defaultShapeLabel("confidence"), fmt.Sprintf("%.2f", item.Confidence))
	if item.Agentic != nil {
		addFieldOption(&diagnostics, "agentic.status", "top_level", defaultShapeLabel("agentic.status"), item.Agentic.Status)
		addFieldOption(&diagnostics, "agentic.summary", "top_level", defaultShapeLabel("agentic.summary"), item.Agentic.Summary)
		addFieldOption(&diagnostics, "agentic.objective", "top_level", defaultShapeLabel("agentic.objective"), item.Agentic.Objective)
		addFieldOption(&diagnostics, "agentic.error", "top_level", defaultShapeLabel("agentic.error"), item.Agentic.Error)
	}
	if len(item.Evidence) == 0 {
		diagnostics.Issues = append(diagnostics.Issues, "research result contains no evidence items")
	}
	var sampleEvidence struct {
		URL         string  `json:"url"`
		Title       string  `json:"title"`
		Snippet     string  `json:"snippet"`
		Score       float64 `json:"score"`
		SimHash     uint64  `json:"simhash"`
		ClusterID   string  `json:"clusterId"`
		Confidence  float64 `json:"confidence"`
		CitationURL string  `json:"citationUrl"`
	}
	if len(item.Evidence) > 0 {
		sampleEvidence = item.Evidence[0]
	}
	addFieldOption(&diagnostics, "evidence.url", "evidence", defaultShapeLabel("evidence.url"), sampleEvidence.URL)
	addFieldOption(&diagnostics, "evidence.title", "evidence", defaultShapeLabel("evidence.title"), sampleEvidence.Title)
	addFieldOption(&diagnostics, "evidence.score", "evidence", defaultShapeLabel("evidence.score"), fmt.Sprintf("%.2f", sampleEvidence.Score))
	addFieldOption(&diagnostics, "evidence.confidence", "evidence", defaultShapeLabel("evidence.confidence"), fmt.Sprintf("%.2f", sampleEvidence.Confidence))
	addFieldOption(&diagnostics, "evidence.clusterId", "evidence", defaultShapeLabel("evidence.clusterId"), sampleEvidence.ClusterID)
	addFieldOption(&diagnostics, "evidence.citationUrl", "evidence", defaultShapeLabel("evidence.citationUrl"), sampleEvidence.CitationURL)
	addFieldOption(&diagnostics, "evidence.snippet", "evidence", defaultShapeLabel("evidence.snippet"), sampleEvidence.Snippet)
	return diagnostics, nil
}

func newExportShapeDiagnostics(sampleCount int) exportShapeDiagnostics {
	return exportShapeDiagnostics{
		AllowedTopLevel:   map[string]struct{}{},
		AllowedNormalized: map[string]struct{}{},
		AllowedEvidence:   map[string]struct{}{},
		AllowedSummary:    map[string]struct{}{},
		AllowedLabels:     map[string]struct{}{},
		SampleRecordCount: sampleCount,
	}
}

func addFieldOption(diagnostics *exportShapeDiagnostics, key string, category string, label string, sample string) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return
	}
	option := piai.ExportShapeFieldOption{
		Key:      trimmedKey,
		Category: category,
		Label:    strings.TrimSpace(label),
	}
	if sample = strings.TrimSpace(sample); sample != "" {
		option.SampleValues = []string{sample}
	}
	diagnostics.FieldOptions = append(diagnostics.FieldOptions, option)
	diagnostics.AllowedLabels[trimmedKey] = struct{}{}
	switch category {
	case "top_level":
		diagnostics.AllowedTopLevel[trimmedKey] = struct{}{}
		diagnostics.AllowedSummary[trimmedKey] = struct{}{}
		diagnostics.TopLevelFieldCount++
	case "normalized":
		diagnostics.AllowedNormalized[trimmedKey] = struct{}{}
		diagnostics.AllowedSummary[trimmedKey] = struct{}{}
		diagnostics.NormalizedCount++
	case "evidence":
		diagnostics.AllowedEvidence[trimmedKey] = struct{}{}
		diagnostics.EvidenceCount++
	}
}

func validateExportShapeCandidate(kind model.Kind, format string, shape exporter.ShapeConfig, diagnostics exportShapeDiagnostics) []string {
	issues := []string{}
	if !exporter.SupportsShapeFormat(strings.TrimSpace(format)) {
		issues = append(issues, "shape format must be md, csv, or xlsx")
	}
	if !exporter.HasMeaningfulShape(shape) {
		issues = append(issues, "shape must include at least one field selection, label, or formatting hint")
	}
	issues = append(issues, validateFieldRefs(shape.TopLevelFields, diagnostics.AllowedTopLevel, "shape.topLevelFields")...)
	issues = append(issues, validateFieldRefs(shape.NormalizedFields, diagnostics.AllowedNormalized, "shape.normalizedFields")...)
	issues = append(issues, validateFieldRefs(shape.EvidenceFields, diagnostics.AllowedEvidence, "shape.evidenceFields")...)
	issues = append(issues, validateFieldRefs(shape.SummaryFields, diagnostics.AllowedSummary, "shape.summaryFields")...)
	for key := range shape.FieldLabels {
		if _, ok := diagnostics.AllowedLabels[key]; !ok {
			issues = append(issues, fmt.Sprintf("shape.fieldLabels[%q] must reference an available field", key))
		}
	}
	if kind != model.KindResearch && len(shape.EvidenceFields) > 0 {
		issues = append(issues, "shape.evidenceFields is only valid for research exports")
	}
	if kind == model.KindResearch && len(shape.NormalizedFields) > 0 {
		issues = append(issues, "shape.normalizedFields is not used for research exports")
	}
	return dedupeIssues(issues)
}

func validateFieldRefs(values []string, allowed map[string]struct{}, label string) []string {
	issues := []string{}
	for idx, value := range values {
		if _, ok := allowed[value]; !ok {
			issues = append(issues, fmt.Sprintf("%s[%d] must reference an available field", label, idx))
		}
	}
	return issues
}

func bridgeExportShapeConfig(shape exporter.ShapeConfig) piai.BridgeExportShapeConfig {
	shape = exporter.NormalizeShapeConfig(shape)
	return piai.BridgeExportShapeConfig{
		TopLevelFields:   append([]string(nil), shape.TopLevelFields...),
		NormalizedFields: append([]string(nil), shape.NormalizedFields...),
		EvidenceFields:   append([]string(nil), shape.EvidenceFields...),
		SummaryFields:    append([]string(nil), shape.SummaryFields...),
		FieldLabels:      cloneStringMap(shape.FieldLabels),
		Formatting: piai.ExportFormattingHints{
			EmptyValue:     shape.Formatting.EmptyValue,
			MultiValueJoin: shape.Formatting.MultiValueJoin,
			MarkdownTitle:  shape.Formatting.MarkdownTitle,
		},
	}
}

func exportShapeConfig(shape piai.BridgeExportShapeConfig) exporter.ShapeConfig {
	return exporter.ShapeConfig{
		TopLevelFields:   append([]string(nil), shape.TopLevelFields...),
		NormalizedFields: append([]string(nil), shape.NormalizedFields...),
		EvidenceFields:   append([]string(nil), shape.EvidenceFields...),
		SummaryFields:    append([]string(nil), shape.SummaryFields...),
		FieldLabels:      cloneStringMap(shape.FieldLabels),
		Formatting: exporter.ExportFormattingHints{
			EmptyValue:     strings.TrimSpace(shape.Formatting.EmptyValue),
			MultiValueJoin: strings.TrimSpace(shape.Formatting.MultiValueJoin),
			MarkdownTitle:  strings.TrimSpace(shape.Formatting.MarkdownTitle),
		},
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func sortedKeys[T any](input map[string]T) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys(input map[string]string) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func defaultShapeLabel(key string) string {
	replacer := strings.NewReplacer("field.", "", "evidence.", "", ".", " ", "_", " ")
	trimmed := strings.TrimSpace(replacer.Replace(key))
	if trimmed == "" {
		return key
	}
	parts := strings.Fields(trimmed)
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
