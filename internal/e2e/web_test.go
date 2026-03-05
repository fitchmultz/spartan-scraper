// Package e2e provides end-to-end integration tests for the web frontend preview.
// Tests cover the built web assets being served correctly via Vite preview.
// Does NOT test CLI commands, API workflows, or authentication mechanisms.
package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWebPreview(t *testing.T) {
	dist := filepath.Join(projectRoot, "web", "dist", "index.html")
	if _, err := os.Stat(dist); err != nil {
		t.Skip("web dist missing; run make build")
	}

	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, cleanup := startProcess(ctx, t, []string{"BROWSER=none"}, filepath.Join(projectRoot, "web"), "pnpm", "exec", "vite", "preview", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	defer cleanup()

	client := &http.Client{Timeout: 2 * time.Second}
	waitForPreview(t, client, port)

	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatalf("fetch web preview: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("web preview status: %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Spartan Scraper") {
		t.Fatalf("web preview missing title")
	}
}

func waitForPreview(t *testing.T, client *http.Client, port int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	for i := 0; i < 50; i++ {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("web preview not reachable on port %d", port)
}
