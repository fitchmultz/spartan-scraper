package manage

import (
	"context"
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
