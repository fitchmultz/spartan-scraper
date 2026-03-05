// Package common contains tests for CLI flag parsing utilities.
//
// Responsibilities:
// - Testing StringSliceFlag parsing and conversion to maps
// - Validating key:value pair extraction and edge cases
//
// Non-goals:
// - Testing actual CLI flag parsing from command line
// - Testing integration with flag package
//
// Assumptions:
// - Tests are isolated and use only in-memory data structures
// - No environment variables required
package common

import "testing"

func TestStringSliceFlag_ToMap_Empty(t *testing.T) {
	var f StringSliceFlag
	got := f.ToMap()
	if got != nil {
		t.Errorf("expected nil for empty flag, got %v", got)
	}
}

func TestStringSliceFlag_ToMap_ValidPairs(t *testing.T) {
	f := StringSliceFlag{"Authorization: Bearer token", "Content-Type: application/json"}
	got := f.ToMap()

	expected := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(got))
	}

	for k, v := range expected {
		if got[k] != v {
			t.Errorf("expected %q = %q, got %q", k, v, got[k])
		}
	}
}

func TestStringSliceFlag_ToMap_InvalidEntries(t *testing.T) {
	f := StringSliceFlag{
		"Valid-Key: valid-value",
		"NoColonHere",
		":MissingKey",
		"EmptyKey :",
		"  SpacedKey  :  SpacedValue  ",
		"",
	}

	got := f.ToMap()

	expected := map[string]string{
		"Valid-Key": "valid-value",
		"SpacedKey": "SpacedValue",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(got))
	}

	for k, v := range expected {
		if got[k] != v {
			t.Errorf("expected %q = %q, got %q", k, v, got[k])
		}
	}
}

func TestStringSliceFlag_ToMap_MultipleColons(t *testing.T) {
	f := StringSliceFlag{"Key: Value: With: Colons"}
	got := f.ToMap()

	expected := map[string]string{
		"Key": "Value: With: Colons",
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}

	if got["Key"] != expected["Key"] {
		t.Errorf("expected %q, got %q", expected["Key"], got["Key"])
	}
}

func TestStringSliceFlag_Set(t *testing.T) {
	var f StringSliceFlag

	if err := f.Set("first"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f) != 1 {
		t.Errorf("expected 1 item, got %d", len(f))
	}

	if err := f.Set("second"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f) != 2 {
		t.Errorf("expected 2 items, got %d", len(f))
	}

	if f[0] != "first" || f[1] != "second" {
		t.Errorf("expected [first second], got %v", f)
	}
}

func TestStringSliceFlag_String(t *testing.T) {
	f := StringSliceFlag{"first", "second", "third"}
	got := f.String()
	expected := "first,second,third"

	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
