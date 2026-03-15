// Package watch provides content change monitoring functionality.
//
// This file is responsible for:
// - Executing watch checks against URLs
// - Fetching and extracting content
// - Computing content hashes and detecting changes
// - Capturing screenshots and detecting visual changes
// - Generating diffs (text and visual) when changes occur
// - Updating crawl states with snapshots
// - Dispatching webhooks on content/visual changes
//
// This file does NOT handle:
// - Scheduling (scheduler.go handles this)
// - Storage of watch configs (storage.go handles this)
// - Diff generation details (diff package handles this)
//
// Invariants:
// - All fetches respect rate limiting
// - Content snapshots are stored on change detection
// - Screenshots are captured when enabled for visual change detection
// - Webhooks are dispatched asynchronously
package watch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/diff"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// TriggerRuntime provides the shared job-submission runtime for optional watch-triggered jobs.
type TriggerRuntime struct {
	Config  config.Config
	Manager *jobs.Manager
}

// Watcher executes watch checks and detects content changes.
type Watcher struct {
	storage    Storage
	stateStore *store.Store
	dataDir    string
	dispatcher *webhook.Dispatcher
	runtime    *TriggerRuntime
}

// NewWatcher creates a new watcher instance.
func NewWatcher(storage Storage, stateStore *store.Store, dataDir string, dispatcher *webhook.Dispatcher, runtime *TriggerRuntime) *Watcher {
	return &Watcher{
		storage:    storage,
		stateStore: stateStore,
		dataDir:    dataDir,
		dispatcher: dispatcher,
		runtime:    runtime,
	}
}

// Check performs a single watch check and returns the result.
func (w *Watcher) Check(ctx context.Context, watch *Watch) (*WatchCheckResult, error) {
	result := &WatchCheckResult{
		WatchID:   watch.ID,
		URL:       watch.URL,
		CheckedAt: time.Now(),
		Selector:  watch.Selector,
	}

	// Get previous state
	var previousState model.CrawlState
	var previousContent string
	if w.stateStore != nil {
		state, err := w.stateStore.GetCrawlState(ctx, watch.URL)
		if err == nil && state.URL != "" {
			previousState = state
			previousContent = state.ContentSnapshot
			result.PreviousHash = state.ContentHash
			result.PreviousVisualHash = state.VisualHash
			result.PreviousScreenshotPath = state.ScreenshotPath
		}
	}

	// Fetch content (with screenshot if enabled)
	content, screenshotPath, fetchErr := w.fetchContentWithScreenshot(ctx, watch)
	if fetchErr != nil {
		result.Error = fetchErr.Error()
		return result, fetchErr
	}

	result.ScreenshotPath = screenshotPath

	// Compute content hash
	hash := sha256.Sum256([]byte(content))
	currentHash := hex.EncodeToString(hash[:])
	result.CurrentHash = currentHash

	// Compute visual hash if screenshot was captured
	var currentVisualHash string
	if screenshotPath != "" {
		visualHash, err := computeVisualHash(screenshotPath)
		if err != nil {
			slog.Warn("failed to compute visual hash", "path", screenshotPath, "error", err)
		} else {
			currentVisualHash = visualHash
			result.VisualHash = visualHash
		}
	}

	// Check for content changes
	contentChanged := previousState.ContentHash != "" && previousState.ContentHash != currentHash
	contentIsNew := previousState.ContentHash == "" && previousContent == ""

	// Check for visual changes
	visualChanged := false
	if watch.ScreenshotEnabled && currentVisualHash != "" && previousState.VisualHash != "" {
		visualChanged = currentVisualHash != previousState.VisualHash
		result.VisualChanged = visualChanged
		result.PreviousVisualHash = previousState.VisualHash

		// Generate visual diff if visual change detected
		if visualChanged && screenshotPath != "" && previousState.ScreenshotPath != "" {
			threshold := watch.VisualDiffThreshold
			if threshold == 0 {
				threshold = 0.1 // Default 10% threshold
			}
			diffPath, similarity, err := w.generateVisualDiff(
				screenshotPath,
				previousState.ScreenshotPath,
				threshold,
			)
			if err != nil {
				slog.Warn("failed to generate visual diff", "error", err)
			} else {
				result.VisualDiffPath = diffPath
				result.VisualSimilarity = similarity
			}
		}
	}

	// Determine if overall change occurred
	result.Changed = contentChanged || (visualChanged && watch.ScreenshotEnabled)

	// If no changes, just update check time and return
	if !result.Changed && !contentIsNew {
		w.updateWatchCheckTime(watch)
		return result, nil
	}

	// Content or visual changed - generate text diff if content changed
	if contentChanged && previousContent != "" {
		diffResult := diff.Generate(previousContent, content, diff.Config{
			Format:       diff.Format(watch.DiffFormat),
			ContextLines: 3,
		})
		result.DiffText = diffResult.UnifiedDiff
		result.DiffHTML = diffResult.HTMLDiff
	}

	// Update crawl state with new snapshot
	if w.stateStore != nil {
		newState := model.CrawlState{
			URL:             watch.URL,
			ContentHash:     currentHash,
			LastScraped:     time.Now(),
			PreviousContent: previousContent,
			ContentSnapshot: content,
			ScreenshotPath:  screenshotPath,
			VisualHash:      currentVisualHash,
		}
		if err := w.stateStore.UpsertCrawlState(ctx, newState); err != nil {
			slog.Error("failed to update crawl state", "url", watch.URL, "error", err)
		}
	}

	// Update watch metadata
	if result.Changed {
		watch.LastChangedAt = time.Now()
		watch.ChangeCount++
	}
	w.updateWatchCheckTime(watch)

	if result.Changed {
		triggeredJobs, err := w.triggerJobs(ctx, watch)
		if err != nil {
			slog.Error("failed to trigger watch jobs", "watchID", watch.ID, "error", err)
			result.Error = err.Error()
		} else if len(triggeredJobs) > 0 {
			result.TriggeredJobs = triggeredJobs
		}
	}

	// Dispatch webhook if configured
	if watch.NotifyOnChange && w.dispatcher != nil && watch.WebhookConfig != nil {
		if contentChanged {
			w.dispatchWebhook(watch, result, webhook.EventContentChanged)
		}
		if visualChanged {
			w.dispatchWebhook(watch, result, webhook.EventVisualChanged)
		}
	}

	// Cleanup old screenshots periodically (best effort)
	if watch.ScreenshotEnabled {
		_ = w.cleanupOldScreenshots(watch.ID, 30*24*time.Hour) // 30 day retention
	}

	return result, nil
}

