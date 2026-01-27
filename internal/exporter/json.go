// Package exporter provides JSON export implementation.
//
// JSON export transforms job results into indented JSON format.
// Functions include:
// - exportJSONStream: Stream export to JSON with reader/writer
//
// This file does NOT handle other formats (JSONL, Markdown, CSV).
package exporter

import (
	"encoding/json"
	"io"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// exportJSONStream exports job results to JSON format with streaming.
// For JSON output, we need to parse the input first since we need to
// re-encode it as proper JSON (not JSONL). This still benefits from
// streaming the input parsing, but the final output is buffered.
func exportJSONStream(job model.Job, r io.Reader, w io.Writer) error {
	var data []byte
	var err error

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(item, "", "  ")
		if err != nil {
			return err
		}
	case model.KindCrawl:
		items, err := parseLinesReader[CrawlResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(item, "", "  ")
		if err != nil {
			return err
		}
	default:
		return apperrors.Internal("unknown job kind")
	}

	_, err = w.Write(data)
	return err
}
