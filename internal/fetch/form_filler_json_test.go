// Package fetch provides tests for form filler JSON output.
//
// Purpose:
//   - Keep machine-readable form detection responses stable for zero-result cases.
//
// Responsibilities:
//   - Verify empty form detection results serialize arrays as [] instead of null.
//
// Scope:
//   - JSON marshaling of FormDetectResponse only.
//
// Usage:
//   - Run with `go test ./internal/fetch`.
//
// Invariants/Assumptions:
//   - Clients should not need special-case null handling when formCount is zero.
package fetch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormDetectResponseMarshalJSON_NormalizesEmptySlices(t *testing.T) {
	payload, err := json.Marshal(FormDetectResponse{
		URL:       "https://example.com",
		FormCount: 0,
	})
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	jsonText := string(payload)
	for _, expected := range []string{
		`"forms":[]`,
		`"detectedTypes":[]`,
		`"formCount":0`,
	} {
		if !strings.Contains(jsonText, expected) {
			t.Fatalf("marshal output missing %s: %s", expected, jsonText)
		}
	}
	if strings.Contains(jsonText, `"forms":null`) {
		t.Fatalf("marshal output should not use null forms: %s", jsonText)
	}
}
