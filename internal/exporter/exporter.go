// Package exporter provides functionality for exporting job results to various formats.
//
// This package supports JSON, JSONL, Markdown, CSV, XLSX, and Parquet output formats with both
// buffered and streaming interfaces for memory-efficient processing of large results.
//
// Public API:
// - Export: Export job results and return as string
// - ExportStream: Stream export job results to writer
//
// This file contains only the public API entry points. Format-specific logic,
// result types, parsing helpers, and pagination utilities are split into separate
// files to maintain focus and keep files under 400 LOC.
package exporter

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// Export exports job results to the specified format and returns the output as a string.
// For large result files, consider using ExportStream instead to avoid loading the entire
// output into memory.
func Export(job model.Job, raw []byte, format string) (string, error) {
	var buf strings.Builder
	if err := ExportStream(job, bytes.NewReader(raw), format, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExportStream exports job results to the specified format, writing the output directly
// to the provided writer. This is more memory-efficient for large result files as it
// streams the input and processes it incrementally where possible.
func ExportStream(job model.Job, r io.Reader, format string, w io.Writer) error {
	switch format {
	case "json":
		return exportJSONStream(job, r, w)
	case "jsonl":
		return exportJSONLStream(r, w)
	case "md":
		return exportMarkdownStream(job, r, w)
	case "csv":
		return exportCSVStream(job, r, w)
	case "xlsx":
		return exportXLSXStream(job, r, w)
	case "parquet":
		return exportParquetStream(job, r, w)
	case "har":
		return exportHARStream(job, r, w)
	default:
		return apperrors.Validation(fmt.Sprintf("unsupported format: %s", format))
	}
}

// ExportStreamWithDatabase exports job results to the specified format.
// For database formats (postgres, mysql, mongodb), the writer is ignored and data is written
// directly to the configured database. For file-based formats, behaves like ExportStream.
func ExportStreamWithDatabase(job model.Job, r io.Reader, format string, w io.Writer, dbCfg *DatabaseExportConfig) error {
	switch format {
	case "json":
		return exportJSONStream(job, r, w)
	case "jsonl":
		return exportJSONLStream(r, w)
	case "md":
		return exportMarkdownStream(job, r, w)
	case "csv":
		return exportCSVStream(job, r, w)
	case "xlsx":
		return exportXLSXStream(job, r, w)
	case "parquet":
		return exportParquetStream(job, r, w)
	case "har":
		return exportHARStream(job, r, w)
	case "postgres":
		if dbCfg == nil {
			return apperrors.Validation("database config required for postgres export")
		}
		return exportPostgresStream(job, r, *dbCfg)
	case "mysql":
		if dbCfg == nil {
			return apperrors.Validation("database config required for mysql export")
		}
		return exportMySQLStream(job, r, *dbCfg)
	case "mongodb":
		if dbCfg == nil {
			return apperrors.Validation("database config required for mongodb export")
		}
		return exportMongoDBStream(job, r, *dbCfg)
	default:
		return apperrors.Validation(fmt.Sprintf("unsupported format: %s", format))
	}
}
