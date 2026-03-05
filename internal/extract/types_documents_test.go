// Package extract provides tests for document type structs (FieldValue, Extracted, NormalizedDocument, ValidationResult).
// Tests cover struct creation, field access, and JSON serialization.
// Does NOT test extraction logic or validation behavior.
package extract

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFieldValueStruct(t *testing.T) {
	t.Run("single value", func(t *testing.T) {
		fv := FieldValue{
			Values: []string{"test"},
			Source: FieldSourceSelector,
		}

		if len(fv.Values) != 1 {
			t.Errorf("expected 1 value, got %d", len(fv.Values))
		}
		if fv.Values[0] != "test" {
			t.Errorf("expected 'test', got %q", fv.Values[0])
		}
	})

	t.Run("multiple values", func(t *testing.T) {
		fv := FieldValue{
			Values: []string{"val1", "val2", "val3"},
			Source: FieldSourceJSONLD,
		}

		if len(fv.Values) != 3 {
			t.Errorf("expected 3 values, got %d", len(fv.Values))
		}
	})

	t.Run("with RawObject", func(t *testing.T) {
		fv := FieldValue{
			Values:    []string{"test"},
			Source:    FieldSourceRegex,
			RawObject: `{"key":"value"}`,
		}

		if fv.RawObject != `{"key":"value"}` {
			t.Errorf("unexpected RawObject: %q", fv.RawObject)
		}
	})

	t.Run("field source constants", func(t *testing.T) {
		sources := []FieldSource{
			FieldSourceSelector,
			FieldSourceJSONLD,
			FieldSourceRegex,
			FieldSourceDerived,
		}

		for _, source := range sources {
			fv := FieldValue{Source: source}
			if fv.Source != source {
				t.Errorf("source not preserved: expected %v, got %v", source, fv.Source)
			}
		}
	})
}

func TestExtractedStruct(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test Title",
			Text:        "Test text content",
			Links:       []string{"/link1", "/link2"},
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		if ext.URL != "http://example.com" {
			t.Errorf("URL not set correctly")
		}
		if ext.Title != "Test Title" {
			t.Errorf("Title not set correctly")
		}
		if ext.Text != "Test text content" {
			t.Errorf("Text not set correctly")
		}
		if len(ext.Links) != 2 {
			t.Errorf("expected 2 links, got %d", len(ext.Links))
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		metadata := map[string]string{
			"author": "John Doe",
			"date":   "2024-01-01",
		}
		fields := map[string]FieldValue{
			"headline": {Values: []string{"Headline"}, Source: FieldSourceSelector},
		}
		jsonld := []map[string]any{
			{"@type": "Article", "headline": "Headline"},
		}

		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test Title",
			Text:        "Test text",
			Links:       []string{},
			Metadata:    metadata,
			Fields:      fields,
			JSONLD:      jsonld,
			Template:    "article",
			ExtractedAt: time.Now(),
		}

		if ext.Metadata == nil {
			t.Error("Metadata should be set")
		}
		if ext.Metadata["author"] != "John Doe" {
			t.Errorf("Metadata author not set correctly")
		}
		if ext.Fields == nil {
			t.Error("Fields should be set")
		}
		if len(ext.JSONLD) != 1 {
			t.Errorf("expected 1 JSON-LD object, got %d", len(ext.JSONLD))
		}
	})

	t.Run("with Raw field", func(t *testing.T) {
		raw := map[string][]string{
			"title":   {"Page Title"},
			"content": {"Content here"},
		}

		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test",
			Text:        "Text",
			Links:       []string{},
			Raw:         raw,
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		if ext.Raw == nil {
			t.Error("Raw should be set")
		}
		if len(ext.Raw) != 2 {
			t.Errorf("expected 2 Raw entries, got %d", len(ext.Raw))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test",
			Text:        "Text",
			Links:       []string{"/link"},
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		data, err := json.Marshal(ext)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled Extracted
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.URL != ext.URL {
			t.Error("URL not preserved")
		}
		if unmarshaled.Title != ext.Title {
			t.Error("Title not preserved")
		}
	})
}

func TestNormalizedDocumentStruct(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test Title",
			Description: "Test description",
			Text:        "Test text",
			Links:       []string{"/link1"},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		if norm.URL != "http://example.com" {
			t.Errorf("URL not set correctly")
		}
		if norm.Title != "Test Title" {
			t.Errorf("Title not set correctly")
		}
		if norm.Description != "Test description" {
			t.Errorf("Description not set correctly")
		}
		if norm.Text != "Test text" {
			t.Errorf("Text not set correctly")
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		metadata := map[string]string{
			"author": "Jane Doe",
		}
		fields := map[string]FieldValue{
			"category": {Values: []string{"Tech"}, Source: FieldSourceSelector},
		}
		jsonld := []map[string]any{
			{"@type": "Article", "headline": "Headline"},
		}

		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Metadata:    metadata,
			Fields:      fields,
			JSONLD:      jsonld,
			Template:    "article",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		if norm.Metadata == nil {
			t.Error("Metadata should be set")
		}
		if norm.Metadata["author"] != "Jane Doe" {
			t.Errorf("Metadata author incorrect")
		}
		if len(norm.JSONLD) != 1 {
			t.Errorf("expected 1 JSON-LD object")
		}
	})

	t.Run("with invalid validation", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation: ValidationResult{
				Valid:  false,
				Errors: []string{"title is required"},
			},
		}

		if norm.Validation.Valid {
			t.Error("expected Valid to be false")
		}
		if len(norm.Validation.Errors) != 1 {
			t.Errorf("expected 1 validation error, got %d", len(norm.Validation.Errors))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		data, err := json.Marshal(norm)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled NormalizedDocument
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Title != norm.Title {
			t.Error("Title not preserved")
		}
		if unmarshaled.Description != norm.Description {
			t.Error("Description not preserved")
		}
	})
}

func TestValidationResultStruct(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  true,
			Errors: []string{},
		}

		if !vr.Valid {
			t.Error("expected Valid to be true")
		}
		if vr.Errors == nil {
			t.Error("expected Errors to be initialized")
		}
		if len(vr.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d", len(vr.Errors))
		}
	})

	t.Run("invalid result", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  false,
			Errors: []string{"title is required", "description too short"},
		}

		if vr.Valid {
			t.Error("expected Valid to be false")
		}
		if len(vr.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(vr.Errors))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  false,
			Errors: []string{"error 1", "error 2"},
		}

		data, err := json.Marshal(vr)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ValidationResult
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Valid != vr.Valid {
			t.Error("Valid not preserved")
		}
		if len(unmarshaled.Errors) != len(vr.Errors) {
			t.Error("Errors count not preserved")
		}
	})
}
