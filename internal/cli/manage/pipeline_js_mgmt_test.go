// Package manage provides tests for the pipeline JavaScript management CLI subcommand.
// Tests cover listing pipeline JS scripts from the data store.
// Does NOT test add/remove/update operations or script execution.
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

func TestRunPipelineJS_List(t *testing.T) {
	tmp := t.TempDir()
	jsonContent := `{
		"scripts": [
			{"name": "s1", "hostPatterns": ["s1.com"]},
			{"name": "s2", "hostPatterns": ["s2.com"]}
		]
	}`
	path := filepath.Join(tmp, "pipeline_js.json")
	if err := os.WriteFile(path, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{DataDir: tmp}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := RunPipelineJS(context.Background(), cfg, []string{"list"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output, "s1") {
		t.Errorf("expected output to contain s1, got %q", output)
	}
	if !strings.Contains(output, "s2") {
		t.Errorf("expected output to contain s2, got %q", output)
	}
}
