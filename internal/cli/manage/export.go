// Package manage contains export CLI command wiring.
//
// It does NOT define export formats; internal/exporter does that.
package manage

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func RunExport(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	jobID := fs.String("job-id", "", "Job id to export")
	format := fs.String("format", "jsonl", "Output format: jsonl|json|md|csv|xlsx|parquet|har|postgres|mysql|mongodb|s3|gcs|azure")
	out := fs.String("out", "", "Output file (defaults to stdout)")

	// Cloud export flags
	cloudProvider := fs.String("cloud-provider", "", "Cloud provider: s3|gcs|azure (for cloud export)")
	cloudBucket := fs.String("cloud-bucket", "", "Cloud storage bucket/container name")
	cloudPath := fs.String("cloud-path", "", "Path template with {job_id}, {timestamp}, {kind}, {format} variables (default: \"{kind}/{timestamp}.{format}\")")
	cloudRegion := fs.String("cloud-region", "", "Cloud region (S3 only, optional)")
	cloudStorageClass := fs.String("cloud-storage-class", "", "S3 storage class (optional: STANDARD, STANDARD_IA, GLACIER)")
	cloudFormat := fs.String("cloud-format", "jsonl", "Content format for cloud export: jsonl|json|md|csv|xlsx|parquet|har")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan export --job-id <id> [options]

Examples:
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan export --job-id <id> --format csv
  spartan export --job-id <id> --format s3 --cloud-provider s3 --cloud-bucket my-bucket
  spartan export --job-id <id> --format jsonl --cloud-provider gcs --cloud-bucket my-bucket --cloud-path "exports/{job_id}.jsonl"

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *jobID == "" {
		fmt.Fprintln(os.Stderr, "--job-id is required")
		return 1
	}

	// Build cloud config if cloud provider is specified
	var cloudCfg *exporter.CloudExportConfig
	if *cloudProvider != "" {
		cloudCfg = &exporter.CloudExportConfig{
			Provider:      *cloudProvider,
			Bucket:        *cloudBucket,
			Path:          *cloudPath,
			Region:        *cloudRegion,
			StorageClass:  *cloudStorageClass,
			ContentFormat: *cloudFormat,
		}
	}

	// Validate cloud config if needed
	if cloudCfg != nil || exporter.IsCloudFormat(*format) {
		cfg := cloudCfg
		if cfg == nil {
			cfg = &exporter.CloudExportConfig{}
		}
		if err := exporter.ValidateCloudConfig(*format, *cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	job, err := st.Get(ctx, *jobID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if job.ResultPath == "" {
		fmt.Fprintln(os.Stderr, "no result path for job")
		return 1
	}

	f, err := os.Open(job.ResultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer f.Close()

	var outWriter io.Writer
	if *out == "" {
		outWriter = os.Stdout
	} else {
		if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		outFile, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer outFile.Close()
		outWriter = outFile
	}

	// Use cloud export if configured or if format is a cloud format
	if cloudCfg != nil || exporter.IsCloudFormat(*format) {
		result, err := exporter.ExportStreamWithCloudResult(job, f, *format, outWriter, cloudCfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if result != nil {
			fmt.Println(result.String())
		}
		return 0
	}

	// Regular export
	if err := exporter.ExportStream(job, f, *format, outWriter); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *out != "" {
		fmt.Println(*out)
	}
	return 0
}
