// Package scheduler provides event-driven export triggering functionality.
//
// This file is responsible for:
// - Listening for job completion events from the jobs manager
// - Matching jobs against export schedule filters
// - Executing exports to configured destinations
// - Managing retry logic with exponential backoff
// - Preventing duplicate exports via history tracking
//
// This file does NOT handle:
// - Schedule persistence (export_storage.go handles that)
// - Export validation (export_validation.go handles that)
// - History tracking (export_history.go handles that)
//
// Invariants:
// - ExportTrigger must be started before it will process events
// - Events are processed asynchronously via a buffered channel
// - Duplicate detection prevents exporting the same job twice for a schedule
// - Failed exports are retried with exponential backoff
// - Export execution runs in goroutines to avoid blocking event processing
package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// ExportTrigger handles event-driven export scheduling.
type ExportTrigger struct {
	mu                sync.RWMutex
	schedules         map[string]*ExportSchedule
	store             *ExportStorage
	historyStore      *ExportHistoryStore
	jobManager        *jobs.Manager
	webhookDispatcher *webhook.Dispatcher
	dataDir           string

	eventCh chan jobs.JobEvent
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewExportTrigger creates a new export trigger.
func NewExportTrigger(dataDir string, store *ExportStorage, historyStore *ExportHistoryStore, jobManager *jobs.Manager, webhookDispatcher *webhook.Dispatcher) *ExportTrigger {
	return &ExportTrigger{
		schedules:         make(map[string]*ExportSchedule),
		store:             store,
		historyStore:      historyStore,
		jobManager:        jobManager,
		webhookDispatcher: webhookDispatcher,
		dataDir:           dataDir,
		eventCh:           make(chan jobs.JobEvent, 128),
		stopCh:            make(chan struct{}),
	}
}

// Start begins listening for job completion events.
func (et *ExportTrigger) Start() error {
	// Load schedules from storage
	schedules, err := et.store.LoadAll()
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to load export schedules", err)
	}

	et.mu.Lock()
	for i := range schedules {
		et.schedules[schedules[i].ID] = &schedules[i]
	}
	et.mu.Unlock()

	slog.Info("export trigger started", "scheduleCount", len(et.schedules))

	// Start event processing goroutine
	et.wg.Add(1)
	go et.processEvents()

	return nil
}

// Stop halts the export trigger.
func (et *ExportTrigger) Stop() error {
	close(et.stopCh)
	et.wg.Wait()
	slog.Info("export trigger stopped")
	return nil
}

// HandleJobEvent processes a job event (called by jobs.Manager).
func (et *ExportTrigger) HandleJobEvent(event jobs.JobEvent) {
	// Only process completed events
	if event.Type != jobs.JobEventCompleted {
		return
	}

	// Only process terminal statuses
	if !event.Job.Status.IsTerminal() {
		return
	}

	select {
	case et.eventCh <- event:
		// Event queued successfully
	case <-et.stopCh:
		// Trigger is stopping, ignore event
	default:
		// Channel full, log and drop
		slog.Warn("export trigger event channel full, dropping event",
			"jobID", event.Job.ID,
			"status", event.Job.Status)
	}
}

// processEvents processes job events from the channel.
func (et *ExportTrigger) processEvents() {
	defer et.wg.Done()

	for {
		select {
		case <-et.stopCh:
			return
		case event := <-et.eventCh:
			et.handleJobEvent(event)
		}
	}
}

// handleJobEvent processes a single job event.
func (et *ExportTrigger) handleJobEvent(event jobs.JobEvent) {
	et.mu.RLock()
	schedules := make([]*ExportSchedule, 0, len(et.schedules))
	for _, s := range et.schedules {
		schedules = append(schedules, s)
	}
	et.mu.RUnlock()

	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		if et.matchSchedule(&event.Job, schedule) {
			// Check for duplicates
			if et.historyStore.HasExported(schedule.ID, event.Job.ID) {
				slog.Debug("skipping duplicate export",
					"scheduleID", schedule.ID,
					"jobID", event.Job.ID)
				continue
			}

			// Execute export asynchronously
			et.wg.Add(1)
			go func(s *ExportSchedule, job model.Job) {
				defer et.wg.Done()
				if err := et.executeExport(context.Background(), &job, s); err != nil {
					slog.Error("export execution failed",
						"scheduleID", s.ID,
						"jobID", job.ID,
						"error", err)
				}
			}(schedule, event.Job)
		}
	}
}

