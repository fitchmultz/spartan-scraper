// Package common contains tests for general utility functions.
//
// Responsibilities:
// - Testing SplitCSV for comma-separated value parsing
// - Validating whitespace trimming and empty value handling
//
// Non-goals:
// - Testing CSV format compliance with RFC 4180
// - Testing integration with external CSV parsers
//
// Assumptions:
// - Tests are isolated and use only in-memory data structures
// - No environment variables required
package common

import "testing"

func TestSplitCSV_EmptyString(t *testing.T) {
	got := SplitCSV("")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestSplitCSV_SingleValue(t *testing.T) {
	got := SplitCSV("value1")

	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0] != "value1" {
		t.Errorf("expected 'value1', got %q", got[0])
	}
}

func TestSplitCSV_MultipleValues(t *testing.T) {
	got := SplitCSV("value1,value2,value3")

	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}

	expected := []string{"value1", "value2", "value3"}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("expected %q at index %d, got %q", v, i, got[i])
		}
	}
}

func TestSplitCSV_WithSpaces(t *testing.T) {
	got := SplitCSV(" value1 , value2 , value3 ")

	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}

	if got[0] != "value1" || got[1] != "value2" || got[2] != "value3" {
		t.Errorf("expected ['value1', 'value2', 'value3'], got %v", got)
	}
}

func TestSplitCSV_EmptyValuesSkipped(t *testing.T) {
	got := SplitCSV("value1,,value2, ,value3")

	if len(got) != 3 {
		t.Fatalf("expected 3 non-empty items, got %d", len(got))
	}

	expected := []string{"value1", "value2", "value3"}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("expected %q at index %d, got %q", v, i, got[i])
		}
	}
}

func TestSplitCSV_AllEmpty(t *testing.T) {
	tests := []string{
		",,",
		" , , ",
		", ,",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := SplitCSV(input)
			if len(got) != 0 {
				t.Errorf("expected empty slice for %q, got %v", input, got)
			}
		})
	}
}
