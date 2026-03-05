// Package exporter provides cloud storage export implementations.
//
// This file implements export to AWS S3, Google Cloud Storage (GCS), and Azure Blob Storage.
// It uses gocloud.dev/blob for a unified API across all providers.
//
// Credential handling (automatic via gocloud.dev):
// - AWS S3: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, IAM roles, instance profiles
// - GCS: GOOGLE_APPLICATION_CREDENTIALS, Application Default Credentials, Workload Identity
// - Azure: AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY, AZURE_STORAGE_SAS_TOKEN, Managed Identity
//
// This file does NOT handle:
// - Custom credential configuration (relies on standard credential chains)
// - Server-side encryption configuration (uses provider defaults)
// - Cross-region replication or lifecycle policies
// - Pre-signed URL generation
package exporter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// exportToCloud handles the actual cloud upload.
func exportToCloud(ctx context.Context, job model.Job, r io.Reader, format string, cfg CloudExportConfig) error {
	// Validate provider
	provider := cfg.Provider
	if provider == "" {
		return apperrors.Validation("cloud provider is required (s3, gcs, azure)")
	}

	// Validate bucket
	if cfg.Bucket == "" {
		return apperrors.Validation("cloud bucket is required")
	}

	// Build bucket URL
	bucketURL, err := resolveBucketURL(cfg)
	if err != nil {
		return err
	}

	// Open bucket
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to open cloud bucket", err)
	}
	defer bucket.Close()

	// Render path template
	path := RenderPathTemplate(cfg.Path, job, format)

	// Determine content type
	contentType := cfg.ContentType
	if contentType == "" {
		contentType = contentTypeForFormat(format)
	}

	// Create writer options
	var opts *blob.WriterOptions
	if contentType != "" {
		opts = &blob.WriterOptions{
			ContentType: contentType,
		}
	}

	// Add S3 storage class if specified
	if provider == "s3" && cfg.StorageClass != "" && opts != nil {
		opts.Metadata = map[string]string{
			"x-amz-storage-class": cfg.StorageClass,
		}
	}

	// Create writer
	writer, err := bucket.NewWriter(ctx, path, opts)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create cloud writer", err)
	}

	// First, we need to convert the input to the desired format
	// We'll buffer it to a pipe to stream the formatted output
	pr, pw := io.Pipe()

	// Start format conversion in a goroutine
	formatErr := make(chan error, 1)
	go func() {
		defer pw.Close()
		formatErr <- ExportStream(job, r, format, pw)
	}()

	// Copy formatted data to cloud
	_, err = io.Copy(writer, pr)
	if err != nil {
		writer.Close()
		return apperrors.Wrap(apperrors.KindInternal, "failed to write to cloud storage", err)
	}

	// Wait for format conversion to complete
	if err := <-formatErr; err != nil {
		writer.Close()
		return apperrors.Wrap(apperrors.KindInternal, "failed to format data for cloud export", err)
	}

	// Close writer to finalize upload
	if err := writer.Close(); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to finalize cloud upload", err)
	}

	return nil
}

// resolveBucketURL constructs the gocloud.dev bucket URL from config.
func resolveBucketURL(cfg CloudExportConfig) (string, error) {
	switch cfg.Provider {
	case "s3":
		// S3 URL format: s3://bucket?region=us-east-1
		url := fmt.Sprintf("s3://%s", cfg.Bucket)
		if cfg.Region != "" {
			url = fmt.Sprintf("%s?region=%s", url, cfg.Region)
		}
		return url, nil
	case "gcs":
		// GCS URL format: gs://bucket
		return fmt.Sprintf("gs://%s", cfg.Bucket), nil
	case "azure":
		// Azure URL format: azblob://container
		return fmt.Sprintf("azblob://%s", cfg.Bucket), nil
	default:
		return "", apperrors.Validation(fmt.Sprintf("unsupported cloud provider: %s", cfg.Provider))
	}
}

