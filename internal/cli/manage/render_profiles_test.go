// Package manage provides tests for the render profiles management CLI subcommand.
// Tests cover listing render profiles from the data store.
// Does NOT test add/remove/update operations or profile application.
package manage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRunRenderProfiles_List(t *testing.T) {
	tmp := t.TempDir()
	jsonContent := `{
		"profiles": [
			{"name": "p1", "hostPatterns": ["p1.com"]},
			{"name": "p2", "hostPatterns": ["p2.com"]}
		]
	}`
	path := filepath.Join(tmp, "render_profiles.json")
	if err := os.WriteFile(path, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{DataDir: tmp}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := RunRenderProfiles(context.Background(), cfg, []string{"list"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output, "p1") {
		t.Errorf("expected output to contain p1, got %q", output)
	}
	if !strings.Contains(output, "p2") {
		t.Errorf("expected output to contain p2, got %q", output)
	}
}
