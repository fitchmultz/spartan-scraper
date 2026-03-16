// Package webhook exposes operator-facing inspection views for persisted webhook deliveries.
//
// Purpose:
// - Provide one shared, sanitized delivery-history representation for API, CLI, and MCP surfaces.
//
// Responsibilities:
// - Convert internal DeliveryRecord values into stable operator-facing payloads.
// - Redact secrets from delivery URLs and error text before they leave the process.
// - Format timestamps consistently for user-facing inspection responses.
//
// Scope:
// - Delivery inspection payload shaping only; persistence and dispatch remain in other files.
//
// Usage:
// - Call ToInspectableDelivery or ToInspectableDeliveries before returning delivery records to operators.
//
// Invariants/Assumptions:
// - URLs and error text are sanitized on every conversion.
// - Timestamps use RFC3339 strings.
// - Nil delivery records convert to zero values instead of panicking.
package webhook

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// InspectableDelivery is the stable, sanitized operator-facing delivery shape.
type InspectableDelivery struct {
	ID           string         `json:"id"`
	EventID      string         `json:"eventId"`
	EventType    EventType      `json:"eventType"`
	JobID        string         `json:"jobId"`
	URL          string         `json:"url"`
	Status       DeliveryStatus `json:"status"`
	Attempts     int            `json:"attempts"`
	LastError    string         `json:"lastError,omitempty"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
	DeliveredAt  string         `json:"deliveredAt,omitempty"`
	ResponseCode int            `json:"responseCode,omitempty"`
}

// ToInspectableDelivery converts a persisted delivery record into its shared
// operator-facing representation.
func ToInspectableDelivery(record *DeliveryRecord) InspectableDelivery {
	if record == nil {
		return InspectableDelivery{}
	}

	delivery := InspectableDelivery{
		ID:           record.ID,
		EventID:      record.EventID,
		EventType:    record.EventType,
		JobID:        record.JobID,
		URL:          record.URL,
		Status:       record.Status,
		Attempts:     record.Attempts,
		LastError:    record.LastError,
		CreatedAt:    formatInspectableTime(record.CreatedAt),
		UpdatedAt:    formatInspectableTime(record.UpdatedAt),
		ResponseCode: record.ResponseCode,
	}
	if record.DeliveredAt != nil {
		delivery.DeliveredAt = formatInspectableTime(*record.DeliveredAt)
	}
	return SanitizeInspectableDelivery(delivery)
}

// ToInspectableDeliveries converts a list of persisted delivery records into
// shared operator-facing payloads.
func ToInspectableDeliveries(records []*DeliveryRecord) []InspectableDelivery {
	if len(records) == 0 {
		return []InspectableDelivery{}
	}
	result := make([]InspectableDelivery, 0, len(records))
	for _, record := range records {
		result = append(result, ToInspectableDelivery(record))
	}
	return result
}

// SanitizeInspectableDelivery redacts sensitive fields on an already-shaped
// operator-facing delivery payload.
func SanitizeInspectableDelivery(delivery InspectableDelivery) InspectableDelivery {
	delivery.URL = apperrors.SanitizeURL(delivery.URL)
	if delivery.LastError != "" {
		delivery.LastError = apperrors.RedactString(delivery.LastError)
	}
	return delivery
}

func formatInspectableTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339)
}
