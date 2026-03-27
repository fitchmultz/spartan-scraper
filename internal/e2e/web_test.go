// Package e2e provides end-to-end integration tests for the web frontend preview.
//
// Purpose:
//   - Verify the built web preview serves the operator shell and key route layouts correctly.
//
// Responsibilities:
//   - Start isolated Vite preview instances against built assets.
//   - Start local backend dependencies when browser-visible route checks need API data.
//   - Assert the preview responds and critical route affordances stay reachable.
//
// Scope:
//   - Web preview coverage only; CLI, API workflow, and auth behavior live in other e2e tests.
//
// Usage:
//   - Runs as part of go test ./internal/e2e/...
//
// Invariants/Assumptions:
//   - Uses the already-built web dist output.
//   - Uses only local backend services and headless browser checks.
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

	"github.com/chromedp/chromedp"
)

type domRect struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type newJobLayoutSnapshot struct {
	ViewportHeight float64  `json:"viewportHeight"`
	SidebarMode    string   `json:"sidebarMode"`
	QuickStart     *domRect `json:"quickStart"`
	Stepper        *domRect `json:"stepper"`
	Actions        *domRect `json:"actions"`
}

func TestWebPreview(t *testing.T) {
	dist := filepath.Join(projectRoot, "web", "dist", "index.html")
	if _, err := os.Stat(dist); err != nil {
		t.Skip("web dist missing; run make build")
	}

	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, cleanup := startWebPreview(ctx, t, []string{"BROWSER=none"}, port)
	defer cleanup()

	client := &http.Client{Timeout: 2 * time.Second}

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

func TestNewJobCompactHiddenAssistantKeepsWorkflowVisible(t *testing.T) {
	dist := filepath.Join(projectRoot, "web", "dist", "index.html")
	if _, err := os.Stat(dist); err != nil {
		t.Skip("web dist missing; run make build")
	}

	backendPort := freePort(t)
	previewPort := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backendEnv := append(baseEnv(t.TempDir()), "PORT="+strconv.Itoa(backendPort))
	_, backendCleanup := startProcess(ctx, t, backendEnv, t.TempDir(), spartanPath, "server")
	defer backendCleanup()

	client := &http.Client{Timeout: 2 * time.Second}
	waitForHealth(t, client, backendPort)

	previewEnv := []string{
		"BROWSER=none",
		"DEV_API_PROXY_TARGET=http://127.0.0.1:" + strconv.Itoa(backendPort),
	}
	_, previewCleanup := startWebPreview(ctx, t, previewEnv, previewPort)
	defer previewCleanup()

	browserCtx, browserCancel := newHeadlessBrowserContext(t)
	defer browserCancel()

	var snapshot newJobLayoutSnapshot
	previewURL := fmt.Sprintf("http://127.0.0.1:%d", previewPort)
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(previewURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`localStorage.setItem("spartan.ai-assistant.open", "false")`, nil),
		chromedp.Navigate(previewURL+"/jobs/new"),
		chromedp.WaitVisible(".job-quickstart", chromedp.ByQuery),
		chromedp.WaitVisible(".job-wizard__stepper", chromedp.ByQuery),
		chromedp.WaitVisible(".job-wizard__actions", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`(() => {
			const rect = (selector) => {
				const element = document.querySelector(selector);
				if (!element) return null;
				const bounds = element.getBoundingClientRect();
				return {
					top: bounds.top,
					bottom: bounds.bottom,
					left: bounds.left,
					right: bounds.right,
					width: bounds.width,
					height: bounds.height,
				};
			};
			const workspace = document.querySelector(".job-wizard__workspace");
			return {
				viewportHeight: window.innerHeight,
				sidebarMode: workspace?.dataset.sidebarMode ?? "",
				quickStart: rect(".job-quickstart"),
				stepper: rect(".job-wizard__stepper"),
				actions: rect(".job-wizard__actions"),
			};
		})()`, &snapshot),
	); err != nil {
		t.Fatalf("capture compact new-job layout: %v", err)
	}

	if snapshot.SidebarMode != "none" {
		t.Fatalf("expected compact hidden-assistant route to inline presets, got sidebar mode %q", snapshot.SidebarMode)
	}
	if snapshot.QuickStart == nil || snapshot.Stepper == nil || snapshot.Actions == nil {
		t.Fatalf("missing layout anchors: %#v", snapshot)
	}
	if snapshot.QuickStart.Bottom > snapshot.ViewportHeight {
		t.Fatalf("quick start falls below the viewport: bottom=%.1f viewport=%.1f", snapshot.QuickStart.Bottom, snapshot.ViewportHeight)
	}
	if snapshot.Stepper.Top >= snapshot.ViewportHeight {
		t.Fatalf("wizard stepper starts below the viewport: top=%.1f viewport=%.1f", snapshot.Stepper.Top, snapshot.ViewportHeight)
	}
	if snapshot.Actions.Top >= snapshot.ViewportHeight {
		t.Fatalf("sticky wizard actions start below the viewport: top=%.1f viewport=%.1f", snapshot.Actions.Top, snapshot.ViewportHeight)
	}
}
