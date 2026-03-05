// Package api provides integration tests for the transform validation endpoint.
// Tests cover JMESPath and JSONata expression validation, including valid/invalid
// expressions, language validation, and missing parameter handling.
// Does NOT test the preview-transform endpoint or actual transformations.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleValidateTransform_ValidJMESPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: "{title: title, url: url}",
		Language:   "jmespath",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	var resp TransformValidateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Valid {
		t.Error("expected expression to be valid")
	}

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}

	if resp.Message != "Expression is valid" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestHandleValidateTransform_ValidJSONata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: `{"item": name, "total": price * quantity}`,
		Language:   "jsonata",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	var resp TransformValidateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Valid {
		t.Error("expected expression to be valid")
	}

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleValidateTransform_InvalidJMESPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: "{title: ", // Invalid syntax
		Language:   "jmespath",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp TransformValidateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Valid {
		t.Error("expected expression to be invalid")
	}

	if resp.Error == "" {
		t.Error("expected error message for invalid expression")
	}

	if resp.Message != "Invalid expression" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestHandleValidateTransform_InvalidJSONata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: "($invalid:", // Invalid syntax
		Language:   "jsonata",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp TransformValidateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Valid {
		t.Error("expected expression to be invalid")
	}

	if resp.Error == "" {
		t.Error("expected error message for invalid expression")
	}
}

func TestHandleValidateTransform_InvalidLanguage(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: "{title: title}",
		Language:   "invalid-lang",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %v: %s", status, rr.Body.String())
	}
}

func TestHandleValidateTransform_MissingExpression(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformValidateRequest{
		Expression: "",
		Language:   "jmespath",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/transform/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %v: %s", status, rr.Body.String())
	}
}