// matchSchedule checks if a job matches schedule filters.
func (et *ExportTrigger) matchSchedule(job *model.Job, schedule *ExportSchedule) bool {
	filters := schedule.Filters

	// Check job kinds
	if len(filters.JobKinds) > 0 {
		matched := false
		for _, kind := range filters.JobKinds {
			if string(job.Kind) == kind {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check job status
	if len(filters.JobStatus) > 0 {
		matched := false
		for _, status := range filters.JobStatus {
			// Handle aliases: "completed" matches succeeded/failed/canceled
			if status == "completed" && job.Status.IsTerminal() {
				matched = true
				break
			}
			if string(job.Status) == status {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check tags (all specified tags must be present)
	if len(filters.Tags) > 0 {
		jobTags, _ := job.Params["tags"].([]interface{})
		jobTagSet := make(map[string]bool)
		for _, t := range jobTags {
			if tag, ok := t.(string); ok {
				jobTagSet[tag] = true
			}
		}

		for _, tag := range filters.Tags {
			if !jobTagSet[tag] {
				return false
			}
		}
	}

	// Check has_results
	if filters.HasResults {
		if job.ResultPath == "" {
			return false
		}
		// Check if file exists and has content
		info, err := os.Stat(job.ResultPath)
		if err != nil || info.Size() == 0 {
			return false
		}
	}

	return true
}

// executeExport performs the actual export.
func (et *ExportTrigger) executeExport(ctx context.Context, job *model.Job, schedule *ExportSchedule) error {
	// Create history record
	record, err := et.historyStore.CreateRecord(schedule.ID, job.ID, schedule.Export.DestinationType)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create export record", err)
	}

	slog.Info("starting automated export",
		"scheduleID", schedule.ID,
		"jobID", job.ID,
		"destination", schedule.Export.DestinationType,
		"format", schedule.Export.Format)

	// Perform export based on destination type
	var exportErr error
	switch schedule.Export.DestinationType {
	case "local":
		exportErr = et.exportToLocal(ctx, job, schedule, record)
	case "webhook":
		exportErr = et.exportToWebhook(ctx, job, schedule, record)
	default:
		exportErr = apperrors.Validation(fmt.Sprintf("unsupported destination type: %s", schedule.Export.DestinationType))
	}

	if exportErr != nil {
		// Mark as failed and potentially retry
		if err := et.historyStore.MarkFailed(record.ID, exportErr.Error()); err != nil {
			slog.Error("failed to mark export as failed", "recordID", record.ID, "error", err)
		}

		// Retry if under max retries
		if record.RetryCount < schedule.Retry.GetMaxRetries() {
			return et.retryExport(ctx, job, schedule, record)
		}

		return exportErr
	}

	slog.Info("automated export completed successfully",
		"scheduleID", schedule.ID,
		"jobID", job.ID,
		"recordID", record.ID)

	return nil
}

// exportToLocal exports job results to local file.
func (et *ExportTrigger) exportToLocal(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	// Determine output path
	pathTemplate := schedule.Export.PathTemplate
	if pathTemplate == "" {
		pathTemplate = schedule.Export.LocalPath
	}
	if pathTemplate == "" {
		pathTemplate = "exports/{kind}/{job_id}.{format}"
	}

	outputPath := exporter.RenderPathTemplate(pathTemplate, *job, schedule.Export.Format)

	// Ensure absolute path
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(et.dataDir, outputPath)
	}

	// Create directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create export directory", err)
	}

	// Read job results
	resultData, err := et.readJobResults(job)
	if err != nil {
		return err
	}

	// Export to file
	file, err := os.Create(outputPath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create export file", err)
	}
	defer file.Close()

	if err := exporter.ExportStream(*job, bytes.NewReader(resultData), schedule.Export.Format, file); err != nil {
		return err
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		slog.Warn("failed to get export file size", "path", outputPath, "error", err)
	}

	// Mark as successful
	var size int64
	if info != nil {
		size = info.Size()
	}
	return et.historyStore.MarkSuccess(record.ID, size, 0)
}

// exportToWebhook exports job results via webhook.
func (et *ExportTrigger) exportToWebhook(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	if schedule.Export.WebhookURL == "" {
		return apperrors.Validation("webhook URL is required for webhook export")
	}

	// Read job results
	resultData, err := et.readJobResults(job)
	if err != nil {
		return err
	}

	// Export to buffer to get formatted output
	var buf bytes.Buffer
	if err := exporter.ExportStream(*job, bytes.NewReader(resultData), schedule.Export.Format, &buf); err != nil {
		return err
	}

	// Send webhook if dispatcher is available
	if et.webhookDispatcher != nil {
		webhookPayload := webhook.Payload{
			EventID:      record.ID,
			EventType:    webhook.EventExportCompleted,
			Timestamp:    time.Now(),
			JobID:        job.ID,
			JobKind:      string(job.Kind),
			Status:       string(job.Status),
			ExportFormat: schedule.Export.Format,
			ExportPath:   schedule.Export.WebhookURL,
		}
		et.webhookDispatcher.Dispatch(ctx, schedule.Export.WebhookURL, webhookPayload, "")
	}

	// Mark as successful
	return et.historyStore.MarkSuccess(record.ID, int64(buf.Len()), 0)
}

// retryExport retries a failed export with exponential backoff.
func (et *ExportTrigger) retryExport(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	// Increment retry count
	if err := et.historyStore.IncrementRetry(record.ID); err != nil {
		slog.Error("failed to increment retry count", "recordID", record.ID, "error", err)
	}

	// Calculate delay with exponential backoff
	delay := schedule.Retry.GetBaseDelay() * time.Duration(1<<record.RetryCount)
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	slog.Info("scheduling export retry",
		"scheduleID", schedule.ID,
		"jobID", job.ID,
		"recordID", record.ID,
		"retryCount", record.RetryCount+1,
		"maxRetries", schedule.Retry.GetMaxRetries(),
		"delay", delay)

	// Wait and retry
	select {
	case <-time.After(delay):
		return et.executeExport(ctx, job, schedule)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// readJobResults reads the job result file.
func (et *ExportTrigger) readJobResults(job *model.Job) ([]byte, error) {
	if job.ResultPath == "" {
		return nil, apperrors.Validation("job has no results")
	}

	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read job results", err)
	}

	return data, nil
}

// AddSchedule adds a new schedule to the trigger.
func (et *ExportTrigger) AddSchedule(schedule *ExportSchedule) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.schedules[schedule.ID] = schedule
}

// UpdateSchedule updates a schedule in the trigger.
func (et *ExportTrigger) UpdateSchedule(schedule *ExportSchedule) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.schedules[schedule.ID] = schedule
}

// RemoveSchedule removes a schedule from the trigger.
func (et *ExportTrigger) RemoveSchedule(scheduleID string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	delete(et.schedules, scheduleID)
}

// ReloadSchedules reloads all schedules from storage.
func (et *ExportTrigger) ReloadSchedules() error {
	schedules, err := et.store.LoadAll()
	if err != nil {
		return err
	}

	et.mu.Lock()
	defer et.mu.Unlock()

	et.schedules = make(map[string]*ExportSchedule)
	for i := range schedules {
		et.schedules[schedules[i].ID] = &schedules[i]
	}

	return nil
}

// Export executes an export for testing purposes.
func (et *ExportTrigger) Export(ctx context.Context, job *model.Job, schedule *ExportSchedule) error {
	return et.executeExport(ctx, job, schedule)
}

// compile-time interface checks
var _ io.Closer = (*ExportTrigger)(nil)

// Close implements io.Closer.
func (et *ExportTrigger) Close() error {
	return et.Stop()
}
