// Package scheduler provides event-driven export triggering functionality.
//
// Purpose:
// - React to terminal job events and execute matching automated export schedules.
//
// Responsibilities:
// - Listen for completed job events from the jobs manager.
// - Match jobs against export schedule filters and deduplicate scheduled exports.
// - Execute local and webhook exports, including retry handling.
// - Keep in-flight export work bound to the export-trigger lifecycle.
//
// Scope:
// - Event-driven export execution only; schedule persistence, validation, and history storage live in sibling files.
//
// Usage:
// - Started by the server runtime after schedule storage is initialized.
//
// Invariants/Assumptions:
// - ExportTrigger must be started before it will process events.
// - Events are processed asynchronously via a buffered channel.
// - Duplicate detection prevents exporting the same job twice for a schedule.
// - Failed exports are retried with exponential backoff until the trigger lifecycle stops.
package scheduler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
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

	eventCh  chan jobs.JobEvent
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewExportTrigger creates a new export trigger.
func NewExportTrigger(dataDir string, store *ExportStorage, historyStore *ExportHistoryStore, jobManager *jobs.Manager, webhookDispatcher *webhook.Dispatcher) *ExportTrigger {
	ctx, cancel := context.WithCancel(context.Background())
	return &ExportTrigger{
		schedules:         make(map[string]*ExportSchedule),
		store:             store,
		historyStore:      historyStore,
		jobManager:        jobManager,
		webhookDispatcher: webhookDispatcher,
		dataDir:           dataDir,
		eventCh:           make(chan jobs.JobEvent, 128),
		stopCh:            make(chan struct{}),
		ctx:               ctx,
		cancel:            cancel,
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
	et.stopOnce.Do(func() {
		close(et.stopCh)
		if et.cancel != nil {
			et.cancel()
		}
	})
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
				if err := et.executeExport(et.ctx, &job, s); err != nil {
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
		return false
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
	destination, err := et.destinationForSchedule(job, schedule)
	if err != nil {
		return err
	}

	record, err := et.historyStore.CreateRecord(CreateRecordInput{
		ScheduleID:  schedule.ID,
		JobID:       job.ID,
		Trigger:     exporter.OutcomeTriggerSchedule,
		Destination: destination,
		Request:     resultExportConfigForSchedule(schedule),
	})
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create export record", err)
	}
	return et.executeExportAttempt(ctx, job, schedule, record)
}

func (et *ExportTrigger) executeExportAttempt(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	slog.Info("starting automated export",
		"scheduleID", schedule.ID,
		"jobID", job.ID,
		"recordID", record.ID,
		"destination", schedule.Export.DestinationType,
		"format", schedule.Export.Format,
		"retryCount", record.RetryCount)

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
		if err := et.historyStore.MarkFailed(record.ID, exportErr); err != nil {
			slog.Error("failed to mark export as failed", "recordID", record.ID, "error", err)
		}

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
	_ = ctx

	outputPath := record.Destination
	if outputPath == "" {
		var err error
		outputPath, err = et.destinationForSchedule(job, schedule)
		if err != nil {
			return err
		}
	}

	resultData, err := et.readJobResults(job)
	if err != nil {
		return err
	}

	rendered, err := exporter.RenderResultExport(*job, resultData, resultExportConfigForSchedule(schedule))
	if err != nil {
		return err
	}
	if _, err := fsutil.WritePrivateFileWithinRoot(filepath.Join(et.dataDir, "exports"), outputPath, rendered.Content); err != nil {
		return err
	}

	return et.historyStore.MarkSuccess(record.ID, rendered)
}

// exportToWebhook exports job results via webhook.
func (et *ExportTrigger) exportToWebhook(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	if schedule.Export.WebhookURL == "" {
		return apperrors.Validation("webhook URL is required for webhook export")
	}
	if et.webhookDispatcher == nil {
		return apperrors.Internal("webhook delivery unavailable: dispatcher is not configured")
	}

	resultData, err := et.readJobResults(job)
	if err != nil {
		return err
	}

	rendered, err := exporter.RenderResultExport(*job, resultData, resultExportConfigForSchedule(schedule))
	if err != nil {
		return err
	}

	webhookPayload := webhook.Payload{
		EventID:      record.ID,
		EventType:    webhook.EventExportCompleted,
		Timestamp:    time.Now(),
		JobID:        job.ID,
		JobKind:      string(job.Kind),
		Status:       string(job.Status),
		ResultURL:    "/v1/jobs/" + job.ID + "/results",
		ExportFormat: rendered.Format,
		Filename:     rendered.Filename,
		ContentType:  rendered.ContentType,
		RecordCount:  rendered.RecordCount,
		ExportSize:   rendered.Size,
	}
	if err := et.webhookDispatcher.DeliverExport(ctx, schedule.Export.WebhookURL, webhookPayload, rendered.Content, ""); err != nil {
		return err
	}

	// Mark as successful only after delivery completes.
	return et.historyStore.MarkSuccess(record.ID, rendered)
}

// retryExport retries a failed export with exponential backoff.
func (et *ExportTrigger) retryExport(ctx context.Context, job *model.Job, schedule *ExportSchedule, record *ExportRecord) error {
	nextRetryCount := record.RetryCount + 1
	record.RetryCount = nextRetryCount
	if err := et.historyStore.IncrementRetry(record.ID); err != nil {
		slog.Error("failed to increment retry count", "recordID", record.ID, "error", err)
	}

	delay := schedule.Retry.GetBaseDelay() * time.Duration(1<<(nextRetryCount-1))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	slog.Info("scheduling export retry",
		"scheduleID", schedule.ID,
		"jobID", job.ID,
		"recordID", record.ID,
		"retryCount", nextRetryCount,
		"maxRetries", schedule.Retry.GetMaxRetries(),
		"delay", delay)

	select {
	case <-time.After(delay):
		return et.executeExportAttempt(ctx, job, schedule, record)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func resultExportConfigForSchedule(schedule *ExportSchedule) exporter.ResultExportConfig {
	return exporter.NormalizeResultExportConfig(exporter.ResultExportConfig{
		Format:    schedule.Export.Format,
		Shape:     schedule.Export.Shape,
		Transform: schedule.Export.Transform,
	})
}

func (et *ExportTrigger) destinationForSchedule(job *model.Job, schedule *ExportSchedule) (string, error) {
	switch schedule.Export.DestinationType {
	case "webhook":
		return strings.TrimSpace(schedule.Export.WebhookURL), nil
	default:
		pathTemplate := schedule.Export.PathTemplate
		if pathTemplate == "" {
			pathTemplate = schedule.Export.LocalPath
		}
		if pathTemplate == "" {
			pathTemplate = "exports/{kind}/{job_id}.{format}"
		}
		outputPath := exporter.RenderPathTemplate(pathTemplate, *job, schedule.Export.Format)
		resolvedPath, err := fsutil.ResolvePathWithinRoot(et.dataDir, outputPath)
		if err != nil {
			return "", err
		}
		exportsRoot, err := fsutil.ResolvePathWithinRoot(et.dataDir, "exports")
		if err != nil {
			return "", apperrors.Wrap(apperrors.KindInternal, "failed to resolve automated export root", err)
		}
		rel, err := filepath.Rel(exportsRoot, resolvedPath)
		if err != nil {
			return "", apperrors.Wrap(apperrors.KindInternal, "failed to validate automated export destination", err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", apperrors.Permission(fmt.Sprintf("automated export destination must stay within %s", exportsRoot))
		}
		return resolvedPath, nil
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
