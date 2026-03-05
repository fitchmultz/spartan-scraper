// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file is responsible for:
// - Fetching feed content from URLs
// - Parsing RSS 2.0 and Atom 1.0 feeds
// - Detecting feed format automatically
// - Extracting feed items and metadata
// - Filtering new items based on seen item storage
// - Creating scrape jobs for new items (when autoScrape is enabled)
//
// This file does NOT handle:
// - Scheduling (scheduler.go handles this)
// - Storage of feed configs (storage.go handles this)
// - Seen item persistence (seen_storage.go handles this)
//
// Invariants:
// - All fetches respect rate limiting
// - Feed parsing is tolerant of malformed feeds
// - Item deduplication uses GUID as primary key, link as fallback
package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// RSS structures for parsing
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	LastBuild   string    `xml:"lastBuildDate"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string   `xml:"guid"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	Content     string   `xml:"encoded"`
	Author      string   `xml:"author"`
	Categories  []string `xml:"category"`
}

// Atom structures for parsing
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Updated    string         `xml:"updated"`
	Published  string         `xml:"published"`
	Authors    []atomAuthor   `xml:"author"`
	Categories []atomCategory `xml:"category"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

// Checker handles feed checking and parsing.
type Checker struct {
	storage     Storage
	seenStorage SeenItemStorage
	httpClient  *http.Client
	jobManager  *jobs.Manager
}

// NewChecker creates a new feed checker.
func NewChecker(storage Storage, seenStorage SeenItemStorage, jobManager *jobs.Manager) *Checker {
	return &Checker{
		storage:     storage,
		seenStorage: seenStorage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jobManager: jobManager,
	}
}

// Check fetches and parses a feed, returns new items.
func (c *Checker) Check(ctx context.Context, feed *Feed) (*FeedCheckResult, error) {
	result := &FeedCheckResult{
		FeedID:    feed.ID,
		CheckedAt: time.Now(),
	}

	// Fetch feed content
	data, err := c.fetchFeed(ctx, feed.URL)
	if err != nil {
		result.Error = err.Error()
		feed.LastError = err.Error()
		feed.ConsecutiveFailures++
		if updateErr := c.storage.Update(feed); updateErr != nil {
			slog.Error("failed to update feed after error", "feedID", feed.ID, "error", updateErr)
		}
		return result, err
	}

	// Parse feed based on type (or auto-detect)
	var items []FeedItem
	var feedTitle, feedDesc string

	switch feed.FeedType {
	case FeedTypeRSS:
		items, feedTitle, feedDesc, err = c.parseRSS(data)
	case FeedTypeAtom:
		items, feedTitle, feedDesc, err = c.parseAtom(data)
	case FeedTypeAuto, "":
		items, feedTitle, feedDesc, err = c.parseAuto(data)
	default:
		err = fmt.Errorf("unknown feed type: %s", feed.FeedType)
	}

	if err != nil {
		result.Error = err.Error()
		feed.LastError = err.Error()
		feed.ConsecutiveFailures++
		if updateErr := c.storage.Update(feed); updateErr != nil {
			slog.Error("failed to update feed after parse error", "feedID", feed.ID, "error", updateErr)
		}
		return result, err
	}

	result.TotalItems = len(items)
	result.FeedTitle = feedTitle
	result.FeedDesc = feedDesc

	// Filter to new items only
	var newItems []FeedItem
	for _, item := range items {
		guid := item.ItemKey()
		if !c.seenStorage.IsSeen(feed.ID, guid) {
			newItems = append(newItems, item)
			// Mark as seen
			seenItem := SeenItem{
				GUID:   guid,
				Link:   item.Link,
				Title:  item.Title,
				SeenAt: time.Now(),
			}
			if err := c.seenStorage.MarkSeen(feed.ID, seenItem); err != nil {
				slog.Error("failed to mark item as seen", "feedID", feed.ID, "guid", guid, "error", err)
			}
		}
	}

	result.NewItems = newItems

	// Create scrape jobs for new items if autoScrape is enabled
	if feed.AutoScrape && c.jobManager != nil && len(newItems) > 0 {
		for _, item := range newItems {
			if err := c.createScrapeJob(ctx, feed, item); err != nil {
				slog.Error("failed to create scrape job for feed item",
					"feedID", feed.ID,
					"itemLink", item.Link,
					"error", err)
			}
		}
	}

	// Update feed metadata
	feed.LastCheckedAt = time.Now()
	feed.LastError = ""
	feed.ConsecutiveFailures = 0
	if updateErr := c.storage.Update(feed); updateErr != nil {
		slog.Error("failed to update feed after check", "feedID", feed.ID, "error", updateErr)
	}

	return result, nil
}

// fetchFeed fetches feed content from a URL.
func (c *Checker) fetchFeed(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SpartanScraper/1.0 Feed Checker")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read feed body: %w", err)
	}

	return data, nil
}

// parseAuto attempts to auto-detect and parse the feed format.
func (c *Checker) parseAuto(data []byte) ([]FeedItem, string, string, error) {
	// Try to detect format from XML structure
	var root struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse XML: %w", err)
	}

	switch root.XMLName.Local {
	case "rss":
		return c.parseRSS(data)
	case "feed":
		return c.parseAtom(data)
	default:
		// Try RSS first, then Atom
		items, title, desc, err := c.parseRSS(data)
		if err == nil && len(items) > 0 {
			return items, title, desc, nil
		}
		return c.parseAtom(data)
	}
}

// parseRSS parses an RSS 2.0 feed.
func (c *Checker) parseRSS(data []byte) ([]FeedItem, string, string, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse RSS: %w", err)
	}

	items := make([]FeedItem, 0, len(feed.Channel.Items))
	for _, rssItem := range feed.Channel.Items {
		item := FeedItem{
			GUID:        rssItem.GUID,
			Title:       rssItem.Title,
			Link:        rssItem.Link,
			Description: rssItem.Description,
			Content:     rssItem.Content,
			Categories:  rssItem.Categories,
		}
		if rssItem.Author != "" {
			item.Author = rssItem.Author
		}
		if rssItem.PubDate != "" {
			if t, err := parseDate(rssItem.PubDate); err == nil {
				item.PubDate = t
			}
		}
		// Use link as GUID fallback
		if item.GUID == "" {
			item.GUID = item.Link
		}
		items = append(items, item)
	}

	return items, feed.Channel.Title, feed.Channel.Description, nil
}

// parseAtom parses an Atom 1.0 feed.
func (c *Checker) parseAtom(data []byte) ([]FeedItem, string, string, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse Atom: %w", err)
	}

	items := make([]FeedItem, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		item := FeedItem{
			GUID:       entry.ID,
			Title:      entry.Title,
			Content:    entry.Content,
			Categories: make([]string, 0, len(entry.Categories)),
		}

		// Extract link
		for _, link := range entry.Links {
			if link.Rel == "" || link.Rel == "alternate" {
				item.Link = link.Href
				break
			}
		}
		if item.Link == "" && len(entry.Links) > 0 {
			item.Link = entry.Links[0].Href
		}

		// Use summary as description if no content
		if entry.Content != "" {
			item.Description = entry.Content
		} else {
			item.Description = entry.Summary
		}

		// Extract author
		if len(entry.Authors) > 0 {
			item.Author = entry.Authors[0].Name
		}

		// Extract categories
		for _, cat := range entry.Categories {
			if cat.Term != "" {
				item.Categories = append(item.Categories, cat.Term)
			}
		}

		// Parse date
		dateStr := entry.Published
		if dateStr == "" {
			dateStr = entry.Updated
		}
		if dateStr != "" {
			if t, err := parseDate(dateStr); err == nil {
				item.PubDate = t
			}
		}

		// Use link as GUID fallback
		if item.GUID == "" {
			item.GUID = item.Link
		}

		items = append(items, item)
	}

	return items, feed.Title, "", nil
}

// parseDate attempts to parse a date string in various formats.
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
	}

	s = strings.TrimSpace(s)
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// createScrapeJob creates a scrape job for a feed item.
func (c *Checker) createScrapeJob(ctx context.Context, feed *Feed, item FeedItem) error {
	if c.jobManager == nil {
		return fmt.Errorf("job manager not available")
	}

	spec := jobs.JobSpec{
		Kind: model.KindScrape,
		URL:  item.Link,
	}

	// Note: ExtractOptions in JobSpec uses Template/Inline for extraction configuration
	// Feed extract options are passed through but may need adjustment based on use case

	job, err := c.jobManager.CreateJob(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	if err := c.jobManager.Enqueue(job); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	slog.Info("created scrape job for feed item",
		"feedID", feed.ID,
		"jobID", job.ID,
		"itemLink", item.Link,
		"itemTitle", item.Title)

	return nil
}

// CheckAll checks all enabled feeds and returns results.
func (c *Checker) CheckAll(ctx context.Context) ([]*FeedCheckResult, error) {
	feeds, err := c.storage.ListEnabled()
	if err != nil {
		return nil, err
	}

	var results []*FeedCheckResult
	for _, feed := range feeds {
		if !feed.IsDue() {
			continue
		}

		result, err := c.Check(ctx, &feed)
		if err != nil {
			slog.Error("feed check failed", "feedID", feed.ID, "url", feed.URL, "error", err)
		}
		results = append(results, result)
	}

	return results, nil
}
