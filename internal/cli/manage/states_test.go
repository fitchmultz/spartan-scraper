// Package manage provides tests for the crawl states management CLI subcommand.
// Tests cover listing, deleting, and clearing crawl states from persistent storage.
// Does NOT test concurrent access or integration with the crawler.
package manage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestRunCrawlStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spartan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.Config{
		DataDir: tempDir,
	}

	st, err := store.Open(tempDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	ctx := context.Background()
	state1 := model.CrawlState{
		URL:         "https://example.com/1",
		LastScraped: time.Now(),
	}
	state2 := model.CrawlState{
		URL:         "https://example.com/2",
		LastScraped: time.Now(),
	}
	_ = st.UpsertCrawlState(ctx, state1)
	_ = st.UpsertCrawlState(ctx, state2)
	st.Close()

	// Test list
	exitCode := RunCrawlStates(ctx, cfg, []string{"list"})
	assert.Equal(t, 0, exitCode)

	// Test delete
	exitCode = RunCrawlStates(ctx, cfg, []string{"delete", "--url", "https://example.com/1"})
	assert.Equal(t, 0, exitCode)

	st, _ = store.Open(tempDir)
	states, _ := st.ListCrawlStates(ctx, store.ListCrawlStatesOptions{})
	assert.Equal(t, 1, len(states))
	assert.Equal(t, "https://example.com/2", states[0].URL)
	st.Close()

	// Test clear
	exitCode = RunCrawlStates(ctx, cfg, []string{"clear", "--force"})
	assert.Equal(t, 0, exitCode)

	st, _ = store.Open(tempDir)
	states, _ = st.ListCrawlStates(ctx, store.ListCrawlStatesOptions{})
	assert.Equal(t, 0, len(states))
	st.Close()

	// Test unknown subcommand
	exitCode = RunCrawlStates(ctx, cfg, []string{"unknown"})
	assert.Equal(t, 1, exitCode)

	// Test help
	exitCode = RunCrawlStates(ctx, cfg, []string{"help"})
	assert.Equal(t, 0, exitCode)
}