// contentTypeForFormat returns the appropriate MIME type for the format.
func contentTypeForFormat(format string) string {
	switch format {
	case "json", "jsonl":
		return "application/json"
	case "md":
		return "text/markdown"
	case "csv":
		return "text/csv"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "parquet":
		return "application/octet-stream"
	case "har":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// ExportToCloud exports job results directly to cloud storage.
// This is a convenience function that wraps ExportStreamWithCloud for cloud targets.
func ExportToCloud(job model.Job, r io.Reader, format string, cfg CloudExportConfig) error {
	return exportToCloud(context.Background(), job, r, format, cfg)
}

// IsCloudFormat returns true if the format is a cloud export format.
func IsCloudFormat(format string) bool {
	switch format {
	case "s3", "gcs", "azure":
		return true
	}
	return false
}

// ValidateCloudConfig validates the cloud export configuration.
func ValidateCloudConfig(format string, cfg CloudExportConfig) error {
	if !IsCloudFormat(format) && cfg.Provider == "" {
		return nil // Not a cloud format, no validation needed
	}

	// If provider is set, validate it regardless of format
	if cfg.Provider != "" {
		switch cfg.Provider {
		case "s3", "gcs", "azure":
			// valid
		default:
			return apperrors.Validation(fmt.Sprintf("unsupported cloud provider: %s (must be s3, gcs, or azure)", cfg.Provider))
		}

		if cfg.Bucket == "" {
			return apperrors.Validation("cloud bucket is required")
		}

		// Validate content format if specified
		if cfg.ContentFormat != "" {
			validFormats := map[string]bool{
				"jsonl": true, "json": true, "md": true, "csv": true,
				"xlsx": true, "parquet": true, "har": true,
			}
			if !validFormats[cfg.ContentFormat] {
				return apperrors.Validation(fmt.Sprintf("invalid content format: %s", cfg.ContentFormat))
			}
		}
	}

	return nil
}

// CloudExportResult holds information about a completed cloud export.
type CloudExportResult struct {
	Provider    string    `json:"provider"`
	Bucket      string    `json:"bucket"`
	Path        string    `json:"path"`
	Format      string    `json:"format"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

// ExportStreamWithCloudResult exports job results to cloud storage and returns result metadata.
func ExportStreamWithCloudResult(job model.Job, r io.Reader, format string, w io.Writer, cloudCfg *CloudExportConfig) (*CloudExportResult, error) {
	if cloudCfg == nil {
		return nil, apperrors.Validation("cloud config is required")
	}

	// Validate provider
	if cloudCfg.Provider == "" {
		return nil, apperrors.Validation("cloud provider is required (s3, gcs, azure)")
	}

	if cloudCfg.Bucket == "" {
		return nil, apperrors.Validation("cloud bucket is required")
	}

	// Determine actual format for cloud-native formats
	actualFormat := format
	if IsCloudFormat(format) {
		actualFormat = cloudCfg.ContentFormat
		if actualFormat == "" {
			actualFormat = "jsonl"
		}
	}

	// Build bucket URL
	bucketURL, err := resolveBucketURL(*cloudCfg)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Open bucket
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to open cloud bucket", err)
	}
	defer bucket.Close()

	// Render path template
	path := RenderPathTemplate(cloudCfg.Path, job, actualFormat)

	// Determine content type
	contentType := cloudCfg.ContentType
	if contentType == "" {
		contentType = contentTypeForFormat(actualFormat)
	}

	// Create a counting reader to track size
	var size int64
	countingReader := &countingReader{r: r, count: &size}

	// Create writer options
	var opts *blob.WriterOptions
	if contentType != "" {
		opts = &blob.WriterOptions{
			ContentType: contentType,
		}
	}

	// Add S3 storage class if specified
	if cloudCfg.Provider == "s3" && cloudCfg.StorageClass != "" && opts != nil {
		if opts.Metadata == nil {
			opts.Metadata = make(map[string]string)
		}
		opts.Metadata["x-amz-storage-class"] = cloudCfg.StorageClass
	}

	// Create writer
	writer, err := bucket.NewWriter(ctx, path, opts)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create cloud writer", err)
	}

	// Stream formatted data to cloud
	if err := ExportStream(job, countingReader, actualFormat, writer); err != nil {
		writer.Close()
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to format and upload data", err)
	}

	// Close writer to finalize upload
	if err := writer.Close(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to finalize cloud upload", err)
	}

	return &CloudExportResult{
		Provider:    cloudCfg.Provider,
		Bucket:      cloudCfg.Bucket,
		Path:        path,
		Format:      actualFormat,
		ContentType: contentType,
		Size:        *countingReader.count,
		UploadedAt:  time.Now(),
	}, nil
}

// countingReader wraps an io.Reader to count bytes read.
type countingReader struct {
	r     io.Reader
	count *int64
}

func (cr *countingReader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	*cr.count += int64(n)
	return n, err
}

// ListCloudProviders returns the list of supported cloud providers.
func ListCloudProviders() []string {
	return []string{"s3", "gcs", "azure"}
}

// CloudProviderDisplayName returns a human-readable name for a provider.
func CloudProviderDisplayName(provider string) string {
	switch provider {
	case "s3":
		return "AWS S3"
	case "gcs":
		return "Google Cloud Storage"
	case "azure":
		return "Azure Blob Storage"
	default:
		return provider
	}
}

// IsValidCloudProvider returns true if the provider is supported.
func IsValidCloudProvider(provider string) bool {
	switch provider {
	case "s3", "gcs", "azure":
		return true
	}
	return false
}

// ExtractCloudConfigFromParams extracts cloud config from job params or query parameters.
// This is useful for API handlers that receive cloud export parameters.
func ExtractCloudConfigFromParams(params map[string]interface{}) *CloudExportConfig {
	cfg := &CloudExportConfig{}

	if v, ok := params["cloudProvider"].(string); ok {
		cfg.Provider = v
	}
	if v, ok := params["cloudBucket"].(string); ok {
		cfg.Bucket = v
	}
	if v, ok := params["cloudPath"].(string); ok {
		cfg.Path = v
	}
	if v, ok := params["cloudRegion"].(string); ok {
		cfg.Region = v
	}
	if v, ok := params["cloudStorageClass"].(string); ok {
		cfg.StorageClass = v
	}
	if v, ok := params["cloudFormat"].(string); ok {
		cfg.ContentFormat = v
	}
	if v, ok := params["cloudContentType"].(string); ok {
		cfg.ContentType = v
	}

	// Return nil if no provider specified
	if cfg.Provider == "" {
		return nil
	}

	return cfg
}

// ToCloudURL returns a URL-like string representation of the cloud location.
func (r *CloudExportResult) ToCloudURL() string {
	switch r.Provider {
	case "s3":
		return fmt.Sprintf("s3://%s/%s", r.Bucket, r.Path)
	case "gcs":
		return fmt.Sprintf("gs://%s/%s", r.Bucket, r.Path)
	case "azure":
		return fmt.Sprintf("azblob://%s/%s", r.Bucket, r.Path)
	default:
		return fmt.Sprintf("%s://%s/%s", r.Provider, r.Bucket, r.Path)
	}
}

// String returns a human-readable representation of the cloud export result.
func (r *CloudExportResult) String() string {
	return fmt.Sprintf("%s (%s, %d bytes)", r.ToCloudURL(), r.ContentType, r.Size)
}

// CloudExportError represents an error during cloud export with additional context.
type CloudExportError struct {
	Provider string
	Bucket   string
	Path     string
	Op       string
	Err      error
}

func (e *CloudExportError) Error() string {
	return fmt.Sprintf("cloud export error [%s/%s]: %s failed: %v", e.Provider, e.Bucket, e.Op, e.Err)
}

func (e *CloudExportError) Unwrap() error {
	return e.Err
}

// NormalizePath ensures the path doesn't start with a leading slash for cloud storage.
func NormalizePath(path string) string {
	return strings.TrimLeft(path, "/")
}
