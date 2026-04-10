// Package webhook serializes webhook payloads into JSON and multipart delivery requests.
//
// Purpose:
// - Build typed delivery requests from webhook payloads for the dispatcher worker pool.
//
// Responsibilities:
// - Define the deliveryRequest type consumed by the dispatcher queue.
// - Serialize JSON event payloads.
// - Serialize multipart export payloads (metadata + file content).
// - Validate export-specific payload requirements.
//
// Scope:
// - Payload serialization only; delivery mechanics live in dispatcher.go.
//
// Usage:
// - Call jsonDeliveryRequest or exportDeliveryRequest to build a deliveryRequest for the queue.
//
// Invariants/Assumptions:
// - Export deliveries require EventExportCompleted, a non-empty Filename, and a non-empty ContentType.
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// deliveryRequest is the internal queue item consumed by dispatcher workers.
type deliveryRequest struct {
	EventID     string
	EventType   EventType
	JobID       string
	ContentType string
	Body        []byte
	Headers     map[string]string
}

// Payload represents webhook event metadata.
//
// Non-export events are delivered as a JSON body containing this payload.
// Export-completed events are delivered as multipart/form-data with this payload
// serialized in the `metadata` part and the rendered export bytes in the `export` part.
type Payload struct {
	EventID     string     `json:"eventId"`
	EventType   EventType  `json:"eventType"`
	Timestamp   time.Time  `json:"timestamp"`
	JobID       string     `json:"jobId"`
	JobKind     string     `json:"jobKind"`
	Status      string     `json:"status"`
	PrevStatus  string     `json:"prevStatus,omitempty"`
	Error       string     `json:"error,omitempty"`
	ResultURL   string     `json:"resultUrl,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Content change fields (populated when EventType is EventContentChanged)
	URL          string `json:"url,omitempty"`
	PreviousHash string `json:"previousHash,omitempty"`
	CurrentHash  string `json:"currentHash,omitempty"`
	DiffText     string `json:"diffText,omitempty"`
	DiffHTML     string `json:"diffHtml,omitempty"`
	Selector     string `json:"selector,omitempty"`

	// Visual change fields (populated when EventType is EventVisualChanged)
	VisualHash         string  `json:"visualHash,omitempty"`
	PreviousVisualHash string  `json:"previousVisualHash,omitempty"`
	VisualSimilarity   float64 `json:"visualSimilarity,omitempty"`

	// Page crawled fields (populated when EventType is EventPageCrawled)
	PageURL     string `json:"pageUrl,omitempty"`
	PageStatus  int    `json:"pageStatus,omitempty"`
	PageTitle   string `json:"pageTitle,omitempty"`
	PageDepth   int    `json:"pageDepth,omitempty"`
	IsDuplicate bool   `json:"isDuplicate,omitempty"`
	DuplicateOf string `json:"duplicateOf,omitempty"`
	CrawlSeqNum int    `json:"crawlSeqNum,omitempty"`

	// Retry attempted fields (populated when EventType is EventRetryAttempted)
	AttemptNumber int    `json:"attemptNumber,omitempty"`
	MaxAttempts   int    `json:"maxAttempts,omitempty"`
	RetryError    string `json:"retryError,omitempty"`
	FetcherType   string `json:"fetcherType,omitempty"`

	// Export completed fields (populated when EventType is EventExportCompleted)
	ExportFormat string `json:"exportFormat,omitempty"`
	Filename     string `json:"filename,omitempty"`
	ContentType  string `json:"contentType,omitempty"`
	RecordCount  int    `json:"recordCount,omitempty"`
	ExportSize   int64  `json:"exportSize,omitempty"`
}

// jsonDeliveryRequest serializes a payload as a JSON delivery request.
func jsonDeliveryRequest(payload Payload) (deliveryRequest, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return deliveryRequest{}, fmt.Errorf("marshal payload: %w", err)
	}
	return deliveryRequest{
		EventID:     payload.EventID,
		EventType:   payload.EventType,
		JobID:       payload.JobID,
		ContentType: "application/json",
		Body:        body,
		Headers: map[string]string{
			"X-Spartan-Webhook-Payload-Type": "event-json",
			"X-Spartan-Webhook-Event-Type":   string(payload.EventType),
		},
	}, nil
}

// exportDeliveryRequest serializes a payload + content bytes as a multipart/form-data delivery request.
func exportDeliveryRequest(payload Payload, content []byte) (deliveryRequest, error) {
	if payload.EventType != EventExportCompleted {
		return deliveryRequest{}, apperrors.Validation("export webhook delivery requires eventType export.completed")
	}
	if payload.Filename == "" {
		return deliveryRequest{}, apperrors.Validation("export webhook delivery requires filename metadata")
	}
	if payload.ContentType == "" {
		return deliveryRequest{}, apperrors.Validation("export webhook delivery requires contentType metadata")
	}
	if payload.ExportSize == 0 {
		payload.ExportSize = int64(len(content))
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metadataHeaders := textproto.MIMEHeader{}
	metadataHeaders.Set("Content-Disposition", `form-data; name="metadata"`)
	metadataHeaders.Set("Content-Type", "application/json")
	metadataPart, err := writer.CreatePart(metadataHeaders)
	if err != nil {
		return deliveryRequest{}, fmt.Errorf("create metadata part: %w", err)
	}
	if err := json.NewEncoder(metadataPart).Encode(payload); err != nil {
		return deliveryRequest{}, fmt.Errorf("encode metadata part: %w", err)
	}

	exportHeaders := textproto.MIMEHeader{}
	exportHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="export"; filename=%q`, payload.Filename))
	exportHeaders.Set("Content-Type", payload.ContentType)
	exportPart, err := writer.CreatePart(exportHeaders)
	if err != nil {
		return deliveryRequest{}, fmt.Errorf("create export part: %w", err)
	}
	if _, err := exportPart.Write(content); err != nil {
		return deliveryRequest{}, fmt.Errorf("write export part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return deliveryRequest{}, fmt.Errorf("close multipart writer: %w", err)
	}

	return deliveryRequest{
		EventID:     payload.EventID,
		EventType:   payload.EventType,
		JobID:       payload.JobID,
		ContentType: writer.FormDataContentType(),
		Body:        body.Bytes(),
		Headers: map[string]string{
			"X-Spartan-Webhook-Payload-Type": "export-multipart",
			"X-Spartan-Webhook-Event-Type":   string(payload.EventType),
		},
	}, nil
}
