package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

	cmd := exec.CommandContext(ctx, "pnpm", "exec", "vite", "preview", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = filepath.Join(projectRoot, "web")
	cmd.Env = append(os.Environ(), "BROWSER=none")
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start web preview: %v", err)
	}
	defer func() {
		cancel()

		waitDone := make(chan error, 1)
		go func() {
			waitDone <- cmd.Wait()
		}()

		select {
		case <-waitDone:
			return
		case <-time.After(3 * time.Second):
			if cmd.Process == nil {
				return
			}
			if runtime.GOOS != "windows" {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
		}

		select {
		case <-waitDone:
			return
		case <-time.After(3 * time.Second):
			return
		}
	}()

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
