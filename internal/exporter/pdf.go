// Package exporter provides PDF export implementation.
//
// PDF export transforms job results into PDF format using headless browser
// capabilities via Chromedp. It generates professional-looking PDF documents
// from scrape, crawl, and research results with proper formatting and layout.
//
// This file does NOT handle other formats (JSON, JSONL, Markdown, CSV).
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// exportPDFStream exports job results to PDF format with streaming.
// Uses Chromedp for PDF generation (always available).
func exportPDFStream(job model.Job, r io.Reader, w io.Writer) error {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return generateScrapePDF(item, w)
	case model.KindCrawl:
		items, err := parseLinesReader[CrawlResult](r)
		if err != nil {
			return err
		}
		return generateCrawlPDF(items, w)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return generateResearchPDF(item, w)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// generateScrapePDF generates a PDF from a single scrape result.
func generateScrapePDF(item ScrapeResult, w io.Writer) error {
	html := buildScrapeHTML(item)
	return generatePDF(context.Background(), html, w)
}

// generateCrawlPDF generates a PDF from crawl results.
func generateCrawlPDF(items []CrawlResult, w io.Writer) error {
	html := buildCrawlHTML(items)
	return generatePDF(context.Background(), html, w)
}

// generateResearchPDF generates a PDF from research results.
func generateResearchPDF(item ResearchResult, w io.Writer) error {
	html := buildResearchHTML(item)
	return generatePDF(context.Background(), html, w)
}

// buildScrapeHTML builds HTML content for a scrape result.
func buildScrapeHTML(item ScrapeResult) string {
	title := item.Title
	desc := item.Metadata.Description
	text := item.Text
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}
	if item.Normalized.Text != "" {
		text = item.Normalized.Text
	}

	var fieldsHTML strings.Builder
	if len(item.Normalized.Fields) > 0 {
		fieldsHTML.WriteString("<h2>Extracted Fields</h2><table class='fields'>")
		for k, v := range item.Normalized.Fields {
			values := strings.Join(v.Values, ", ")
			fieldsHTML.WriteString(fmt.Sprintf("<tr><th>%s</th><td>%s</td></tr>", escapeHTML(k), escapeHTML(values)))
		}
		fieldsHTML.WriteString("</table>")
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>%s</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; max-width: 800px; margin: 40px auto; padding: 20px; color: #333; }
h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
h2 { color: #34495e; margin-top: 30px; }
.meta { background: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0; }
.meta p { margin: 5px 0; }
table.fields { width: 100%%; border-collapse: collapse; margin: 20px 0; }
table.fields th, table.fields td { text-align: left; padding: 10px; border-bottom: 1px solid #ddd; }
table.fields th { background: #f8f9fa; width: 30%%; }
.content { white-space: pre-wrap; margin-top: 20px; }
.url { color: #3498db; word-break: break-all; }
</style>
</head>
<body>
<h1>%s</h1>
<div class="meta">
<p><strong>URL:</strong> <span class="url">%s</span></p>
<p><strong>Status:</strong> %d</p>
%s
</div>
%s
<div class="content">%s</div>
</body>
</html>`,
		escapeHTML(title),
		escapeHTML(title),
		escapeHTML(item.URL),
		item.Status,
		formatDescriptionHTML(desc),
		fieldsHTML.String(),
		escapeHTML(text),
	)
}

// buildCrawlHTML builds HTML content for crawl results.
func buildCrawlHTML(items []CrawlResult) string {
	var itemsHTML strings.Builder
	for i, item := range items {
		title := item.Title
		if item.Normalized.Title != "" {
			title = item.Normalized.Title
		}

		var fieldsHTML strings.Builder
		if len(item.Normalized.Fields) > 0 {
			fieldsHTML.WriteString("<table class='fields'>")
			for k, v := range item.Normalized.Fields {
				values := strings.Join(v.Values, ", ")
				fieldsHTML.WriteString(fmt.Sprintf("<tr><th>%s</th><td>%s</td></tr>", escapeHTML(k), escapeHTML(values)))
			}
			fieldsHTML.WriteString("</table>")
		}

		itemsHTML.WriteString(fmt.Sprintf(`
<div class="crawl-item">
<h2>%d. %s</h2>
<p><strong>URL:</strong> <span class="url">%s</span></p>
<p><strong>Status:</strong> %d</p>
%s
</div>
`, i+1, escapeHTML(title), escapeHTML(item.URL), item.Status, fieldsHTML.String()))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Crawl Results</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; max-width: 800px; margin: 40px auto; padding: 20px; color: #333; }
h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
h2 { color: #34495e; margin-top: 30px; font-size: 1.3em; }
.crawl-item { margin: 30px 0; padding: 20px; background: #f8f9fa; border-radius: 5px; page-break-inside: avoid; }
.crawl-item h2 { margin-top: 0; color: #2c3e50; }
table.fields { width: 100%%; border-collapse: collapse; margin: 15px 0; }
table.fields th, table.fields td { text-align: left; padding: 8px; border-bottom: 1px solid #ddd; }
table.fields th { background: #e9ecef; width: 30%%; font-size: 0.9em; }
.url { color: #3498db; word-break: break-all; }
.summary { background: #e8f4f8; padding: 15px; border-radius: 5px; margin: 20px 0; }
</style>
</head>
<body>
<h1>Crawl Results</h1>
<div class="summary">
<p><strong>Total Pages:</strong> %d</p>
<p><strong>Generated:</strong> %s</p>
</div>
%s
</body>
</html>`, len(items), time.Now().Format("2006-01-02 15:04:05"), itemsHTML.String())
}

// buildResearchHTML builds HTML content for research results.
func buildResearchHTML(item ResearchResult) string {
	var clustersHTML strings.Builder
	if len(item.Clusters) > 0 {
		clustersHTML.WriteString("<h2>Evidence Clusters</h2>")
		for _, cluster := range item.Clusters {
			clustersHTML.WriteString(fmt.Sprintf(`
<div class="cluster">
<h3>%s</h3>
<p>Confidence: %.2f | Items: %d</p>
</div>
`, escapeHTML(cluster.Label), cluster.Confidence, len(cluster.Evidence)))
		}
	}

	var citationsHTML strings.Builder
	if len(item.Citations) > 0 {
		citationsHTML.WriteString("<h2>Citations</h2><ol>")
		for _, citation := range item.Citations {
			target := citation.Canonical
			if citation.Anchor != "" {
				target = citation.Canonical + "#" + citation.Anchor
			}
			citationsHTML.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", escapeHTML(target), escapeHTML(target)))
		}
		citationsHTML.WriteString("</ol>")
	}

	var evidenceHTML strings.Builder
	if len(item.Evidence) > 0 {
		evidenceHTML.WriteString("<h2>Evidence</h2>")
		for _, ev := range item.Evidence {
			evidenceHTML.WriteString(fmt.Sprintf(`
<div class="evidence-item">
<h3>%s</h3>
<p class="url"><a href="%s">%s</a></p>
<p>Score: %.2f | Confidence: %.2f</p>
<p class="snippet">%s</p>
</div>
`, escapeHTML(ev.Title), escapeHTML(ev.URL), escapeHTML(ev.URL), ev.Score, ev.Confidence, escapeHTML(ev.Snippet)))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Research Report: %s</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; max-width: 800px; margin: 40px auto; padding: 20px; color: #333; }
h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
h2 { color: #34495e; margin-top: 30px; border-bottom: 1px solid #ecf0f1; padding-bottom: 8px; }
h3 { color: #2c3e50; margin-top: 20px; }
.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 8px; margin-bottom: 30px; }
.header h1 { color: white; border-bottom: none; margin: 0 0 10px 0; }
.meta { margin-top: 15px; }
.meta p { margin: 5px 0; }
.summary { background: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #3498db; }
.cluster { background: #e8f4f8; padding: 15px; margin: 10px 0; border-radius: 5px; }
.cluster h3 { margin-top: 0; color: #2980b9; }
.evidence-item { margin: 25px 0; padding: 20px; background: #f8f9fa; border-radius: 5px; page-break-inside: avoid; }
.evidence-item h3 { margin-top: 0; color: #2c3e50; font-size: 1.1em; }
.evidence-item .url { font-size: 0.9em; margin: 5px 0; }
.evidence-item .url a { color: #3498db; word-break: break-all; }
.evidence-item .snippet { color: #555; font-style: italic; margin-top: 10px; }
ol { padding-left: 20px; }
ol li { margin: 8px 0; }
ol li a { color: #3498db; word-break: break-all; }
</style>
</head>
<body>
<div class="header">
<h1>Research Report</h1>
<div class="meta">
<p><strong>Query:</strong> %s</p>
<p><strong>Confidence:</strong> %.2f</p>
<p><strong>Generated:</strong> %s</p>
</div>
</div>
<div class="summary">
<h2>Summary</h2>
<p>%s</p>
</div>
%s
%s
%s
</body>
</html>`,
		escapeHTML(item.Query),
		escapeHTML(item.Query),
		item.Confidence,
		time.Now().Format("2006-01-02 15:04:05"),
		escapeHTML(item.Summary),
		clustersHTML.String(),
		citationsHTML.String(),
		evidenceHTML.String(),
	)
}

// generatePDF generates a PDF from HTML content using Chromedp.
func generatePDF(ctx context.Context, html string, w io.Writer) error {
	// Create a new allocator context
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAlloc()

	// Create a new browser context
	browserCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Set a timeout for the PDF generation
	ctx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelTimeout()

	// Navigate to a data URL with the HTML content
	dataURL := "data:text/html;charset=utf-8," + html

	var pdfData []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(dataURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.5).
				WithPaperHeight(11).
				WithMarginTop(0.5).
				WithMarginBottom(0.5).
				WithMarginLeft(0.5).
				WithMarginRight(0.5).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to generate PDF", err)
	}

	_, err = w.Write(pdfData)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write PDF data", err)
	}

	return nil
}

// escapeHTML escapes HTML special characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// formatDescriptionHTML formats the description for HTML display.
func formatDescriptionHTML(desc string) string {
	if desc == "" {
		return ""
	}
	return fmt.Sprintf("<p><strong>Description:</strong> %s</p>", escapeHTML(desc))
}

// PDFExportConfig holds configuration for PDF export.
type PDFExportConfig struct {
	// PageSize is the paper size (A4, Letter, Legal)
	PageSize string `json:"pageSize,omitempty"`

	// Orientation is the page orientation (portrait, landscape)
	Orientation string `json:"orientation,omitempty"`

	// MarginTop is the top margin in inches
	MarginTop float64 `json:"marginTop,omitempty"`

	// MarginBottom is the bottom margin in inches
	MarginBottom float64 `json:"marginBottom,omitempty"`

	// MarginLeft is the left margin in inches
	MarginLeft float64 `json:"marginLeft,omitempty"`

	// MarginRight is the right margin in inches
	MarginRight float64 `json:"marginRight,omitempty"`

	// PrintBackground includes background graphics in the PDF
	PrintBackground bool `json:"printBackground,omitempty"`
}

// DefaultPDFConfig returns the default PDF export configuration.
func DefaultPDFConfig() PDFExportConfig {
	return PDFExportConfig{
		PageSize:        "Letter",
		Orientation:     "portrait",
		MarginTop:       0.5,
		MarginBottom:    0.5,
		MarginLeft:      0.5,
		MarginRight:     0.5,
		PrintBackground: true,
	}
}

// parsePDFConfig parses PDF export configuration from job params.
func parsePDFConfig(params map[string]interface{}) PDFExportConfig {
	cfg := DefaultPDFConfig()

	if params == nil {
		return cfg
	}

	// Try to get pdfConfig from params
	if pdfConfigRaw, ok := params["pdfConfig"]; ok {
		if pdfConfigJSON, err := json.Marshal(pdfConfigRaw); err == nil {
			_ = json.Unmarshal(pdfConfigJSON, &cfg)
		}
	}

	return cfg
}