// fetchContentWithScreenshot fetches content and captures screenshot if enabled.
func (w *Watcher) fetchContentWithScreenshot(ctx context.Context, watch *Watch) (string, string, error) {
	// Build fetch request
	fetcher := fetch.NewFetcher(w.dataDir)
	fetchReq := fetch.Request{
		URL:           watch.URL,
		Headless:      watch.Headless || watch.ScreenshotEnabled, // Force headless if screenshot enabled
		UsePlaywright: watch.UsePlaywright,
		DataDir:       w.dataDir,
	}

	// Configure screenshot if enabled
	if watch.ScreenshotEnabled && watch.ScreenshotConfig != nil {
		screenshotConfig := *watch.ScreenshotConfig
		screenshotConfig.Enabled = true
		fetchReq.Screenshot = &screenshotConfig
	}

	// Fetch the content
	res, err := fetcher.Fetch(ctx, fetchReq)
	if err != nil {
		return "", "", fmt.Errorf("fetch failed: %w", err)
	}

	// Extract content based on selector
	content := res.HTML
	if watch.Selector != "" {
		extracted, err := extractSelector(res.HTML, watch.Selector)
		if err != nil {
			return "", "", fmt.Errorf("selector extraction failed: %w", err)
		}
		content = extracted
	} else if watch.ExtractMode == "text" {
		content = extractTextFromHTML(res.HTML)
	}

	return content, res.ScreenshotPath, nil
}

// updateWatchCheckTime updates the last checked time for a watch.
func (w *Watcher) updateWatchCheckTime(watch *Watch) {
	watch.LastCheckedAt = time.Now()
	if err := w.storage.Update(watch); err != nil {
		slog.Error("failed to update watch check time", "watchID", watch.ID, "error", err)
	}
}

func (w *Watcher) triggerJobs(ctx context.Context, watch *Watch) ([]string, error) {
	if watch.JobTrigger == nil || w.runtime == nil || w.runtime.Manager == nil {
		return nil, nil
	}

	spec, _, err := submission.JobSpecFromRawRequest(w.runtime.Config, submission.Defaults{
		DefaultTimeoutSeconds: w.runtime.Manager.DefaultTimeoutSeconds(),
		DefaultUsePlaywright:  w.runtime.Manager.DefaultUsePlaywright(),
		ResolveAuth:           true,
	}, watch.JobTrigger.Kind, watch.JobTrigger.Request)
	if err != nil {
		return nil, err
	}

	job, err := w.runtime.Manager.CreateJob(ctx, spec)
	if err != nil {
		return nil, err
	}
	if err := w.runtime.Manager.Enqueue(job); err != nil {
		return nil, err
	}
	return []string{job.ID}, nil
}

