// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file contains tests for the feed parser and checker.
package feed

import (
	"encoding/xml"
	"testing"
	"time"
)

func TestParseRSS(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>A test RSS feed</description>
    <link>https://example.com</link>
    <lastBuildDate>Mon, 01 Jan 2024 00:00:00 GMT</lastBuildDate>
    <item>
      <guid>item-1</guid>
      <title>First Item</title>
      <link>https://example.com/item1</link>
      <description>Description of first item</description>
      <pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
      <author>John Doe</author>
      <category>Tech</category>
      <category>News</category>
    </item>
    <item>
      <guid>item-2</guid>
      <title>Second Item</title>
      <link>https://example.com/item2</link>
      <description>Description of second item</description>
      <pubDate>Tue, 02 Jan 2024 00:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	items, title, desc, err := checker.parseRSS([]byte(rssXML))
	if err != nil {
		t.Fatalf("parseRSS() error = %v", err)
	}

	if title != "Test Feed" {
		t.Errorf("title = %q, want %q", title, "Test Feed")
	}

	if desc != "A test RSS feed" {
		t.Errorf("description = %q, want %q", desc, "A test RSS feed")
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Check first item
	item1 := items[0]
	if item1.GUID != "item-1" {
		t.Errorf("item1.GUID = %q, want %q", item1.GUID, "item-1")
	}
	if item1.Title != "First Item" {
		t.Errorf("item1.Title = %q, want %q", item1.Title, "First Item")
	}
	if item1.Link != "https://example.com/item1" {
		t.Errorf("item1.Link = %q, want %q", item1.Link, "https://example.com/item1")
	}
	if item1.Author != "John Doe" {
		t.Errorf("item1.Author = %q, want %q", item1.Author, "John Doe")
	}
	if len(item1.Categories) != 2 {
		t.Errorf("len(item1.Categories) = %d, want 2", len(item1.Categories))
	}
	if item1.PubDate.IsZero() {
		t.Error("item1.PubDate should not be zero")
	}
}

func TestParseRSS_NoGUID(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Item Without GUID</title>
      <link>https://example.com/item</link>
    </item>
  </channel>
</rss>`

	items, _, _, err := checker.parseRSS([]byte(rssXML))
	if err != nil {
		t.Fatalf("parseRSS() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	// GUID should fall back to link
	if items[0].GUID != "https://example.com/item" {
		t.Errorf("GUID = %q, want %q (link fallback)", items[0].GUID, "https://example.com/item")
	}
}

func TestParseAtom(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test Feed</title>
  <id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>
  <updated>2024-01-01T00:00:00Z</updated>
  <entry>
    <id>entry-1</id>
    <title>First Entry</title>
    <link href="https://example.com/entry1"/>
    <summary>Summary of first entry</summary>
    <published>2024-01-01T00:00:00Z</published>
    <updated>2024-01-01T12:00:00Z</updated>
    <author>
      <name>Jane Doe</name>
    </author>
    <category term="Technology"/>
  </entry>
  <entry>
    <id>entry-2</id>
    <title>Second Entry</title>
    <link href="https://example.com/entry2"/>
    <content>Full content of second entry</content>
    <published>2024-01-02T00:00:00Z</published>
  </entry>
</feed>`

	items, title, desc, err := checker.parseAtom([]byte(atomXML))
	if err != nil {
		t.Fatalf("parseAtom() error = %v", err)
	}

	if title != "Atom Test Feed" {
		t.Errorf("title = %q, want %q", title, "Atom Test Feed")
	}

	if desc != "" {
		t.Errorf("description = %q, want empty", desc)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Check first entry
	item1 := items[0]
	if item1.GUID != "entry-1" {
		t.Errorf("item1.GUID = %q, want %q", item1.GUID, "entry-1")
	}
	if item1.Title != "First Entry" {
		t.Errorf("item1.Title = %q, want %q", item1.Title, "First Entry")
	}
	if item1.Link != "https://example.com/entry1" {
		t.Errorf("item1.Link = %q, want %q", item1.Link, "https://example.com/entry1")
	}
	if item1.Author != "Jane Doe" {
		t.Errorf("item1.Author = %q, want %q", item1.Author, "Jane Doe")
	}
	if len(item1.Categories) != 1 || item1.Categories[0] != "Technology" {
		t.Errorf("item1.Categories = %v, want [Technology]", item1.Categories)
	}

	// Check second entry - should use content as description
	item2 := items[1]
	if item2.Description != "Full content of second entry" {
		t.Errorf("item2.Description = %q, want %q", item2.Description, "Full content of second entry")
	}
}

func TestParseAtom_AlternateLink(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test</title>
  <entry>
    <id>entry-1</id>
    <title>Entry</title>
    <link rel="alternate" href="https://example.com/entry"/>
  </entry>
</feed>`

	items, _, _, err := checker.parseAtom([]byte(atomXML))
	if err != nil {
		t.Fatalf("parseAtom() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	if items[0].Link != "https://example.com/entry" {
		t.Errorf("Link = %q, want %q", items[0].Link, "https://example.com/entry")
	}
}

func TestParseAuto_RSS(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>RSS Feed</title>
    <item>
      <title>Item</title>
      <link>https://example.com/item</link>
    </item>
  </channel>
</rss>`

	items, title, _, err := checker.parseAuto([]byte(rssXML))
	if err != nil {
		t.Fatalf("parseAuto() error = %v", err)
	}

	if title != "RSS Feed" {
		t.Errorf("title = %q, want %q", title, "RSS Feed")
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestParseAuto_Atom(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Feed</title>
  <id>test-id</id>
  <updated>2024-01-01T00:00:00Z</updated>
  <entry>
    <id>entry-1</id>
    <title>Entry</title>
    <link href="https://example.com/entry"/>
  </entry>
</feed>`

	items, title, _, err := checker.parseAuto([]byte(atomXML))
	if err != nil {
		t.Fatalf("parseAuto() error = %v", err)
	}

	if title != "Atom Feed" {
		t.Errorf("title = %q, want %q", title, "Atom Feed")
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestParseAuto_InvalidXML(t *testing.T) {
	checker := NewChecker(nil, nil, nil)

	invalidXML := `not valid xml`

	_, _, _, err := checker.parseAuto([]byte(invalidXML))
	if err == nil {
		t.Error("parseAuto() expected error for invalid XML, got nil")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{
			name:     "RFC3339",
			input:    "2024-01-01T00:00:00Z",
			wantZero: false,
		},
		{
			name:     "RFC1123",
			input:    "Mon, 01 Jan 2024 00:00:00 GMT",
			wantZero: false,
		},
		{
			name:     "RFC1123Z",
			input:    "Mon, 01 Jan 2024 00:00:00 +0000",
			wantZero: false,
		},
		{
			name:     "RSS date format",
			input:    "01 Jan 2024 00:00:00 GMT",
			wantZero: false,
		},
		{
			name:     "invalid date",
			input:    "not a date",
			wantZero: true,
		},
		{
			name:     "empty date",
			input:    "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("parseDate() = %v, want zero time", got)
				}
				if err == nil {
					t.Error("parseDate() expected error for invalid date")
				}
			} else {
				if got.IsZero() {
					t.Errorf("parseDate() = zero time, want non-zero")
				}
			}
		})
	}
}

func TestParseDate_TrimWhitespace(t *testing.T) {
	// Test that whitespace is trimmed
	date, err := parseDate("  2024-01-01T00:00:00Z  ")
	if err != nil {
		t.Errorf("parseDate() with whitespace error = %v", err)
	}
	if date.IsZero() {
		t.Error("parseDate() with whitespace returned zero time")
	}
}

// Mock implementations for testing

type mockStorage struct {
	feeds []Feed
	err   error
}

func (m *mockStorage) Add(feed *Feed) (*Feed, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.feeds = append(m.feeds, *feed)
	return feed, nil
}

func (m *mockStorage) Update(feed *Feed) error {
	if m.err != nil {
		return m.err
	}
	for i, f := range m.feeds {
		if f.ID == feed.ID {
			m.feeds[i] = *feed
			return nil
		}
	}
	return &NotFoundError{ID: feed.ID}
}

func (m *mockStorage) Delete(id string) error {
	if m.err != nil {
		return m.err
	}
	var filtered []Feed
	for _, f := range m.feeds {
		if f.ID != id {
			filtered = append(filtered, f)
		}
	}
	m.feeds = filtered
	return nil
}

func (m *mockStorage) Get(id string) (*Feed, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, f := range m.feeds {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, &NotFoundError{ID: id}
}

func (m *mockStorage) List() ([]Feed, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.feeds, nil
}

func (m *mockStorage) ListEnabled() ([]Feed, error) {
	if m.err != nil {
		return nil, m.err
	}
	var enabled []Feed
	for _, f := range m.feeds {
		if f.Enabled {
			enabled = append(enabled, f)
		}
	}
	return enabled, nil
}

type mockSeenStorage struct {
	items map[string]map[string]SeenItem
	err   error
}

func newMockSeenStorage() *mockSeenStorage {
	return &mockSeenStorage{
		items: make(map[string]map[string]SeenItem),
	}
}

func (m *mockSeenStorage) IsSeen(feedID, guid string) bool {
	if m.items[feedID] == nil {
		return false
	}
	_, seen := m.items[feedID][guid]
	return seen
}

func (m *mockSeenStorage) MarkSeen(feedID string, item SeenItem) error {
	if m.err != nil {
		return m.err
	}
	if m.items[feedID] == nil {
		m.items[feedID] = make(map[string]SeenItem)
	}
	m.items[feedID][item.GUID] = item
	return nil
}

func (m *mockSeenStorage) GetSeen(feedID string) ([]SeenItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []SeenItem
	for _, item := range m.items[feedID] {
		result = append(result, item)
	}
	return result, nil
}

func (m *mockSeenStorage) Cleanup(feedID string, before time.Time) error {
	return m.err
}

func (m *mockSeenStorage) CleanupAll(before time.Time) error {
	return m.err
}

func TestChecker_Check_WithMockStorage(t *testing.T) {
	// This test would require a mock HTTP server to test the full Check method
	// For now, we test the parsing logic which is the core functionality
}

func TestRSSStructures(t *testing.T) {
	// Test that RSS structures marshal/unmarshal correctly
	rss := rssFeed{
		Channel: rssChannel{
			Title:       "Test",
			Description: "Desc",
			Link:        "https://example.com",
			Items: []rssItem{
				{
					GUID:    "guid1",
					Title:   "Title",
					Link:    "https://example.com/item",
					PubDate: "Mon, 01 Jan 2024 00:00:00 GMT",
				},
			},
		},
	}

	data, err := xml.Marshal(rss)
	if err != nil {
		t.Fatalf("xml.Marshal() error = %v", err)
	}

	var parsed rssFeed
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("xml.Unmarshal() error = %v", err)
	}

	if parsed.Channel.Title != "Test" {
		t.Errorf("Title = %q, want %q", parsed.Channel.Title, "Test")
	}
}

func TestAtomStructures(t *testing.T) {
	// Test that Atom structures marshal/unmarshal correctly
	atom := atomFeed{
		Title: "Test Feed",
		ID:    "urn:test",
		Entries: []atomEntry{
			{
				ID:      "entry1",
				Title:   "Entry Title",
				Content: "Content",
				Links: []atomLink{
					{Href: "https://example.com/entry", Rel: "alternate"},
				},
				Authors: []atomAuthor{
					{Name: "Author"},
				},
				Categories: []atomCategory{
					{Term: "Category"},
				},
			},
		},
	}

	data, err := xml.Marshal(atom)
	if err != nil {
		t.Fatalf("xml.Marshal() error = %v", err)
	}

	var parsed atomFeed
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("xml.Unmarshal() error = %v", err)
	}

	if parsed.Title != "Test Feed" {
		t.Errorf("Title = %q, want %q", parsed.Title, "Test Feed")
	}
}
