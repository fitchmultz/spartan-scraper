// Package exporter provides JSONL export implementation.
//
// JSONL export passes through JSON objects one per line without transformation.
// Functions include:
// - exportJSONLStream: Stream JSONL from reader to writer
//
// This file does NOT handle other formats (JSON, Markdown, CSV).
package exporter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// exportJSONLStream exports job results to JSONL format with streaming.
// JSONL format passes through JSON objects one per line.
func exportJSONLStream(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}
