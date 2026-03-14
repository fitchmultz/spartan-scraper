package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportTriggerExportAppliesConfiguredTransform(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	trigger := NewExportTrigger(dataDir, store, history, nil, nil)

	jobDir := filepath.Join(dataDir, "jobs", "job-transform")
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(jobDir): %v", err)
	}
	resultPath := filepath.Join(jobDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(strings.Join([]string{
		`{"url":"https://example.com/a","title":"A","status":200}`,
		`{"url":"https://example.com/b","title":"B","status":200}`,
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(resultPath): %v", err)
	}

	job := model.Job{
		ID:         "job-transform",
		Kind:       model.KindCrawl,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: resultPath,
	}
	schedule := &ExportSchedule{
		ID:      "schedule-transform",
		Name:    "Projected Export",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"crawl"}},
		Export: ExportConfig{
			Format:          "csv",
			DestinationType: "local",
			LocalPath:       "exports/{job_id}.csv",
			PathTemplate:    "exports/{job_id}.csv",
			Transform: exporter.TransformConfig{
				Expression: "{title: title, url: url}",
				Language:   "jmespath",
			},
		},
		Retry: DefaultRetryConfig(),
	}

	if err := trigger.Export(context.Background(), &job, schedule); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	outputPath := filepath.Join(dataDir, "exports", "job-transform.csv")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath): %v", err)
	}
	content := strings.TrimSpace(string(data))
	if !strings.Contains(content, "title,url") || strings.Contains(content, "status") {
		t.Fatalf("unexpected transformed export content: %s", content)
	}

	records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBySchedule() failed: %v", err)
	}
	if total != 1 || len(records) != 1 || records[0].Status != "success" {
		t.Fatalf("unexpected export history: %#v total=%d", records, total)
	}
}
