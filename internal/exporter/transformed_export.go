package exporter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xuri/excelize/v2"
)

// ExportStreamWithShapeAndTransform exports job results with optional deterministic
// shaping and optional pre-export transformation. Transform and shape are mutually
// exclusive because shape field references target the canonical persisted result
// contract while transform output can be arbitrary.
func ExportStreamWithShapeAndTransform(job model.Job, r io.Reader, format string, shape ShapeConfig, transform TransformConfig, w io.Writer) error {
	shape = NormalizeShapeConfig(shape)
	transform = NormalizeTransformConfig(transform)
	if err := ValidateTransformConfig(transform); err != nil {
		return err
	}
	if HasMeaningfulTransform(transform) {
		if HasMeaningfulShape(shape) {
			return apperrors.Validation("export shape and transform cannot be combined")
		}
		results, err := loadResultsForTransform(job, r)
		if err != nil {
			return err
		}
		transformed, err := ApplyTransformConfig(results, transform)
		if err != nil {
			return err
		}
		return exportTransformedResults(format, transformed, w)
	}
	return ExportStreamWithShape(job, r, format, shape, w)
}

// ExportWithShapeAndTransform exports job results with optional deterministic
// shaping and pre-export transformation.
func ExportWithShapeAndTransform(job model.Job, raw []byte, format string, shape ShapeConfig, transform TransformConfig) (string, error) {
	var buf strings.Builder
	if err := ExportStreamWithShapeAndTransform(job, bytes.NewReader(raw), format, shape, transform, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func loadResultsForTransform(job model.Job, r io.Reader) ([]any, error) {
	switch job.Kind {
	case model.KindScrape, model.KindResearch:
		item, err := parseSingleReader[any](r)
		if err != nil {
			return nil, err
		}
		return []any{item}, nil
	case model.KindCrawl:
		items, err := parseLinesReader[any](r)
		if err != nil {
			return nil, err
		}
		results := make([]any, 0, len(items))
		for _, item := range items {
			results = append(results, item)
		}
		return results, nil
	default:
		return nil, apperrors.Internal("unknown job kind")
	}
}

// ExportTransformedResults exports already transformed results in the requested format.
func ExportTransformedResults(format string, results []any, w io.Writer) error {
	return exportTransformedResults(strings.TrimSpace(format), results, w)
}

func exportTransformedResults(format string, results []any, w io.Writer) error {
	switch strings.TrimSpace(format) {
	case "jsonl":
		return exportTransformedJSONL(results, w)
	case "json":
		return exportTransformedJSON(results, w)
	case "csv":
		return exportTransformedCSV(results, w)
	case "md":
		return exportTransformedMarkdown(results, w)
	case "xlsx":
		return exportTransformedXLSX(results, w)
	default:
		return apperrors.Validation("unsupported format for transformed export: " + format)
	}
}

func exportTransformedJSONL(results []any, w io.Writer) error {
	encoder := json.NewEncoder(w)
	for _, result := range results {
		if err := encoder.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func exportTransformedJSON(results []any, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func exportTransformedCSV(results []any, w io.Writer) error {
	return writeGenericCSVResults(results, w)
}

func exportTransformedMarkdown(results []any, w io.Writer) error {
	return writeGenericMarkdownResults(results, w)
}

func exportTransformedXLSX(results []any, w io.Writer) error {
	return writeGenericXLSXResults(results, w)
}

func writeGenericCSVResults(results []any, w io.Writer) error {
	if len(results) == 0 {
		return nil
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	allKeys := make(map[string]bool)
	for _, result := range results {
		if m, ok := result.(map[string]any); ok {
			for key := range m {
				allKeys[key] = true
			}
		}
	}

	keys := make([]string, 0, len(allKeys))
	for key := range allKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		for _, result := range results {
			jsonBytes, _ := json.Marshal(result)
			if err := writer.Write([]string{string(jsonBytes)}); err != nil {
				return err
			}
		}
		return writer.Error()
	}

	if err := writer.Write(keys); err != nil {
		return err
	}

	for _, result := range results {
		row := make([]string, len(keys))
		if m, ok := result.(map[string]any); ok {
			for i, key := range keys {
				if value, exists := m[key]; exists {
					switch typed := value.(type) {
					case string:
						row[i] = typed
					case nil:
						row[i] = ""
					default:
						row[i] = fmt.Sprint(typed)
					}
				}
			}
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return writer.Error()
}

func writeGenericMarkdownResults(results []any, w io.Writer) error {
	fmt.Fprint(w, "# Transformed Results\n\n")

	for i, result := range results {
		fmt.Fprintf(w, "## Item %d\n\n", i+1)

		switch value := result.(type) {
		case map[string]any:
			keys := make([]string, 0, len(value))
			for key := range value {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, key := range keys {
				switch typed := value[key].(type) {
				case string:
					fmt.Fprintf(w, "- **%s**: %s\n", key, typed)
				case nil:
					fmt.Fprintf(w, "- **%s**: (null)\n", key)
				default:
					fmt.Fprintf(w, "- **%s**: %v\n", key, typed)
				}
			}
		default:
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintf(w, "```json\n%s\n```\n", string(jsonBytes))
		}

		fmt.Fprint(w, "\n")
	}

	return nil
}

func writeGenericXLSXResults(results []any, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	if len(results) == 0 {
		return f.Write(w)
	}

	allKeys := make(map[string]bool)
	for _, result := range results {
		if m, ok := result.(map[string]any); ok {
			for key := range m {
				allKeys[key] = true
			}
		}
	}

	keys := make([]string, 0, len(allKeys))
	for key := range allKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		f.SetCellValue(sheetName, "A1", "Data")
		for i, result := range results {
			jsonBytes, _ := json.Marshal(result)
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+2), string(jsonBytes))
		}
		return f.Write(w)
	}

	for i, key := range keys {
		cell := fmt.Sprintf("%s1", columnName(i))
		f.SetCellValue(sheetName, cell, key)
	}

	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9D9D9"},
			Pattern: 1,
		},
	})
	if err != nil {
		return err
	}
	f.SetRowStyle(sheetName, 1, 1, style)

	for rowIdx, result := range results {
		if m, ok := result.(map[string]any); ok {
			for colIdx, key := range keys {
				if value, exists := m[key]; exists {
					f.SetCellValue(sheetName, fmt.Sprintf("%s%d", columnName(colIdx), rowIdx+2), value)
				}
			}
		}
	}

	for i, key := range keys {
		col := columnName(i)
		width := float64(len(key)) * 1.5
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		f.SetColWidth(sheetName, col, col, width)
	}

	return f.Write(w)
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+(index%26))) + name
		index = index/26 - 1
	}
	return name
}
