// Package watch provides content change monitoring functionality.
//
// This file is responsible for:
// - Executing watch checks against URLs
// - Fetching and extracting content
// - Computing content hashes and detecting changes
// - Generating diffs when content changes
// - Updating crawl states with snapshots
// - Dispatching webhooks on content changes
//
// This file does NOT handle:
// - Scheduling (scheduler.go handles this)
// - Storage of watch configs (storage.go handles this)
// - Diff generation details (diff package handles this)
//
// Invariants:
// - All fetches respect rate limiting
// - Content snapshots are stored on change detection
// - Webhooks are dispatched asynchronously
package watch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fitchmultz/spartan-scraper/internal/diff"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// Watcher executes watch checks and detects content changes.
type Watcher struct {
	storage    Storage
	stateStore *store.Store
	dataDir    string
	dispatcher *webhook.Dispatcher
}

// NewWatcher creates a new watcher instance.
func NewWatcher(storage Storage, stateStore *store.Store, dataDir string, dispatcher *webhook.Dispatcher) *Watcher {
	return &Watcher{
		storage:    storage,
		stateStore: stateStore,
		dataDir:    dataDir,
		dispatcher: dispatcher,
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
		}
	}

	// Fetch content
	content, err := w.fetchContent(ctx, watch)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Compute hash
	hash := sha256.Sum256([]byte(content))
	currentHash := hex.EncodeToString(hash[:])
	result.CurrentHash = currentHash

	// Check for changes
	if previousState.ContentHash == currentHash {
		// No change
		result.Changed = false
		w.updateWatchCheckTime(watch)
		return result, nil
	}

	// Content changed
	result.Changed = true

	// Generate diff
	if previousContent != "" {
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
		}
		if err := w.stateStore.UpsertCrawlState(ctx, newState); err != nil {
			slog.Error("failed to update crawl state", "url", watch.URL, "error", err)
		}
	}

	// Update watch metadata
	watch.LastChangedAt = time.Now()
	watch.ChangeCount++
	w.updateWatchCheckTime(watch)

	// Dispatch webhook if configured
	if watch.NotifyOnChange && w.dispatcher != nil && watch.WebhookConfig != nil {
		w.dispatchWebhook(watch, result)
	}

	return result, nil
}

// fetchContent fetches and extracts content from the watch URL.
func (w *Watcher) fetchContent(ctx context.Context, watch *Watch) (string, error) {
	// Fetch the content
	fetcher := fetch.NewFetcher(w.dataDir)
	fetchReq := fetch.Request{
		URL:           watch.URL,
		Headless:      watch.Headless,
		UsePlaywright: watch.UsePlaywright,
		DataDir:       w.dataDir,
	}

	res, err := fetcher.Fetch(ctx, fetchReq)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}

	// Extract content based on selector
	if watch.Selector != "" {
		extracted, err := extractSelector(res.HTML, watch.Selector)
		if err != nil {
			return "", fmt.Errorf("selector extraction failed: %w", err)
		}
		return extracted, nil
	}

	// Return full HTML or normalized text based on extract mode
	if watch.ExtractMode == "text" {
		return extractTextFromHTML(res.HTML), nil
	}

	return res.HTML, nil
}

// updateWatchCheckTime updates the last checked time for a watch.
func (w *Watcher) updateWatchCheckTime(watch *Watch) {
	watch.LastCheckedAt = time.Now()
	if err := w.storage.Update(watch); err != nil {
		slog.Error("failed to update watch check time", "watchID", watch.ID, "error", err)
	}
}

// dispatchWebhook sends a webhook notification for a content change.
func (w *Watcher) dispatchWebhook(watch *Watch, result *WatchCheckResult) {
	if watch.WebhookConfig == nil || watch.WebhookConfig.URL == "" {
		return
	}

	payload := webhook.Payload{
		EventID:      generateEventID(),
		EventType:    webhook.EventContentChanged,
		Timestamp:    time.Now(),
		URL:          result.URL,
		PreviousHash: result.PreviousHash,
		CurrentHash:  result.CurrentHash,
		DiffText:     result.DiffText,
		DiffHTML:     result.DiffHTML,
		Selector:     result.Selector,
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