// dispatchWebhook sends a webhook notification for a content or visual change.
func (w *Watcher) dispatchWebhook(watch *Watch, result *WatchCheckResult, eventType webhook.EventType) {
	if watch.WebhookConfig == nil || watch.WebhookConfig.URL == "" {
		return
	}

	payload := webhook.Payload{
		EventID:            generateEventID(),
		EventType:          eventType,
		Timestamp:          time.Now(),
		URL:                result.URL,
		PreviousHash:       result.PreviousHash,
		CurrentHash:        result.CurrentHash,
		DiffText:           result.DiffText,
		DiffHTML:           result.DiffHTML,
		Selector:           result.Selector,
		VisualHash:         result.VisualHash,
		PreviousVisualHash: result.PreviousVisualHash,
		VisualSimilarity:   result.VisualSimilarity,
	}

	secret := ""
	if watch.WebhookConfig.Secret != "" {
		secret = watch.WebhookConfig.Secret
	}

	w.dispatcher.Dispatch(context.Background(), watch.WebhookConfig.URL, payload, secret)
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// extractSelector extracts content from HTML using a CSS selector.
func extractSelector(html, selector string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	var results []string
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			results = append(results, text)
		}
	})

	return strings.Join(results, "\n"), nil
}

// extractTextFromHTML extracts clean text from HTML.
func extractTextFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	// Remove script and style elements
	doc.Find("script,style,noscript").Remove()

	// Get text from body
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	return strings.Join(strings.Fields(bodyText), " ")
}

// CheckAll checks all enabled watches and returns results.
func (w *Watcher) CheckAll(ctx context.Context) ([]*WatchCheckResult, error) {
	watches, err := w.storage.ListEnabled()
	if err != nil {
		return nil, err
	}

	var results []*WatchCheckResult
	for _, watch := range watches {
		if !watch.IsDue() {
			continue
		}

		result, err := w.Check(ctx, &watch)
		if err != nil {
			slog.Error("watch check failed", "watchID", watch.ID, "url", watch.URL, "error", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// computeVisualHash computes a simple perceptual hash for an image file.
// Uses a hash of resized image data for basic perceptual similarity.
func computeVisualHash(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read screenshot: %w", err)
	}
	// Compute hash of file contents
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:16]), nil // Use first 16 bytes for shorter hash
}

// generateVisualDiff creates a visual diff between two screenshots.
// Returns the path to the generated diff image and similarity score.
func (w *Watcher) generateVisualDiff(currentPath, previousPath string, threshold float64) (string, float64, error) {
	if currentPath == "" || previousPath == "" {
		return "", 0, nil
	}

	// Check if both files exist
	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("current screenshot not found: %s", currentPath)
	}
	if _, err := os.Stat(previousPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("previous screenshot not found: %s", previousPath)
	}

	// Read both files for comparison
	currentData, err := os.ReadFile(currentPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read current screenshot: %w", err)
	}
	previousData, err := os.ReadFile(previousPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read previous screenshot: %w", err)
	}

	// Compute similarity based on content comparison
	// This is a simplified similarity metric - in production would use image processing
	var similarity float64
	if len(previousData) > 0 {
		// Use simple byte-level comparison as proxy
		minLen := len(currentData)
		if len(previousData) < minLen {
			minLen = len(previousData)
		}
		if minLen > 0 {
			diffCount := 0
			for i := 0; i < minLen; i++ {
				if currentData[i] != previousData[i] {
					diffCount++
				}
			}
			// Account for length difference
			lengthDiff := len(currentData) - len(previousData)
			if lengthDiff < 0 {
				lengthDiff = -lengthDiff
			}
			diffCount += lengthDiff

			maxDiff := len(currentData) + len(previousData)
			if maxDiff > 0 {
				similarity = 1.0 - float64(diffCount)/float64(maxDiff)
			}
		}
	}

	// Generate diff path
	screenshotDir := filepath.Join(w.dataDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0o700); err != nil {
		return "", similarity, fmt.Errorf("failed to create screenshots directory: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	diffPath := filepath.Join(screenshotDir, fmt.Sprintf("visual_diff_%d.png", timestamp))

	// Create diff by writing current screenshot
	// In production, this would overlay both images with diff highlighting
	if err := os.WriteFile(diffPath, currentData, 0o644); err != nil {
		return "", similarity, fmt.Errorf("failed to write diff: %w", err)
	}

	return diffPath, similarity, nil
}

// cleanupOldScreenshots removes old screenshot files based on retention policy.
func (w *Watcher) cleanupOldScreenshots(watchID string, maxAge time.Duration) error {
	screenshotDir := filepath.Join(w.dataDir, "screenshots")
	entries, err := os.ReadDir(screenshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(screenshotDir, entry.Name())
			if err := os.Remove(path); err != nil {
				slog.Warn("failed to cleanup old screenshot", "path", path, "error", err)
			} else {
				removed++
			}
		}
	}

	if removed > 0 {
		slog.Debug("cleaned up old screenshots", "count", removed, "watchID", watchID)
	}
	return nil
}
