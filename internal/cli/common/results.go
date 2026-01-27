// Package common contains CLI helpers for waiting on jobs and rendering results.
//
// It does NOT define export formats or storage layout; internal/exporter and internal/store do.
package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"spartan-scraper/internal/apperrors"
	"spartan-scraper/internal/exporter"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/store"
)

func HandleJobResult(ctx context.Context, st *store.Store, job model.Job, wait bool, waitTimeout time.Duration, out string) int {
	if !wait && out == "" {
		payload, _ := json.MarshalIndent(job, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	if err := waitForJob(ctx, st, job.ID, waitTimeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if out != "" {
		if err := copyResults(ctx, st, job.ID, out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(job.ID)
		return 0
	}

	if err := printResults(ctx, st, job.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func waitForJob(ctx context.Context, st *store.Store, id string, timeout time.Duration) error {
	start := time.Now()
	for {
		if timeout > 0 && time.Since(start) > timeout {
			return apperrors.Internal(fmt.Sprintf("wait timeout after %s", timeout))
		}
		job, err := st.Get(ctx, id)
		if err != nil {
			return err
		}
		switch job.Status {
		case model.StatusSucceeded:
			return nil
		case model.StatusFailed:
			if job.Error != "" {
				return apperrors.Internal(fmt.Sprintf("job failed: %s", job.Error))
			}
			return apperrors.Internal("job failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func copyResults(ctx context.Context, st *store.Store, id, outPath string) error {
	job, err := st.Get(ctx, id)
	if err != nil {
		return err
	}
	if job.ResultPath == "" {
		return apperrors.NotFound("no result path for job")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(job.ResultPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func printResults(ctx context.Context, st *store.Store, id string) error {
	job, err := st.Get(ctx, id)
	if err != nil {
		return err
	}
	if job.ResultPath == "" {
		return apperrors.NotFound("no result path for job")
	}

	f, err := os.Open(job.ResultPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Always jsonl raw output for printResults.
	return exporter.ExportStream(job, f, "jsonl", os.Stdout)
}
