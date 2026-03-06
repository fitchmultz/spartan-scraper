// Package api provides HTTP handlers for template A/B testing endpoints.
// This file verifies that template A/B responses match the OpenAPI contract
// rather than leaking raw store record fields.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func createTestABRecord(t *testing.T, srv *Server) *store.TemplateABTestRecord {
	t.Helper()

	now := time.Now().UTC()
	record := &store.TemplateABTestRecord{
		ID:                  "ab-test-1",
		Name:                "Template QA",
		Description:         "Verify API response shape",
		BaselineTemplate:    "article",
		VariantTemplate:     "default",
		AllocationJSON:      `{"baseline":50,"variant":50}`,
		StartTime:           now,
		Status:              "pending",
		SuccessCriteriaJSON: `{"metric":"success_rate","min_improvement":0.05,"min_field_coverage":0.8}`,
		MinSampleSize:       100,
		ConfidenceLevel:     0.95,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := srv.store.CreateABTest(t.Context(), record); err != nil {
		t.Fatalf("create AB test: %v", err)
	}

	return record
}

func TestHandleListABTestsUsesOpenAPIFieldNames(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	record := createTestABRecord(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/template-ab-tests", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	tests, ok := payload["tests"].([]any)
	if !ok || len(tests) != 1 {
		t.Fatalf("expected one test, got %#v", payload["tests"])
	}

	item, ok := tests[0].(map[string]any)
	if !ok {
		t.Fatalf("expected test object, got %#v", tests[0])
	}

	if got := item["id"]; got != record.ID {
		t.Fatalf("expected id %q, got %#v", record.ID, got)
	}
	if got := item["name"]; got != record.Name {
		t.Fatalf("expected name %q, got %#v", record.Name, got)
	}
	if got := item["baseline_template"]; got != record.BaselineTemplate {
		t.Fatalf("expected baseline_template %q, got %#v", record.BaselineTemplate, got)
	}
	if _, exists := item["ID"]; exists {
		t.Fatalf("did not expect raw store field ID in response: %#v", item)
	}
	if _, exists := item["BaselineTemplate"]; exists {
		t.Fatalf("did not expect raw store field BaselineTemplate in response: %#v", item)
	}
}

func TestHandleGetABTestUsesOpenAPIFieldNames(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	record := createTestABRecord(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/template-ab-tests/"+record.ID, nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got := payload["id"]; got != record.ID {
		t.Fatalf("expected id %q, got %#v", record.ID, got)
	}
	if got := payload["variant_template"]; got != record.VariantTemplate {
		t.Fatalf("expected variant_template %q, got %#v", record.VariantTemplate, got)
	}
	if _, exists := payload["ID"]; exists {
		t.Fatalf("did not expect raw store field ID in response: %#v", payload)
	}
}
