// Package exporter provides Parquet export implementation.
//
// Parquet export transforms job results into Apache Parquet columnar format:
// - Efficient compression (SNAPPY by default, GZIP/LZ4/ZSTD optional)
// - Schema inference from extraction fields
// - Optimized for analytics pipelines (BigQuery, Athena, Spark)
//
// This file does NOT handle other formats (JSON, JSONL, Markdown, CSV, XLSX).
package exporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
)

// ParquetScrapeRow represents a single scrape result row in Parquet format.
type ParquetScrapeRow struct {
	URL         string `parquet:"name=url, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Status      int32  `parquet:"name=status, type=INT32"`
	Title       string `parquet:"name=title, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Description string `parquet:"name=description, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Text        string `parquet:"name=text, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
}

// ParquetCrawlRow represents a single crawl result row in Parquet format.
type ParquetCrawlRow struct {
	URL    string `parquet:"name=url, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Status int32  `parquet:"name=status, type=INT32"`
	Title  string `parquet:"name=title, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Text   string `parquet:"name=text, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
}

// ParquetEvidenceRow represents a single evidence item in Parquet format.
type ParquetEvidenceRow struct {
	URL         string  `parquet:"name=url, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Title       string  `parquet:"name=title, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Snippet     string  `parquet:"name=snippet, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Score       float64 `parquet:"name=score, type=DOUBLE"`
	Confidence  float64 `parquet:"name=confidence, type=DOUBLE"`
	ClusterID   string  `parquet:"name=cluster_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	CitationURL string  `parquet:"name=citation_url, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
}

// ParquetSummaryRow represents the research summary in Parquet format.
type ParquetSummaryRow struct {
	Query      string  `parquet:"name=query, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Summary    string  `parquet:"name=summary, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Confidence float64 `parquet:"name=confidence, type=DOUBLE"`
}

// exportParquetStream exports job results to Parquet format with streaming.
func exportParquetStream(job model.Job, r io.Reader, w io.Writer) error {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return writeScrapeParquet(item, w)
	case model.KindCrawl:
		rs, cleanup, err := ensureSeekable(r)
		if err != nil {
			return err
		}
		defer cleanup()
		return writeCrawlParquetStream(rs, w)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchParquet(item, w)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// writeScrapeParquet writes a single scrape result to Parquet format.
func writeScrapeParquet(item ScrapeResult, w io.Writer) error {
	// Create Parquet writer with SNAPPY compression
	pw, err := writer.NewParquetWriterFromWriter(w, new(ParquetScrapeRow), 1)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create parquet writer", err)
	}
	defer pw.WriteStop()

	// Configure compression
	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	// Prepare data
	title := item.Title
	desc := item.Metadata.Description
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}

	row := ParquetScrapeRow{
		URL:         item.URL,
		Status:      int32(item.Status),
		Title:       title,
		Description: desc,
		Text:        item.Text,
	}

	if err := pw.Write(row); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write parquet row", err)
	}

	return nil
}

// writeCrawlParquetStream writes multiple crawl results to Parquet format using two-pass streaming.
func writeCrawlParquetStream(rs io.ReadSeeker, w io.Writer) error {
	// Create Parquet writer
	pw, err := writer.NewParquetWriterFromWriter(w, new(ParquetCrawlRow), 1)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create parquet writer", err)
	}
	defer pw.WriteStop()

	// Configure compression
	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	// Write all rows
	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
		title := item.Title
		if item.Normalized.Title != "" {
			title = item.Normalized.Title
		}

		row := ParquetCrawlRow{
			URL:    item.URL,
			Status: int32(item.Status),
			Title:  title,
			Text:   item.Text,
		}

		if err := pw.Write(row); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to write parquet row", err)
		}
		return nil
	})

	return err
}

// writeResearchParquet writes a research result to Parquet format with evidence rows.
func writeResearchParquet(item ResearchResult, w io.Writer) error {
	// For research jobs, we create a single Parquet file with evidence rows
	// This is the most useful format for analytics pipelines
	pw, err := writer.NewParquetWriterFromWriter(w, new(ParquetEvidenceRow), 1)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create parquet writer", err)
	}
	defer pw.WriteStop()

	// Configure compression
	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	// Write evidence rows
	for _, ev := range item.Evidence {
		row := ParquetEvidenceRow{
			URL:         ev.URL,
			Title:       ev.Title,
			Snippet:     ev.Snippet,
			Score:       ev.Score,
			Confidence:  ev.Confidence,
			ClusterID:   ev.ClusterID,
			CitationURL: ev.CitationURL,
		}

		if err := pw.Write(row); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to write parquet row", err)
		}
	}

	return nil
}

// ParquetCrawlRowWithFields represents a crawl row with dynamic fields.
type ParquetCrawlRowWithFields struct {
	URL    string            `parquet:"name=url, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Status int32             `parquet:"name=status, type=INT32"`
	Title  string            `parquet:"name=title, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Text   string            `parquet:"name=text, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Fields map[string]string `parquet:"name=fields, type=MAP, convertedtype=MAP, keytype=BYTE_ARRAY, keyconvertedtype=UTF8, valuetype=BYTE_ARRAY, valconvertedtype=UTF8"`
}

// writeCrawlParquetStreamWithFields writes crawl results with dynamic fields as a map column.
// This is an alternative implementation that preserves extraction fields in a map column.
//
//nolint:unused // Reserved for future use if field-level schema is needed
func writeCrawlParquetStreamWithFields(rs io.ReadSeeker, w io.Writer) error {
	// Collect unique field keys
	fieldSet := make(map[string]bool)
	var allRows []CrawlResult

	err := scanReader[CrawlResult](rs, func(item CrawlResult) error {
		for k := range item.Normalized.Fields {
			fieldSet[k] = true
		}
		allRows = append(allRows, item)
		return nil
	})
	if err != nil {
		return err
	}

	// Sort field names for consistent ordering
	fieldNames := make([]string, 0, len(fieldSet))
	for k := range fieldSet {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	// If we have dynamic fields, include them in the text as JSON for now
	// Full dynamic schema support would require generating struct types at runtime
	pw, err := writer.NewParquetWriterFromWriter(w, new(ParquetCrawlRow), 1)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create parquet writer", err)
	}
	defer pw.WriteStop()

	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	for _, item := range allRows {
		title := item.Title
		if item.Normalized.Title != "" {
			title = item.Normalized.Title
		}

		// Append field values to text for preservation
		text := item.Text
		if len(item.Normalized.Fields) > 0 {
			var fieldParts []string
			for _, k := range fieldNames {
				if v, ok := item.Normalized.Fields[k]; ok {
					fieldParts = append(fieldParts, fmt.Sprintf("%s: %s", k, strings.Join(v.Values, "; ")))
				}
			}
			if len(fieldParts) > 0 {
				text = text + "\n\nFields:\n" + strings.Join(fieldParts, "\n")
			}
		}

		row := ParquetCrawlRow{
			URL:    item.URL,
			Status: int32(item.Status),
			Title:  title,
			Text:   text,
		}

		if err := pw.Write(row); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to write parquet row", err)
		}
	}

	return nil
}
