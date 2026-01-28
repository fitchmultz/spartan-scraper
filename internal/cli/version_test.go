// Package cli_test contains tests for the CLI package.
package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/stretchr/testify/assert"
)

func TestRunVersion(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set test values
	buildinfo.Version = "1.2.3"
	buildinfo.Commit = "abcdef"
	buildinfo.Date = "2026-01-28T00:00:00Z"

	err := RunVersion()
	assert.NoError(t, err)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.True(t, strings.Contains(output, "Spartan version: 1.2.3"))
	assert.True(t, strings.Contains(output, "Commit:          abcdef"))
	assert.True(t, strings.Contains(output, "Build date:      2026-01-28T00:00:00Z"))
	assert.True(t, strings.Contains(output, "Go version:"))
	assert.True(t, strings.Contains(output, "OS/Arch:"))
}
