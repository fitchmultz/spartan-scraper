package manage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestRunExportWithScheduleSeed(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{DataDir: tmpDir}
	jobID := writeExportTestJob(t, tmpDir, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Example"}`)

	scheduleStore := scheduler.NewExportStorage(tmpDir)
	schedule, err := scheduleStore.Add(scheduler.ExportSchedule{
		Name:    "Projected export",
		Enabled: true,
		Filters: scheduler.ExportFilters{JobKinds: []string{"scrape"}},
		Export: scheduler.ExportConfig{
			Format:          "json",
			DestinationType: "local",
			LocalPath:       "exports/{job_id}.json",
			Transform: exporter.TransformConfig{
				Expression: "{title: title}",
				Language:   "jmespath",
			},
		},
		Retry: scheduler.DefaultRetryConfig(),
	})
	if err != nil {
		t.Fatalf("add schedule: %v", err)
	}

	outPath := filepath.Join(tmpDir, "out", "projected.json")
	if code := RunExport(context.Background(), cfg, []string{"--job-id", jobID, "--schedule-id", schedule.ID, "--out", outPath}); code != 0 {
		t.Fatalf("RunExport returned %d", code)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if strings.Contains(string(data), "status") || !strings.Contains(string(data), "title") {
		t.Fatalf("unexpected projected export: %s", data)
	}
}

func TestRunExportWithShapeFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{DataDir: tmpDir}
	jobID := writeExportTestJob(t, tmpDir, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Example","normalized":{"fields":{"price":{"values":["$10"]}}}}`)

	shapePath := filepath.Join(tmpDir, "shape.json")
	if err := os.WriteFile(shapePath, []byte(`{"summaryFields":["title","url"],"normalizedFields":["field.price"]}`), 0o644); err != nil {
		t.Fatalf("write shape file: %v", err)
	}
	outPath := filepath.Join(tmpDir, "out", "report.md")
	if code := RunExport(context.Background(), cfg, []string{"--job-id", jobID, "--format", "md", "--shape-file", shapePath, "--out", outPath}); code != 0 {
		t.Fatalf("RunExport returned %d", code)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if !strings.Contains(string(data), "Example") || !strings.Contains(string(data), "$10") {
		t.Fatalf("unexpected shaped markdown: %s", data)
	}
}

func TestRunExportRejectsShapeAndTransform(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{DataDir: tmpDir}
	_ = writeExportTestJob(t, tmpDir, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Example"}`)

	shapePath := filepath.Join(tmpDir, "shape.json")
	if err := os.WriteFile(shapePath, []byte(`{"topLevelFields":["url"]}`), 0o644); err != nil {
		t.Fatalf("write shape file: %v", err)
	}
	code := RunExport(context.Background(), cfg, []string{"--job-id", "job-export-test", "--format", "csv", "--shape-file", shapePath, "--transform-language", "jmespath", "--transform-expression", "{url: url}"})
	if code == 0 {
		t.Fatal("expected validation failure")
	}
}

func TestRunExportInspectAndHistory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{DataDir: tmpDir}
	jobID := writeExportTestJob(t, tmpDir, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Example"}`)
	outPath := filepath.Join(tmpDir, "out", "results.json")
	if code := RunExport(context.Background(), cfg, []string{"--job-id", jobID, "--format", "json", "--out", outPath}); code != 0 {
		t.Fatalf("RunExport returned %d", code)
	}

	historyStore := scheduler.NewExportHistoryStore(tmpDir)
	records, total, err := historyStore.GetByJob(jobID, 10, 0)
	if err != nil {
		t.Fatalf("GetByJob failed: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("unexpected export history: total=%d records=%#v", total, records)
	}

	historyJSON := captureStdout(t, func() {
		if code := RunExport(context.Background(), cfg, []string{"--history-job-id", jobID, "--json"}); code != 0 {
			t.Fatalf("history command returned %d", code)
		}
	})
	if !strings.Contains(historyJSON, `"exports"`) || !strings.Contains(historyJSON, records[0].ID) {
		t.Fatalf("unexpected history json: %s", historyJSON)
	}

	inspectText := captureStdout(t, func() {
		if code := RunExport(context.Background(), cfg, []string{"--inspect-id", records[0].ID}); code != 0 {
			t.Fatalf("inspect command returned %d", code)
		}
	})
	if !strings.Contains(inspectText, "EXPORT READY") || !strings.Contains(inspectText, records[0].ID) {
		t.Fatalf("unexpected inspect output: %s", inspectText)
	}
}

func writeExportTestJob(t *testing.T, dataDir string, kind model.Kind, resultContent string) string {
	t.Helper()
	jobID := "job-export-test"
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	job := model.Job{
		ID:         jobID,
		Kind:       kind,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]any{"url": "https://example.com"},
		ResultPath: filepath.Join(dataDir, "jobs", jobID, "results.jsonl"),
	}
	if err := os.MkdirAll(filepath.Dir(job.ResultPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(job.ResultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}
	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	return jobID
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close: %v", err)
	}
	return string(output)
}
